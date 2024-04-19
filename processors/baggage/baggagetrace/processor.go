// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package baggagetrace // import "go.opentelemetry.io/contrib/processors/baggage/baggagetrace"

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	otelbaggage "go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/sdk/trace"
)

type SpanProcessor struct{}

var _ trace.SpanProcessor = (*SpanProcessor)(nil)

// NewBaggageSpanProcessor returns a new SpanProcessor.
//
// The Baggage span processor duplicates onto a span the attributes found
// in Baggage in the parent context at the moment the span is started.
func NewBaggageSpanProcessor() trace.SpanProcessor {
	return &SpanProcessor{}
}

func (processor SpanProcessor) OnStart(ctx context.Context, span trace.ReadWriteSpan) {
	for _, entry := range otelbaggage.FromContext(ctx).Members() {
		span.SetAttributes(attribute.String(entry.Key(), entry.Value()))
	}
}

func (processor SpanProcessor) OnEnd(s trace.ReadOnlySpan)       {}
func (processor SpanProcessor) Shutdown(context.Context) error   { return nil }
func (processor SpanProcessor) ForceFlush(context.Context) error { return nil }
