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

// ParentProbabilityBased is an implementation of the OpenTelemetry
// Trace Sampler interface that provides additional checks for tracestate
// Probability Sampling fields.
func ParentProbabilityBased(root sdktrace.Sampler, samplers ...sdktrace.ParentBasedSamplerOption) sdktrace.Sampler {
	return &parentProbabilitySampler{
		delegate: sdktrace.ParentBased(root, samplers...),
	}
}

// ShouldSample implements "go.opentelemetry.io/otel/sdk/trace".Sampler.
func (p *parentProbabilitySampler) ShouldSample(params sdktrace.SamplingParameters) sdktrace.SamplingResult {
	psc := trace.SpanContextFromContext(params.ParentContext)

	// Note: We do not check psc.IsValid(), i.e., we repair the tracestate
	// with or without a parent TraceId and SpanId.
	state := psc.TraceState()

	otts, err := parseOTelTraceState(state.Get(traceStateKey), psc.IsSampled())
	if err != nil {
		otel.Handle(err)
		value := otts.serialize()
		if len(value) > 0 {
			// Note: see the note in
			// "go.opentelemetry.io/otel/trace".TraceState.Insert(). The
			// error below is not a condition we're supposed to handle.
			state, _ = state.Insert(traceStateKey, value)
		} else {
			state = state.Delete(traceStateKey)
		}

		// Fix the broken tracestate before calling the delegate.
		params.ParentContext = trace.ContextWithSpanContext(params.ParentContext, psc.WithTraceState(state))
	}

	return p.delegate.ShouldSample(params)
}

// Description returns the same description as the built-in
// ParentBased sampler, with "ParentBased" replaced by
// "ParentProbabilityBased".
func (p *parentProbabilitySampler) Description() string {
	return "ParentProbabilityBased" + strings.TrimPrefix(p.delegate.Description(), "ParentBased")
}
