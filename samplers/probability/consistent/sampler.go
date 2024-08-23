// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package consistent provides a consistent probability based sampler.
package consistent // import "go.opentelemetry.io/contrib/samplers/probability/consistent"

import (
	"fmt"
	"math/bits"
	"math/rand"
	"sync"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type (
	// ProbabilityBasedOption is an option to the
	// ConssitentProbabilityBased sampler.
	ProbabilityBasedOption interface {
		apply(*consistentProbabilityBasedConfig)
	}

	consistentProbabilityBasedConfig struct {
		source rand.Source
	}

	consistentProbabilityBasedRandomSource struct {
		rand.Source
	}

	consistentProbabilityBased struct {
		// "LAC" is an abbreviation for the logarithm of
		// adjusted count.  Greater values have greater
		// representivity, therefore lesser sampling
		// probability.

		// lowLAC is the lower-probability log-adjusted count
		lowLAC uint8
		// highLAC is the higher-probability log-adjusted
		// count.  except for the zero probability special
		// case, highLAC == lowLAC - 1.
		highLAC uint8
		// lowProb is the probability that lowLAC should be used,
		// in the interval (0, 1].  For exact powers of two and the
		// special case of 0 probability, lowProb == 1.
		lowProb float64

		// lock protects rnd
		lock sync.Mutex
		rnd  *rand.Rand
	}
)

// WithRandomSource sets the source of the randomness used by the Sampler.
func WithRandomSource(source rand.Source) ProbabilityBasedOption {
	return consistentProbabilityBasedRandomSource{source}
}

func (s consistentProbabilityBasedRandomSource) apply(cfg *consistentProbabilityBasedConfig) {
	cfg.source = s.Source
}

// ProbabilityBased samples a given fraction of traces.  Based on the
// OpenTelemetry specification, this Sampler supports only power-of-two
// fractions.  When the input fraction is not a power of two, it will
// be rounded down.
// - Fractions >= 1 will always sample.
// - Fractions < 2^-62 are treated as zero.
//
// This Sampler sets the OpenTelemetry tracestate p-value and/or r-value.
//
// To respect the parent trace's `SampledFlag`, this sampler should be
// used as the root delegate of a `Parent` sampler.
func ProbabilityBased(fraction float64, opts ...ProbabilityBasedOption) sdktrace.Sampler {
	cfg := consistentProbabilityBasedConfig{
		source: rand.NewSource(rand.Int63()), //nolint:gosec // G404: Use of weak random number generator (math/rand instead of crypto/rand) is ignored as this is not security-sensitive.
	}
	for _, opt := range opts {
		opt.apply(&cfg)
	}

	if fraction < 0 {
		fraction = 0
	} else if fraction > 1 {
		fraction = 1
	}

	lowLAC, highLAC, lowProb := splitProb(fraction)

	return &consistentProbabilityBased{
		lowLAC:  lowLAC,
		highLAC: highLAC,
		lowProb: lowProb,
		rnd:     rand.New(cfg.source), //nolint:gosec // G404: Use of weak random number generator (math/rand instead of crypto/rand) is ignored as this is not security-sensitive.
	}
}

func (cs *consistentProbabilityBased) newR() uint8 {
	cs.lock.Lock()
	defer cs.lock.Unlock()
	return uint8(bits.LeadingZeros64(uint64(cs.rnd.Int63())) - 1) // nolint: gosec  // 8-bit sample.
}

func (cs *consistentProbabilityBased) lowChoice() bool {
	cs.lock.Lock()
	defer cs.lock.Unlock()
	return cs.rnd.Float64() < cs.lowProb
}

// ShouldSample implements "go.opentelemetry.io/otel/sdk/trace".Sampler.
func (cs *consistentProbabilityBased) ShouldSample(p sdktrace.SamplingParameters) sdktrace.SamplingResult {
	psc := trace.SpanContextFromContext(p.ParentContext)

	// Note: this ignores whether psc.IsValid() because this
	// allows other otel trace state keys to pass through even
	// for root decisions.
	state := psc.TraceState()

	otts, err := parseOTelTraceState(state.Get(traceStateKey), psc.IsSampled())
	if err != nil {
		// Note: a state.Insert(traceStateKey)
		// follows, nothing else needs to be done here.
		otel.Handle(err)
	}

	if !otts.hasRValue() {
		otts.rvalue = cs.newR()
	}

	var decision sdktrace.SamplingDecision
	var lac uint8

	if cs.lowProb == 1 || cs.lowChoice() {
		lac = cs.lowLAC
	} else {
		lac = cs.highLAC
	}

	if lac <= otts.rvalue {
		decision = sdktrace.RecordAndSample
		otts.pvalue = lac
	} else {
		decision = sdktrace.Drop
		otts.pvalue = invalidValue
	}

	// Note: see the note in
	// "go.opentelemetry.io/otel/trace".TraceState.Insert(). The
	// error below is not a condition we're supposed to handle.
	state, _ = state.Insert(traceStateKey, otts.serialize())

	return sdktrace.SamplingResult{
		Decision:   decision,
		Tracestate: state,
	}
}

// Description returns "ProbabilityBased{%g}" with the configured probability.
func (cs *consistentProbabilityBased) Description() string {
	var prob float64
	if cs.lowLAC != pZeroValue {
		prob = cs.lowProb * expToFloat64(-int(cs.lowLAC))
		prob += (1 - cs.lowProb) * expToFloat64(-int(cs.highLAC))
	}
	return fmt.Sprintf("ProbabilityBased{%g}", prob)
}
