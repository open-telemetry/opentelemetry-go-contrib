// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package traceidratio provides a trace ID ratio-based sampler per the
// OpenTelemetry specification.
package traceidratio // import "go.opentelemetry.io/contrib/samplers/probability/traceidratio"

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

const (
	// DefaultSamplingPrecision is the default precision for threshold encoding.
	DefaultSamplingPrecision = 4
	maxAdjustedCount        = 1 << 56
	// randomnessMask masks the least significant 56 bits of the trace ID per
	// W3C Trace Context Level 2 Random Trace ID Flag.
	// https://www.w3.org/TR/trace-context-2/#random-trace-id-flag
	randomnessMask = maxAdjustedCount - 1

	probabilityZeroThreshold = 1 / float64(maxAdjustedCount)
	probabilityOneThreshold  = 1 - 0x1p-52
)

// TraceIDRatioSampler is a sampler that samples a fraction of traces based on
// the trace ID. It is exported for testing (e.g., to assert threshold values).
type TraceIDRatioSampler struct {
	threshold   uint64
	thkv        string
	description string
}

// Threshold returns the rejection threshold for testing.
func (ts *TraceIDRatioSampler) Threshold() uint64 {
	return ts.threshold
}

// ShouldSample implements sdktrace.Sampler.
func (ts *TraceIDRatioSampler) ShouldSample(p sdktrace.SamplingParameters) sdktrace.SamplingResult {
	psc := trace.SpanContextFromContext(p.ParentContext)
	state := psc.TraceState()

	existingOtts := state.Get("ot")

	var randomness uint64
	var hasRandomness bool
	if existingOtts != "" {
		randomness, hasRandomness = tracestateRandomness(existingOtts)
	}

	if !hasRandomness {
		randomness = binary.BigEndian.Uint64(p.TraceID[8:16]) & randomnessMask
	}

	if ts.threshold > randomness {
		return sdktrace.SamplingResult{
			Decision:   sdktrace.Drop,
			Tracestate: state,
		}
	}

	var newOtts string
	if !psc.TraceFlags().IsRandom() {
		newOtts = eraseTraceStateThKeyValue(existingOtts)
	} else {
		newOtts = InsertOrUpdateTraceStateThKeyValue(existingOtts, ts.thkv)
	}

	if combined, err := state.Insert("ot", newOtts); err != nil {
		otel.Handle(fmt.Errorf("could not combine tracestate: %w", err))
		return sdktrace.SamplingResult{Decision: sdktrace.Drop, Tracestate: state}
	} else {
		state = combined
	}
	return sdktrace.SamplingResult{Decision: sdktrace.RecordAndSample, Tracestate: state}
}

// Description implements sdktrace.Sampler.
func (ts *TraceIDRatioSampler) Description() string {
	return ts.description
}

// TraceIDRatioBased samples a given fraction of traces. Fractions >= 1 will
// always sample. Fractions < 0 are treated as zero. To respect the parent
// trace's SampledFlag, the TraceIDRatioBased sampler should be used as a
// delegate of a Parent sampler.
func TraceIDRatioBased(fraction float64) sdktrace.Sampler {
	const (
		maxp  = 14
		defp  = DefaultSamplingPrecision
		hbits = 4
	)
	if fraction > probabilityOneThreshold {
		return sdktrace.AlwaysSample()
	}
	if fraction < probabilityZeroThreshold {
		return sdktrace.NeverSample()
	}

	_, expF := math.Frexp(fraction)
	_, expR := math.Frexp(1 - fraction)
	precision := min(maxp, max(defp+expF/-hbits, defp+expR/-hbits))

	scaled := uint64(math.Round(fraction * float64(maxAdjustedCount)))
	threshold := maxAdjustedCount - scaled

	if shift := hbits * (maxp - precision); shift != 0 {
		half := uint64(1) << (shift - 1)
		threshold += half
		threshold >>= shift
		threshold <<= shift
	}

	tvalue := strings.TrimRight(strconv.FormatUint(maxAdjustedCount+threshold, 16)[1:], "0")
	return &TraceIDRatioSampler{
		threshold:   threshold,
		thkv:        "th:" + tvalue,
		description: fmt.Sprintf("TraceIDRatioBased{%g}", fraction),
	}
}

