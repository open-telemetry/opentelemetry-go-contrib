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

package trace

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"

	otelcore "go.opentelemetry.io/otel/api/core"
	oteltrace "go.opentelemetry.io/otel/api/trace"
)

// Span is a mock span used in association with Tracer for
// testing purpose only.
type Span struct {
	sc           otelcore.SpanContext
	tracer       *Tracer
	Name         string
	Attributes   map[otelcore.Key]otelcore.Value
	Kind         oteltrace.SpanKind
	Status       codes.Code
	ParentSpanID otelcore.SpanID
}

var _ oteltrace.Span = (*Span)(nil)

// SpanContext returns associated otelcore.SpanContext.
//
// If the receiver is nil it returns an empty otelcore.SpanContext.
func (ms *Span) SpanContext() otelcore.SpanContext {
	if ms == nil {
		return otelcore.EmptySpanContext()
	}
	return ms.sc
}

// IsRecording always returns false for Span.
func (ms *Span) IsRecording() bool {
	return false
}

// SetStatus sets the Status member.
func (ms *Span) SetStatus(status codes.Code, msg string) {
	ms.Status = status
}

// SetAttributes adds an attribute to Attributes member.
func (ms *Span) SetAttributes(attributes ...otelcore.KeyValue) {
	if ms.Attributes == nil {
		ms.Attributes = make(map[otelcore.Key]otelcore.Value)
	}
	for _, kv := range attributes {
		ms.Attributes[kv.Key] = kv.Value
	}
}

// End puts the span into tracers ended spans.
func (ms *Span) End(options ...oteltrace.EndOption) {
	ms.tracer.addEndedSpan(ms)
}

// RecordError does nothing.
func (ms *Span) RecordError(ctx context.Context, err error, opts ...oteltrace.ErrorOption) {
}

// SetName sets the span name.
func (ms *Span) SetName(name string) {
	ms.Name = name
}

// Tracer returns the mock tracer implementation of Tracer.
func (ms *Span) Tracer() oteltrace.Tracer {
	return ms.tracer
}

// AddEvent does nothing.
func (ms *Span) AddEvent(ctx context.Context, name string, attrs ...otelcore.KeyValue) {
}

// AddEvent does nothing.
func (ms *Span) AddEventWithTimestamp(ctx context.Context, timestamp time.Time, name string, attrs ...otelcore.KeyValue) {
}
