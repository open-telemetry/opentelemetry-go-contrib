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
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"

	jaeger_api_v2 "go.opentelemetry.io/contrib/samplers/jaegerremote/internal/proto-gen/jaeger-idl/proto/api_v2"
	"go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const (
	testOperationName          = "op"
	testFirstTimeOperationName = "firstTimeOp"

	testDefaultSamplingProbability = 0.5
	testMaxID                      = uint64(1) << 62
	testDefaultMaxOperations       = 10
)

func TestProbabilisticSampler(t *testing.T) {
	var traceID oteltrace.TraceID

	sampler := newProbabilisticSampler(0.5)
	binary.BigEndian.PutUint64(traceID[:], testMaxID+10)
	result := sampler.ShouldSample(trace.SamplingParameters{TraceID: traceID})
	assert.Equal(t, trace.Drop, result.Decision)
	binary.BigEndian.PutUint64(traceID[:], testMaxID-20)
	result = sampler.ShouldSample(trace.SamplingParameters{TraceID: traceID})
	assert.Equal(t, trace.RecordAndSample, result.Decision)

	t.Run("test_64bit_id", func(t *testing.T) {
		binary.BigEndian.PutUint64(traceID[:], (testMaxID+10)|1<<63)
		result = sampler.ShouldSample(trace.SamplingParameters{TraceID: traceID})
		assert.Equal(t, trace.Drop, result.Decision)
		binary.BigEndian.PutUint64(traceID[:], (testMaxID-20)|1<<63)
		result = sampler.ShouldSample(trace.SamplingParameters{TraceID: traceID})
		assert.Equal(t, trace.RecordAndSample, result.Decision)
	})
}

func TestRateLimitingSampler(t *testing.T) {
	sampler := newRateLimitingSampler(2)
	result := sampler.ShouldSample(trace.SamplingParameters{Name: testOperationName})
	assert.Equal(t, trace.RecordAndSample, result.Decision)
	result = sampler.ShouldSample(trace.SamplingParameters{Name: testOperationName})
	assert.Equal(t, trace.RecordAndSample, result.Decision)
	result = sampler.ShouldSample(trace.SamplingParameters{Name: testOperationName})
	assert.Equal(t, trace.Drop, result.Decision)

	sampler = newRateLimitingSampler(0.1)
	result = sampler.ShouldSample(trace.SamplingParameters{Name: testOperationName})
	assert.Equal(t, trace.RecordAndSample, result.Decision)
	result = sampler.ShouldSample(trace.SamplingParameters{Name: testOperationName})
	assert.Equal(t, trace.Drop, result.Decision)

	sampler = newRateLimitingSampler(0)
	result = sampler.ShouldSample(trace.SamplingParameters{Name: testOperationName})
	assert.Equal(t, trace.Drop, result.Decision)
}

func TestGuaranteedThroughputProbabilisticSamplerUpdate(t *testing.T) {
	samplingRate := 0.5
	lowerBound := 2.0
	sampler := newGuaranteedThroughputProbabilisticSampler(lowerBound, samplingRate)
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
	})
	assert.Equal(t, 42, sampler.maxOperations)

	sampler = newPerOperationSampler(perOperationSamplerParams{Strategies: strategies})
	assert.Equal(t, 2000, sampler.maxOperations, "default MaxOperations applied")

	sampler = newPerOperationSampler(perOperationSamplerParams{
		MaxOperations: testDefaultMaxOperations,
		Strategies:    strategies,
	})

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
	})
	assert.Equal(t, 0.0, sampler.samplers[testOperationName].samplingRate)

	strategies.PerOperationStrategies[0].ProbabilisticSampling.SamplingRate = 1.1
	sampler = newPerOperationSampler(perOperationSamplerParams{
		MaxOperations: testDefaultMaxOperations,
		Strategies:    strategies,
	})
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
	})

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
	})

	result := sampler.ShouldSample(makeSamplingParameters(testMaxID-10, testFirstTimeOperationName))
	assert.Equal(t, trace.RecordAndSample, result.Decision)
}
