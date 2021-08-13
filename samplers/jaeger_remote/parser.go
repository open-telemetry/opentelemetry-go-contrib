package jaeger_remote

import (
	"fmt"

	jaeger_api_v2 "github.com/jaegertracing/jaeger/proto-gen/api_v2"
	"go.opentelemetry.io/otel/sdk/trace"
)

type samplingStrategyParser interface {
	Parse(response jaeger_api_v2.SamplingStrategyResponse) (trace.Sampler, error)
}

type samplingStrategyParseImpl struct{}

var _ samplingStrategyParser = samplingStrategyParseImpl{}

func (p samplingStrategyParseImpl) Parse(strategies jaeger_api_v2.SamplingStrategyResponse) (trace.Sampler, error) {
	// TODO add support for rate limiting
	if strategies.StrategyType != jaeger_api_v2.SamplingStrategyType_PROBABILISTIC {
		return nil, fmt.Errorf("only strategy type PROBABILISTC is supported, got %s", strategies.StrategyType)
	}
	// TODO add support for per operation sampling
	if strategies.OperationSampling != nil {
		return nil, fmt.Errorf("per operation sampling is not supported")
	}

	// TODO should we implement this validation ourselves?
	if strategies.ProbabilisticSampling == nil {
		return nil, fmt.Errorf("strategy is probabilistic, but struct is empty")
	}

	return trace.TraceIDRatioBased(strategies.ProbabilisticSampling.SamplingRate), nil
}
