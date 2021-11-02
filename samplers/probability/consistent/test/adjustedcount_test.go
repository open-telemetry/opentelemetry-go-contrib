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

package test

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/contrib/samplers/probability/consistent"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var (
	populationSize = int64(1e6)
	trials         = 20
	significance   = 1 / float64(trials)

	// import "gonum.org/v1/gonum/stat/distuv"
	// with significance = 0.05
	// chiSquaredDF1  = distuv.ChiSquared{K: 1}.Quantile(significance)
	// chiSquaredDF2  = distuv.ChiSquared{K: 2}.Quantile(significance)
	chiSquaredDF1 = 0.003932140000019522
	chiSquaredDF2 = 0.1025865887751011

	chiSquaredByDF = [3]float64{
		0,
		chiSquaredDF1,
		chiSquaredDF2,
	}
)

func init() {
	fmt.Println("chi-squared", chiSquaredByDF)
}

func parsePR(s string) (p, r string) {
	for _, kvf := range strings.Split(s, ";") {
		kv := strings.SplitN(kvf, ":", 2)
		switch kv[0] {
		case "p":
			p = kv[1]
		case "r":
			r = kv[1]
		}
	}
	return
}

type testSpanRecorder struct {
	spans []sdktrace.ReadOnlySpan
}

func (tsr *testSpanRecorder) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	tsr.spans = append(tsr.spans, spans...)
	return nil
}

func (tsr *testSpanRecorder) Shutdown(ctx context.Context) error {
	return nil
}

func sampleTrials(t *testing.T, prob float64, degrees, upperP, spans int64, source rand.Source) float64 {
	ctx := context.Background()

	sampler := consistent.ConsistentProbabilityBased(
		prob,
		consistent.WithRandomSource(source),
	)

	recorder := &testSpanRecorder{}
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(recorder),
		sdktrace.WithSampler(sampler),
	)

	tracer := provider.Tracer("test")

	for i := int64(0); i < spans; i++ {
		_, span := tracer.Start(ctx, "span")
		span.End()
	}

	var minP, maxP int64

	counts := map[int64]int64{}

	for idx, r := range recorder.spans {
		ts := r.SpanContext().TraceState()
		p, _ := parsePR(ts.Get("ot"))

		pi, err := strconv.ParseUint(p, 10, 64)
		require.NoError(t, err)

		if idx == 0 {
			maxP = int64(pi)
			minP = maxP
		} else {
			if int64(pi) < minP {
				minP = int64(pi)
			}
			if int64(pi) > maxP {
				maxP = int64(pi)
			}
		}
		counts[int64(pi)]++
	}

	require.Less(t, maxP, minP+degrees, "%v %v %v", minP, maxP, degrees)
	require.Less(t, maxP, int64(63))
	require.LessOrEqual(t, len(counts), 2)

	var upperProb, lowerProb, lowerChoice float64

	if degrees == 2 {
		if len(counts) != 0 {
			require.Equal(t, minP+1, maxP)
			require.Equal(t, upperP, maxP)
		}
		upperProb = 1 / float64(int64(1)<<minP)
		lowerProb = 1 / float64(int64(1)<<maxP)
		lowerChoice = (upperProb - prob) / (upperProb - lowerProb)
	} else {
		if len(counts) != 0 {
			require.Equal(t, minP, maxP)
			require.Equal(t, upperP, maxP)
		}
		upperProb = 0
		lowerProb = prob
		lowerChoice = 1
	}

	expectLowerCount := lowerChoice * lowerProb * float64(spans)
	expectUpperCount := (1 - lowerChoice) * upperProb * float64(spans)
	expectUnsampled := (1 - prob) * float64(spans)

	fmt.Println("Prob", prob, "Low", expectLowerCount, "Up", expectUpperCount, "Uns", expectUnsampled)

	upperCount := int64(0)
	lowerCount := counts[maxP]
	if degrees == 2 {
		upperCount = counts[minP]
	}
	unsampled := spans - upperCount - lowerCount

	chi2 := 0.0
	chi2 += math.Pow(float64(unsampled)-expectUnsampled, 2) / expectUnsampled
	chi2 += math.Pow(float64(lowerCount)-expectLowerCount, 2) / expectLowerCount
	if degrees == 2 {
		chi2 += math.Pow(float64(upperCount)-expectUpperCount, 2) / expectUpperCount
	}

	return chi2
}

type probSeed struct {
	prob    float64
	upperP  int64
	degrees int64
	seed    int64
}

func TestPowerOfTwoSampling(t *testing.T) {
	// Note that each of the seeds used in this test comes from the
	// source below.  The seed's position in the sequence is listed
	// in the comments below.
	newSource := func() rand.Source {
		return rand.NewSource(77777677777)
	}
	for _, ps := range []probSeed{
		// Non-powers of two
		{0.6, 1, 2, 0x7e436d5ff98928b8},     // 14th
		{0.33, 2, 2, 0x3b7603f0b2596e2f},    // 4th
		{0.1, 4, 2, 0x31fb79834649508c},     // 1st
		{0.05, 5, 2, 0x31fb79834649508c},    // 1st
		{0.01, 7, 2, 0x3b7603f0b2596e2f},    // 4th
		{0.005, 8, 2, 0x222159400ee1080},    // 2nd
		{0.001, 10, 2, 0x62e520750be95257},  // 6th
		{0.0005, 11, 2, 0x222159400ee1080},  // 2nd
		{0.0001, 14, 2, 0x53c7cf004663c656}, // 3rd

		// Powers of two
		{0x1p-1, 1, 1, 0x31fb79834649508c},   // 1st
		{0x1p-4, 4, 1, 0x53c7cf004663c656},   // 3rd
		{0x1p-7, 7, 1, 0x3b7603f0b2596e2f},   // 4th
		{0x1p-10, 10, 1, 0x31fb79834649508c}, // 1st
		{0x1p-13, 13, 1, 0x222159400ee1080},  // 2nd
	} {
		t.Run(fmt.Sprint(ps.prob), func(t *testing.T) {
			rnd := rand.New(newSource())

			for {
				seed := ps.seed
				if seed == 0 {
					seed = rnd.Int63()
				}
				src := rand.NewSource(seed)
				less := 0

				for j := 0; j < trials; j++ {
					x := sampleTrials(t, ps.prob, ps.degrees, ps.upperP, populationSize, src)

					if x < chiSquaredByDF[ps.degrees] {
						less++
					}
				}

				if less != 1 && ps.seed == 0 {
					t.Logf("%d probabilistic failures, trying a new seed for %g was 0x%x", less, ps.prob, seed)
					continue
				} else if less != 1 {
					t.Errorf("incorrect number of probabilistic failures, should be 1 was %d", less)
				} else if ps.seed == 0 {
					t.Logf("update the test for %g to use seed 0x%x", ps.prob, seed)
					t.Fail()
					return
				} else {
					// pass
					break
				}
			}
		})
	}
}
