// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package b3_test

import (
	"context"
	"net/http"
	"testing"

	"go.opentelemetry.io/contrib/propagators/b3"
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
		propagator := b3.B3{}
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
					_ = propagator.Extract(ctx, req.Header)
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
			propagator := b3.B3{InjectEncoding: tt.encoding}
			traceBenchmark(tg.name+"/"+tt.name, b, func(b *testing.B) {
				req, _ := http.NewRequest("GET", "http://example.com", nil)
				ctx := trace.ContextWithSpan(
					context.Background(),
					testSpan{sc: tt.sc},
				)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					propagator.Inject(ctx, req.Header)
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
