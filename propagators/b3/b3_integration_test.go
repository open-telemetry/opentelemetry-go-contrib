// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package b3_test

import (
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/contrib/propagators/b3"
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

				ctx := t.Context()
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

func TestExtractB3ReplacesSamplingState(t *testing.T) {
	const (
		traceID = "463ac35c9f6413ad48485a3953bb6124"
		spanID  = "a2fb4a1d1a96d312"
	)
	for _, previous := range []string{"debug", "deferred"} {
		for _, encoding := range []string{"single", "multiple"} {
			for _, sampling := range []string{"0", "1", "d", ""} {
				t.Run(previous+"/"+encoding+"/"+sampling, func(t *testing.T) {
					propagator := b3.New()
					previousHeader := "4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7"
					if previous == "debug" {
						previousHeader += "-d"
					}
					parent := propagator.Extract(t.Context(), propagation.MapCarrier{b3Context: previousHeader})
					wantHeader := traceID + "-" + spanID
					if sampling != "" {
						wantHeader += "-" + sampling
					}
					header := propagation.MapCarrier{b3Context: wantHeader}
					if encoding == "multiple" {
						header = propagation.MapCarrier{b3TraceID: traceID, b3SpanID: spanID}
						if sampling == "d" {
							header[b3Flags] = "1"
						} else if sampling != "" {
							header[b3Sampled] = sampling
						}
					}
					ctx := propagator.Extract(parent, header)
					assert.True(t, trace.SpanContextFromContext(ctx).IsRemote())
					assert.Equal(t, sampling == "1" || sampling == "d", trace.SpanContextFromContext(ctx).IsSampled())
					injected := propagation.MapCarrier{}
					propagator.Inject(ctx, injected)
					assert.Equal(t, wantHeader, injected[b3Context])
					propagator.Inject(parent, injected)
					assert.Equal(t, previousHeader, injected[b3Context])
				})
			}
		}
	}
}

func TestExtractB3PreservesDebugWithoutValidSpanContext(t *testing.T) {
	const parentHeader = "4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-d"
	propagator := b3.New()
	parent := propagator.Extract(t.Context(), propagation.MapCarrier{b3Context: parentHeader})
	for _, header := range []propagation.MapCarrier{
		{},
		{b3TraceID: "invalid", b3SpanID: "invalid"},
		{b3Context: "0"},
	} {
		ctx := propagator.Extract(parent, header)
		injected := propagation.MapCarrier{}
		propagator.Inject(ctx, injected)
		assert.Equal(t, parentHeader, injected[b3Context])
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
					t.Context(),
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
			// B3Unspecified defaults to single-header injection, so Fields must
			// report the `b3` header to match what Inject writes.
			want: []string{
				b3Context,
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
