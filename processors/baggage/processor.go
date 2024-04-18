// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package baggage

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/sdk/trace"
)

type baggageSpanProcessor struct{}

var _ trace.SpanProcessor = (*baggageSpanProcessor)(nil)

// Returns a new baggageSpanProcessor.
//
// The Baggage span processor duplicates onto a span the attributes found
// in Baggage in the parent context at the moment the span is started.
func NewBaggageSpanProcessor() trace.SpanProcessor {
	return &baggageSpanProcessor{}
}

func (processor baggageSpanProcessor) OnStart(ctx context.Context, span trace.ReadWriteSpan) {
	baggage := baggage.FromContext(ctx)
	for _, entry := range baggage.Members() {
		span.SetAttributes(attribute.String(entry.Key(), entry.Value()))
	}
}

func (processor baggageSpanProcessor) OnEnd(s trace.ReadOnlySpan)       {}
func (processor baggageSpanProcessor) Shutdown(context.Context) error   { return nil }
func (processor baggageSpanProcessor) ForceFlush(context.Context) error { return nil }
