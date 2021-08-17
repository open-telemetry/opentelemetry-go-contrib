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
	"fmt"

	jaeger_api_v2 "go.opentelemetry.io/contrib/samplers/jaegerremote/internal/proto-gen/jaeger-idl/proto/api_v2"
	"go.opentelemetry.io/otel/sdk/trace"
)

type samplingStrategyParser interface {
	Parse(response jaeger_api_v2.SamplingStrategyResponse) (trace.Sampler, error)
}

type samplingStrategyParseImpl struct{}

var _ samplingStrategyParser = samplingStrategyParseImpl{}

func (p samplingStrategyParseImpl) Parse(strategies jaeger_api_v2.SamplingStrategyResponse) (trace.Sampler, error) {
	perOperationSamplingStrategy := strategies.GetOperationSampling()
	if perOperationSamplingStrategy != nil {
		return newPerOperationSampler(perOperationSamplingStrategy), nil
	}

	switch strategies.StrategyType {
	case jaeger_api_v2.SamplingStrategyType_PROBABILISTIC:
		return trace.TraceIDRatioBased(strategies.GetProbabilisticSampling().SamplingRate), nil
	case jaeger_api_v2.SamplingStrategyType_RATE_LIMITING:
		return nil, fmt.Errorf("strategy type RATE_LIMITING is not supported")
	}

	return nil, fmt.Errorf("got unrecognized strategy type %s", strategies.StrategyType)
}
