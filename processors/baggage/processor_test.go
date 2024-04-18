// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package baggage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/attribute"
	otelbaggage "go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/sdk/trace"
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

func TestBaggageSpanProcessorAppendsBaggageAttributes(t *testing.T) {
	// create ctx with some baggage
	ctx := context.Background()
	suitcase := otelbaggage.FromContext(ctx)
	packingCube, err := otelbaggage.NewMemberRaw("baggage.test", "baggage value")
	require.NoError(t, err)
	suitcase, err = suitcase.SetMember(packingCube)
	require.NoError(t, err)
	ctx = otelbaggage.ContextWithBaggage(ctx, suitcase)

	// create trace provider with baggage processor and test exporter
	exporter := NewTestExporter()
	tp := trace.NewTracerProvider(
		trace.WithSpanProcessor(NewBaggageSpanProcessor()),
		trace.WithSpanProcessor(trace.NewSimpleSpanProcessor(exporter)),
	)

	// create tracer and start/end span
	tracer := tp.Tracer("test")
	_, span := tracer.Start(ctx, "test")
	span.End()

	assert.Len(t, exporter.spans, 1)
	assert.Len(t, exporter.spans[0].Attributes(), 1)

	for _, attr := range exporter.spans[0].Attributes() {
		assert.Equal(t, attribute.Key("baggage.test"), attr.Key)
		assert.Equal(t, "baggage value", attr.Value.AsString())
	}
}