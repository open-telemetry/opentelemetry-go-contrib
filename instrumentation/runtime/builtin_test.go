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

package runtime

import (
	"context"
	"runtime/metrics"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metrictest"
)

var expectLib = metrictest.Library{
	InstrumentationName:    LibraryName,
	InstrumentationVersion: SemVersion(),
	SchemaURL:              "",
}

// TestBuiltinRuntimeMetrics tests the real output of the library to
// ensure expected prefix, instrumentation scope, and empty
// attributes.
func TestBuiltinRuntimeMetrics(t *testing.T) {
	provider, exp := metrictest.NewTestMeterProvider()

	err := Start(
		WithUseGoRuntimeMetrics(true),
		WithMeterProvider(provider),
	)

	require.NoError(t, err)

	require.NoError(t, exp.Collect(context.Background()))

	const prefix = "process.runtime.go."

	// Note: metrictest library lacks a way to distinguish
	// monotonic vs not or to test the unit. This will be fixed in
	// the new SDK, all the pieces untested here.
	// TODO: add testing in the new SDK's metrictest.
	for _, rec := range exp.Records {
		require.Regexp(t, `^process\.runtime\.go\..+`, rec.InstrumentName)
		require.Equal(t, expectLib, rec.InstrumentationLibrary)
		require.Equal(t, []attribute.KeyValue(nil), rec.Attributes)
	}

}

func makeTestCase() (allFunc, readFunc, map[string]metrics.Value) {
	// Note: the library provides no way to generate values, so use the
	// builtin library to get some.  Since we can't generate a Float64 value
	// we can't even test the Gauge logic in this package.
	ints := map[metrics.Value]bool{}

	real := metrics.All()
	realSamples := make([]metrics.Sample, len(real))
	for i := range real {
		realSamples[i].Name = real[i].Name
	}
	metrics.Read(realSamples)
	for i, rs := range realSamples {
		switch real[i].Kind {
		case metrics.KindUint64:
			ints[rs.Value] = true
		default:
			// Histograms and Floats are not tested.
			// The 1.19 runtime generates no Floats and
			// exports no test constructors.
		}
	}

	var allInts []metrics.Value

	for iv := range ints {
		allInts = append(allInts, iv)
	}

	af := func() []metrics.Description {
		return []metrics.Description{
			{
				Name:        "/cntr/things:things",
				Description: "a counter of things",
				Kind:        metrics.KindUint64,
				Cumulative:  true,
			},
			{
				Name:        "/updowncntr/things:things",
				Description: "an updowncounter of things",
				Kind:        metrics.KindUint64,
				Cumulative:  false,
			},
			{
				Name:        "/process/count:things",
				Description: "a process counter of things",
				Kind:        metrics.KindUint64,
				Cumulative:  true,
			},
			{
				Name:        "/process/count:parts",
				Description: "a process counter of parts",
				Kind:        metrics.KindUint64,
				Cumulative:  true,
			},
		}
	}
	mapping := map[string]metrics.Value{
		"/cntr/things:things":       allInts[0],
		"/updowncntr/things:things": allInts[1],
		"/process/cntr:things":      allInts[2],
		"/process/cntr:parts":       allInts[3],
	}
	rf := func(samples []metrics.Sample) {
		for i := range samples {
			v, ok := mapping[samples[i].Name]
			if ok {
				samples[i].Value = v
			}
		}
	}
	return af, rf, map[string]metrics.Value{
		"cntr.things":         allInts[0],
		"updowncntr.things":   allInts[1],
		"process.cntr.things": allInts[2],
		"process.cntr.parts":  allInts[3],
	}
}

// TestMetricTranslation validates the translation logic using
// synthetic metric names and values.
func TestMetricTranslation(t *testing.T) {
	provider, exp := metrictest.NewTestMeterProvider()

	af, rf, mapping := makeTestCase()
	br := newBuiltinRuntime(provider.Meter("test"), af, rf)
	br.register()

	const prefix = "process.runtime.go."

	for _, rec := range exp.Records {
		require.Regexp(t, `^process\.runtime\.go\..+`, rec.InstrumentName)
		require.Equal(t, expectLib, rec.InstrumentationLibrary)
		require.Equal(t, []attribute.KeyValue(nil), rec.Attributes)

		name := rec.InstrumentName[len("process.runtime.go."):]

		// Note: only int64 is tested, we have no way to
		// generate Float64 values and Float64Hist values are
		// not implemented for testing.

		require.Equal(t, mapping[name].Uint64, uint64(rec.Sum.AsInt64()))
	}

}
