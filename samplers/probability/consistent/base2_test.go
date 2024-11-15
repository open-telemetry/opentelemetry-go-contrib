// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consistent

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitProb(t *testing.T) {
	require.Equal(t, -1, expFromFloat64(0.6)) //nolint:testifylint // false positive on expected-actual
	require.Equal(t, -2, expFromFloat64(0.4)) //nolint:testifylint // false positive on expected-actual
	require.Equal(t, 0.5, expToFloat64(-1))
	require.Equal(t, 0.25, expToFloat64(-2))

	for _, tc := range []struct {
		in      float64
		low     uint8
		lowProb float64
	}{
		// Probability 0.75 corresponds with choosing S=1 (the
		// "low" probability) 50% of the time and S=0 (the
		// "high" probability) 50% of the time.
		{0.75, 1, 0.5},
		{0.6, 1, 0.8},
		{0.9, 1, 0.2},

		// Powers of 2 exactly
		{1, 0, 1},
		{0.5, 1, 1},
		{0.25, 2, 1},

		// Smaller numbers
		{0.05, 5, 0.4},
		{0.1, 4, 0.4}, // 0.1 == 0.4 * 1/16 + 0.6 * 1/8
		{0.003, 9, 0.464},

		// Special cases:
		{0, 63, 1},
	} {
		low, high, lowProb := splitProb(tc.in)
		require.Equal(t, tc.low, low, "got %v want %v", low, tc.low)
		if lowProb != 1 {
			require.Equal(t, tc.low-1, high, "got %v want %v", high, tc.low-1)
		}
		require.InEpsilon(t, tc.lowProb, lowProb, 1e-6, "got %v want %v", lowProb, tc.lowProb)
	}
}
