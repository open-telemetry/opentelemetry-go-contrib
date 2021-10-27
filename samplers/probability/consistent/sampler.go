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
	"fmt"
	"math/bits"
	"math/rand"
	"sync"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type (
	ConsistentProbabilityBasedOption interface {
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

// WithRandomSource sets the source of the random number used in this
// prototype of OTEP 170, which assumes the randomness is not derived
// from the TraceID.
func WithRandomSource(source rand.Source) ConsistentProbabilityBasedOption {
	return consistentProbabilityBasedRandomSource{source}
}

func (s consistentProbabilityBasedRandomSource) apply(cfg *consistentProbabilityBasedConfig) {
	cfg.source = s.Source
}

// ConsistentProbabilityBased samples a given fraction of traces.  Based on the
// OpenTelemetry specification, this Sampler supports only power-of-two
// fractions.  When the input fraction is not a power of two, it will
// be rounded down.
// - Fractions >= 1 will always sample.
// - Fractions < 2^-62 are treated as zero.
//
// This Sampler sets the `sampler.adjusted_count` attribute.
//
// To respect the parent trace's `SampledFlag`, this sampler should be
// used as the root delegate of a `Parent` sampler.
func ConsistentProbabilityBased(fraction float64, opts ...ConsistentProbabilityBasedOption) sdktrace.Sampler {
	cfg := consistentProbabilityBasedConfig{}
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
		rnd:     rand.New(cfg.source),
	}
}

func (cs *consistentProbabilityBased) newR() uint8 {
	cs.lock.Lock()
	defer cs.lock.Unlock()
	return uint8(bits.LeadingZeros64(uint64(cs.rnd.Int63())) - 1)
}

func (cs *consistentProbabilityBased) lowChoice() bool {
	cs.lock.Lock()
	defer cs.lock.Unlock()
	return cs.rnd.Float64() < cs.lowProb
}

// ShouldSample implements Sampler.
func (cs *consistentProbabilityBased) ShouldSample(p sdktrace.SamplingParameters) sdktrace.SamplingResult {
	psc := trace.SpanContextFromContext(p.ParentContext)
	var otts otelTraceState
	var state trace.TraceState

	if !psc.IsValid() {
		// A new root is happening.  Compute the r-value.
		otts = newTraceState()
		otts.rvalue = cs.newR()
	} else {
		// A valid parent context.
		state = psc.TraceState()

		var err error
		otts, err = parseOTelTraceState(state.Get(traceStateKey), psc.IsSampled())

		if err != nil {
			// Note: a state.Insert(traceStateKey)
			// follows, nothing else needs to be done here.
			otel.Handle(err)
		}

		if !otts.hasRValue() {
			// Specification says to set r-value if missing.
			otts.rvalue = cs.newR()
		}
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
	} else {
		decision = sdktrace.Drop
	}

	otts.pvalue = lac

	state, err := state.Insert(traceStateKey, otts.serialize())
	if err != nil {
		otel.Handle(err)
		// Note: see the note in
		// "go.opentelemetry.io/otel/trace".TraceState.Insert()
		// this is not a condition we're supposed to handle.
	}

	return sdktrace.SamplingResult{
		Decision:   decision,
		Tracestate: state,
	}
}

// Description implements Sampler.
func (cs *consistentProbabilityBased) Description() string {
	var prob float64
	if cs.lowLAC != pZeroValue {
		prob = cs.lowProb * expToFloat64(-int(cs.lowLAC))
		prob += (1 - cs.lowProb) * expToFloat64(-int(cs.highLAC))
	}
	return fmt.Sprintf("ConsistentProbabilityBased{%g}", prob)
}
