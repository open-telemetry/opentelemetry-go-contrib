// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelmongo // import "go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/event"

	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo/internal/semconv"
)

type spanKey struct {
	ConnectionID string
	RequestID    int64
}

type monitor struct {
	sync.Mutex
	spans   map[spanKey]trace.Span
	cfg     config
	semconv semconv.EventMonitor
}

func (m *monitor) Started(ctx context.Context, evt *event.CommandStartedEvent) {
	attrOptions := []semconv.AttributeOption{
		semconv.WithCommandAttributeDisabled(m.cfg.CommandAttributeDisabled),
	}

	var spanName string
	if collection, err := extractCollection(evt); err == nil && collection != "" {
		spanName = collection + "."

		attrOptions = append(attrOptions, semconv.WithCollectionName(collection))
	}

	spanName += evt.CommandName
	opts := []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(m.semconv.CommandStartedTraceAttrs(evt, attrOptions...)...),
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
	m.Finished(&evt.CommandFinishedEvent, fmt.Errorf("%s", evt.Failure))
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

// NewMonitor creates a new mongodb event CommandMonitor.
func NewMonitor(opts ...Option) *event.CommandMonitor {
	cfg := newConfig(opts...)
	m := &monitor{
		spans:   make(map[spanKey]trace.Span),
		cfg:     cfg,
		semconv: semconv.NewEventMonitor(),
	}
	return &event.CommandMonitor{
		Started:   m.Started,
		Succeeded: m.Succeeded,
		Failed:    m.Failed,
	}
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
