// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ot_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"

	"go.opentelemetry.io/contrib/propagators/ot"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
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
				h := make(http.Header, len(tc.headers))
				for k, v := range tc.headers {
					h.Set(k, v)
				}

				ctx := context.Background()
				ctx = propagator.Extract(ctx, propagation.HeaderCarrier(h))
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

				members := baggage.FromContext(ctx).Members()
				actualBaggage := map[string]string{}
				for _, m := range members {
					actualBaggage[m.Key()] = m.Value()
				}

				if diff := cmp.Diff(tc.baggage, actualBaggage); tc.baggage != nil && diff != "" {
					t.Errorf("%s: %s: -got +want %s", tg.name, tc.name, diff)
				}
			})
		}
	}
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
				members := []baggage.Member{}
				for k, v := range tc.baggage {
					m, err := baggage.NewMember(k, v)
					if err != nil {
						t.Errorf("%s: %s, unexpected error creating baggage member: %s", tg.name, tc.name, err.Error())
					}
					members = append(members, m)
				}
				bag, err := baggage.New(members...)
				if err != nil {
					t.Errorf("%s: %s, unexpected error creating baggage: %s", tg.name, tc.name, err.Error())
				}
				ctx := baggage.ContextWithBaggage(context.Background(), bag)
				ctx = trace.ContextWithSpanContext(ctx, trace.NewSpanContext(tc.sc))
				header := http.Header{}
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
