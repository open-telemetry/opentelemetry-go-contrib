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

package consistent

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitProb(t *testing.T) {
	require.Equal(t, -1, expFromFloat64(0.6))
	require.Equal(t, -2, expFromFloat64(0.4))
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

func TestDescription(t *testing.T) {
	for _, tc := range []struct {
		prob   float64
		expect string
	}{
		{0.75, "ConsistentProbabilityBased{0.75}"},
		{0.05, "ConsistentProbabilityBased{0.05}"},
		{0.003, "ConsistentProbabilityBased{0.003}"},
		{0.99999999, "ConsistentProbabilityBased{0.99999999}"},
		{0.00000001, "ConsistentProbabilityBased{1e-08}"},
		{1, "ConsistentProbabilityBased{1}"},
		{0, "ConsistentProbabilityBased{0}"},
	} {
		s := ConsistentProbabilityBased(tc.prob)
		require.Equal(t, tc.expect, s.Description())
	}
}

func TestNewTraceState(t *testing.T) {
	otts := newTraceState()
	require.False(t, otts.hasPValue())
	require.False(t, otts.hasRValue())
	require.Equal(t, "", otts.serialize())
}

func TestParseTraceState(t *testing.T) {
	type testCase struct {
		in         string
		pval, rval uint8
		expectErr  error
	}
	const notset = 255
	for _, test := range []testCase{
		{"r:1;p:2", 2, 1, nil},
		{"r:1;p:2;", 2, 1, nil},
		{"p:2;r:1;", 2, 1, nil},
		{"p:2;r:1", 2, 1, nil},
		{"r:1;", notset, 1, nil},
		{"r:1", notset, 1, nil},
		{"r:1=p:2", notset, notset, strconv.ErrSyntax},
		{"r:1;p:2=s:3", notset, notset, strconv.ErrSyntax},
		{":1;p:2=s:3", notset, notset, strconv.ErrSyntax},
		{":;p:2=s:3", notset, notset, strconv.ErrSyntax},
		{":;:", notset, notset, strconv.ErrSyntax},
		{":", notset, notset, strconv.ErrSyntax},
		{"", notset, notset, nil},
		{"r:", notset, notset, strconv.ErrSyntax},
		{"r:;p=1", notset, notset, strconv.ErrSyntax},
		{"r:1", notset, 1, nil},
		{"r:10", notset, 10, nil},
		{"r:33", notset, 33, nil},
		{"r:61", notset, 61, nil},
		{"r:62", notset, 62, nil},                      // max r-value
		{"r:63", notset, notset, strconv.ErrRange},     // out-of-range
		{"r:100", notset, notset, strconv.ErrRange},    // out-of-range
		{"r:100001", notset, notset, strconv.ErrRange}, // out-of-range
		{"p:1", 1, notset, nil},
		{"p:62", 62, notset, nil},
		{"p:63", 63, notset, nil},
		{"p:64", notset, notset, strconv.ErrRange},
		{"p:100", notset, notset, strconv.ErrRange},
		{"r:1a", notset, notset, strconv.ErrSyntax}, // not-hexadecimal
		{"p:-1", notset, notset, strconv.ErrSyntax}, // non-negative
	} {
		t.Run(strings.NewReplacer(":", "_", ";", "_").Replace(test.in), func(t *testing.T) {
			otts, err := parseOTelTraceState(test.in)

			if test.expectErr != nil {
				require.True(t, errors.Is(err, test.expectErr), "not expecting %v", err)
			}
			if test.pval != notset {
				require.True(t, otts.hasPValue())
				require.Equal(t, test.pval, otts.pvalue)
			} else {
				require.False(t, otts.hasPValue(), "should have no p-value")
			}
			if test.rval != notset {
				require.True(t, otts.hasRValue())
				require.Equal(t, test.rval, otts.rvalue)
			} else {
				require.False(t, otts.hasRValue(), "should have no r-value")
			}
		})
	}
}
