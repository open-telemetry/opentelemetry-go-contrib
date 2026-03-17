// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package traceidratio

import (
	"context"
	"math/rand"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func TestTraceIDRatioBased(t *testing.T) {
	t.Run("description", func(t *testing.T) {
		for _, tc := range []struct {
			prob float64
			desc string
		}{
			{0.5, "TraceIDRatioBased{0.5}"},
			{1. / 3, "TraceIDRatioBased{0.3333333333333333}"},
			{1. / 10000, "TraceIDRatioBased{0.0001}"},
			{1, "AlwaysOnSampler"},
			{1.5, "AlwaysOnSampler"},
			{0, "AlwaysOffSampler"},
			{-0.5, "AlwaysOffSampler"},
		} {
			require.Equal(t, tc.desc, TraceIDRatioBased(tc.prob).Description())
		}
	})

	t.Run("threshold", func(t *testing.T) {
		for _, tc := range []struct {
			prob      float64
			threshold uint64
		}{
			{0.5, 0x80000000000000},
			{1 / 3.0, 0xaaab0000000000},
			{2 / 3.0, 0x55550000000000},
			{1 / 10.0, 0xe6660000000000},
			{1 / 256.0, 0xff000000000000},
			{1 / 65536.0, 0xffff0000000000},
			{1 / 1048576.0, 0xfffff000000000},
		} {
			sampler := TraceIDRatioBased(tc.prob).(*TraceIDRatioSampler)
			require.Equal(t, tc.threshold, sampler.Threshold())
		}
	})

	t.Run("inclusive sampling", func(t *testing.T) {
		const numSamplers = 100
		const numTraces = 50
		for range numSamplers {
			ratioLo, ratioHi := rand.Float64(), rand.Float64()
			if ratioHi < ratioLo {
				ratioLo, ratioHi = ratioHi, ratioLo
			}
			samplerHi := TraceIDRatioBased(ratioHi)
			samplerLo := TraceIDRatioBased(ratioLo)
			for range numTraces {
				traceID := trace.TraceID{}
				rand.Read(traceID[:])
				params := sdktrace.SamplingParameters{
					ParentContext: trace.ContextWithSpanContext(
						context.Background(),
						trace.NewSpanContext(trace.SpanContextConfig{
							TraceID:    traceID,
							TraceFlags: trace.FlagsRandom,
						}),
					),
					TraceID: traceID,
				}
				if samplerLo.ShouldSample(params).Decision == sdktrace.RecordAndSample {
					assert.Equal(t, sdktrace.RecordAndSample, samplerHi.ShouldSample(params).Decision,
						"%s sampled but %s did not", samplerLo.Description(), samplerHi.Description())
				}
			}
		}
	})

	t.Run("RecordAndSample adds ot.th to tracestate", func(t *testing.T) {
		const traceIDWillSample = "00000000000000000080000000000000"
		sampler := TraceIDRatioBased(0.5)
		traceID, _ := trace.TraceIDFromHex(traceIDWillSample)
		spanID, _ := trace.SpanIDFromHex("00f067aa0ba902b7")
		initialState, err := trace.ParseTraceState("vendor=value")
		require.NoError(t, err)

		parentCtx := trace.ContextWithSpanContext(
			context.Background(),
			trace.NewSpanContext(trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: trace.FlagsRandom,
				TraceState: initialState,
			}),
		)
		params := sdktrace.SamplingParameters{
			ParentContext: parentCtx,
			TraceID:       traceID,
		}

		result := sampler.ShouldSample(params)

		assert.Equal(t, sdktrace.RecordAndSample, result.Decision)
		ot := result.Tracestate.Get("ot")
		require.NotEmpty(t, ot)
		assert.True(t, strings.HasPrefix(ot, "th:"), "ot value should contain th key, got %q", ot)
		assert.Equal(t, "value", result.Tracestate.Get("vendor"))
	})

	t.Run("Drop when randomness < threshold", func(t *testing.T) {
		const traceIDWillDrop = "0000000000000000007fffffffffffff"
		sampler := TraceIDRatioBased(0.5)
		traceID, _ := trace.TraceIDFromHex(traceIDWillDrop)
		spanID, _ := trace.SpanIDFromHex("00f067aa0ba902b7")
		initialState, err := trace.ParseTraceState("ot=th:0;rv:0123456789abcd,vendor=value")
		require.NoError(t, err)

		parentCtx := trace.ContextWithSpanContext(
			context.Background(),
			trace.NewSpanContext(trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: trace.FlagsRandom,
				TraceState: initialState,
			}),
		)
		params := sdktrace.SamplingParameters{
			ParentContext: parentCtx,
			TraceID:       traceID,
		}

		result := sampler.ShouldSample(params)

		assert.Equal(t, sdktrace.Drop, result.Decision)
		assert.Equal(t, initialState, result.Tracestate)
	})
}
