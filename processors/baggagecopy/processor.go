// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package baggagecopy

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/sdk/trace"
)

// Filter returns true if the baggage member should be copied to a span or log record.
type Filter func(member baggage.Member) bool

// AllowAllMembers allows all baggage members to be copied to a span or log record.
var AllowAllMembers Filter = func(baggage.Member) bool { return true }

// SpanProcessor is a [trace.SpanProcessor] implementation that adds baggage
// members onto a span as attributes.
type SpanProcessor struct {
	filter Filter
}

var _ trace.SpanProcessor = (*SpanProcessor)(nil)

// NewSpanProcessor returns a new [SpanProcessor].
//
// The Baggage span processor duplicates onto a span the attributes found
// in Baggage in the parent context at the moment the span is started.
// The passed filter determines which baggage members are added to the span.
//
// If filter is nil, all baggage members will be added.
func NewSpanProcessor(filter Filter) *SpanProcessor {
	return &SpanProcessor{
		filter: filter,
	}
}

// OnStart is called when a span is started and adds span attributes for baggage contents.
func (processor SpanProcessor) OnStart(ctx context.Context, span trace.ReadWriteSpan) {
	filter := processor.filter
	if filter == nil {
		filter = AllowAllMembers
	}

	for _, member := range baggage.FromContext(ctx).Members() {
		if filter(member) {
			span.SetAttributes(attribute.String(member.Key(), member.Value()))
		}
	}
}

// OnEnd is called when span is finished and is a no-op for this processor.
func (SpanProcessor) OnEnd(trace.ReadOnlySpan) {}

// Shutdown is called when the SDK shuts down and is a no-op for this processor.
func (SpanProcessor) Shutdown(context.Context) error { return nil }

// ForceFlush is a no-op for this processor.
func (SpanProcessor) ForceFlush(context.Context) error { return nil }
