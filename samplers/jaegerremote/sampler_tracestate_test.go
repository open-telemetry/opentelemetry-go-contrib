// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package jaegerremote

import (
	"context"
	"encoding/binary"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// contextWithTraceState builds a ParentContext carrying the given TraceID and
// (parsed) W3C tracestate, for exercising shouldSampleWithTraceState.
func contextWithTraceState(t *testing.T, traceID oteltrace.TraceID, tracestate string) context.Context {
	t.Helper()
	state, err := oteltrace.ParseTraceState(tracestate)
	require.NoError(t, err)
	return oteltrace.ContextWithSpanContext(t.Context(), oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID:    traceID,
		TraceState: state,
	}))
}

func TestProbabilisticSamplerTraceStateThreshold(t *testing.T) {
	for _, tc := range []struct {
		prob      float64
		threshold uint64
	}{
		{0.5, 0x80000000000000},
		{1 / 3.0, 0xaaab0000000000},
		{2 / 3.0, 0x55550000000000},
		{1, 0},
		{1.5, 0},
		{0, maxAdjustedCount},
	} {
		s := newProbabilisticSampler(tc.prob, false, true)
		assert.Equal(t, tc.threshold, s.threshold, "probability %g", tc.prob)
	}
}

func TestProbabilisticSamplerTraceStateShouldSample(t *testing.T) {
	const threshold50 = uint64(0x80000000000000)

	t.Run("disabled path is unaffected", func(t *testing.T) {
		s := newProbabilisticSampler(0.5, false, false)
		var traceID oteltrace.TraceID
		binary.BigEndian.PutUint64(traceID[8:], threshold50)
		result := s.ShouldSample(trace.SamplingParameters{TraceID: traceID})
		assert.Equal(t, trace.RecordAndSample, result.Decision)
		assert.Empty(t, result.Tracestate.Get("ot"))
	})

	t.Run("root span sample and drop via trace ID randomness", func(t *testing.T) {
		s := newProbabilisticSampler(0.5, false, true)

		var traceIDSample oteltrace.TraceID
		binary.BigEndian.PutUint64(traceIDSample[8:], threshold50)
		result := s.ShouldSample(trace.SamplingParameters{TraceID: traceIDSample})
		assert.Equal(t, trace.RecordAndSample, result.Decision)
		ot := result.Tracestate.Get("ot")
		require.NotEmpty(t, ot)
		assert.True(t, strings.HasPrefix(ot, "th:8"), "got %q", ot)

		var traceIDDrop oteltrace.TraceID
		binary.BigEndian.PutUint64(traceIDDrop[8:], threshold50-1)
		result = s.ShouldSample(trace.SamplingParameters{TraceID: traceIDDrop})
		assert.Equal(t, trace.Drop, result.Decision)
		assert.Empty(t, result.Tracestate.Get("ot"))
	})

	t.Run("existing th and other ot keys are replaced in place, vendors preserved", func(t *testing.T) {
		s := newProbabilisticSampler(0.5, false, true)
		var traceID oteltrace.TraceID
		binary.BigEndian.PutUint64(traceID[8:], threshold50)
		ctx := contextWithTraceState(t, traceID, "ot=th:0ad;other:value,vendor=v")

		result := s.ShouldSample(trace.SamplingParameters{ParentContext: ctx, TraceID: traceID})
		assert.Equal(t, trace.RecordAndSample, result.Decision)
		ot := result.Tracestate.Get("ot")
		assert.True(t, strings.HasPrefix(ot, "th:8"), "got %q", ot)
		assert.Contains(t, ot, "other:value")
		assert.Equal(t, "v", result.Tracestate.Get("vendor"))
	})

	t.Run("explicit rv in tracestate is honored over trace ID", func(t *testing.T) {
		s := newProbabilisticSampler(0.5, false, true)
		var traceID oteltrace.TraceID
		binary.BigEndian.PutUint64(traceID[8:], 1) // would drop if used directly
		ctx := contextWithTraceState(t, traceID, "ot=rv:80000000000000,vendor=value")

		result := s.ShouldSample(trace.SamplingParameters{ParentContext: ctx, TraceID: traceID})
		assert.Equal(t, trace.RecordAndSample, result.Decision, "rv value should be used for sampling decision")
		assert.Contains(t, result.Tracestate.Get("ot"), "th:")
		assert.Equal(t, "value", result.Tracestate.Get("vendor"))
	})

	t.Run("probability one uses th:0 and always samples", func(t *testing.T) {
		s := newProbabilisticSampler(1, false, true)
		var traceID oteltrace.TraceID // all zero, would drop at any positive threshold
		result := s.ShouldSample(trace.SamplingParameters{TraceID: traceID})
		assert.Equal(t, trace.RecordAndSample, result.Decision)
		assert.Equal(t, "th:0", result.Tracestate.Get("ot"))
	})
}

// TestRateLimitingDoesNotSetTracestateThreshold guards the documented
// limitation that rate-limiting decisions - including the guaranteed lower
// bound used by guaranteedThroughputProbabilisticSampler - never add a
// tracestate "th" value, because they are not fixed-probability decisions.
func TestRateLimitingDoesNotSetTracestateThreshold(t *testing.T) {
	t.Run("standalone rate limiting sampler", func(t *testing.T) {
		s := newRateLimitingSampler(1000, false)
		result := s.ShouldSample(trace.SamplingParameters{Name: testOperationName})
		assert.Equal(t, trace.RecordAndSample, result.Decision)
		assert.Empty(t, result.Tracestate.Get("ot"))
	})

	t.Run("guaranteed throughput fallback", func(t *testing.T) {
		// samplingRate 0 with traceStateSamplingEnabled makes the probabilistic
		// branch always drop, forcing the rate-limiting lower bound to decide.
		gt := newGuaranteedThroughputProbabilisticSampler(1000, 0, false, true)
		var traceID oteltrace.TraceID
		result := gt.ShouldSample(trace.SamplingParameters{TraceID: traceID})
		assert.Equal(t, trace.RecordAndSample, result.Decision)
		assert.Empty(t, result.Tracestate.Get("ot"))
	})
}

func TestWithTraceStateSamplingEnabled(t *testing.T) {
	c := newConfig(WithTraceStateSamplingEnabled())
	require.True(t, c.traceStateSamplingEnabled)
	sampler, ok := c.sampler.(*probabilisticSampler)
	require.True(t, ok)
	assert.True(t, sampler.traceStateSamplingEnabled)
}

func BenchmarkProbabilisticSamplerShouldSample(b *testing.B) {
	var traceIDSample oteltrace.TraceID
	binary.BigEndian.PutUint64(traceIDSample[8:], uint64(0x80000000000000))
	params := trace.SamplingParameters{TraceID: traceIDSample}

	cases := []struct {
		name    string
		sampler *probabilisticSampler
	}{
		{"trace_state_disabled", newProbabilisticSampler(0.5, false, false)},
		{"trace_state_enabled", newProbabilisticSampler(0.5, false, true)},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = tc.sampler.ShouldSample(params)
			}
		})
	}
}
