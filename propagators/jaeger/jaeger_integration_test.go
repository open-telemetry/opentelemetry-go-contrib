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

package jaeger_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"

	"go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/otel/oteltest"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

var (
	mockTracer  = oteltest.NewTracerProvider().Tracer("")
	_, mockSpan = mockTracer.Start(context.Background(), "")
)

func TestExtractJaeger(t *testing.T) {
	testGroup := []struct {
		name      string
		testcases []extractTest
	}{
		{
			name:      "valid test case",
			testcases: extractHeaders,
		},
		{
			name:      "invalid test case",
			testcases: invalidExtractHeaders,
		},
	}

	for _, tg := range testGroup {
		propagator := jaeger.Jaeger{}

		for _, tc := range tg.testcases {
			t.Run(tc.name, func(t *testing.T) {
				req, _ := http.NewRequest("GET", "http://example.com", nil)
				for k, v := range tc.headers {
					req.Header.Set(k, v)
				}

				ctx := context.Background()
				ctx = propagator.Extract(ctx, propagation.HeaderCarrier(req.Header))
				resSc := trace.RemoteSpanContextFromContext(ctx)
				if diff := cmp.Diff(resSc, tc.expected, cmp.AllowUnexported(trace.TraceState{})); diff != "" {
					t.Errorf("%s: %s: -got +want %s", tg.name, tc.name, diff)
				}
			})
		}
	}
}

type testSpan struct {
	trace.Span
	sc trace.SpanContext
}

func (s testSpan) SpanContext() trace.SpanContext {
	return s.sc
}

func TestInjectJaeger(t *testing.T) {
	testGroup := []struct {
		name      string
		testcases []injectTest
	}{
		{
			name:      "valid test case",
			testcases: injectHeaders,
		},
		{
			name:      "invalid test case",
			testcases: invalidInjectHeaders,
		},
	}

	for _, tg := range testGroup {
		for _, tc := range tg.testcases {
			propagator := jaeger.Jaeger{}
			t.Run(tc.name, func(t *testing.T) {
				req, _ := http.NewRequest("GET", "http://example.com", nil)
				ctx := trace.ContextWithSpan(
					context.Background(),
					testSpan{
						Span: mockSpan,
						sc:   tc.sc,
					},
				)
				propagator.Inject(ctx, propagation.HeaderCarrier(req.Header))

				for h, v := range tc.wantHeaders {
					result, want := req.Header.Get(h), v
					if diff := cmp.Diff(result, want); diff != "" {
						t.Errorf("%s: %s, header=%s: -got +want %s", tg.name, tc.name, h, diff)
					}
				}
			})
		}
	}
}
