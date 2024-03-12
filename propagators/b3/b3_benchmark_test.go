// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package b3_test

import (
	"context"
	"net/http"
	"testing"

	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func BenchmarkExtractB3(b *testing.B) {
	testGroup := []struct {
		name  string
		tests []extractTest
	}{
		{
			name:  "valid headers",
			tests: extractHeaders,
		},
		{
			name:  "invalid headers",
			tests: extractInvalidHeaders,
		},
	}

	for _, tg := range testGroup {
		propagator := b3.New()
		for _, tt := range tg.tests {
			traceBenchmark(tg.name+"/"+tt.name, b, func(b *testing.B) {
				ctx := context.Background()
				req, _ := http.NewRequest("GET", "http://example.com", nil)
				for h, v := range tt.headers {
					req.Header.Set(h, v)
				}
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_ = propagator.Extract(ctx, propagation.HeaderCarrier(req.Header))
				}
			})
		}
	}
}

func BenchmarkInjectB3(b *testing.B) {
	testGroup := []struct {
		name  string
		tests []injectTest
	}{
		{
			name:  "valid headers",
			tests: injectHeader,
		},
		{
			name:  "invalid headers",
			tests: injectInvalidHeader,
		},
	}

	for _, tg := range testGroup {
		for _, tt := range tg.tests {
			propagator := b3.New(b3.WithInjectEncoding(tt.encoding))
			traceBenchmark(tg.name+"/"+tt.name, b, func(b *testing.B) {
				req, _ := http.NewRequest("GET", "http://example.com", nil)
				ctx := trace.ContextWithSpan(
					context.Background(),
					testSpan{sc: trace.NewSpanContext(tt.scc)},
				)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					propagator.Inject(ctx, propagation.HeaderCarrier(req.Header))
				}
			})
		}
	}
}

func traceBenchmark(name string, b *testing.B, fn func(*testing.B)) {
	b.Run(name, func(b *testing.B) {
		b.ReportAllocs()
		fn(b)
	})
	b.Run(name, func(b *testing.B) {
		b.ReportAllocs()
		fn(b)
	})
}
