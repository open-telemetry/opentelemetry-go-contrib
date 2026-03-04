// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package runtime // import "go.opentelemetry.io/contrib/instrumentation/runtime"

import (
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
	"go.opentelemetry.io/otel/semconv/v1.39.0/goconv"
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
	debug.SetMemoryLimit(1234567890)
	// reset to default
	defer debug.SetMemoryLimit(math.MaxInt64)

	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))
	err := Start(WithMeterProvider(mp))
	assert.NoError(t, err)
	rm := metricdata.ResourceMetrics{}
	err = reader.Collect(t.Context(), &rm)
	assert.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	require.Len(t, rm.ScopeMetrics[0].Metrics, 8)

	expectedScopeMetric := metricdata.ScopeMetrics{
		Scope: instrumentation.Scope{
			Name:    "go.opentelemetry.io/contrib/instrumentation/runtime",
			Version: Version,
		},
		Metrics: []metricdata.Metrics{
			{
				Name:        goconv.MemoryUsed{}.Name(),
				Description: goconv.MemoryUsed{}.Description(),
				Unit:        goconv.MemoryUsed{}.Unit(),
				Data: metricdata.Sum[int64]{
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: false,
					DataPoints: []metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								goconv.MemoryUsed{}.AttrMemoryType(goconv.MemoryTypeStack),
							),
						},
						{
							Attributes: attribute.NewSet(
								goconv.MemoryUsed{}.AttrMemoryType(goconv.MemoryTypeOther),
							),
						},
					},
				},
			},
			{
				Name:        goconv.MemoryLimit{}.Name(),
				Description: goconv.MemoryLimit{}.Description(),
				Unit:        goconv.MemoryLimit{}.Unit(),
				Data: metricdata.Sum[int64]{
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: false,
					DataPoints:  []metricdata.DataPoint[int64]{{}},
				},
			},
			{
				Name:        goconv.MemoryAllocated{}.Name(),
				Description: goconv.MemoryAllocated{}.Description(),
				Unit:        goconv.MemoryAllocated{}.Unit(),
				Data: metricdata.Sum[int64]{
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: true,
					DataPoints:  []metricdata.DataPoint[int64]{{}},
				},
			},
			{
				Name:        goconv.MemoryAllocations{}.Name(),
				Description: goconv.MemoryAllocations{}.Description(),
				Unit:        goconv.MemoryAllocations{}.Unit(),
				Data: metricdata.Sum[int64]{
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: true,
					DataPoints:  []metricdata.DataPoint[int64]{{}},
				},
			},
			{
				Name:        goconv.MemoryGCGoal{}.Name(),
				Description: goconv.MemoryGCGoal{}.Description(),
				Unit:        goconv.MemoryGCGoal{}.Unit(),
				Data: metricdata.Sum[int64]{
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: false,
					DataPoints:  []metricdata.DataPoint[int64]{{}},
				},
			},
			{
				Name:        goconv.GoroutineCount{}.Name(),
				Description: goconv.GoroutineCount{}.Description(),
				Unit:        goconv.GoroutineCount{}.Unit(),
				Data: metricdata.Sum[int64]{
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: false,
					DataPoints:  []metricdata.DataPoint[int64]{{}},
				},
			},
			{
				Name:        goconv.ProcessorLimit{}.Name(),
				Description: goconv.ProcessorLimit{}.Description(),
				Unit:        goconv.ProcessorLimit{}.Unit(),
				Data: metricdata.Sum[int64]{
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: false,
					DataPoints:  []metricdata.DataPoint[int64]{{}},
				},
			},
			{
				Name:        goconv.ConfigGogc{}.Name(),
				Description: goconv.ConfigGogc{}.Description(),
				Unit:        goconv.ConfigGogc{}.Unit(),
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
