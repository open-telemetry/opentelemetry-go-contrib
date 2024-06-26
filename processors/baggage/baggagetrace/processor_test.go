// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package baggagetrace

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/attribute"
	otelbaggage "go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

var _ trace.SpanExporter = &testExporter{}

type testExporter struct {
	spans []trace.ReadOnlySpan
}

func (e *testExporter) Start(ctx context.Context) error    { return nil }
func (e *testExporter) Shutdown(ctx context.Context) error { return nil }

func (e *testExporter) ExportSpans(ctx context.Context, ss []trace.ReadOnlySpan) error {
	e.spans = append(e.spans, ss...)
	return nil
}

func NewTestExporter() *testExporter {
	return &testExporter{}
}

func TestSpanProcessorAppendsAllBaggageAttributes(t *testing.T) {
	baggage, _ := otelbaggage.New()
	baggage = addEntryToBaggage(t, baggage, "baggage.test", "baggage value")
	ctx := otelbaggage.ContextWithBaggage(context.Background(), baggage)

	// create trace provider with baggage processor and test exporter
	exporter := NewTestExporter()
	tp := trace.NewTracerProvider(
		trace.WithSpanProcessor(New(AllowAllBaggageKeys)),
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
	baggage, _ := otelbaggage.New()
	baggage = addEntryToBaggage(t, baggage, "baggage.test", "baggage value")
	ctx := otelbaggage.ContextWithBaggage(context.Background(), baggage)

	baggageKeyPredicate := func(key string) bool {
		return strings.HasPrefix(key, "baggage.")
	}

	// create trace provider with baggage processor and test exporter
	exporter := NewTestExporter()
	tp := trace.NewTracerProvider(
		trace.WithSpanProcessor(New(baggageKeyPredicate)),
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
	baggage, _ := otelbaggage.New()
	baggage = addEntryToBaggage(t, baggage, "baggage.test", "baggage value")
	ctx := otelbaggage.ContextWithBaggage(context.Background(), baggage)

	expr := regexp.MustCompile(`^baggage\..*`)
	baggageKeyPredicate := func(key string) bool {
		return expr.MatchString(key)
	}

	// create trace provider with baggage processor and test exporter
	exporter := NewTestExporter()
	tp := trace.NewTracerProvider(
		trace.WithSpanProcessor(New(baggageKeyPredicate)),
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
	baggage, _ := otelbaggage.New()
	baggage = addEntryToBaggage(t, baggage, "baggage.test", "baggage value")
	baggage = addEntryToBaggage(t, baggage, "foo", "bar")
	ctx := otelbaggage.ContextWithBaggage(context.Background(), baggage)

	baggageKeyPredicate := func(key string) bool {
		return key == "baggage.test"
	}

	// create trace provider with baggage processor and test exporter
	exporter := NewTestExporter()
	tp := trace.NewTracerProvider(
		trace.WithSpanProcessor(New(baggageKeyPredicate)),
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

func addEntryToBaggage(t *testing.T, baggage otelbaggage.Baggage, key, value string) otelbaggage.Baggage {
	member, err := otelbaggage.NewMemberRaw(key, value)
	require.NoError(t, err)
	baggage, err = baggage.SetMember(member)
	require.NoError(t, err)
	return baggage
}

func TestZeroSpanProcessorNoPanic(t *testing.T) {
	sp := new(SpanProcessor)

	m, err := otelbaggage.NewMember("key", "val")
	require.NoError(t, err)
	b, err := otelbaggage.New(m)
	require.NoError(t, err)

	ctx := otelbaggage.ContextWithBaggage(context.Background(), b)
	roS := (tracetest.SpanStub{}).Snapshot()
	rwS := rwSpan{}
	assert.NotPanics(t, func() {
		sp.OnStart(ctx, rwS)
		sp.OnEnd(roS)
		_ = sp.ForceFlush(ctx)
		_ = sp.Shutdown(ctx)
	})
}

type rwSpan struct {
	trace.ReadWriteSpan
}

func (s rwSpan) SetAttributes(kv ...attribute.KeyValue) {}
