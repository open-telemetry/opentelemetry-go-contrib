// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgrpc

import (
	"testing"

	"go.opentelemetry.io/otel"
)

func BenchmarkMetadataSupplier(b *testing.B) {
	ctx := b.Context()
	propagator := otel.GetTextMapPropagator()

	b.Run("extract", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = extract(ctx, propagator)
		}
	})

	b.Run("inject", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = inject(ctx, propagator)
		}
	})
}
