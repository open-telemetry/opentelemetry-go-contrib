// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Copyright (c) 2021 The Jaeger Authors.
// Copyright (c) 2017 Uber Technologies, Inc.
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

package jaegerremote

import (
	crand "crypto/rand"
	"encoding/binary"
	"math"
	"math/rand"
	"strings"
	"testing"

	jaeger_api_v2 "github.com/jaegertracing/jaeger-idl/proto-gen/api_v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/x"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const (
	testOperationName          = "op"
	testFirstTimeOperationName = "firstTimeOp"

	testDefaultSamplingProbability = 0.5
	testMaxID                      = uint64(1) << 63
	testDefaultMaxOperations       = 10
)

type randomIDGenerator struct {
	randSource *rand.Rand
}

// NewTraceID returns a non-zero trace ID from a randomly-chosen sequence.
func (gen *randomIDGenerator) NewTraceID() oteltrace.TraceID {
	tid := oteltrace.TraceID{}
	for {
		_, _ = gen.randSource.Read(tid[:])
		if tid.IsValid() {
			break
		}
	}
	return tid
}

func defaultIDGenerator() *randomIDGenerator {
	gen := &randomIDGenerator{}
	var rngSeed int64
	_ = binary.Read(crand.Reader, binary.LittleEndian, &rngSeed)
	gen.randSource = rand.New(rand.NewSource(rngSeed))
	return gen
}

func TestProbabilisticSampler(t *testing.T) {
	var traceID oteltrace.TraceID

	sampler := newProbabilisticSampler(0.5, false, false)
	binary.BigEndian.PutUint64(traceID[8:], testMaxID+10)
	result := sampler.ShouldSample(trace.SamplingParameters{TraceID: traceID})
	assert.Equal(t, trace.Drop, result.Decision)
	binary.BigEndian.PutUint64(traceID[8:], testMaxID-20)
	result = sampler.ShouldSample(trace.SamplingParameters{TraceID: traceID})
	assert.Equal(t, trace.RecordAndSample, result.Decision)

	t.Run("test_64bit_id", func(t *testing.T) {
		binary.BigEndian.PutUint64(traceID[:8], math.MaxUint64)
		binary.BigEndian.PutUint64(traceID[8:], testMaxID+10)
		result = sampler.ShouldSample(trace.SamplingParameters{TraceID: traceID})
		assert.Equal(t, trace.Drop, result.Decision)
		binary.BigEndian.PutUint64(traceID[8:], testMaxID-20)
		result = sampler.ShouldSample(trace.SamplingParameters{TraceID: traceID})
		assert.Equal(t, trace.RecordAndSample, result.Decision)
	})

	t.Run("test_parity", func(t *testing.T) {
		for _, tc := range []struct {
			name                string
			probabilitySampling bool
			oracle              trace.Sampler
		}{
			{name: "TraceIDRatioBased", oracle: trace.TraceIDRatioBased(0.5)},
			{name: "ProbabilitySampler", probabilitySampling: true, oracle: x.ProbabilitySampler(0.5)},
		} {
			t.Run(tc.name, func(t *testing.T) {
				numTests := 1000

				sampler := newProbabilisticSampler(0.5, true, tc.probabilitySampling)
				assert.Equal(t, tc.oracle.Description(), sampler.Description())
				idGenerator := defaultIDGenerator()

				for range numTests {
					traceID := idGenerator.NewTraceID()
					assert.Equal(
						t,
						tc.oracle.ShouldSample(trace.SamplingParameters{TraceID: traceID}),
						sampler.ShouldSample(trace.SamplingParameters{TraceID: traceID}),
					)
				}
			})
		}
	})

	t.Run("Equals", func(t *testing.T) {
		sampler := newProbabilisticSampler(0.5, false, false)
		assert.True(t, sampler.Equal(newProbabilisticSampler(0.5, false, false)))
		assert.False(t, sampler.Equal(newProbabilisticSampler(0.0, false, false)))
		assert.False(t, sampler.Equal(newProbabilisticSampler(0.75, false, false)))
		assert.False(t, sampler.Equal(newProbabilisticSampler(1.0, false, false)))
	})
}

func TestRateLimitingSampler(t *testing.T) {
	sampler := newRateLimitingSampler(2, false)
	result := sampler.ShouldSample(trace.SamplingParameters{Name: testOperationName})
	assert.Equal(t, trace.RecordAndSample, result.Decision)
	result = sampler.ShouldSample(trace.SamplingParameters{Name: testOperationName})
	assert.Equal(t, trace.RecordAndSample, result.Decision)
	result = sampler.ShouldSample(trace.SamplingParameters{Name: testOperationName})
	assert.Equal(t, trace.Drop, result.Decision)

	sampler = newRateLimitingSampler(0.1, false)
	result = sampler.ShouldSample(trace.SamplingParameters{Name: testOperationName})
	assert.Equal(t, trace.RecordAndSample, result.Decision)
	result = sampler.ShouldSample(trace.SamplingParameters{Name: testOperationName})
	assert.Equal(t, trace.Drop, result.Decision)

	sampler = newRateLimitingSampler(0, false)
	result = sampler.ShouldSample(trace.SamplingParameters{Name: testOperationName})
	assert.Equal(t, trace.Drop, result.Decision)
}

func TestGuaranteedThroughputProbabilisticSamplerUpdate(t *testing.T) {
	samplingRate := 0.5
	lowerBound := 2.0
	sampler := newGuaranteedThroughputProbabilisticSampler(lowerBound, samplingRate, false, false)
	assert.Equal(t, lowerBound, sampler.lowerBound)
	assert.Equal(t, samplingRate, sampler.samplingRate)

	newSamplingRate := 0.6
	newLowerBound := 1.0
	sampler.update(newLowerBound, newSamplingRate)
	assert.Equal(t, newLowerBound, sampler.lowerBound)
	assert.Equal(t, newSamplingRate, sampler.samplingRate)

	newSamplingRate = 1.1
	sampler.update(newLowerBound, newSamplingRate)
	assert.Equal(t, 1.0, sampler.samplingRate)
}

func TestAdaptiveSampler(t *testing.T) {
	samplingRates := []*jaeger_api_v2.OperationSamplingStrategy{
		{
			Operation:             testOperationName,
			ProbabilisticSampling: &jaeger_api_v2.ProbabilisticSamplingStrategy{SamplingRate: testDefaultSamplingProbability},
		},
	}
	strategies := &jaeger_api_v2.PerOperationSamplingStrategies{
		DefaultSamplingProbability:       testDefaultSamplingProbability,
		DefaultLowerBoundTracesPerSecond: 1.0,
		PerOperationStrategies:           samplingRates,
	}

	sampler := newPerOperationSampler(perOperationSamplerParams{
		Strategies:    strategies,
		MaxOperations: 42,
	}, false, false)
	assert.Equal(t, 42, sampler.maxOperations)

	sampler = newPerOperationSampler(perOperationSamplerParams{Strategies: strategies}, false, false)
	assert.Equal(t, 2000, sampler.maxOperations, "default MaxOperations applied")

	sampler = newPerOperationSampler(perOperationSamplerParams{
		MaxOperations: testDefaultMaxOperations,
		Strategies:    strategies,
	}, false, false)

	result := sampler.ShouldSample(makeSamplingParameters(testMaxID+10, testOperationName))
	assert.Equal(t, trace.RecordAndSample, result.Decision)

	result = sampler.ShouldSample(makeSamplingParameters(testMaxID-20, testOperationName))
	assert.Equal(t, trace.RecordAndSample, result.Decision)

	result = sampler.ShouldSample(makeSamplingParameters(testMaxID+10, testOperationName))
	assert.Equal(t, trace.Drop, result.Decision)

	// This operation is seen for the first time by the sampler
	result = sampler.ShouldSample(makeSamplingParameters(testMaxID, testFirstTimeOperationName))
	assert.Equal(t, trace.RecordAndSample, result.Decision)
}

func TestAdaptiveSamplerProbabilitySampling(t *testing.T) {
	strategies := func(operation string, defaultSamplingProbability float64) *jaeger_api_v2.PerOperationSamplingStrategies {
		return &jaeger_api_v2.PerOperationSamplingStrategies{
			DefaultSamplingProbability:       defaultSamplingProbability,
			DefaultLowerBoundTracesPerSecond: 1.0,
			PerOperationStrategies: []*jaeger_api_v2.OperationSamplingStrategy{
				{
					Operation:             operation,
					ProbabilisticSampling: &jaeger_api_v2.ProbabilisticSamplingStrategy{SamplingRate: 0.25},
				},
			},
		}
	}

	for _, tc := range []struct {
		probabilitySampling bool
		want                string
	}{
		{probabilitySampling: false, want: "TraceIDRatioBased"},
		{probabilitySampling: true, want: "ProbabilitySampler"},
	} {
		t.Run(tc.want, func(t *testing.T) {
			// Covers samplers created at construction, lazily, and on update.
			assertAlgorithm := func(t *testing.T, sampler *perOperationSampler) {
				t.Helper()
				require.Len(t, sampler.samplers, 2)
				for operation, s := range sampler.samplers {
					assert.Truef(t, strings.HasPrefix(s.probabilisticSampler.Description(), tc.want), "operation %q: %s", operation, s.probabilisticSampler.Description())
				}
				assert.True(t, strings.HasPrefix(sampler.defaultSampler.Description(), tc.want), sampler.defaultSampler.Description())
			}

			sampler := newPerOperationSampler(perOperationSamplerParams{
				MaxOperations: 2,
				Strategies:    strategies(testOperationName, testDefaultSamplingProbability),
			}, false, tc.probabilitySampling)
			_ = sampler.getSamplerForOperation(testFirstTimeOperationName)
			assertAlgorithm(t, sampler)

			// A new default probability rebuilds the default sampler.
			sampler.update(strategies("updated", 0.75))
			_ = sampler.getSamplerForOperation(testFirstTimeOperationName)
			assertAlgorithm(t, sampler)
		})
	}
}

func TestAdaptiveSamplerErrors(t *testing.T) {
	strategies := &jaeger_api_v2.PerOperationSamplingStrategies{
		DefaultSamplingProbability:       testDefaultSamplingProbability,
		DefaultLowerBoundTracesPerSecond: 2.0,
		PerOperationStrategies: []*jaeger_api_v2.OperationSamplingStrategy{
			{
				Operation:             testOperationName,
				ProbabilisticSampling: &jaeger_api_v2.ProbabilisticSamplingStrategy{SamplingRate: -0.1},
			},
		},
	}

	sampler := newPerOperationSampler(perOperationSamplerParams{
		MaxOperations: testDefaultMaxOperations,
		Strategies:    strategies,
	}, false, false)
	assert.Equal(t, 0.0, sampler.samplers[testOperationName].samplingRate)

	strategies.PerOperationStrategies[0].ProbabilisticSampling.SamplingRate = 1.1
	sampler = newPerOperationSampler(perOperationSamplerParams{
		MaxOperations: testDefaultMaxOperations,
		Strategies:    strategies,
	}, false, false)
	assert.Equal(t, 1.0, sampler.samplers[testOperationName].samplingRate)
}

func TestAdaptiveSamplerUpdate(t *testing.T) {
	samplingRate := 0.1
	lowerBound := 2.0
	samplingRates := []*jaeger_api_v2.OperationSamplingStrategy{
		{
			Operation:             testOperationName,
			ProbabilisticSampling: &jaeger_api_v2.ProbabilisticSamplingStrategy{SamplingRate: samplingRate},
		},
	}
	strategies := &jaeger_api_v2.PerOperationSamplingStrategies{
		DefaultSamplingProbability:       testDefaultSamplingProbability,
		DefaultLowerBoundTracesPerSecond: lowerBound,
		PerOperationStrategies:           samplingRates,
	}

	sampler := newPerOperationSampler(perOperationSamplerParams{
		MaxOperations: testDefaultMaxOperations,
		Strategies:    strategies,
	}, false, false)

	assert.Equal(t, lowerBound, sampler.lowerBound)
	assert.Equal(t, testDefaultSamplingProbability, sampler.defaultSampler.SamplingRate())
	assert.Len(t, sampler.samplers, 1)

	// Update the sampler with new sampling rates
	newSamplingRate := 0.2
	newLowerBound := 3.0
	newDefaultSamplingProbability := 0.1
	newSamplingRates := []*jaeger_api_v2.OperationSamplingStrategy{
		{
			Operation:             testOperationName,
			ProbabilisticSampling: &jaeger_api_v2.ProbabilisticSamplingStrategy{SamplingRate: newSamplingRate},
		},
		{
			Operation:             testFirstTimeOperationName,
			ProbabilisticSampling: &jaeger_api_v2.ProbabilisticSamplingStrategy{SamplingRate: newSamplingRate},
		},
	}
	strategies = &jaeger_api_v2.PerOperationSamplingStrategies{
		DefaultSamplingProbability:       newDefaultSamplingProbability,
		DefaultLowerBoundTracesPerSecond: newLowerBound,
		PerOperationStrategies:           newSamplingRates,
	}

	sampler.update(strategies)
	assert.Equal(t, newLowerBound, sampler.lowerBound)
	assert.Equal(t, newDefaultSamplingProbability, sampler.defaultSampler.SamplingRate())
	assert.Len(t, sampler.samplers, 2)
}

func TestMaxOperations(t *testing.T) {
	samplingRates := []*jaeger_api_v2.OperationSamplingStrategy{
		{
			Operation:             testOperationName,
			ProbabilisticSampling: &jaeger_api_v2.ProbabilisticSamplingStrategy{SamplingRate: 0.1},
		},
	}
	strategies := &jaeger_api_v2.PerOperationSamplingStrategies{
		DefaultSamplingProbability:       testDefaultSamplingProbability,
		DefaultLowerBoundTracesPerSecond: 2.0,
		PerOperationStrategies:           samplingRates,
	}

	sampler := newPerOperationSampler(perOperationSamplerParams{
		MaxOperations: 1,
		Strategies:    strategies,
	}, false, false)

	result := sampler.ShouldSample(makeSamplingParameters(testMaxID-10, testFirstTimeOperationName))
	assert.Equal(t, trace.RecordAndSample, result.Decision)
}

func TestAttributes(t *testing.T) {
	t.Parallel()

	t.Run("probabilistic", func(t *testing.T) {
		t.Parallel()

		var traceID oteltrace.TraceID
		s := newProbabilisticSampler(0.5, false, false)
		binary.BigEndian.PutUint64(traceID[:8], math.MaxUint64)
		binary.BigEndian.PutUint64(traceID[8:], testMaxID+10)
		result := s.ShouldSample(trace.SamplingParameters{TraceID: traceID})
		assert.Equal(t, trace.Drop, result.Decision)
		assert.Nil(t, result.Attributes)

		binary.BigEndian.PutUint64(traceID[8:], testMaxID-20)
		result = s.ShouldSample(trace.SamplingParameters{TraceID: traceID})
		assert.Equal(t, trace.RecordAndSample, result.Decision)
		assert.Equal(t, []attribute.KeyValue{attribute.String(samplerTypeKey, samplerTypeValueProbabilistic), attribute.Float64(samplerParamKey, 0.5)}, result.Attributes)

		s = newProbabilisticSampler(1.0, false, false)
		result = s.ShouldSample(trace.SamplingParameters{TraceID: traceID})
		assert.Equal(t, trace.RecordAndSample, result.Decision)
		assert.Equal(t, []attribute.KeyValue{attribute.String(samplerTypeKey, samplerTypeValueProbabilistic), attribute.Float64(samplerParamKey, 1.0)}, result.Attributes)
	})

	t.Run("probabilistic attributes disabled", func(t *testing.T) {
		t.Parallel()

		var traceID oteltrace.TraceID
		s := newProbabilisticSampler(0.5, true, false)
		binary.BigEndian.PutUint64(traceID[:8], math.MaxUint64)
		binary.BigEndian.PutUint64(traceID[8:], testMaxID+10)
		result := s.ShouldSample(trace.SamplingParameters{TraceID: traceID})
		assert.Equal(t, trace.Drop, result.Decision)
		assert.Nil(t, result.Attributes)

		binary.BigEndian.PutUint64(traceID[8:], testMaxID-20)
		result = s.ShouldSample(trace.SamplingParameters{TraceID: traceID})
		assert.Equal(t, trace.RecordAndSample, result.Decision)
		assert.Nil(t, result.Attributes)
	})

	t.Run("ratelimiting", func(t *testing.T) {
		t.Parallel()

		s := newRateLimitingSampler(1, false)
		result := s.ShouldSample(trace.SamplingParameters{Name: testOperationName})
		assert.Equal(t, trace.RecordAndSample, result.Decision)
		assert.Equal(t, []attribute.KeyValue{attribute.String(samplerTypeKey, samplerTypeValueRateLimiting), attribute.Float64(samplerParamKey, 1)}, result.Attributes)
		result = s.ShouldSample(trace.SamplingParameters{Name: testOperationName})
		assert.Equal(t, trace.Drop, result.Decision)
		assert.Nil(t, result.Attributes)

		s = newRateLimitingSampler(0.1, false)
		result = s.ShouldSample(trace.SamplingParameters{Name: testOperationName})
		assert.Equal(t, trace.RecordAndSample, result.Decision)
		assert.Equal(t, []attribute.KeyValue{attribute.String(samplerTypeKey, samplerTypeValueRateLimiting), attribute.Float64(samplerParamKey, 0.1)}, result.Attributes)
		result = s.ShouldSample(trace.SamplingParameters{Name: testOperationName})
		assert.Equal(t, trace.Drop, result.Decision)
		assert.Nil(t, result.Attributes)
	})

	t.Run("ratelimiting attributes disabled", func(t *testing.T) {
		t.Parallel()

		s := newRateLimitingSampler(1, true)
		result := s.ShouldSample(trace.SamplingParameters{Name: testOperationName})
		assert.Equal(t, trace.RecordAndSample, result.Decision)
		assert.Nil(t, result.Attributes)
		result = s.ShouldSample(trace.SamplingParameters{Name: testOperationName})
		assert.Equal(t, trace.Drop, result.Decision)
		assert.Nil(t, result.Attributes)
	})

	t.Run("per operation", func(t *testing.T) {
		t.Parallel()

		samplingRates := []*jaeger_api_v2.OperationSamplingStrategy{
			{
				Operation:             testOperationName,
				ProbabilisticSampling: &jaeger_api_v2.ProbabilisticSamplingStrategy{SamplingRate: testDefaultSamplingProbability},
			},
		}
		strategies := &jaeger_api_v2.PerOperationSamplingStrategies{
			DefaultSamplingProbability:       testDefaultSamplingProbability,
			DefaultLowerBoundTracesPerSecond: 1.0,
			PerOperationStrategies:           samplingRates,
		}
		s := newPerOperationSampler(perOperationSamplerParams{
			MaxOperations: testDefaultMaxOperations,
			Strategies:    strategies,
		}, false, false)

		result := s.ShouldSample(makeSamplingParameters(testMaxID+10, testOperationName))
		assert.Equal(t, trace.RecordAndSample, result.Decision)
		assert.Equal(t, []attribute.KeyValue{attribute.String(samplerTypeKey, samplerTypeValueRateLimiting), attribute.Float64(samplerParamKey, 1)}, result.Attributes)

		result = s.ShouldSample(makeSamplingParameters(testMaxID-20, testOperationName))
		assert.Equal(t, trace.RecordAndSample, result.Decision)
		assert.Equal(t, []attribute.KeyValue{attribute.String(samplerTypeKey, samplerTypeValueProbabilistic), attribute.Float64(samplerParamKey, 0.5)}, result.Attributes)

		result = s.ShouldSample(makeSamplingParameters(testMaxID+10, testOperationName))
		assert.Equal(t, trace.Drop, result.Decision)
		assert.Nil(t, result.Attributes)

		// This operation is seen for the first time by the s
		result = s.ShouldSample(makeSamplingParameters(testMaxID, testFirstTimeOperationName))
		assert.Equal(t, trace.RecordAndSample, result.Decision)
		assert.Equal(t, []attribute.KeyValue{attribute.String(samplerTypeKey, samplerTypeValueRateLimiting), attribute.Float64(samplerParamKey, 1)}, result.Attributes)
	})

	t.Run("per operation attributes disabled", func(t *testing.T) {
		t.Parallel()

		samplingRates := []*jaeger_api_v2.OperationSamplingStrategy{
			{
				Operation:             testOperationName,
				ProbabilisticSampling: &jaeger_api_v2.ProbabilisticSamplingStrategy{SamplingRate: testDefaultSamplingProbability},
			},
		}
		strategies := &jaeger_api_v2.PerOperationSamplingStrategies{
			DefaultSamplingProbability:       testDefaultSamplingProbability,
			DefaultLowerBoundTracesPerSecond: 1.0,
			PerOperationStrategies:           samplingRates,
		}
		s := newPerOperationSampler(perOperationSamplerParams{
			MaxOperations: testDefaultMaxOperations,
			Strategies:    strategies,
		}, true, false)

		result := s.ShouldSample(makeSamplingParameters(testMaxID+10, testOperationName))
		assert.Equal(t, trace.RecordAndSample, result.Decision)
		assert.Nil(t, result.Attributes)

		result = s.ShouldSample(makeSamplingParameters(testMaxID-20, testOperationName))
		assert.Equal(t, trace.RecordAndSample, result.Decision)
		assert.Nil(t, result.Attributes)

		result = s.ShouldSample(makeSamplingParameters(testMaxID+10, testOperationName))
		assert.Equal(t, trace.Drop, result.Decision)
		assert.Nil(t, result.Attributes)

		// This operation is seen for the first time by the s
		result = s.ShouldSample(makeSamplingParameters(testMaxID, testFirstTimeOperationName))
		assert.Equal(t, trace.RecordAndSample, result.Decision)
		assert.Nil(t, result.Attributes)
	})
}

const (
	// Randomness values just at and below a 50% sampler's threshold.
	testSampledRandomness = uint64(1) << 55
	testDroppedRandomness = uint64(0)

	// testHalfThreshold is the "th" sub-value a 50% probability sampler records.
	testHalfThreshold = "th:8"

	// testRandomnessValue is the "rv" sub-key for testSampledRandomness.
	testRandomnessValue = "rv:80000000000000"
)

var testSpanID = oteltrace.SpanID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}

// traceIDWithRandomness returns a trace ID whose low 56 bits are r.
func traceIDWithRandomness(r uint64) oteltrace.TraceID {
	tid := oteltrace.TraceID{0x01}
	binary.BigEndian.PutUint64(tid[8:], r)
	return tid
}

// samplingParameters builds parameters for the child of a sampled parent with
// randomness r and "ot" tracestate entry otts (unset when empty).
func samplingParameters(t testing.TB, r uint64, otts string) trace.SamplingParameters {
	t.Helper()

	ts := oteltrace.TraceState{}
	if otts != "" {
		var err error
		ts, err = ts.Insert(traceStateKey, otts)
		require.NoError(t, err)
	}

	traceID := traceIDWithRandomness(r)
	sc := oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     testSpanID,
		TraceFlags: oteltrace.FlagsSampled,
		TraceState: ts,
	})
	return trace.SamplingParameters{
		ParentContext: oteltrace.ContextWithSpanContext(t.Context(), sc),
		TraceID:       traceID,
		Name:          testOperationName,
	}
}

func TestProbabilisticSamplerTraceState(t *testing.T) {
	sampler := newProbabilisticSampler(testDefaultSamplingProbability, true, true)

	t.Run("records threshold when sampled", func(t *testing.T) {
		result := sampler.ShouldSample(samplingParameters(t, testSampledRandomness, ""))
		require.Equal(t, trace.RecordAndSample, result.Decision)
		assert.Equal(t, testHalfThreshold, result.Tracestate.Get(traceStateKey))
	})

	t.Run("replaces an inbound threshold", func(t *testing.T) {
		result := sampler.ShouldSample(samplingParameters(t, testSampledRandomness, "th:1;foo:bar"))
		require.Equal(t, trace.RecordAndSample, result.Decision)
		assert.Equal(t, testHalfThreshold+";foo:bar", result.Tracestate.Get(traceStateKey))
	})

	t.Run("prefers explicit randomness over the trace ID", func(t *testing.T) {
		// rv overrides a trace ID that would be dropped.
		result := sampler.ShouldSample(samplingParameters(t, testDroppedRandomness, testRandomnessValue))
		require.Equal(t, trace.RecordAndSample, result.Decision)
		assert.Equal(t, testHalfThreshold+";"+testRandomnessValue, result.Tracestate.Get(traceStateKey))

		// And one that would be sampled.
		result = sampler.ShouldSample(samplingParameters(t, testSampledRandomness, "rv:00000000000000"))
		assert.Equal(t, trace.Drop, result.Decision)
	})

	t.Run("always samples at probability one", func(t *testing.T) {
		result := newProbabilisticSampler(1.0, true, true).ShouldSample(samplingParameters(t, testDroppedRandomness, ""))
		require.Equal(t, trace.RecordAndSample, result.Decision)
		assert.Equal(t, "th:0", result.Tracestate.Get(traceStateKey))
	})

	t.Run("never samples at probability zero", func(t *testing.T) {
		result := newProbabilisticSampler(0.0, true, true).ShouldSample(samplingParameters(t, testSampledRandomness, ""))
		assert.Equal(t, trace.Drop, result.Decision)
	})

	t.Run("leaves the tracestate untouched when disabled", func(t *testing.T) {
		sampler := newProbabilisticSampler(1.0, true, false)
		result := sampler.ShouldSample(samplingParameters(t, testDroppedRandomness, "th:1;"+testRandomnessValue))
		require.Equal(t, trace.RecordAndSample, result.Decision)
		assert.Equal(t, "th:1;"+testRandomnessValue, result.Tracestate.Get(traceStateKey))
	})
}

func TestRateLimitingSamplerClearsThreshold(t *testing.T) {
	sampler := newRateLimitingSampler(1, true)

	result := sampler.ShouldSample(samplingParameters(t, testSampledRandomness, testHalfThreshold+";"+testRandomnessValue))
	require.Equal(t, trace.RecordAndSample, result.Decision)
	assert.Equal(t, testRandomnessValue, result.Tracestate.Get(traceStateKey))

	// Out of credit.
	result = sampler.ShouldSample(samplingParameters(t, testSampledRandomness, testHalfThreshold+";"+testRandomnessValue))
	require.Equal(t, trace.Drop, result.Decision)
	assert.Equal(t, testRandomnessValue, result.Tracestate.Get(traceStateKey))

	t.Run("drops the ot entry when the threshold was its only sub-key", func(t *testing.T) {
		sampler := newRateLimitingSampler(1, true)
		result := sampler.ShouldSample(samplingParameters(t, testSampledRandomness, testHalfThreshold))
		require.Equal(t, trace.RecordAndSample, result.Decision)
		assert.Empty(t, result.Tracestate.Get(traceStateKey))
	})
}

func TestGuaranteedThroughputProbabilisticSamplerTraceState(t *testing.T) {
	t.Run("probabilistic decision records the threshold", func(t *testing.T) {
		sampler := newGuaranteedThroughputProbabilisticSampler(1, testDefaultSamplingProbability, true, true)
		result := sampler.ShouldSample(samplingParameters(t, testSampledRandomness, "th:1"))
		require.Equal(t, trace.RecordAndSample, result.Decision)
		assert.Equal(t, testHalfThreshold, result.Tracestate.Get(traceStateKey))
	})

	t.Run("lower bound decision clears the threshold", func(t *testing.T) {
		// Zero probability forces the lower bound limiter to decide.
		sampler := newGuaranteedThroughputProbabilisticSampler(1, 0, true, true)
		result := sampler.ShouldSample(samplingParameters(t, testSampledRandomness, testHalfThreshold+";"+testRandomnessValue))
		require.Equal(t, trace.RecordAndSample, result.Decision)
		assert.Equal(t, testRandomnessValue, result.Tracestate.Get(traceStateKey))
	})
}
