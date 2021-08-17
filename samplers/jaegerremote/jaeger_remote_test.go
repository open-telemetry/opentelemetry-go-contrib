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

package jaegerremote

import (
	"testing"

	"github.com/stretchr/testify/assert"

	jaeger_api_v2 "go.opentelemetry.io/contrib/samplers/jaegerremote/internal/proto-gen/jaeger-idl/proto/api_v2"
	"go.opentelemetry.io/otel/sdk/trace"
)

func Test_sampler_ShouldSample_probabilistic(t *testing.T) {
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
	jaegerRemoteSampler.fetcher = mockStrategyFetcher{
		response: genProbabilisticStrategy(0),
	}

	err := jaegerRemoteSampler.updateSamplingStrategies()
	assert.NoError(t, err)

	traceID := [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}

	result := jaegerRemoteSampler.ShouldSample(trace.SamplingParameters{
		TraceID: traceID,
	})
	assert.Equal(t, trace.Drop, result.Decision)

	// set fraction to 0, this should sample every trace
	jaegerRemoteSampler.fetcher = mockStrategyFetcher{
		response: genProbabilisticStrategy(1),
	}

	err = jaegerRemoteSampler.updateSamplingStrategies()
	assert.NoError(t, err)

	traceID = [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2}

	result = jaegerRemoteSampler.ShouldSample(trace.SamplingParameters{
		TraceID: traceID,
	})
	assert.Equal(t, trace.RecordAndSample, result.Decision)
}

func Test_sampler_updateSamplingStrategies(t *testing.T) {
	jaegerRemoteSampler := New().(*sampler)

	defaultSampler := trace.TraceIDRatioBased(defaultSamplingRate)
	assert.Equal(t, defaultSampler, jaegerRemoteSampler.sampler)

	tests := []struct {
		name      string
		strategy  jaeger_api_v2.SamplingStrategyResponse
		expectErr bool
		sampler   trace.Sampler
	}{
		{
			name:     "no change, sampler stays the same",
			strategy: jaeger_api_v2.SamplingStrategyResponse{},
			sampler:  defaultSampler,
		},
		{
			name: "new strategy, sampler is updated",
			strategy: jaeger_api_v2.SamplingStrategyResponse{
				StrategyType: jaeger_api_v2.SamplingStrategyType_PROBABILISTIC,
				ProbabilisticSampling: &jaeger_api_v2.ProbabilisticSamplingStrategy{
					SamplingRate: 0.8,
				},
			},
			sampler: trace.TraceIDRatioBased(0.8),
		},
		{
			name: "strategy with RATE_LIMITING, update fails and sampler stays the same",
			strategy: jaeger_api_v2.SamplingStrategyResponse{
				StrategyType: jaeger_api_v2.SamplingStrategyType_RATE_LIMITING,
				RateLimitingSampling: &jaeger_api_v2.RateLimitingSamplingStrategy{
					MaxTracesPerSecond: 100,
				},
			},
			expectErr: true,
			sampler:   trace.TraceIDRatioBased(0.8),
		},
		{
			name: "strategy with per operation sampling, update fails and sampler stays the same",
			strategy: jaeger_api_v2.SamplingStrategyResponse{
				StrategyType: jaeger_api_v2.SamplingStrategyType_PROBABILISTIC,
				ProbabilisticSampling: &jaeger_api_v2.ProbabilisticSamplingStrategy{
					SamplingRate: 1,
				},
				OperationSampling: &jaeger_api_v2.PerOperationSamplingStrategies{
					DefaultSamplingProbability: 1,
				},
			},
			sampler: &perOperationSampler{
				defaultSampler: trace.TraceIDRatioBased(1),
				operationMap:   map[string]trace.Sampler{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jaegerRemoteSampler.fetcher = mockStrategyFetcher{
				response: tt.strategy,
			}

			err := jaegerRemoteSampler.updateSamplingStrategies()

			if tt.expectErr {
				assert.Error(t, err)
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

var _ samplingStrategyFetcher = mockStrategyFetcher{}

func (m mockStrategyFetcher) Fetch() (jaeger_api_v2.SamplingStrategyResponse, error) {
	return m.response, m.err
}
