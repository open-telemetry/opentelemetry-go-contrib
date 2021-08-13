package jaeger_remote

import (
	"testing"

	jaeger_api_v2 "github.com/jaegertracing/jaeger/proto-gen/api_v2"
	"github.com/stretchr/testify/assert"
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
			expectedErr: "only strategy type PROBABILISTC is supported, got RATE_LIMITING",
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
			expectedErr: "per operation sampling is not supported",
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
