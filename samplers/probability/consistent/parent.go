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

package consistent // import "go.opentelemetry.io/contrib/samplers/probability/consistent"

import (
	"strings"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type (
	parentProbabilitySampler struct {
		delegate sdktrace.Sampler
	}
)

func ConsistentParentProbabilityBased(root sdktrace.Sampler, samplers ...sdktrace.ParentBasedSamplerOption) sdktrace.Sampler {
	return &parentProbabilitySampler{
		delegate: sdktrace.ParentBased(root, samplers...),
	}
}

func (p *parentProbabilitySampler) ShouldSample(params sdktrace.SamplingParameters) sdktrace.SamplingResult {
	psc := trace.SpanContextFromContext(params.ParentContext)

	if !psc.IsValid() {
		return p.delegate.ShouldSample(params)
	}

	state := psc.TraceState()

	otts, err := parseOTelTraceState(state.Get(traceStateKey), psc.IsSampled())

	if err != nil {
		otel.Handle(err)
		state.Insert(traceStateKey, otts.serialize())

		// Fix the broken tracestate before calling the delegate.
		params.ParentContext =
			trace.ContextWithSpanContext(params.ParentContext, psc.WithTraceState(state))
	}

	return p.delegate.ShouldSample(params)
}

func (p *parentProbabilitySampler) Description() string {
	s := p.delegate.Description()
	if strings.HasPrefix(s, "ParentBased") {
		s = s[len("ParentBased"):]
	}
	return "ConsistentParentProbabilityBased" + s
}
