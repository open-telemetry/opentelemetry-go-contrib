// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package runtime // import "go.opentelemetry.io/contrib/instrumentation/runtime"

import (
	"context"
	"math"
	"runtime/debug"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
)

func TestRefreshGoCollector(t *testing.T) {
	// buffer for allocating memory
	var buffer [][]byte
	collector := newCollector(10*time.Second, runtimeMetrics)
	testClock := newClock()
	collector.now = testClock.now
	// before the first refresh, all counters are zero
	assert.Zero(t, collector.getInt(goMemoryAllocations))
	// after the first refresh, counters are non-zero
	buffer = allocateMemory(buffer)
	collector.refresh()
	initialAllocations := collector.getInt(goMemoryAllocations)
	assert.NotZero(t, initialAllocations)
	// if less than the refresh time has elapsed, the value is not updated
	// on refresh.
	testClock.increment(9 * time.Second)
	collector.refresh()
	buffer = allocateMemory(buffer)
	assert.Equal(t, initialAllocations, collector.getInt(goMemoryAllocations))
	// if greater than the refresh time has elapsed, the value changes.
	testClock.increment(2 * time.Second)
	collector.refresh()
	_ = allocateMemory(buffer)
	assert.NotEqual(t, initialAllocations, collector.getInt(goMemoryAllocations))
}

func newClock() *clock {
	return &clock{current: time.Now()}
}

type clock struct {
	current time.Time
}

func (c *clock) now() time.Time { return c.current }

func (c *clock) increment(d time.Duration) { c.current = c.current.Add(d) }

func TestRuntimeWithLimit(t *testing.T) {
	// buffer for allocating memory
	var buffer [][]byte
	_ = allocateMemory(buffer)
	t.Setenv("OTEL_GO_X_DEPRECATED_RUNTIME_METRICS", "false")
	debug.SetMemoryLimit(1234567890)
	// reset to default
	defer debug.SetMemoryLimit(math.MaxInt64)

	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))
	err := Start(WithMeterProvider(mp))
	assert.NoError(t, err)
	rm := metricdata.ResourceMetrics{}
	err = reader.Collect(context.Background(), &rm)
	assert.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	require.Len(t, rm.ScopeMetrics[0].Metrics, 8)

	expectedScopeMetric := metricdata.ScopeMetrics{
		Scope: instrumentation.Scope{
			Name:    "go.opentelemetry.io/contrib/instrumentation/runtime",
			Version: Version(),
		},
		Metrics: []metricdata.Metrics{
			{
				Name:        "go.memory.used",
				Description: "Memory used by the Go runtime.",
				Unit:        "By",
				Data: metricdata.Sum[int64]{
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: false,
					DataPoints: []metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(attribute.String("go.memory.type", "stack")),
						},
						{
							Attributes: attribute.NewSet(attribute.String("go.memory.type", "other")),
						},
					},
				},
			},
			{
				Name:        "go.memory.limit",
				Description: "Go runtime memory limit configured by the user, if a limit exists.",
				Unit:        "By",
				Data: metricdata.Sum[int64]{
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: false,
					DataPoints:  []metricdata.DataPoint[int64]{{}},
				},
			},
			{
				Name:        "go.memory.allocated",
				Description: "Memory allocated to the heap by the application.",
				Unit:        "By",
				Data: metricdata.Sum[int64]{
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: true,
					DataPoints:  []metricdata.DataPoint[int64]{{}},
				},
			},
			{
				Name:        "go.memory.allocations",
				Description: "Count of allocations to the heap by the application.",
				Unit:        "{allocation}",
				Data: metricdata.Sum[int64]{
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: true,
					DataPoints:  []metricdata.DataPoint[int64]{{}},
				},
			},
			{
				Name:        "go.memory.gc.goal",
				Description: "Heap size target for the end of the GC cycle.",
				Unit:        "By",
				Data: metricdata.Sum[int64]{
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: false,
					DataPoints:  []metricdata.DataPoint[int64]{{}},
				},
			},
			{
				Name:        "go.goroutine.count",
				Description: "Count of live goroutines.",
				Unit:        "{goroutine}",
				Data: metricdata.Sum[int64]{
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: false,
					DataPoints:  []metricdata.DataPoint[int64]{{}},
				},
			},
			{
				Name:        "go.processor.limit",
				Description: "The number of OS threads that can execute user-level Go code simultaneously.",
				Unit:        "{thread}",
				Data: metricdata.Sum[int64]{
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: false,
					DataPoints:  []metricdata.DataPoint[int64]{{}},
				},
			},
			{
				Name:        "go.config.gogc",
				Description: "Heap size target percentage configured by the user, otherwise 100.",
				Unit:        "%",
				Data: metricdata.Sum[int64]{
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: false,
					DataPoints:  []metricdata.DataPoint[int64]{{}},
				},
			},
		},
	}
	metricdatatest.AssertEqual(t, expectedScopeMetric, rm.ScopeMetrics[0], metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue())
	assertNonZeroValues(t, rm.ScopeMetrics[0])
}

func assertNonZeroValues(t *testing.T, sm metricdata.ScopeMetrics) {
	for _, m := range sm.Metrics {
		switch a := m.Data.(type) {
		case metricdata.Sum[int64]:
			for _, dp := range a.DataPoints {
				assert.Positivef(t, dp.Value, "Metric %q should have a non-zero value for point with attributes %+v", m.Name, dp.Attributes)
			}
		default:
			t.Fatalf("unexpected data type %v", a)
		}
	}
}

func allocateMemory(buffer [][]byte) [][]byte {
	return append(buffer, make([]byte, 1000000))
}
