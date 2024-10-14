// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !race
// +build !race

package consistent

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

const (
	oneDegree  testDegrees = 1
	twoDegrees testDegrees = 2
)

var (
	trials         = 20
	populationSize = 1e5

	// These may be computed using Gonum, e.g.,
	// import "gonum.org/v1/gonum/stat/distuv"
	// with significance = 1 / float64(trials) = 0.05
	// chiSquaredDF1  = distuv.ChiSquared{K: 1}.Quantile(significance)
	// chiSquaredDF2  = distuv.ChiSquared{K: 2}.Quantile(significance)
	//
	// These have been specified using significance = 0.05:
	chiSquaredDF1 = 0.003932140000019522
	chiSquaredDF2 = 0.1025865887751011

	chiSquaredByDF = [3]float64{
		0,
		chiSquaredDF1,
		chiSquaredDF2,
	}
)

func TestSamplerStatistics(t *testing.T) {
	seedBankRng := rand.New(rand.NewSource(77777677777))
	seedBank := make([]int64, 7) // N.B. Max=6 below.
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
			{0.90000, 1, twoDegrees, 3},
			{0.60000, 1, twoDegrees, 2},
			{0.33000, 2, twoDegrees, 2},
			{0.13000, 3, twoDegrees, 1},
			{0.10000, 4, twoDegrees, 0},
			{0.05000, 5, twoDegrees, 0},
			{0.01700, 6, twoDegrees, 2},
			{0.01000, 7, twoDegrees, 2},
			{0.00500, 8, twoDegrees, 2},
			{0.00290, 9, twoDegrees, 4},
			{0.00100, 10, twoDegrees, 6},
			{0.00050, 11, twoDegrees, 0},

			// Powers of two
			{0x1p-1, 1, oneDegree, 0},
			{0x1p-4, 4, oneDegree, 0},
			{0x1p-7, 7, oneDegree, 1},
		}
	)

	// Limit the test runtime by choosing 3 of the above
	// non-deterministically
	rand.New(rand.NewSource(time.Now().UnixNano())).Shuffle(len(allTests), func(i, j int) {
		allTests[i], allTests[j] = allTests[j], allTests[i]
	})
	allTests = allTests[0:3]

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
					// however this just doubles runtime and adds little evidence.  For example:
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

	// Note: This produces a table that should match what is in
	// the specification if it's the same test.
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

func sampleTrials(t *testing.T, prob float64, degrees testDegrees, upperP pValue, source rand.Source) (float64, []float64) {
	ctx := context.Background()

	sampler := ProbabilityBased(
		prob,
		WithRandomSource(source),
	)

	recorder := &tracetest.InMemoryExporter{}
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(recorder),
		sdktrace.WithSampler(sampler),
	)

	tracer := provider.Tracer("test")

	for i := 0; i < int(populationSize); i++ {
		_, span := tracer.Start(ctx, "span")
		span.End()
	}

	var minP, maxP pValue

	counts := map[pValue]int64{}

	for idx, r := range recorder.GetSpans() {
		ts := r.SpanContext.TraceState()
		p, _ := parsePR(ts.Get("ot"))

		pi, err := strconv.ParseUint(p, 10, 64)
		require.NoError(t, err)

		if idx == 0 {
			maxP = pValue(pi)
			minP = maxP
		} else {
			if pValue(pi) < minP {
				minP = pValue(pi)
			}
			if pValue(pi) > maxP {
				maxP = pValue(pi)
			}
		}
		counts[pValue(pi)]++
	}

	require.Less(t, maxP, minP+pValue(degrees), "%v %v %v", minP, maxP, degrees)
	require.Less(t, maxP, pValue(63))
	require.LessOrEqual(t, len(counts), 2)

	var ceilingProb, floorProb, floorChoice float64

	// Note: we have to test len(counts) == 0 because this outcome
	// is actually possible, just very unlikely.  If this happens
	// during development, a new initial seed must be used for
	// this test.
	//
	// The test specification ensures the test ensures there are
	// at least 20 expected items per category in these tests.
	require.NotEmpty(t, counts)

	if degrees == 2 {
		// Note: because the test is probabilistic, we can't be
		// sure that both the min and max P values happen.  We
		// can only assert that one of these is true.
		require.GreaterOrEqual(t, maxP, upperP-1)
		require.GreaterOrEqual(t, minP, upperP-1)
		require.LessOrEqual(t, maxP, upperP)
		require.LessOrEqual(t, minP, upperP)
		require.LessOrEqual(t, maxP-minP, 1)

		ceilingProb = 1 / float64(int64(1)<<(upperP-1))
		floorProb = 1 / float64(int64(1)<<upperP)
		floorChoice = (ceilingProb - prob) / (ceilingProb - floorProb)
	} else {
		require.Equal(t, minP, maxP)
		require.Equal(t, upperP, maxP)
		ceilingProb = 0
		floorProb = prob
		floorChoice = 1
	}

	expectLowerCount := floorChoice * floorProb * populationSize
	expectUpperCount := (1 - floorChoice) * ceilingProb * populationSize
	expectUnsampled := (1 - prob) * populationSize

	upperCount := int64(0)
	lowerCount := counts[maxP]
	if degrees == 2 {
		upperCount = counts[minP]
	}
	unsampled := int64(populationSize) - upperCount - lowerCount

	expected := []float64{
		expectUnsampled,
		expectLowerCount,
		expectUpperCount,
	}

	chi2 := 0.0
	chi2 += math.Pow(float64(unsampled)-expectUnsampled, 2) / expectUnsampled
	chi2 += math.Pow(float64(lowerCount)-expectLowerCount, 2) / expectLowerCount
	if degrees == 2 {
		chi2 += math.Pow(float64(upperCount)-expectUpperCount, 2) / expectUpperCount
	}

	return chi2, expected
}
