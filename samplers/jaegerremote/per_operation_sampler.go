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
	"strings"

	jaeger_api_v2 "go.opentelemetry.io/contrib/samplers/jaegerremote/internal/proto-gen/jaeger-idl/proto/api_v2"
	"go.opentelemetry.io/otel/sdk/trace"
)

type perOperationSampler struct {
	defaultSampler trace.Sampler
	operationMap   map[string]trace.Sampler
}

var _ trace.Sampler = &perOperationSampler{}

func (p *perOperationSampler) ShouldSample(parameters trace.SamplingParameters) trace.SamplingResult {
	sampler := p.defaultSampler
	if s, ok := p.operationMap[parameters.Name]; ok {
		sampler = s
	}
	return sampler.ShouldSample(parameters)
}

func (p *perOperationSampler) Description() string {
	var ops []string
	for op, sampler := range p.operationMap {
		ops = append(ops, fmt.Sprintf("%s:%s", op, sampler.Description()))
	}
	mapStr := strings.Join(ops, ",")

	return fmt.Sprintf("PerOperationSampler{default=%s,perOperation={%v}}", p.defaultSampler.Description(), mapStr)
}

func newPerOperationSampler(strategies *jaeger_api_v2.PerOperationSamplingStrategies) trace.Sampler {
	operationMap := make(map[string]trace.Sampler)
	for _, operationSamplingStrategy := range strategies.PerOperationStrategies {
		operationMap[operationSamplingStrategy.Operation] = trace.TraceIDRatioBased(operationSamplingStrategy.ProbabilisticSampling.SamplingRate)
	}

	return &perOperationSampler{
		defaultSampler: trace.TraceIDRatioBased(strategies.DefaultSamplingProbability),
		operationMap:   operationMap,
	}
}
