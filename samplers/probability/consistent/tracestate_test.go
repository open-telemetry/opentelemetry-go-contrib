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

func TestNewTraceState(t *testing.T) {
	otts := newTraceState()
	require.False(t, otts.hasPValue())
	require.False(t, otts.hasRValue())
	require.Equal(t, "", otts.serialize())
}

func TestParseTraceStateUnsampled(t *testing.T) {
	type testCase struct {
		in         string
		pval, rval uint8
		expectErr  error
	}
	const notset = 255
	for _, test := range []testCase{
		// All are unsampled tests, i.e., `sampled` is not set in traceparent.
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
			// Note: passing isSampled=false as stated above.
			otts, err := parseOTelTraceState(test.in, false)

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

func TestParseTraceStateSampled(t *testing.T) {
	type testCase struct {
		in         string
		rval, pval uint8
		expectErr  error
	}
	const notset = 255
	for _, test := range []testCase{
		// All are sampled tests, i.e., `sampled` is set in traceparent.
		{"r:2;p:2", 2, 2, nil},
		{"r:2;p:1", 2, 1, nil},
		{"r:2;p:0", 2, 0, nil},

		{"r:1;p:1", 1, 1, nil},
		{"r:1;p:0", 1, 0, nil},

		{"r:0;p:0", 0, 0, nil},

		{"r:62;p:0", 62, 0, nil},
		{"r:62;p:62", 62, 62, nil},

		// The important special case:
		{"r:0;p:63", 0, 63, nil},
		{"r:2;p:63", 2, 63, nil},
		{"r:62;p:63", 62, 63, nil},

		// Inconsistencies cause unset p-value.
		{"r:2;p:3", 2, notset, errTraceStateInconsistent},
		{"r:2;p:4", 2, notset, errTraceStateInconsistent},
		{"r:2;p:62", 2, notset, errTraceStateInconsistent},
		{"r:0;p:1", 0, notset, errTraceStateInconsistent},
		{"r:1;p:2", 1, notset, errTraceStateInconsistent},
		{"r:61;p:62", 61, notset, errTraceStateInconsistent},
	} {
		t.Run(strings.NewReplacer(":", "_", ";", "_").Replace(test.in), func(t *testing.T) {
			// Note: passing isSampled=true as stated above.
			otts, err := parseOTelTraceState(test.in, true)

			if test.expectErr != nil {
				require.True(t, errors.Is(err, test.expectErr), "not expecting %v", err)
			} else {
				require.NoError(t, err)
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
