// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package otelmongo // import "go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/event"
)

type requestKey struct {
	ConnectionID string
	RequestID    int64
}

type request struct {
	span  trace.Span
	attrs []attribute.KeyValue
}

type monitor struct {
	sync.Mutex
	requests  map[requestKey]request
	cfg       config
	durations metric.Float64Histogram
}

func (m *monitor) Started(ctx context.Context, evt *event.CommandStartedEvent) {
	var spanName string

	hostname, port := peerInfo(evt)

	attrs := make([]attribute.KeyValue, 0, 8)
	attrs = append(attrs,
		semconv.DBSystemMongoDB,
		semconv.DBOperation(evt.CommandName),
		semconv.DBName(evt.DatabaseName),
		semconv.NetPeerName(hostname),
		semconv.NetPeerPort(port),
		semconv.NetTransportTCP,
	)
	if collection, err := extractCollection(evt); err == nil && collection != "" {
		spanName = collection + "."
		attrs = append(attrs, semconv.DBMongoDBCollection(collection))
	}
	spanAttrs := attrs
	if !m.cfg.CommandAttributeDisabled {
		spanAttrs = append(spanAttrs, semconv.DBStatement(sanitizeCommand(evt.Command)))
	}
	spanName += evt.CommandName
	opts := []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(spanAttrs...),
	}
	_, span := m.cfg.Tracer.Start(ctx, spanName, opts...)
	key := requestKey{
		ConnectionID: evt.ConnectionID,
		RequestID:    evt.RequestID,
	}
	m.Lock()
	m.requests[key] = request{span, attrs}
	m.Unlock()
}

func (m *monitor) Succeeded(ctx context.Context, evt *event.CommandSucceededEvent) {
	m.Finished(ctx, &evt.CommandFinishedEvent, nil)
}

func (m *monitor) Failed(ctx context.Context, evt *event.CommandFailedEvent) {
	m.Finished(ctx, &evt.CommandFinishedEvent, fmt.Errorf("%s", evt.Failure))
}

func (m *monitor) Finished(ctx context.Context, evt *event.CommandFinishedEvent, err error) {
	key := requestKey{
		ConnectionID: evt.ConnectionID,
		RequestID:    evt.RequestID,
	}
	m.Lock()
	request, ok := m.requests[key]
	if ok {
		delete(m.requests, key)
	}
	m.Unlock()
	if !ok {
		return
	}

	if err != nil {
		request.span.SetStatus(codes.Error, err.Error())
		request.attrs = append(request.attrs, semconv.OtelStatusCodeError)
	} else {
		request.attrs = append(request.attrs, semconv.OtelStatusCodeOk)
	}
	m.durations.Record(ctx, evt.Duration.Seconds()/1000, metric.WithAttributeSet(attribute.NewSet(request.attrs...)))

	request.span.End()
}

// TODO sanitize values where possible, then reenable `db.statement` span attributes default.
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
	return "", fmt.Errorf("collection name not found")
}

// NewMonitor creates a new mongodb event CommandMonitor.
func NewMonitor(opts ...Option) *event.CommandMonitor {
	cfg := newConfig(opts...)
	histogram, err := cfg.Meter.Float64Histogram("command.duration", metric.WithUnit("ms"), metric.WithDescription("Duration of finished commands"))
	if err != nil {
		otel.Handle(err)
	}
	m := &monitor{
		requests:  make(map[requestKey]request),
		cfg:       cfg,
		durations: histogram,
	}
	return &event.CommandMonitor{
		Started:   m.Started,
		Succeeded: m.Succeeded,
		Failed:    m.Failed,
	}
}

func peerInfo(evt *event.CommandStartedEvent) (hostname string, port int) {
	hostname = evt.ConnectionID
	port = 27017
	if idx := strings.IndexByte(hostname, '['); idx >= 0 {
		hostname = hostname[:idx]
	}
	if idx := strings.IndexByte(hostname, ':'); idx >= 0 {
		port = func(p int, e error) int { return p }(strconv.Atoi(hostname[idx+1:]))
		hostname = hostname[:idx]
	}
	return hostname, port
}
