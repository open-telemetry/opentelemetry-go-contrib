// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package b3_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/google/go-cmp/cmp"

	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
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
		propagator := b3.New()

		for _, tt := range tg.tests {
			t.Run(tt.name, func(t *testing.T) {
				header := make(http.Header, len(tt.headers))
				for h, v := range tt.headers {
					header.Set(h, v)
				}

				ctx := context.Background()
				ctx = propagator.Extract(ctx, propagation.HeaderCarrier(header))
				gotSc := trace.SpanContextFromContext(ctx)

				comparer := cmp.Comparer(func(a, b trace.SpanContext) bool {
					// Do not compare remote field, it is unset on empty
					// SpanContext.
					newA := a.WithRemote(b.IsRemote())
					return newA.Equal(b)
				})
				if diff := cmp.Diff(gotSc, trace.NewSpanContext(tt.wantScc), comparer); diff != "" {
					t.Errorf("%s: %s: -got +want %s", tg.name, tt.name, diff)
				}
				assert.Equal(t, tt.debug, b3.DebugFromContext(ctx))
				assert.Equal(t, tt.deferred, b3.DeferredFromContext(ctx))
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
			propagator := b3.New(b3.WithInjectEncoding(tt.encoding))
			t.Run(tt.name, func(t *testing.T) {
				header := http.Header{}
				ctx := trace.ContextWithSpanContext(
					context.Background(),
					trace.NewSpanContext(tt.scc),
				)
				ctx = b3.WithDebug(ctx, tt.debug)
				ctx = b3.WithDeferred(ctx, tt.deferred)
				propagator.Inject(ctx, propagation.HeaderCarrier(header))

				for h, v := range tt.wantHeaders {
					got, want := header.Get(h), v
					if diff := cmp.Diff(got, want); diff != "" {
						t.Errorf("%s: %s, header=%s: -got +want %s", tg.name, tt.name, h, diff)
					}
				}
				for _, h := range tt.doNotWantHeaders {
					v, gotOk := header[h]
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
		propagator propagation.TextMapPropagator
		want       []string
	}{
		{
			name:       "no encoding specified",
			propagator: b3.New(),
			want: []string{
				b3TraceID,
				b3SpanID,
				b3Sampled,
				b3Flags,
			},
		},
		{
			name:       "B3MultipleHeader encoding specified",
			propagator: b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader)),
			want: []string{
				b3TraceID,
				b3SpanID,
				b3Sampled,
				b3Flags,
			},
		},
		{
			name:       "B3SingleHeader encoding specified",
			propagator: b3.New(b3.WithInjectEncoding(b3.B3SingleHeader)),
			want: []string{
				b3Context,
			},
		},
		{
			name:       "B3SingleHeader and B3MultipleHeader encoding specified",
			propagator: b3.New(b3.WithInjectEncoding(b3.B3SingleHeader | b3.B3MultipleHeader)),
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
