// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consistent

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func TestParentSamplerDescription(t *testing.T) {
	opts := []sdktrace.ParentBasedSamplerOption{
		sdktrace.WithRemoteParentNotSampled(sdktrace.AlwaysSample()),
	}
	root := ProbabilityBased(1)
	compare := sdktrace.ParentBased(root, opts...)
	parent := ParentProbabilityBased(root, opts...)
	require.Equal(t,
		strings.Replace(
			compare.Description(),
			"ParentBased",
			"ParentProbabilityBased",
			1,
		),
		parent.Description(),
	)
}

func TestParentSamplerValidContext(t *testing.T) {
	parent := ParentProbabilityBased(sdktrace.NeverSample())
	type testCase struct {
		in      string
		sampled bool
	}
	for _, valid := range []testCase{
		// sampled tests
		{"r:10", true},
		{"r:10;a:b", true},
		{"r:10;p:1", true},
		{"r:10;p:10", true},
		{"r:10;p:10;a:b", true},
		{"r:10;p:63", true},
		{"r:10;p:63;a:b", true},
		{"p:0", true},
		{"p:10;a:b", true},
		{"p:63", true},
		{"p:63;a:b", true},

		// unsampled tests
		{"r:10", false},
		{"r:10;a:b", false},
	} {
		t.Run(testName(valid.in), func(t *testing.T) {
			traceID, _ := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
			spanID, _ := trace.SpanIDFromHex("00f067aa0ba902b7")
			traceState, err := trace.TraceState{}.Insert(traceStateKey, valid.in)
			require.NoError(t, err)

			sccfg := trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceState: traceState,
			}

			if valid.sampled {
				sccfg.TraceFlags = trace.FlagsSampled
			}

			parentCtx := trace.ContextWithSpanContext(
				context.Background(),
				trace.NewSpanContext(sccfg),
			)

			result := parent.ShouldSample(
				sdktrace.SamplingParameters{
					ParentContext: parentCtx,
					TraceID:       traceID,
					Name:          "test",
					Kind:          trace.SpanKindServer,
				},
			)

			if valid.sampled {
				require.Equal(t, sdktrace.RecordAndSample, result.Decision)
			} else {
				require.Equal(t, sdktrace.Drop, result.Decision)
			}
			require.Equal(t, []attribute.KeyValue(nil), result.Attributes)
			require.Equal(t, valid.in, result.Tracestate.Get(traceStateKey))
		})
	}
}

func TestParentSamplerInvalidContext(t *testing.T) {
	parent := ParentProbabilityBased(sdktrace.NeverSample())
	type testCase struct {
		in      string
		sampled bool
		expect  string
	}
	for _, invalid := range []testCase{
		// sampled
		{"r:100", true, ""},
		{"r:100;p:1", true, ""},
		{"r:100;p:1;a:b", true, "a:b"},
		{"r:10;p:100", true, "r:10"},
		{"r:10;p:100;a:b", true, "r:10;a:b"},

		// unsampled
		{"r:63;p:1", false, ""},
		{"r:10;p:1", false, "r:10"},
		{"r:10;p:1;a:b", false, "r:10;a:b"},
	} {
		testInvalid := func(t *testing.T, isChildContext bool) {
			traceID, _ := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
			traceState, err := trace.TraceState{}.Insert(traceStateKey, invalid.in)
			require.NoError(t, err)

			sccfg := trace.SpanContextConfig{
				TraceState: traceState,
			}
			if isChildContext {
				spanID, _ := trace.SpanIDFromHex("00f067aa0ba902b7")

				sccfg.TraceID = traceID
				sccfg.SpanID = spanID
				// Note: the other branch is testing a fabricated
				// situation where the context has a tracestate and
				// no TraceID.
			}
			if invalid.sampled {
				sccfg.TraceFlags = trace.FlagsSampled
			}

			parentCtx := trace.ContextWithSpanContext(
				context.Background(),
				trace.NewSpanContext(sccfg),
			)

			result := parent.ShouldSample(
				sdktrace.SamplingParameters{
					ParentContext: parentCtx,
					TraceID:       sccfg.TraceID,
					Name:          "test",
					Kind:          trace.SpanKindServer,
				},
			)

			if isChildContext && invalid.sampled {
				require.Equal(t, sdktrace.RecordAndSample, result.Decision)
			} else {
				// if we're not a child context, ShouldSample
				// falls through to the delegate, which is NeverSample.
				require.Equal(t, sdktrace.Drop, result.Decision)
			}
			require.Equal(t, []attribute.KeyValue(nil), result.Attributes)
			require.Equal(t, invalid.expect, result.Tracestate.Get(traceStateKey))
		}

		t.Run(testName(invalid.in)+"_with_parent", func(t *testing.T) {
			testInvalid(t, true)
		})
		t.Run(testName(invalid.in)+"_no_parent", func(t *testing.T) {
			testInvalid(t, false)
		})
	}
}
