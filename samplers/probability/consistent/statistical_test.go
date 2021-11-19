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

// +build !race

package consistent

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestSamplerStatistics(t *testing.T) {

	seedBankRng := rand.New(rand.NewSource(77777677777))
	seedBank := make([]int64, 15) // N.B. Max=14 below.
	for i := range seedBank {
		seedBank[i] = seedBankRng.Int63()
	}
	type (
		testCase struct {
			// prob is the sampling probability under test.
			prob float64

			// upperP reflects the larger of the one or two
			// distinct adjusted counts represented in the test.
			//
			// For power-of-two tests, there is one distinct p-value,
			// and each span counts as 2**upperP representative spans.
			//
			// For non-power-of-two tests, there are two distinct
			// p-values expected, the test is specified using the
			// larger of these values corresponding with the
			// smaller sampling probability.  The sampling
			// probability under test rounded down to the nearest
			// power of two is expected to equal 2**(-upperP).
			upperP pValue

			// degrees is 1 for power-of-two tests and 2 for
			// non-power-of-two tests.
			degrees testDegrees

			// seedIndex is the index into seedBank of the test seed.
			// If this is -1 the code below will search for the smallest
			// seed index that passes the test.
			seedIndex int
		}
		testResult struct {
			test     testCase
			expected []float64
		}
	)
	var (
		testSummary []testResult

		allTests = []testCase{
			// Non-powers of two
			{0.90000, 1, twoDegrees, 5},
			{0.60000, 1, twoDegrees, 14},
			{0.33000, 2, twoDegrees, 3},
			{0.13000, 3, twoDegrees, 2},
			{0.10000, 4, twoDegrees, 0},
			{0.05000, 5, twoDegrees, 0},
			{0.01700, 6, twoDegrees, 2},
			{0.01000, 7, twoDegrees, 3},
			{0.00500, 8, twoDegrees, 1},
			{0.00290, 9, twoDegrees, 1},
			{0.00100, 10, twoDegrees, 5},
			{0.00050, 11, twoDegrees, 1},
			{0.00026, 12, twoDegrees, 3},
			{0.00023, 13, twoDegrees, 0},
			{0.00010, 14, twoDegrees, 2},

			// Powers of two
			{0x1p-1, 1, oneDegree, 0},
			{0x1p-4, 4, oneDegree, 2},
			{0x1p-7, 7, oneDegree, 3},
			{0x1p-10, 10, oneDegree, 0},
			{0x1p-13, 13, oneDegree, 1},
		}
	)

	if testing.Short() {
		one := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(len(allTests))
		allTests = allTests[one : one+1]
	}

	for _, test := range allTests {
		t.Run(fmt.Sprint(test.prob), func(t *testing.T) {
			var expected []float64
			trySeedIndex := 0

			for {
				var seed int64
				seedIndex := test.seedIndex
				if seedIndex >= 0 {
					seed = seedBank[seedIndex]
				} else {
					seedIndex = trySeedIndex
					seed = seedBank[trySeedIndex]
					trySeedIndex++
				}

				countFailures := func(src rand.Source) int {
					failed := 0

					for j := 0; j < trials; j++ {
						var x float64
						x, expected = sampleTrials(t, test.prob, test.degrees, test.upperP, src)

						if x < chiSquaredByDF[test.degrees] {
							failed++
						}
					}
					return failed
				}

				failed := countFailures(rand.NewSource(seed))

				if failed != 1 && test.seedIndex < 0 {
					t.Logf("%d probabilistic failures, trying a new seed for %g was 0x%x", failed, test.prob, seed)
					continue
				} else if failed != 1 {
					t.Errorf("wrong number of probabilistic failures for %g, should be 1 was %d for seed 0x%x", test.prob, failed, seed)
				} else if test.seedIndex < 0 {
					t.Logf("update the test for %g to use seed index %d", test.prob, seedIndex)
					t.Fail()
					return
				} else {
					// Note: this can be uncommented to verify that the preceding seed failed the test,
					// for example:
					// if seedIndex != 0 && countFailures(rand.NewSource(seedBank[seedIndex-1])) == 1 {
					// 	t.Logf("update the test for %g to use seed index < %d", test.prob, seedIndex)
					// 	t.Fail()
					// }
					break
				}
			}
			testSummary = append(testSummary, testResult{
				test:     test,
				expected: expected,
			})
		})
	}

	for idx, res := range testSummary {
		var probability, pvalues, expectLower, expectUpper, expectUnsampled string
		if res.test.degrees == twoDegrees {
			probability = fmt.Sprintf("%.6f", res.test.prob)
			pvalues = fmt.Sprint(res.test.upperP-1, ", ", res.test.upperP)
			expectUnsampled = fmt.Sprintf("%.10g", res.expected[0])
			expectLower = fmt.Sprintf("%.10g", res.expected[1])
			expectUpper = fmt.Sprintf("%.10g", res.expected[2])
		} else {
			probability = fmt.Sprintf("%x (%.6f)", res.test.prob, res.test.prob)
			pvalues = fmt.Sprint(res.test.upperP)
			expectUnsampled = fmt.Sprintf("%.10g", res.expected[0])
			expectLower = fmt.Sprintf("%.10g", res.expected[1])
			expectUpper = "n/a"
		}
		t.Logf("| %d | %s | %s | %s | %s | %s |\n", idx+1, probability, pvalues, expectLower, expectUpper, expectUnsampled)
	}
}
