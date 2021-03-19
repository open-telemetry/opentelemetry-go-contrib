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

package ot_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"

	"go.opentelemetry.io/contrib/propagators/ot"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/oteltest"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

var (
	mockTracer  = oteltest.NewTracerProvider().Tracer("")
	_, mockSpan = mockTracer.Start(context.Background(), "")
)

func TestExtractOT(t *testing.T) {
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
		propagator := ot.OT{}

		for _, tc := range tg.testcases {
			t.Run(tc.name, func(t *testing.T) {
				req, _ := http.NewRequest("GET", "http://example.com", nil)
				for k, v := range tc.headers {
					req.Header.Set(k, v)
				}

				ctx := context.Background()
				ctx = propagator.Extract(ctx, propagation.HeaderCarrier(req.Header))
				resSc := trace.RemoteSpanContextFromContext(ctx)

				comparer := cmp.Comparer(func(a, b trace.SpanContext) bool {
					// Do not compare remote field, it is unset on empty
					// SpanContext.
					newA := a.WithRemote(b.IsRemote())
					return newA.Equal(b)
				})
				if diff := cmp.Diff(resSc, trace.NewSpanContext(tc.expected), comparer); diff != "" {
					t.Errorf("%s: %s: -got +want %s", tg.name, tc.name, diff)
				}

				m := baggage.Set(ctx)
				mi := tc.baggage.Iter()
				for mi.Next() {
					attribute := mi.Attribute()
					val, ok := m.Value(attribute.Key)
					if !ok {
						t.Errorf("%s: %s: expected key '%s'", tg.name, tc.name, attribute.Key)
					}
					if diff := cmp.Diff(attribute.Value.AsString(), val.AsString()); diff != "" {
						t.Errorf("%s: %s: -got +want %s", tg.name, tc.name, diff)
					}
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

func TestInjectOT(t *testing.T) {
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
			propagator := ot.OT{}
			t.Run(tc.name, func(t *testing.T) {
				req, _ := http.NewRequest("GET", "http://example.com", nil)

				ctx := baggage.ContextWithValues(context.Background(),
					tc.baggage...,
				)
				ctx = trace.ContextWithSpan(
					ctx,
					testSpan{
						Span: mockSpan,
						sc:   trace.NewSpanContext(tc.sc),
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
