// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelmongo // import "go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/v2/mongo/otelmongo"

import (
	"context"
	"errors"
	"net"
	"strconv"
	"sync"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/event"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.39.0"
	"go.opentelemetry.io/otel/semconv/v1.39.0/dbconv"
	"go.opentelemetry.io/otel/trace"
)

type spanKey struct {
	ConnectionID string
	RequestID    int64
}

type monitor struct {
	ClientOperationDuration *dbconv.ClientOperationDuration

	sync.Mutex
	spans map[spanKey]trace.Span
	cfg   config
}

func (m *monitor) Started(ctx context.Context, evt *event.CommandStartedEvent) {
	hostname, port := peerInfo(evt.ConnectionID)

	attrs := []attribute.KeyValue{
		semconv.DBSystemNameMongoDB,
		semconv.DBOperationName(evt.CommandName),
		semconv.DBNamespace(evt.DatabaseName),
		semconv.NetworkPeerAddress(hostname),
		semconv.NetworkPeerPort(port),
		semconv.NetworkTransportTCP,
	}
	if !m.cfg.CommandAttributeDisabled {
		attrs = append(attrs, semconv.DBQueryText(sanitizeCommand(evt.Command)))
	}

	collection, err := extractCollection(evt)
	if err == nil && collection != "" {
		attrs = append(attrs, semconv.DBCollectionName(collection))
	}

	spanName := m.cfg.SpanNameFormatter(evt)

	opts := []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attrs...),
	}
	_, span := m.cfg.Tracer.Start(ctx, spanName, opts...)
	key := spanKey{
		ConnectionID: evt.ConnectionID,
		RequestID:    evt.RequestID,
	}
	m.Lock()
	m.spans[key] = span
	m.Unlock()
}

func (m *monitor) Succeeded(ctx context.Context, evt *event.CommandSucceededEvent) {
	m.Finished(&evt.CommandFinishedEvent, nil)
	if m.ClientOperationDuration == nil {
		return
	}

	hostname, port := peerInfo(evt.ConnectionID)
	attrs := attribute.NewSet(
		semconv.DBSystemNameMongoDB,
		// No need to add semconv.DBSystemMongoDB, it will be added by metrics recorder.
		semconv.DBOperationName(evt.CommandName),
		semconv.DBNamespace(evt.DatabaseName),
		semconv.NetworkPeerAddress(hostname),
		semconv.NetworkPeerPort(port),
		semconv.NetworkTransportTCP,
		// `db.response.status_code` is excluded for succeeded events.
		// Succeeded processes an [go.mongodb.org/mongo-driver/v2/event.CommandSucceededEvent] for OTel,
		// including collecting metrics. The status code metric is excluded since MongoDB server indicates
		// a successful operation with {ok: 1}, which doesn't map to a traditional status code.
	)
	// TODO: db.query.text attribute is currently disabled by default.
	// Because event does not provide the query text directly.
	// command := m.extractCommand(evt)
	// attrs = append(attrs, semconv.DBQueryText(sanitizeCommand(evt.Command)))

	m.ClientOperationDuration.RecordSet(
		ctx,
		evt.Duration.Seconds(),
		attrs,
	)
}

func (m *monitor) Failed(ctx context.Context, evt *event.CommandFailedEvent) {
	m.Finished(&evt.CommandFinishedEvent, evt.Failure)
	if m.ClientOperationDuration == nil {
		return
	}

	hostname, port := peerInfo(evt.ConnectionID)
	attrs := attribute.NewSet(
		semconv.DBSystemNameMongoDB,
		semconv.DBOperationName(evt.CommandName),
		semconv.NetworkPeerAddress(hostname),
		semconv.NetworkPeerPort(port),
		semconv.NetworkTransportTCP,
		// TODO: The status code should not be static, but reflect server behavior.
		// Assert the error as [go.mongodb.org/mongo-driver/v2/x/mongo/driver.Error] and pull the code from there.
		// ref. https://jira.mongodb.org/browse/GODRIVER-3690
		semconv.ErrorType(evt.Failure),
	)
	// TODO: db.query.text attribute is currently disabled by default.
	// Because event does not provide the query text directly.
	// command := m.extractCommand(evt)
	// attrs = append(attrs, semconv.DBQueryText(sanitizeCommand(evt.Command)))

	m.ClientOperationDuration.RecordSet(
		ctx,
		evt.Duration.Seconds(),
		attrs,
	)
}

func (m *monitor) Finished(evt *event.CommandFinishedEvent, err error) {
	key := spanKey{
		ConnectionID: evt.ConnectionID,
		RequestID:    evt.RequestID,
	}
	m.Lock()
	span, ok := m.spans[key]
	if ok {
		delete(m.spans, key)
	}
	m.Unlock()
	if !ok {
		return
	}

	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}

	span.End()
}

// TODO sanitize values where possible, then re-enable `db.statement` span attributes default.
// TODO limit maximum size.
func sanitizeCommand(command bson.Raw) string {
	b, _ := bson.MarshalExtJSON(command, false, false)
	return string(b)
}

// extractCollection extracts the collection for the given mongodb command event.
// For CRUD operations, this is the first key/value string pair in the bson
// document where key == "<operation>" (e.g. key == "insert").
// For database meta-level operations, such a key may not exist.
// This function returns the collection name or an error if no collection can be determined.
func extractCollection(evt *event.CommandStartedEvent) (string, error) {
	elt, err := evt.Command.IndexErr(0)
	if err != nil {
		return "", err
	}
	if key, err := elt.KeyErr(); err == nil && key == evt.CommandName {
		var v bson.RawValue
		if v, err = elt.ValueErr(); err != nil || v.Type != bson.TypeString {
			return "", err
		}
		return v.StringValue(), nil
	}
	return "", errors.New("collection name not found")
}

// NewMonitor creates a new mongodb event CommandMonitor.
func NewMonitor(opts ...Option) *event.CommandMonitor {
	cfg := newConfig(opts...)
	var clientOperationDuration *dbconv.ClientOperationDuration
	operationDuration, err := dbconv.NewClientOperationDuration(
		cfg.Meter,
		metric.WithExplicitBucketBoundaries(0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5, 10),
	)
	if err != nil {
		clientOperationDuration = nil
		otel.Handle(err)
	} else {
		clientOperationDuration = &operationDuration
	}

	m := &monitor{
		spans: make(map[spanKey]trace.Span),
		cfg:   cfg,

		ClientOperationDuration: clientOperationDuration,
	}
	return &event.CommandMonitor{
		Started:   m.Started,
		Succeeded: m.Succeeded,
		Failed:    m.Failed,
	}
}

// peerInfo will parse the hostname and port from the mongo connection ID.
func peerInfo(connectionID string) (hostname string, port int) {
	defaultMongoPort := 27017
	hostname, portStr, err := net.SplitHostPort(connectionID)
	if err != nil {
		// If parsing fails, assume default MongoDB port and return the entire ConnectionID as hostname
		hostname = connectionID
		port = defaultMongoPort
		return hostname, port
	}

	port, err = strconv.Atoi(portStr)
	if err != nil || port < 1 {
		// If port parsing fails, fallback to default MongoDB port
		port = defaultMongoPort
	}

	return hostname, port
}
