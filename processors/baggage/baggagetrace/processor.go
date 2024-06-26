// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package baggagetrace // import "go.opentelemetry.io/contrib/processors/baggage/baggagetrace"

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/sdk/trace"
)

// BaggageKeyPredicate is a function that returns true if the baggage key should be added to the span.
type BaggageKeyPredicate func(baggageKey string) bool

// AllowAllBaggageKeys is a BaggageKeyPredicate that allows all baggage keys.
var AllowAllBaggageKeys = func(string) bool { return true }

// SpanProcessor is a processing pipeline for spans in the trace signal.
type SpanProcessor struct {
	baggageKeyPredicate BaggageKeyPredicate
}

var _ trace.SpanProcessor = (*SpanProcessor)(nil)

// New returns a new SpanProcessor.
//
// The Baggage span processor duplicates onto a span the attributes found
// in Baggage in the parent context at the moment the span is started.
// The predicate function is used to filter which baggage keys are added to the span.
func New(baggageKeyPredicate BaggageKeyPredicate) trace.SpanProcessor {
	return &SpanProcessor{
		baggageKeyPredicate: baggageKeyPredicate,
	}
}

// OnStart is called when a span is started and adds span attributes for baggage contents.
func (processor SpanProcessor) OnStart(ctx context.Context, span trace.ReadWriteSpan) {
	for _, entry := range baggage.FromContext(ctx).Members() {
		if processor.baggageKeyPredicate(entry.Key()) {
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
