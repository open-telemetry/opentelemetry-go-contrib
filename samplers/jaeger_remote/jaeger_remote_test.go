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

package jaeger_remote

import (
	"testing"

	jaeger_api_v2 "github.com/jaegertracing/jaeger/proto-gen/api_v2"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/sdk/trace"
)

func TestJaegerRemoteSampler_ShouldSample_probabilistic(t *testing.T) {
	genProbabilisticStrategy := func(fraction float64) jaeger_api_v2.SamplingStrategyResponse {
		return jaeger_api_v2.SamplingStrategyResponse{
			StrategyType: jaeger_api_v2.SamplingStrategyType_PROBABILISTIC,
			ProbabilisticSampling: &jaeger_api_v2.ProbabilisticSamplingStrategy{
				SamplingRate: fraction,
			},
		}
	}

	jaegerRemoteSampler := New().(*sampler)

	// set fraction to 0, this should drop every trace
	err := jaegerRemoteSampler.loadSamplingStrategies(genProbabilisticStrategy(0))
	assert.NoError(t, err)

	traceID := [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}

	result := jaegerRemoteSampler.ShouldSample(trace.SamplingParameters{
		TraceID: traceID,
	})
	assert.Equal(t, trace.Drop, result.Decision)

	// set fraction to 0, this should sample every trace
	err = jaegerRemoteSampler.loadSamplingStrategies(genProbabilisticStrategy(1))
	assert.NoError(t, err)

	traceID = [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2}

	result = jaegerRemoteSampler.ShouldSample(trace.SamplingParameters{
		TraceID: traceID,
	})
	assert.Equal(t, trace.RecordAndSample, result.Decision)
}

func TestJaegerRemoteSampler_ShouldSample_rateLimiting(t *testing.T) {
	rateLimitingStrategy := jaeger_api_v2.SamplingStrategyResponse{
		StrategyType: jaeger_api_v2.SamplingStrategyType_RATE_LIMITING,
		RateLimitingSampling: &jaeger_api_v2.RateLimitingSamplingStrategy{
			MaxTracesPerSecond: 100,
		},
	}

	jaegerRemoteSampler := New().(*sampler)

	err := jaegerRemoteSampler.loadSamplingStrategies(rateLimitingStrategy)
	assert.Error(t, err)
}

func TestJaegerRemoteSampler_ShouldSample_perOperation(t *testing.T) {
	perOperationStrategy := jaeger_api_v2.SamplingStrategyResponse{
		StrategyType: jaeger_api_v2.SamplingStrategyType_PROBABILISTIC,
		ProbabilisticSampling: &jaeger_api_v2.ProbabilisticSamplingStrategy{
			SamplingRate: 1,
		},
		OperationSampling: &jaeger_api_v2.PerOperationSamplingStrategies{
			DefaultSamplingProbability: 0.1,
			PerOperationStrategies: []*jaeger_api_v2.OperationSamplingStrategy{
				{
					Operation: "test",
					ProbabilisticSampling: &jaeger_api_v2.ProbabilisticSamplingStrategy{
						SamplingRate: 1,
					},
				},
			},
		},
	}

	jaegerRemoteSampler := New().(*sampler)

	err := jaegerRemoteSampler.loadSamplingStrategies(perOperationStrategy)
	assert.Error(t, err)
}

func TestJaegerRemoteSampler_updateSamplingStrategies(t *testing.T) {
	jaegerRemoteSampler := New().(*sampler)

	defaultSampler := trace.TraceIDRatioBased(defaultSamplingRate)
	assert.Equal(t, defaultSampler, jaegerRemoteSampler.sampler)

	tests := []struct {
		name        string
		strategy    jaeger_api_v2.SamplingStrategyResponse
		expectedErr string
		sampler     trace.Sampler
	}{
		{
			name:     "update strategy without changes",
			strategy: jaeger_api_v2.SamplingStrategyResponse{},
			sampler:  defaultSampler,
		},
		{
			name: "update strategy with PROBABILISTIC and sampling rate 0.8",
			strategy: jaeger_api_v2.SamplingStrategyResponse{
				StrategyType: jaeger_api_v2.SamplingStrategyType_PROBABILISTIC,
				ProbabilisticSampling: &jaeger_api_v2.ProbabilisticSamplingStrategy{
					SamplingRate: 0.8,
				},
			},
			sampler: trace.TraceIDRatioBased(0.8),
		},
		{
			name: "update strategy with RATE_LIMITING",
			strategy: jaeger_api_v2.SamplingStrategyResponse{
				StrategyType: jaeger_api_v2.SamplingStrategyType_RATE_LIMITING,
				RateLimitingSampling: &jaeger_api_v2.RateLimitingSamplingStrategy{
					MaxTracesPerSecond: 100,
				},
			},
			expectedErr: "loading failed: only strategy type PROBABILISTC is supported, got RATE_LIMITING",
			sampler:     trace.TraceIDRatioBased(0.8),
		},
		{
			name: "update strategy with per operation sampling",
			strategy: jaeger_api_v2.SamplingStrategyResponse{
				StrategyType: jaeger_api_v2.SamplingStrategyType_PROBABILISTIC,
				ProbabilisticSampling: &jaeger_api_v2.ProbabilisticSamplingStrategy{
					SamplingRate: 1,
				},
				OperationSampling: &jaeger_api_v2.PerOperationSamplingStrategies{
					DefaultSamplingProbability: 1,
				},
			},
			expectedErr: "loading failed: per operation sampling is not supported",
			sampler:     trace.TraceIDRatioBased(0.8),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jaegerRemoteSampler.fetcher = mockStrategyFetcher{
				response: tt.strategy,
			}

			err := jaegerRemoteSampler.updateSamplingStrategies()
			// TODO this feels awkward
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.sampler, jaegerRemoteSampler.sampler)
		})
	}
}

type mockStrategyFetcher struct {
	response jaeger_api_v2.SamplingStrategyResponse
	err      error
}

func (m mockStrategyFetcher) Fetch() (jaeger_api_v2.SamplingStrategyResponse, error) {
	return m.response, m.err
}
