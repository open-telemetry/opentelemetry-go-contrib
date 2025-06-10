// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelmongo // import "go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/v2/mongo/otelmongo"

import (
	"context"
	"errors"
	"net"
	"strconv"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.32.0"
	"go.opentelemetry.io/otel/trace"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/event"
)

type spanKey struct {
	ConnectionID string
	RequestID    int64
}

type monitor struct {
	sync.Mutex
	spans map[spanKey]trace.Span
	cfg   config
}

func (m *monitor) Started(ctx context.Context, evt *event.CommandStartedEvent) {
	var spanName string

	hostname, port := peerInfo(evt)

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
	if collection, err := extractCollection(evt); err == nil && collection != "" {
		spanName = collection + "."
		attrs = append(attrs, semconv.DBCollectionName(collection))
	}
	spanName += evt.CommandName
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
}

func (m *monitor) Failed(ctx context.Context, evt *event.CommandFailedEvent) {
	m.Finished(&evt.CommandFinishedEvent, evt.Failure)
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
	m := &monitor{
		spans: make(map[spanKey]trace.Span),
		cfg:   cfg,
	}
	return &event.CommandMonitor{
		Started:   m.Started,
		Succeeded: m.Succeeded,
		Failed:    m.Failed,
	}
}

// peerInfo will parse the hostname and port from the mongo connection ID.
func peerInfo(evt *event.CommandStartedEvent) (hostname string, port int) {
	defaultMongoPort := 27017
	hostname, portStr, err := net.SplitHostPort(evt.ConnectionID)
	if err != nil {
		// If parsing fails, assume default MongoDB port and return the entire ConnectionID as hostname
		hostname = evt.ConnectionID
		port = defaultMongoPort
		return
	}

	port, err = strconv.Atoi(portStr)
	if err != nil || port < 1 {
		// If port parsing fails, fallback to default MongoDB port
		port = defaultMongoPort
	}

	return hostname, port
}
