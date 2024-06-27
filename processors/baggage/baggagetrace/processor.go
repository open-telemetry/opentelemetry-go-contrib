// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package baggagetrace // import "go.opentelemetry.io/contrib/processors/baggage/baggagetrace"

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	otelbaggage "go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/sdk/trace"
)

// Filter returns true if the baggage member with key should be added to a
// span.
type Filter func(key string) bool

// AllowAllBaggageKeys allows all baggage members to be added to a span.
var AllowAllBaggageKeys Filter = func(string) bool { return true }

// SpanProcessor is a [trace.SpanProcessor] implementation that adds baggage
// members onto a span as attributes.
type SpanProcessor struct {
	filter Filter
}

var _ trace.SpanProcessor = (*SpanProcessor)(nil)

// New returns a new [SpanProcessor].
//
// The Baggage span processor duplicates onto a span the attributes found
// in Baggage in the parent context at the moment the span is started.
// The passed filter determines which baggage members are added to the span.
//
// If filter is nil, all baggage members will be added.
func New(filter Filter) *SpanProcessor {
	return &SpanProcessor{
		filter: filter,
	}
}

// OnStart is called when a span is started and adds span attributes for baggage contents.
func (processor SpanProcessor) OnStart(ctx context.Context, span trace.ReadWriteSpan) {
	filter := processor.filter
	if filter == nil {
		filter = AllowAllBaggageKeys
	}

	for _, entry := range otelbaggage.FromContext(ctx).Members() {
		if filter(entry.Key()) {
			span.SetAttributes(attribute.String(entry.Key(), entry.Value()))
		}
	}
}

// OnEnd is called when span is finished and is a no-op for this processor.
func (processor SpanProcessor) OnEnd(s trace.ReadOnlySpan) {}

// Shutdown is called when the SDK shuts down and is a no-op for this processor.
func (processor SpanProcessor) Shutdown(context.Context) error { return nil }

// ForceFlush exports all ended spans to the configured Exporter that have not yet
// been exported and is a no-op for this processor.
func (processor SpanProcessor) ForceFlush(context.Context) error { return nil }
