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
	"context"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type (
	testDegrees int
	pValue      int

	testSpanRecorder struct {
		lock  sync.Mutex
		spans []sdktrace.ReadOnlySpan
	}

	testErrorHandler struct {
		lock   sync.Mutex
		errors []error
	}
)

const (
	oneDegree  testDegrees = 1
	twoDegrees testDegrees = 2
)

var (
	populationSize = 1e6
	trials         = 20

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

func (tsr *testSpanRecorder) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	tsr.lock.Lock()
	defer tsr.lock.Unlock()
	tsr.spans = append(tsr.spans, spans...)
	return nil
}

func (tsr *testSpanRecorder) Shutdown(ctx context.Context) error {
	return nil
}

func (eh *testErrorHandler) Handle(err error) {
	eh.lock.Lock()
	defer eh.lock.Unlock()
	eh.errors = append(eh.errors, err)
}

func (eh *testErrorHandler) Errors() []error {
	eh.lock.Lock()
	defer eh.lock.Unlock()
	return eh.errors
}

func TestSamplerDescription(t *testing.T) {
	const minProb = 0x1p-62 // 2.168404344971009e-19

	for _, tc := range []struct {
		prob   float64
		expect string
	}{
		{1, "ProbabilityBased{1}"},
		{0, "ProbabilityBased{0}"},
		{0.75, "ProbabilityBased{0.75}"},
		{0.05, "ProbabilityBased{0.05}"},
		{0.003, "ProbabilityBased{0.003}"},
		{0.99999999, "ProbabilityBased{0.99999999}"},
		{0.00000001, "ProbabilityBased{1e-08}"},
		{minProb, "ProbabilityBased{2.168404344971009e-19}"},
		{minProb * 1.5, "ProbabilityBased{3.2526065174565133e-19}"},
		{3e-19, "ProbabilityBased{3e-19}"},

		// out-of-range > 1
		{1.01, "ProbabilityBased{1}"},
		{101.1, "ProbabilityBased{1}"},

		// out-of-range < 2^-62
		{-1, "ProbabilityBased{0}"},
		{-0.001, "ProbabilityBased{0}"},
		{minProb * 0.999, "ProbabilityBased{0}"},
	} {
		s := ProbabilityBased(tc.prob)
		require.Equal(t, tc.expect, s.Description(), "%#v", tc.prob)
	}
}

func getUnknowns(otts otelTraceState) string {
	otts.pvalue = invalidValue
	otts.rvalue = invalidValue
	return otts.serialize()
}

func TestSamplerBehavior(t *testing.T) {
	type testGroup struct {
		probability float64
		minP        uint8
		maxP        uint8
	}
	type testCase struct {
		isRoot        bool
		parentSampled bool
		ctxTracestate string
		hasErrors     bool
	}

	for _, group := range []testGroup{
		{1.0, 0, 0},
		{0.75, 0, 1},
		{0.5, 1, 1},
		{0, 63, 63},
	} {
		t.Run(fmt.Sprint(group.probability), func(t *testing.T) {
			for _, test := range []testCase{
				// roots do not care if the context is
				// sampled, however preserve other
				// otel tracestate keys
				{true, false, "", false},
				{true, false, "a:b", false},

				// non-roots insert r
				{false, true, "", false},
				{false, true, "a:b", false},
				{false, false, "", false},
				{false, false, "a:b", false},

				// error cases: r-p inconsistency
				{false, true, "r:10;p:20", true},
				{false, true, "r:10;p:20;a:b", true},
				{false, false, "r:10;p:5", true},
				{false, false, "r:10;p:5;a:b", true},

				// error cases: out-of-range
				{false, false, "r:100", true},
				{false, false, "r:100;a:b", true},
				{false, true, "r:100;p:100", true},
				{false, true, "r:100;p:100;a:b", true},
				{false, true, "r:10;p:100", true},
				{false, true, "r:10;p:100;a:b", true},
			} {
				t.Run(testName(test.ctxTracestate), func(t *testing.T) {
					handler := &testErrorHandler{}
					otel.SetErrorHandler(handler)

					src := rand.NewSource(99999199999)
					sampler := ProbabilityBased(group.probability, WithRandomSource(src))

					traceID, _ := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
					spanID, _ := trace.SpanIDFromHex("00f067aa0ba902b7")

					traceState := trace.TraceState{}
					if test.ctxTracestate != "" {
						var err error
						traceState, err = traceState.Insert(traceStateKey, test.ctxTracestate)
						require.NoError(t, err)
					}

					sccfg := trace.SpanContextConfig{
						TraceState: traceState,
					}

					if !test.isRoot {
						sccfg.TraceID = traceID
						sccfg.SpanID = spanID
					}

					if test.parentSampled {
						sccfg.TraceFlags = trace.FlagsSampled
					}

					parentCtx := trace.ContextWithSpanContext(
						context.Background(),
						trace.NewSpanContext(sccfg),
					)

					// Note: the error below is sometimes expected
					testState, _ := parseOTelTraceState(test.ctxTracestate, test.parentSampled)
					hasRValue := testState.hasRValue()

					const repeats = 10
					for i := 0; i < repeats; i++ {
						result := sampler.ShouldSample(
							sdktrace.SamplingParameters{
								ParentContext: parentCtx,
								TraceID:       traceID,
								Name:          "test",
								Kind:          trace.SpanKindServer,
							},
						)
						sampled := result.Decision == sdktrace.RecordAndSample

						// The result is deterministically random. Parse the tracestate
						// to see that it is consistent.
						otts, err := parseOTelTraceState(result.Tracestate.Get(traceStateKey), sampled)
						require.NoError(t, err)
						require.True(t, otts.hasRValue())
						require.Equal(t, []attribute.KeyValue(nil), result.Attributes)

						if otts.hasPValue() {
							require.LessOrEqual(t, group.minP, otts.pvalue)
							require.LessOrEqual(t, otts.pvalue, group.maxP)
							require.Equal(t, sdktrace.RecordAndSample, result.Decision)
						} else {
							require.Equal(t, sdktrace.Drop, result.Decision)
						}

						require.Equal(t, getUnknowns(testState), getUnknowns(otts))

						if hasRValue {
							require.Equal(t, testState.rvalue, otts.rvalue)
						}

						if test.hasErrors {
							require.Less(t, 0, len(handler.Errors()))
						} else {
							require.Equal(t, 0, len(handler.Errors()))
						}
					}
				})
			}
		})
	}
}

func sampleTrials(t *testing.T, prob float64, degrees testDegrees, upperP pValue, source rand.Source) (float64, []float64) {
	ctx := context.Background()

	sampler := ProbabilityBased(
		prob,
		WithRandomSource(source),
	)

	recorder := &testSpanRecorder{}
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

	for idx, r := range recorder.spans {
		ts := r.SpanContext().TraceState()
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
	require.NotEqual(t, 0, len(counts))

	if degrees == 2 {
		require.Equal(t, minP+1, maxP)
		require.Equal(t, upperP, maxP)
		ceilingProb = 1 / float64(int64(1)<<minP)
		floorProb = 1 / float64(int64(1)<<maxP)
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
