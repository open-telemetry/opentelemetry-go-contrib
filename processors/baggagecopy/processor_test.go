// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package baggagecopy

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	oteltrace "go.opentelemetry.io/otel/trace"
)

var _ trace.SpanExporter = &testExporter{}

type testExporter struct {
	spans []trace.ReadOnlySpan
}

func (*testExporter) Start(context.Context) error    { return nil }
func (*testExporter) Shutdown(context.Context) error { return nil }

func (e *testExporter) ExportSpans(_ context.Context, ss []trace.ReadOnlySpan) error {
	e.spans = append(e.spans, ss...)
	return nil
}

func NewTestExporter() *testExporter {
	return &testExporter{}
}

func TestSpanProcessorAppendsAllBaggageAttributes(t *testing.T) {
	b, _ := baggage.New()
	b = addEntryToBaggage(t, b, "baggage.test", "baggage value")
	ctx := baggage.ContextWithBaggage(t.Context(), b)

	// create trace provider with baggage processor and test exporter
	exporter := NewTestExporter()
	tp := trace.NewTracerProvider(
		trace.WithSpanProcessor(NewSpanProcessor(AllowAllMembers)),
		trace.WithSpanProcessor(trace.NewSimpleSpanProcessor(exporter)),
	)

	// create tracer and start/end span
	tracer := tp.Tracer("test")
	_, span := tracer.Start(ctx, "test")
	span.End()

	require.Len(t, exporter.spans, 1)
	require.Len(t, exporter.spans[0].Attributes(), 1)

	want := []attribute.KeyValue{attribute.String("baggage.test", "baggage value")}
	require.Equal(t, want, exporter.spans[0].Attributes())
}

func TestSpanProcessorAppendsBaggageAttributesWithHaPrefixPredicate(t *testing.T) {
	b, _ := baggage.New()
	b = addEntryToBaggage(t, b, "baggage.test", "baggage value")
	ctx := baggage.ContextWithBaggage(t.Context(), b)

	baggageKeyPredicate := func(m baggage.Member) bool {
		return strings.HasPrefix(m.Key(), "baggage.")
	}

	// create trace provider with baggage processor and test exporter
	exporter := NewTestExporter()
	tp := trace.NewTracerProvider(
		trace.WithSpanProcessor(NewSpanProcessor(baggageKeyPredicate)),
		trace.WithSpanProcessor(trace.NewSimpleSpanProcessor(exporter)),
	)

	// create tracer and start/end span
	tracer := tp.Tracer("test")
	_, span := tracer.Start(ctx, "test")
	span.End()

	require.Len(t, exporter.spans, 1)
	require.Len(t, exporter.spans[0].Attributes(), 1)

	want := []attribute.KeyValue{attribute.String("baggage.test", "baggage value")}
	require.Equal(t, want, exporter.spans[0].Attributes())
}

func TestSpanProcessorAppendsBaggageAttributesWithRegexPredicate(t *testing.T) {
	b, _ := baggage.New()
	b = addEntryToBaggage(t, b, "baggage.test", "baggage value")
	ctx := baggage.ContextWithBaggage(t.Context(), b)

	expr := regexp.MustCompile(`^baggage\..*`)
	baggageKeyPredicate := func(m baggage.Member) bool {
		return expr.MatchString(m.Key())
	}

	// create trace provider with baggage processor and test exporter
	exporter := NewTestExporter()
	tp := trace.NewTracerProvider(
		trace.WithSpanProcessor(NewSpanProcessor(baggageKeyPredicate)),
		trace.WithSpanProcessor(trace.NewSimpleSpanProcessor(exporter)),
	)

	// create tracer and start/end span
	tracer := tp.Tracer("test")
	_, span := tracer.Start(ctx, "test")
	span.End()

	require.Len(t, exporter.spans, 1)
	require.Len(t, exporter.spans[0].Attributes(), 1)

	want := []attribute.KeyValue{attribute.String("baggage.test", "baggage value")}
	require.Equal(t, want, exporter.spans[0].Attributes())
}

func TestOnlyAddsBaggageEntriesThatMatchPredicate(t *testing.T) {
	b, _ := baggage.New()
	b = addEntryToBaggage(t, b, "baggage.test", "baggage value")
	b = addEntryToBaggage(t, b, "foo", "bar")
	ctx := baggage.ContextWithBaggage(t.Context(), b)

	baggageKeyPredicate := func(m baggage.Member) bool {
		return m.Key() == "baggage.test"
	}

	// create trace provider with baggage processor and test exporter
	exporter := NewTestExporter()
	tp := trace.NewTracerProvider(
		trace.WithSpanProcessor(NewSpanProcessor(baggageKeyPredicate)),
		trace.WithSpanProcessor(trace.NewSimpleSpanProcessor(exporter)),
	)

	// create tracer and start/end span
	tracer := tp.Tracer("test")
	_, span := tracer.Start(ctx, "test")
	span.End()

	require.Len(t, exporter.spans, 1)
	require.Len(t, exporter.spans[0].Attributes(), 1)

	want := attribute.String("baggage.test", "baggage value")
	require.Equal(t, want, exporter.spans[0].Attributes()[0])
}

func TestSpanProcessorOverwritesStartAttributesByDefault(t *testing.T) {
	want := []attribute.KeyValue{
		attribute.String("network.peer.address", "10.0.0.1"),
		attribute.String("baggage.test", "baggage value"),
	}
	got := startSpanWithPeerAddress(t, NewSpanProcessor(AllowAllMembers))
	assert.ElementsMatch(t, want, got)
}

func TestSpanProcessorKeepsStartAttributes(t *testing.T) {
	want := []attribute.KeyValue{
		attribute.String("network.peer.address", "203.0.113.7"),
		attribute.String("baggage.test", "baggage value"),
	}
	got := startSpanWithPeerAddress(t, NewSpanProcessor(AllowAllMembers, WithOverwriteExistingAttributes(false)))
	assert.ElementsMatch(t, want, got)
}

// startSpanWithPeerAddress starts and ends a span that already has a
// network.peer.address attribute, in a context carrying baggage with the same
// key, and returns the exported attributes.
func startSpanWithPeerAddress(t *testing.T, proc *SpanProcessor) []attribute.KeyValue {
	b, _ := baggage.New()
	b = addEntryToBaggage(t, b, "network.peer.address", "10.0.0.1")
	b = addEntryToBaggage(t, b, "baggage.test", "baggage value")
	ctx := baggage.ContextWithBaggage(t.Context(), b)

	exporter := NewTestExporter()
	tp := trace.NewTracerProvider(
		trace.WithSpanProcessor(proc),
		trace.WithSpanProcessor(trace.NewSimpleSpanProcessor(exporter)),
	)

	_, span := tp.Tracer("test").Start(ctx, "test", oteltrace.WithAttributes(
		attribute.String("network.peer.address", "203.0.113.7"),
	))
	span.End()

	require.Len(t, exporter.spans, 1)
	return exporter.spans[0].Attributes()
}

func addEntryToBaggage(t *testing.T, b baggage.Baggage, key, value string) baggage.Baggage {
	member, err := baggage.NewMemberRaw(key, value)
	require.NoError(t, err)
	b, err = b.SetMember(member)
	require.NoError(t, err)
	return b
}

func TestZeroSpanProcessorNoPanic(t *testing.T) {
	sp := new(SpanProcessor)

	m, err := baggage.NewMember("key", "val")
	require.NoError(t, err)
	b, err := baggage.New(m)
	require.NoError(t, err)

	ctx := baggage.ContextWithBaggage(t.Context(), b)
	roS := (tracetest.SpanStub{}).Snapshot()
	rwS := rwSpan{}
	assert.NotPanics(t, func() {
		sp.OnStart(ctx, rwS)
		sp.OnEnd(roS)
		_ = sp.ForceFlush(ctx)
		_ = sp.Shutdown(ctx)
	})

	keep := NewSpanProcessor(nil, WithOverwriteExistingAttributes(false))
	assert.NotPanics(t, func() { keep.OnStart(ctx, rwS) })
}

type rwSpan struct {
	trace.ReadWriteSpan
}

func (rwSpan) SetAttributes(...attribute.KeyValue) {}

func (rwSpan) Attributes() []attribute.KeyValue { return nil }
