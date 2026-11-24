// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package jaegerremote

import "testing"

// traceStateCases are the incoming "ot" tracestate shapes that change a
// sampler's work.
var traceStateCases = []struct {
	name string
	otts string
}{
	{name: "NoTraceState"},
	{name: "NoThreshold", otts: testRandomnessValue},
	{name: "Threshold", otts: testHalfThreshold + ";" + testRandomnessValue},
}

var probabilitySamplingModes = []struct {
	name    string
	enabled bool
}{
	{name: "TraceIDRatioBased"},
	{name: "ProbabilitySampler", enabled: true},
}

func BenchmarkProbabilisticSamplerShouldSample(b *testing.B) {
	for _, mode := range probabilitySamplingModes {
		for _, tc := range traceStateCases {
			b.Run(mode.name+"/"+tc.name, func(b *testing.B) {
				sampler := newProbabilisticSampler(testDefaultSamplingProbability, false, mode.enabled)
				params := samplingParameters(b, testSampledRandomness, tc.otts)

				b.ReportAllocs()
				for b.Loop() {
					_ = sampler.ShouldSample(params)
				}
			})
		}
	}
}

func BenchmarkRateLimitingSamplerShouldSample(b *testing.B) {
	for _, tc := range traceStateCases {
		b.Run(tc.name, func(b *testing.B) {
			sampler := newRateLimitingSampler(1e9, false)
			params := samplingParameters(b, testSampledRandomness, tc.otts)

			b.ReportAllocs()
			for b.Loop() {
				_ = sampler.ShouldSample(params)
			}
		})
	}
}

func BenchmarkGuaranteedThroughputProbabilisticSamplerShouldSample(b *testing.B) {
	for _, mode := range probabilitySamplingModes {
		for _, tc := range traceStateCases {
			b.Run(mode.name+"/"+tc.name, func(b *testing.B) {
				// Keep the decision on the probabilistic path.
				sampler := newGuaranteedThroughputProbabilisticSampler(1e9, 1.0, false, mode.enabled)
				params := samplingParameters(b, testSampledRandomness, tc.otts)

				b.ReportAllocs()
				for b.Loop() {
					_ = sampler.ShouldSample(params)
				}
			})
		}
	}
}
