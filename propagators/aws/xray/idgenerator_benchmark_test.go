// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xray

import (
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer

func init() {
	idg := NewIDGenerator()

	tracer = sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithIDGenerator(idg),
	).Tracer("sample-app")
}

func BenchmarkStartAndEndSampledSpan(b *testing.B) {
	for range b.N {
		_, span := tracer.Start(b.Context(), "Example Trace")
		span.End()
	}
}

func BenchmarkStartAndEndNestedSampledSpan(b *testing.B) {
	ctx, parent := tracer.Start(b.Context(), "Parent operation...")
	defer parent.End()

	b.ResetTimer()
	for range b.N {
		_, span := tracer.Start(ctx, "Sub operation...")
		span.End()
	}
}
