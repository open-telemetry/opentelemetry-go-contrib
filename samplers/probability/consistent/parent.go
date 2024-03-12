// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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
