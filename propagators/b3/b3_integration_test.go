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

	"github.com/google/go-cmp/cmp"

	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel/oteltest"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

var (
	mockTracer  = oteltest.NewTracerProvider().Tracer("")
	_, mockSpan = mockTracer.Start(context.Background(), "")
)

func TestExtractB3(t *testing.T) {
	testGroup := []struct {
		name  string
		tests []extractTest
	}{
		{
			name:  "valid extract headers",
			tests: extractHeaders,
		},
		{
			name:  "invalid extract headers",
			tests: extractInvalidHeaders,
		},
	}

	for _, tg := range testGroup {
		propagator := b3.B3{}

		for _, tt := range tg.tests {
			t.Run(tt.name, func(t *testing.T) {
				req, _ := http.NewRequest("GET", "http://example.com", nil)
				for h, v := range tt.headers {
					req.Header.Set(h, v)
				}

				ctx := context.Background()
				ctx = propagator.Extract(ctx, propagation.HeaderCarrier(req.Header))
				gotSc := trace.RemoteSpanContextFromContext(ctx)
				if diff := cmp.Diff(gotSc, tt.wantSc, cmp.AllowUnexported(trace.TraceState{})); diff != "" {
					t.Errorf("%s: %s: -got +want %s", tg.name, tt.name, diff)
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

func TestInjectB3(t *testing.T) {
	testGroup := []struct {
		name  string
		tests []injectTest
	}{
		{
			name:  "valid inject headers",
			tests: injectHeader,
		},
		{
			name:  "invalid inject headers",
			tests: injectInvalidHeader,
		},
	}

	for _, tg := range testGroup {
		for _, tt := range tg.tests {
			propagator := b3.B3{InjectEncoding: tt.encoding}
			t.Run(tt.name, func(t *testing.T) {
				req, _ := http.NewRequest("GET", "http://example.com", nil)
				ctx := trace.ContextWithSpan(
					context.Background(),
					testSpan{
						Span: mockSpan,
						sc:   tt.sc,
					},
				)
				propagator.Inject(ctx, propagation.HeaderCarrier(req.Header))

				for h, v := range tt.wantHeaders {
					got, want := req.Header.Get(h), v
					if diff := cmp.Diff(got, want); diff != "" {
						t.Errorf("%s: %s, header=%s: -got +want %s", tg.name, tt.name, h, diff)
					}
				}
				for _, h := range tt.doNotWantHeaders {
					v, gotOk := req.Header[h]
					if diff := cmp.Diff(gotOk, false); diff != "" {
						t.Errorf("%s: %s, header=%s: -got +want %s, value=%s", tg.name, tt.name, h, diff, v)
					}
				}
			})
		}
	}
}

func TestB3Propagator_Fields(t *testing.T) {
	tests := []struct {
		name       string
		propagator b3.B3
		want       []string
	}{
		{
			name:       "no encoding specified",
			propagator: b3.B3{},
			want: []string{
				b3TraceID,
				b3SpanID,
				b3Sampled,
				b3Flags,
			},
		},
		{
			name:       "B3MultipleHeader encoding specified",
			propagator: b3.B3{InjectEncoding: b3.B3MultipleHeader},
			want: []string{
				b3TraceID,
				b3SpanID,
				b3Sampled,
				b3Flags,
			},
		},
		{
			name:       "B3SingleHeader encoding specified",
			propagator: b3.B3{InjectEncoding: b3.B3SingleHeader},
			want: []string{
				b3Context,
			},
		},
		{
			name:       "B3SingleHeader and B3MultipleHeader encoding specified",
			propagator: b3.B3{InjectEncoding: b3.B3SingleHeader | b3.B3MultipleHeader},
			want: []string{
				b3Context,
				b3TraceID,
				b3SpanID,
				b3Sampled,
				b3Flags,
			},
		},
	}

	for _, test := range tests {
		if diff := cmp.Diff(test.propagator.Fields(), test.want); diff != "" {
			t.Errorf("%s: Fields: -got +want %s", test.name, diff)
		}
	}
}
