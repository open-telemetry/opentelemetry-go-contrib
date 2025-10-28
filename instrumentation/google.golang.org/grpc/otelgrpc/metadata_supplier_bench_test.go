// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgrpc

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
)

func BenchmarkMetadataSupplier(b *testing.B) {
	ctx := context.Background()
	propagator := otel.GetTextMapPropagator()

	b.Run("extract", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = extract(ctx, propagator)
		}
	})

	b.Run("inject", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = inject(ctx, propagator)
		}
	})

}
