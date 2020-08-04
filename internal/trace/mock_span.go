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

	otelkv "go.opentelemetry.io/otel/api/kv"
	oteltrace "go.opentelemetry.io/otel/api/trace"
)

// Span is a mock span used in association with Tracer for
// testing purpose only.
type Span struct {
	sc            oteltrace.SpanContext
	tracer        *Tracer
	Name          string
	Attributes    map[otelkv.Key]otelkv.Value
	Kind          oteltrace.SpanKind
	Status        codes.Code
	StatusMessage string
	ParentSpanID  oteltrace.SpanID
	Links         map[oteltrace.SpanContext][]otelkv.KeyValue
}

var _ oteltrace.Span = (*Span)(nil)

// SpanContext returns associated oteltrace.SpanContext.
//
// If the receiver is nil it returns an empty oteltrace.SpanContext.
func (ms *Span) SpanContext() oteltrace.SpanContext {
	if ms == nil {
		return oteltrace.EmptySpanContext()
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
	ms.StatusMessage = msg
}

// SetAttribute adds a single inferred attribute.
func (ms *Span) SetAttribute(key string, value interface{}) {
	ms.SetAttributes(otelkv.Any(key, value))
}

// SetAttributes adds an attribute to Attributes member.
func (ms *Span) SetAttributes(attributes ...otelkv.KeyValue) {
	if ms.Attributes == nil {
		ms.Attributes = make(map[otelkv.Key]otelkv.Value)
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
func (ms *Span) AddEvent(ctx context.Context, name string, attrs ...otelkv.KeyValue) {
}

// AddEvent does nothing.
func (ms *Span) AddEventWithTimestamp(ctx context.Context, timestamp time.Time, name string, attrs ...otelkv.KeyValue) {
}
