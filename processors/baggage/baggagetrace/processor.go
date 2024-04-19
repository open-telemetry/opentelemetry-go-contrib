// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package baggagetrace // import "go.opentelemetry.io/contrib/processors/baggage"

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	otelbaggage "go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/sdk/trace"
)

// SpanProcessor is a processing pipeline for spans in the trace signal.
type SpanProcessor struct{}

var _ trace.SpanProcessor = (*SpanProcessor)(nil)

// New returns a new SpanProcessor.
//
// The Baggage span processor duplicates onto a span the attributes found
// in Baggage in the parent context at the moment the span is started.
func New() trace.SpanProcessor {
	return &SpanProcessor{}
}

// OnStart is called when a span is started and adds span attributes for baggage contents.
func (processor SpanProcessor) OnStart(ctx context.Context, span trace.ReadWriteSpan) {
	for _, entry := range otelbaggage.FromContext(ctx).Members() {
		span.SetAttributes(attribute.String(entry.Key(), entry.Value()))
	}
}

// OnEnd is called when span is finished and is a no-op for this processor.
func (processor SpanProcessor) OnEnd(s trace.ReadOnlySpan) {}

// Shutdown is called when the SDK shuts down and is a no-op for this processor.
func (processor SpanProcessor) Shutdown(context.Context) error { return nil }

// ForceFlush exports all ended spans to the configured Exporter that have not yet
// been exported and is a no-op for this processor.
func (processor SpanProcessor) ForceFlush(context.Context) error { return nil }
