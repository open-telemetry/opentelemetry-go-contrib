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

package runtime_test

import (
	goruntime "runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel/metric/metrictest"
)

func TestRuntime(t *testing.T) {
	err := runtime.Start(
		runtime.WithMinimumReadMemStatsInterval(time.Second),
	)
	assert.NoError(t, err)
	time.Sleep(time.Second)
}

func getGCCount(provider *metrictest.MeterProvider) int {
	for _, b := range provider.MeasurementBatches {
		for _, m := range b.Measurements {
			if m.Instrument.Descriptor().Name() == "runtime.go.gc.count" {
				return int(m.Number.CoerceToInt64(m.Instrument.Descriptor().NumberKind()))
			}
		}
	}
	panic("Could not locate a runtime.go.gc.count metric in test output")
}

func testMinimumInterval(t *testing.T, shouldHappen bool, opts ...runtime.Option) {
	goruntime.GC()

	var mstats0 goruntime.MemStats
	goruntime.ReadMemStats(&mstats0)
	baseline := int(mstats0.NumGC)

	provider := metrictest.NewMeterProvider()

	err := runtime.Start(
		append(
			opts,
			runtime.WithMeterProvider(provider),
		)...,
	)
	assert.NoError(t, err)

	goruntime.GC()

	provider.RunAsyncInstruments()

	require.Equal(t, 1, getGCCount(provider)-baseline)

	provider.MeasurementBatches = nil

	extra := 0
	if shouldHappen {
		extra = 3
	}

	goruntime.GC()
	goruntime.GC()
	goruntime.GC()

	provider.RunAsyncInstruments()

	require.Equal(t, 1+extra, getGCCount(provider)-baseline)
}

func TestDefaultMinimumInterval(t *testing.T) {
	testMinimumInterval(t, false)
}

func TestNoMinimumInterval(t *testing.T) {
	testMinimumInterval(t, true, runtime.WithMinimumReadMemStatsInterval(0))
}

func TestExplicitMinimumInterval(t *testing.T) {
	testMinimumInterval(t, false, runtime.WithMinimumReadMemStatsInterval(time.Hour))
}
