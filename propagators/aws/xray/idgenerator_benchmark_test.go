// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xray

import (
	"context"
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
	for i := 0; i < b.N; i++ {
		_, span := tracer.Start(context.Background(), "Example Trace")
		span.End()
	}
}

func BenchmarkStartAndEndNestedSampledSpan(b *testing.B) {
	ctx, parent := tracer.Start(context.Background(), "Parent operation...")
	defer parent.End()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, span := tracer.Start(ctx, "Sub operation...")
		span.End()
	}
}
