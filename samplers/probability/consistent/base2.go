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

import "math"

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
//	splitProb(0.375) => (2, 1, 0.5)
//
// indicates to sample with probability (2^-2) 50% of the time
// and (2^-1) 50% of the time.
func splitProb(p float64) (uint8, uint8, float64) {
	if p < 2e-62 {
		// Note: spec.
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
