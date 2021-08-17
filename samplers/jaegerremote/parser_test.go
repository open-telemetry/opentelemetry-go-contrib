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

func Test_samplingStrategyParserImpl_Parse(t *testing.T) {
	tests := []struct {
		name             string
		samplingStrategy jaeger_api_v2.SamplingStrategyResponse
		expectedErr      string
		expectedSampler  trace.Sampler
	}{
		{
			name: "PROBABILISTIC only",
			samplingStrategy: jaeger_api_v2.SamplingStrategyResponse{
				StrategyType: jaeger_api_v2.SamplingStrategyType_PROBABILISTIC,
				ProbabilisticSampling: &jaeger_api_v2.ProbabilisticSamplingStrategy{
					SamplingRate: 0.5,
				},
			},
			expectedSampler: trace.TraceIDRatioBased(0.5),
		},
		{
			name: "RATE_LIMITING only",
			samplingStrategy: jaeger_api_v2.SamplingStrategyResponse{
				StrategyType: jaeger_api_v2.SamplingStrategyType_RATE_LIMITING,
				RateLimitingSampling: &jaeger_api_v2.RateLimitingSamplingStrategy{
					MaxTracesPerSecond: 100,
				},
			},
			expectedErr: "strategy type RATE_LIMITING is not supported",
		},
		{
			name: "PROBABILISTIC and per operation",
			samplingStrategy: jaeger_api_v2.SamplingStrategyResponse{
				StrategyType: jaeger_api_v2.SamplingStrategyType_PROBABILISTIC,
				ProbabilisticSampling: &jaeger_api_v2.ProbabilisticSamplingStrategy{
					SamplingRate: 0.5,
				},
				OperationSampling: &jaeger_api_v2.PerOperationSamplingStrategies{
					DefaultSamplingProbability: 0.1,
					PerOperationStrategies: []*jaeger_api_v2.OperationSamplingStrategy{
						{
							Operation: "foo",
							ProbabilisticSampling: &jaeger_api_v2.ProbabilisticSamplingStrategy{
								SamplingRate: 0.5,
							},
						},
					},
				},
			},
			expectedSampler: &perOperationSampler{
				defaultSampler: trace.TraceIDRatioBased(0.1),
				operationMap: map[string]trace.Sampler{
					"foo": trace.TraceIDRatioBased(0.5),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := samplingStrategyParseImpl{}

			sampler, err := parser.Parse(tt.samplingStrategy)

			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedSampler, sampler)
			}
		})
	}
}
