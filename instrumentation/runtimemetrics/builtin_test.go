// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package runtimemetrics

import (
	"context"
	"fmt"
	"runtime/metrics"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

const (
	classKey       = attribute.Key("class")
	subclassKey    = attribute.Key("class2")
	subsubclassKey = attribute.Key("class3")
)

func TestMetricTranslation1(t *testing.T) {
	testMetricTranslation(t, makeTestCase1)
}

func TestMetricTranslation2(t *testing.T) {
	testMetricTranslation(t, makeTestCase2)
}

func TestMetricTranslationBuiltin(t *testing.T) {
	testMetricTranslation(t, makeTestCaseBuiltin)
}

// makeAllInts retrieves real metric.Values.  We use real
// runtime/metrics values b/c these can't be constructed outside the
// library.
//
// Note that all current metrics through go-1.20 are either histogram
// or integer valued, so although Float64 values are supported in
// theory, they are not tested _and can't be tested because the
// library never produces them_.
func makeAllInts() (allInts, allFloats []metrics.Value) {
	// Note: the library provides no way to generate values, so use the
	// builtin library to get some.  Since we can't generate a Float64 value
	// we can't even test the Gauge logic in this package.
	ints := map[metrics.Value]bool{}
	floats := map[metrics.Value]bool{}

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
		case metrics.KindFloat64:
			floats[rs.Value] = true
		default:
			// Histograms are not tested.
		}
	}
	for iv := range ints {
		allInts = append(allInts, iv)
	}
	for fv := range floats {
		allFloats = append(allFloats, fv)
	}
	return allInts, allFloats
}

// testMapping implements a synthetic metrics reader.
type testMapping map[string]metrics.Value

// read is like metrics.Read w/ synthetic data.
func (m testMapping) read(samples []metrics.Sample) {
	for i := range samples {
		v, ok := m[samples[i].Name]
		if ok {
			samples[i].Value = v
		} else {
			panic("outcome uncertain")
		}
	}
}

// readFrom turns real runtime/metrics data into a test expectation.
func (m testMapping) readFrom(af allFunc, rf readFunc) {
	all := af()
	samples := make([]metrics.Sample, len(all))
	for i := range all {
		switch all[i].Kind {
		case metrics.KindUint64, metrics.KindFloat64:
		default:
			continue
		}
		samples[i].Name = all[i].Name
	}
	rf(samples)
	for i := range samples {
		m[samples[i].Name] = samples[i].Value
	}
}

// testExpectation allows validating the behavior using
// hand-constructed test cases.
type testExpectation map[string]*testExpectMetric

// testExpectMetric sets a test expectation consisting of name, unit,
// and known cardinal values.
type testExpectMetric struct {
	desc string
	unit string
	kind builtinKind
	vals map[attribute.Set]metrics.Value
}

// makeTestCase1 covers the following cases:
// - single counter, updowncounter, gauge
// - bytes/objects counter
// - classes counter (gc-cycles)
func makeTestCase1(t *testing.T) (allFunc, readFunc, *builtinDescriptor, testExpectation) {
	allInts, _ := makeAllInts()

	af := func() []metrics.Description {
		return []metrics.Description{
			{
				Name:       "/cntr/things:things",
				Kind:       metrics.KindUint64,
				Cumulative: true,
			},
			{
				Name:       "/updowncntr/things:things",
				Kind:       metrics.KindUint64,
				Cumulative: false,
			},
			{
				Name:       "/process/count:objects",
				Kind:       metrics.KindUint64,
				Cumulative: true,
			},
			{
				Name:       "/cpu/temp:C",
				Kind:       metrics.KindUint64, // TODO: post Go-1.20 make this Float64
				Cumulative: false,
			},
			{
				Name:       "/process/count:bytes",
				Kind:       metrics.KindUint64,
				Cumulative: true,
			},
			{
				Name:       "/waste/cycles/ocean:gc-cycles",
				Kind:       metrics.KindUint64,
				Cumulative: true,
			},
			{
				Name:       "/waste/cycles/sea:gc-cycles",
				Kind:       metrics.KindUint64,
				Cumulative: true,
			},
			{
				Name:       "/waste/cycles/lake:gc-cycles",
				Kind:       metrics.KindUint64,
				Cumulative: true,
			},
			{
				Name:       "/waste/cycles/pond:gc-cycles",
				Kind:       metrics.KindUint64,
				Cumulative: true,
			},
			{
				Name:       "/waste/cycles/puddle:gc-cycles",
				Kind:       metrics.KindUint64,
				Cumulative: true,
			},
			{
				Name:       "/waste/cycles/total:gc-cycles",
				Kind:       metrics.KindUint64,
				Cumulative: true,
			},
		}
	}
	mapping := testMapping{
		"/cntr/things:things":            allInts[0],
		"/updowncntr/things:things":      allInts[1],
		"/process/count:objects":         allInts[2],
		"/process/count:bytes":           allInts[3],
		"/waste/cycles/ocean:gc-cycles":  allInts[4],
		"/waste/cycles/sea:gc-cycles":    allInts[5],
		"/waste/cycles/lake:gc-cycles":   allInts[6],
		"/waste/cycles/pond:gc-cycles":   allInts[7],
		"/waste/cycles/puddle:gc-cycles": allInts[8],
		"/waste/cycles/total:gc-cycles":  allInts[9],

		// Note: this would be a nice float test, but 1.19 doesn't have
		// any, so we wait for this repo's min Go version to support a
		// metrics.KindFloat64 value for testing.
		"/cpu/temp:C": allInts[10],
	}
	bd := newBuiltinDescriptor()
	bd.singleCounter("/cntr/things:things")
	bd.singleUpDownCounter("/updowncntr/things:things")
	bd.singleGauge("/cpu/temp:C")
	bd.objectBytesCounter("/process/count:*")
	bd.classesCounter("/waste/cycles/*:gc-cycles")
	return af, mapping.read, bd, testExpectation{
		"cntr.things": &testExpectMetric{
			unit: "{things}",
			desc: "/cntr/things:things from runtime/metrics",
			kind: builtinCounter,
			vals: map[attribute.Set]metrics.Value{
				emptySet: allInts[0],
			},
		},
		"updowncntr.things": &testExpectMetric{
			unit: "{things}",
			desc: "/updowncntr/things:things from runtime/metrics",
			kind: builtinUpDownCounter,
			vals: map[attribute.Set]metrics.Value{
				emptySet: allInts[1],
			},
		},
		"process.count.objects": &testExpectMetric{
			unit: "",
			desc: "/process/count:objects from runtime/metrics",
			kind: builtinCounter,
			vals: map[attribute.Set]metrics.Value{
				emptySet: allInts[2],
			},
		},
		"process.count": &testExpectMetric{
			unit: "By",
			kind: builtinCounter,
			desc: "/process/count:bytes from runtime/metrics",
			vals: map[attribute.Set]metrics.Value{
				emptySet: allInts[3],
			},
		},
		"waste.cycles": &testExpectMetric{
			unit: "{gc-cycles}",
			desc: "/waste/cycles/*:gc-cycles from runtime/metrics",
			kind: builtinCounter,
			vals: map[attribute.Set]metrics.Value{
				attribute.NewSet(classKey.String("ocean")):  allInts[4],
				attribute.NewSet(classKey.String("sea")):    allInts[5],
				attribute.NewSet(classKey.String("lake")):   allInts[6],
				attribute.NewSet(classKey.String("pond")):   allInts[7],
				attribute.NewSet(classKey.String("puddle")): allInts[8],
			},
		},
		"cpu.temp": &testExpectMetric{
			// This is made-up.  We don't recognize this
			// unit, code defaults to pseudo-units.
			unit: "{C}",
			kind: builtinGauge,
			desc: "/cpu/temp:C from runtime/metrics",
			vals: map[attribute.Set]metrics.Value{
				emptySet: allInts[10],
			},
		},
	}
}

// makeTestCase2 covers the following cases:
// - classes counter (bytes)
// - classes counter (cpu-seconds)
func makeTestCase2(t *testing.T) (allFunc, readFunc, *builtinDescriptor, testExpectation) {
	allInts, _ := makeAllInts()

	af := func() []metrics.Description {
		return []metrics.Description{
			// classes (bytes)
			{
				Name:       "/objsize/classes/presos:bytes",
				Kind:       metrics.KindUint64,
				Cumulative: false,
			},
			{
				Name:       "/objsize/classes/sheets:bytes",
				Kind:       metrics.KindUint64,
				Cumulative: false,
			},
			{
				Name:       "/objsize/classes/docs/word:bytes",
				Kind:       metrics.KindUint64,
				Cumulative: false,
			},
			{
				Name:       "/objsize/classes/docs/pdf:bytes",
				Kind:       metrics.KindUint64,
				Cumulative: false,
			},
			{
				Name:       "/objsize/classes/docs/total:bytes",
				Kind:       metrics.KindUint64,
				Cumulative: false,
			},
			{
				Name:       "/objsize/classes/total:bytes",
				Kind:       metrics.KindUint64,
				Cumulative: false,
			},
			// classes (time)
			{
				Name:       "/socchip/classes/pru:cpu-seconds",
				Kind:       metrics.KindUint64,
				Cumulative: true,
			},
			{
				Name:       "/socchip/classes/dsp:cpu-seconds",
				Kind:       metrics.KindUint64,
				Cumulative: true,
			},
			{
				Name:       "/socchip/classes/total:cpu-seconds",
				Kind:       metrics.KindUint64,
				Cumulative: true,
			},
		}
	}
	mapping := testMapping{
		"/objsize/classes/presos:bytes":      allInts[0],
		"/objsize/classes/sheets:bytes":      allInts[1],
		"/objsize/classes/docs/word:bytes":   allInts[2],
		"/objsize/classes/docs/pdf:bytes":    allInts[3],
		"/objsize/classes/docs/total:bytes":  allInts[4],
		"/objsize/classes/total:bytes":       allInts[5],
		"/socchip/classes/pru:cpu-seconds":   allInts[6],
		"/socchip/classes/dsp:cpu-seconds":   allInts[7],
		"/socchip/classes/total:cpu-seconds": allInts[8],
	}
	bd := newBuiltinDescriptor()
	bd.classesUpDownCounter("/objsize/classes/*:bytes")
	bd.classesCounter("/socchip/classes/*:cpu-seconds")
	return af, mapping.read, bd, testExpectation{
		"objsize.usage": &testExpectMetric{
			unit: "By",
			desc: "/objsize/classes/*:bytes from runtime/metrics",
			kind: builtinUpDownCounter,
			vals: map[attribute.Set]metrics.Value{
				attribute.NewSet(classKey.String("presos")):                           allInts[0],
				attribute.NewSet(classKey.String("sheets")):                           allInts[1],
				attribute.NewSet(classKey.String("docs"), subclassKey.String("word")): allInts[2],
				attribute.NewSet(classKey.String("docs"), subclassKey.String("pdf")):  allInts[3],
			},
		},
		"socchip.time": &testExpectMetric{
			unit: "{cpu-seconds}",
			desc: "/socchip/classes/*:cpu-seconds from runtime/metrics",
			kind: builtinCounter,
			vals: map[attribute.Set]metrics.Value{
				attribute.NewSet(classKey.String("pru")): allInts[6],
				attribute.NewSet(classKey.String("dsp")): allInts[7],
			},
		},
	}
}

// makeTestCaseBuiltin fabricates a test expectation from the
// version-specific portion of the descriptor, synthesizes values and
// checks the result for a match.
func makeTestCaseBuiltin(t *testing.T) (allFunc, readFunc, *builtinDescriptor, testExpectation) {
	testMap := testMapping{}
	testMap.readFrom(metrics.All, metrics.Read)

	realDesc := expectRuntimeMetrics()

	expect := testExpectation{}

	for goname, realval := range testMap {
		mname, munit, descPat, attrs, kind, err := realDesc.findMatch(goname)
		if err != nil || mname == "" || kind == builtinSkip {
			continue // e.g., async histogram data, totalized metrics
		}
		noprefix := mname[len(namePrefix)+1:]
		te, ok := expect[noprefix]
		if !ok {
			te = &testExpectMetric{
				desc: fmt.Sprint(descPat, " from runtime/metrics"),
				unit: munit,
				kind: kind,
				vals: map[attribute.Set]metrics.Value{},
			}
			expect[noprefix] = te
		}
		te.vals[attribute.NewSet(attrs...)] = realval
	}

	return metrics.All, testMap.read, realDesc, expect
}

// testMetricTranslation registers the metrics allFunc and readFunc
// functions using the descriptor and validates the test expectation.
func testMetricTranslation(t *testing.T, makeTestCase func(t *testing.T) (allFunc, readFunc, *builtinDescriptor, testExpectation)) {
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))

	af, rf, desc, expectation := makeTestCase(t)
	br := newBuiltinRuntime(provider.Meter("test"), af, rf)
	err := br.register(desc)
	require.NoError(t, err)

	var data metricdata.ResourceMetrics
	err = reader.Collect(context.Background(), &data)
	require.NoError(t, err)

	require.Equal(t, 1, len(data.ScopeMetrics))

	require.Equal(t, 1, len(data.ScopeMetrics))
	require.Equal(t, "test", data.ScopeMetrics[0].Scope.Name)

	// Compare the name sets, to make the test output readable.
	haveNames := map[string]bool{}
	expectNames := map[string]bool{}

	for _, m := range data.ScopeMetrics[0].Metrics {
		require.True(t, strings.HasPrefix(m.Name, namePrefix))
		haveNames[m.Name[len(namePrefix)+1:]] = true
	}
	for n := range expectation {
		expectNames[n] = true
	}
	require.Equal(t, expectNames, haveNames)

	for _, inst := range data.ScopeMetrics[0].Metrics {
		// Test name, description, and unit.
		require.True(t, strings.HasPrefix(inst.Name, namePrefix+"."), "%s", inst.Name)

		name := inst.Name[len(namePrefix)+1:]
		exm := expectation[name]

		require.Equal(t, exm.desc, inst.Description)
		require.Equal(t, exm.unit, inst.Unit)

		// The counter and gauge branches do make the same
		// checks, just have to be split b/c the underlying
		// types are different.
		switch exm.kind {
		case builtinCounter, builtinUpDownCounter:
			// Handle both int/float cases.  Note: If we could write
			// in-line generic code, this could be less repetitive.
			if _, isInt := inst.Data.(metricdata.Sum[int64]); isInt {
				// Integer

				_, isSum := inst.Data.(metricdata.Sum[int64])
				// Expect a sum data point w/ correct monotonicity.
				require.True(t, isSum, "%v", exm)
				require.Equal(t, exm.kind == builtinCounter, inst.Data.(metricdata.Sum[int64]).IsMonotonic, "%v", exm)
				require.Equal(t, metricdata.CumulativeTemporality, inst.Data.(metricdata.Sum[int64]).Temporality, "%v", exm)

				// Check expected values.
				for _, point := range inst.Data.(metricdata.Sum[int64]).DataPoints {
					lookup, ok := exm.vals[point.Attributes]
					require.True(t, ok, "lookup failed: %v: %v", exm.vals, point.Attributes)
					require.Equal(t, lookup.Uint64(), uint64(point.Value))
				}
			} else {
				// Floating point

				_, isSum := inst.Data.(metricdata.Sum[float64])
				// Expect a sum data point w/ correct monotonicity.
				require.True(t, isSum, "%v", exm)
				require.Equal(t, exm.kind == builtinCounter, inst.Data.(metricdata.Sum[float64]).IsMonotonic, "%v", exm)
				require.Equal(t, metricdata.CumulativeTemporality, inst.Data.(metricdata.Sum[float64]).Temporality, "%v", exm)

				// Check expected values.
				for _, point := range inst.Data.(metricdata.Sum[float64]).DataPoints {
					lookup, ok := exm.vals[point.Attributes]
					require.True(t, ok, "lookup failed: %v: %v", exm.vals, point.Attributes)
					require.Equal(t, lookup.Float64(), float64(point.Value))
				}
			}
		case builtinGauge:
			if _, isInt := inst.Data.(metricdata.Gauge[int64]); isInt {
				// Integer
				_, isGauge := inst.Data.(metricdata.Gauge[int64])
				require.True(t, isGauge, "%v", exm)

				// Check expected values.
				for _, point := range inst.Data.(metricdata.Gauge[int64]).DataPoints {
					lookup, ok := exm.vals[point.Attributes]
					require.True(t, ok, "lookup failed: %v: %v", exm.vals, point.Attributes)
					require.Equal(t, lookup.Uint64(), uint64(point.Value))
				}
			} else {
				// Floating point
				_, isGauge := inst.Data.(metricdata.Gauge[float64])
				require.True(t, isGauge, "%v", exm)

				// Check expected values.
				for _, point := range inst.Data.(metricdata.Gauge[float64]).DataPoints {
					lookup, ok := exm.vals[point.Attributes]
					require.True(t, ok, "lookup failed: %v: %v", exm.vals, point.Attributes)
					require.Equal(t, lookup.Float64(), float64(point.Value))
				}
			}
		default:
			t.Errorf("unexpected runtimes/metric test case: %v", exm)
			continue
		}
	}
}
