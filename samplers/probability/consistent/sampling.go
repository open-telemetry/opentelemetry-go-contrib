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
	"math"
	"math/bits"
	"math/rand"
	"strconv"
	"strings"
	"sync"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

const (
	traceStateKey   = "ot"
	pValueSubkey    = "p"
	rValueSubkey    = "r"
	pZeroValue      = 63
	valueNumberBase = 10
	valueBitWidth   = 6
)

var (
	errTraceStateSyntax = fmt.Errorf("otel tracestate: %w", strconv.ErrSyntax)
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

	otelTraceState struct {
		rvalue  uint8 // valid in the interval [0, 62]
		pvalue  uint8 // valid in the interval [0, 63]
		unknown []string
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

// These are IEEE 754 double-width floating point constants used with
// math.Float64bits.
const (
	offsetExponentMask = 0x7ff0000000000000
	offsetExponentBias = 1023
	significandBits    = 52
)

// expFromFloat64 returns floor(log2(x)).
func expFromFloat64(x float64) int {
	return int((math.Float64bits(x)&offsetExponentMask)>>significandBits) - offsetExponentBias
}

// expToFloat64 returns 2^x.
func expToFloat64(x int) float64 {
	return math.Float64frombits(uint64(offsetExponentBias+x) << significandBits)
}

// splitProb returns the two values of log-adjusted-count nearest to p
// Example:
//
//   splitProb(0.375) => (2, 1, 0.5)
//
// indicates to sample with probability (2^-2) 50% of the time
// and (2^-1) 50% of the time.
func splitProb(p float64) (uint8, uint8, float64) {
	if p < 2e-62 {
		return pZeroValue, pZeroValue, 1
	}
	// Take the exponent and drop the significand to locate the
	// smaller of two powers of two.
	exp := expFromFloat64(p)

	// Low is the smaller of two log-adjusted counts, the negative
	// of the exponent computed above.
	low := -exp
	// High is the greater of two log-adjusted counts (i.e., one
	// less than low, a smaller adjusted count means a larger
	// probability).
	high := low - 1

	// Return these to probability values and use linear
	// interpolation to compute the required probability of
	// choosing the low-probability Sampler.
	lowP := expToFloat64(-low)
	highP := expToFloat64(-high)
	lowProb := (highP - p) / (highP - lowP)

	return uint8(low), uint8(high), lowProb
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

func newTraceState() otelTraceState {
	return otelTraceState{
		rvalue: 64, // out-of-range => !hasRValue()
		pvalue: 64, // out-of-range => !hasPValue()
	}
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
		// It does not matter if psc.IsSampled().
		state = psc.TraceState()
		otts.clearPValue()

		var err error
		otts, err = parseOTelTraceState(state.Get(traceStateKey))

		if err != nil {
			// Note: we've reset trace state.
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
		// TODO: Spec how to handle this.
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

func (otts otelTraceState) serialize() string {
	var sb strings.Builder
	if otts.hasPValue() {
		_, _ = sb.WriteString(fmt.Sprintf("p:%02d;", otts.pvalue))
	}
	if otts.hasRValue() {
		_, _ = sb.WriteString(fmt.Sprintf("r:%02d;", otts.rvalue))
	}
	for _, unk := range otts.unknown {
		_, _ = sb.WriteString(unk)
		_, _ = sb.WriteString(";")
	}
	res := sb.String()
	// Disregard a trailing `;`
	if len(res) != 0 {
		res = res[:len(res)-1]
	}
	return res
}

func parseError(key string, err error) error {
	return fmt.Errorf("otel tracestate: %s-value %w", key, err)
}

func parseOTelTraceState(ts string) (otelTraceState, error) {
	// TODO: Limits to apply from the spec?
	// TODO: Key syntax
	// TODO: Value syntax
	otts := newTraceState()
	for len(ts) > 0 {
		eqPos := 0
		for ; eqPos < len(ts); eqPos++ {
			if ts[eqPos] >= 'a' && ts[eqPos] <= 'z' {
				continue
			}
			break
		}
		if eqPos == 0 || eqPos == len(ts) || ts[eqPos] != ':' {
			return newTraceState(), errTraceStateSyntax
		}

		key := ts[0:eqPos]
		tail := ts[eqPos+1:]

		sepPos := 0

		if key == pValueSubkey || key == rValueSubkey {
			// See TODOs above.  Have one syntax check,
			// then let ParseUint return ErrSyntax if need
			// be.
			for ; sepPos < len(tail); sepPos++ {
				if tail[sepPos] >= '0' && tail[sepPos] <= '9' {
					continue
				}
				break
			}
			value, err := strconv.ParseUint(
				tail[0:sepPos],
				valueNumberBase,
				valueBitWidth,
			)
			if err != nil {
				return newTraceState(), parseError(key, err)
			}
			if key == pValueSubkey {
				if value > pZeroValue {
					return newTraceState(), parseError(key, strconv.ErrRange)
				}
				otts.pvalue = uint8(value)
			} else if key == rValueSubkey {
				if value > (pZeroValue - 1) {
					return newTraceState(), parseError(key, strconv.ErrRange)
				}
				otts.rvalue = uint8(value)
			}

		} else {
			// Note: Spec valid character set for forward compatibility.
			// Should this be the base64 characters?
			for ; sepPos < len(tail); sepPos++ {
				if tail[sepPos] >= '0' && tail[sepPos] <= '9' {
					// See TODOs above.
					continue
				}
			}
			otts.unknown = append(otts.unknown, ts[0:sepPos])
		}

		if sepPos == 0 || (sepPos < len(tail) && tail[sepPos] != ';') {
			return newTraceState(), errTraceStateSyntax
		}

		if sepPos == len(tail) {
			break
		}

		ts = tail[sepPos+1:]
	}

	return otts, nil
}

func (otts otelTraceState) hasRValue() bool {
	return otts.rvalue < pZeroValue
}

func (otts otelTraceState) hasPValue() bool {
	return otts.pvalue <= pZeroValue
}

func (otts otelTraceState) clearRValue() {
	otts.rvalue = pZeroValue
}

func (otts otelTraceState) clearPValue() {
	otts.pvalue = pZeroValue + 1
}
