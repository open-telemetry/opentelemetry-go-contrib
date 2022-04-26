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
	"context"
	"fmt"
	goruntime "runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel/sdk/metric/export/aggregation"
	"go.opentelemetry.io/otel/sdk/metric/metrictest"
)

func TestRuntime(t *testing.T) {
	err := runtime.Start(
		runtime.WithMinimumReadMemStatsInterval(time.Second),
	)
	assert.NoError(t, err)
	time.Sleep(time.Second)
}

func getGCCount(exp *metrictest.Exporter) int {
	for _, r := range exp.GetRecords() {
		if r.InstrumentName == "process.runtime.go.gc.count" {
			switch r.AggregationKind {
			case aggregation.SumKind, aggregation.HistogramKind:
				return int(r.Sum.CoerceToInt64(r.NumberKind))
			case aggregation.LastValueKind:
				return int(r.LastValue.CoerceToInt64(r.NumberKind))
			default:
				panic(fmt.Sprintf("invalid aggregation type: %v", r.AggregationKind))
			}
		}
	}
	panic("Could not locate a process.runtime.go.gc.count metric in test output")
}

func testMinimumInterval(t *testing.T, shouldHappen bool, opts ...runtime.Option) {
	goruntime.GC()

	var mstats0 goruntime.MemStats
	goruntime.ReadMemStats(&mstats0)
	baseline := int(mstats0.NumGC)

	provider, exp := metrictest.NewTestMeterProvider()

	err := runtime.Start(
		append(
			opts,
			runtime.WithMeterProvider(provider),
		)...,
	)
	assert.NoError(t, err)

	goruntime.GC()

	require.NoError(t, exp.Collect(context.Background()))

	require.Equal(t, 1, getGCCount(exp)-baseline)

	extra := 0
	if shouldHappen {
		extra = 3
	}

	goruntime.GC()
	goruntime.GC()
	goruntime.GC()

	require.NoError(t, exp.Collect(context.Background()))

	require.Equal(t, 1+extra, getGCCount(exp)-baseline)
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
