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
	"math/rand"
	"sort"
	"testing"

	"go.opentelemetry.io/contrib/samplers/probability/consistent"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"gonum.org/v1/gonum/stat/distuv"
)

type testSpanRecorder struct {
	spans []sdktrace.ReadOnlySpan
}

func newSource() rand.Source {
	return rand.NewSource(77777677777)
}

func TestAdjustedCount(t *testing.T) {
	// testPowerOfTwo(t, 500, 100, 1000, 0.5, newSource())
	// testPowerOfTwo(t, 500, 100, 2000, 0.25, newSource())
	testPowerOfTwo(t, 5000, 100, 4000, 0.125, newSource())
}

func testPowerOfTwo(t *testing.T, repeats, trials, tosses int, prob float64, source rand.Source) {

	ctx := context.Background()

	sampler := consistent.ConsistentProbabilityBased(
		prob,
		consistent.WithRandomSource(source),
	)

	modelDist := distuv.Binomial{
		N: float64(tosses),
		P: prob,
	}

	var dvalues []float64

	for repeat := 0; repeat < repeats; repeat++ {
		var results []int

		for trial := 0; trial < trials; trial++ {
			recorder := &testSpanRecorder{}
			provider := sdktrace.NewTracerProvider(
				sdktrace.WithSyncer(recorder),
				sdktrace.WithSampler(sampler),
			)

			tracer := provider.Tracer("test")

			recording := 0
			for i := 0; i < tosses; i++ {
				_, span := tracer.Start(ctx, "span")
				if span.IsRecording() {
					recording++
				}
				span.End()
			}
			results = append(results, recording)
		}

		sort.Ints(results)

		// Like Knuth 3.3.1 algorithm B, one-sample, but without the
		// sqrt(trials) term, thus using the exact Kolmogorov D
		// distribution instead of K+ and K- like the text.
		d := 0.0

		for i := 0; i < trials; i++ {
			for i < trials-1 && results[i+1] == results[i] {
				i++ // Scanning past duplicates
			}

			x := float64(results[i])
			low := float64(i+1) / float64(trials)
			high := float64(i) / float64(trials)

			if dPlus := low - modelDist.CDF(x); dPlus > d {
				d = dPlus
			}
			if dMinus := modelDist.CDF(x) - high; dMinus > d {
				d = dMinus
			}
		}

		//fmt.Printf("K single D %f%%\n", 100*kolmogorov(trials, d))

		dvalues = append(dvalues, d)

		//if len(dvalues)%step == 0 {
		show(dvalues, trials)
		//}
	}
}

func show(dvalues []float64, trials int) {
	repeats := len(dvalues)

	sort.Float64s(dvalues)

	d := 0.0

	for i := 0; i < repeats; i++ {
		for i < repeats-1 && dvalues[i+1] == dvalues[i] {
			i++ // Scanning past duplicates
		}

		x := float64(dvalues[i])
		low := float64(i+1) / float64(repeats)
		high := float64(i) / float64(repeats)

		if dPlus := low - kolmogorov(trials, x); dPlus > d {
			d = dPlus
		}
		if dMinus := kolmogorov(trials, x) - high; dMinus > d {
			d = dMinus
		}
	}

	fmt.Printf("K multi D %f%%\n", 100*kolmogorov(repeats, d))
}

func (tsr *testSpanRecorder) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	tsr.spans = append(tsr.spans, spans...)
	return nil
}

func (tsr *testSpanRecorder) Shutdown(ctx context.Context) error {
	return nil
}
