// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package jaeger_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/google/go-cmp/cmp"

	"go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
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
				header := make(http.Header, len(tc.headers))
				for k, v := range tc.headers {
					header.Set(k, v)
				}

				ctx := context.Background()
				ctx = propagator.Extract(ctx, propagation.HeaderCarrier(header))
				resSc := trace.SpanContextFromContext(ctx)
				comparer := cmp.Comparer(func(a, b trace.SpanContext) bool {
					// Do not compare remote field, it is unset on empty
					// SpanContext.
					newA := a.WithRemote(b.IsRemote())
					return newA.Equal(b)
				})
				if diff := cmp.Diff(resSc, trace.NewSpanContext(tc.expected), comparer); diff != "" {
					t.Errorf("%s: %s: -got +want %s", tg.name, tc.name, diff)
				}
				assert.Equal(t, tc.debug, jaeger.DebugFromContext(ctx))
			})
		}
	}
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
				header := http.Header{}
				ctx := trace.ContextWithSpanContext(
					jaeger.WithDebug(context.Background(), tc.debug),
					trace.NewSpanContext(tc.scc),
				)
				propagator.Inject(ctx, propagation.HeaderCarrier(header))

				for h, v := range tc.wantHeaders {
					result, want := header.Get(h), v
					if diff := cmp.Diff(result, want); diff != "" {
						t.Errorf("%s: %s, header=%s: -got +want %s", tg.name, tc.name, h, diff)
					}
				}
			})
		}
	}
}
