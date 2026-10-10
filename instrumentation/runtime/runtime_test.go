// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"math"
	goruntime "runtime"
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
	"go.opentelemetry.io/otel/semconv/v1.43.0/goconv"
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

func TestRuntimeOptInMetrics(t *testing.T) {
	t.Run("option", func(t *testing.T) {
		testRuntimeOptInMetrics(t, WithOptInMetrics(MemoryGCCycles, CPUTime))
	})
	t.Run("environment", func(t *testing.T) {
		t.Setenv("OTEL_GO_X_RUNTIME_METRICS_OPTIN", "go.memory.gc.cycles, go.cpu.time")
		testRuntimeOptInMetrics(t)
	})
}

func testRuntimeOptInMetrics(t *testing.T, opts ...Option) {
	goruntime.GC()

	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))
	err := Start(append(opts, WithMeterProvider(mp))...)
	assert.NoError(t, err)
	rm := metricdata.ResourceMetrics{}
	err = reader.Collect(t.Context(), &rm)
	assert.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)

	metrics := map[string]metricdata.Metrics{}
	for _, m := range rm.ScopeMetrics[0].Metrics {
		metrics[m.Name] = m
	}

	metricdatatest.AssertEqual(t, metricdata.Metrics{
		Name:        goconv.MemoryGCCyclesObservable{}.Name(),
		Description: goconv.MemoryGCCyclesObservable{}.Description(),
		Unit:        goconv.MemoryGCCyclesObservable{}.Unit(),
		Data: metricdata.Sum[int64]{
			Temporality: metricdata.CumulativeTemporality,
			IsMonotonic: true,
			DataPoints:  []metricdata.DataPoint[int64]{{}},
		},
	}, metrics[goconv.MemoryGCCyclesObservable{}.Name()], metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue())
	gcCycles := metrics[goconv.MemoryGCCyclesObservable{}.Name()].Data.(metricdata.Sum[int64])
	assert.Positive(t, gcCycles.DataPoints[0].Value)

	cpuTime := goconv.CPUTimeObservable{}
	metricdatatest.AssertEqual(t, metricdata.Metrics{
		Name:        cpuTime.Name(),
		Description: cpuTime.Description(),
		Unit:        cpuTime.Unit(),
		Data: metricdata.Sum[float64]{
			Temporality: metricdata.CumulativeTemporality,
			IsMonotonic: true,
			DataPoints: []metricdata.DataPoint[float64]{
				{Attributes: attribute.NewSet(cpuTime.AttrCPUState(goconv.CPUStateUser))},
				{Attributes: attribute.NewSet(cpuTime.AttrCPUState(goconv.CPUStateGC))},
				{Attributes: attribute.NewSet(cpuTime.AttrCPUState(goconv.CPUStateScavenge))},
				{Attributes: attribute.NewSet(cpuTime.AttrCPUState(goconv.CPUStateIdle))},
			},
		},
	}, metrics[cpuTime.Name()], metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue())

	cpuStates := map[string]float64{}
	for _, dp := range metrics[cpuTime.Name()].Data.(metricdata.Sum[float64]).DataPoints {
		state, _ := dp.Attributes.Value("go.cpu.state")
		cpuStates[state.AsString()] = dp.Value
		assert.GreaterOrEqualf(t, dp.Value, float64(0), "go.cpu.state %q", state.AsString())
	}
	assert.Positive(t, cpuStates[string(goconv.CPUStateUser)])
	assert.Positive(t, cpuStates[string(goconv.CPUStateGC)])
}

func TestRuntimeIgnoresProducerOptInMetrics(t *testing.T) {
	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))
	err := Start(WithMeterProvider(mp), WithOptInMetrics(MemoryGCPauseDuration))
	assert.NoError(t, err)
	rm := metricdata.ResourceMetrics{}
	err = reader.Collect(t.Context(), &rm)
	assert.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)

	for _, m := range rm.ScopeMetrics[0].Metrics {
		assert.NotEqual(t, goconv.MemoryGCPauseDuration{}.Name(), m.Name)
	}
}

func TestGoCollectorGetFloat(t *testing.T) {
	collector := newCollector(0, []string{goCPUUser, goGCCycles})
	goruntime.GC()
	collector.refresh()
	assert.Positive(t, collector.getFloat(goCPUUser))
	assert.Zero(t, collector.getFloat(goGCCycles), "a uint64 sample is not read as a float")
	assert.Zero(t, collector.getFloat(goCPUIdle), "an unread sample is zero")
}

func TestRuntimeWithoutOptInMetrics(t *testing.T) {
	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))
	err := Start(WithMeterProvider(mp))
	assert.NoError(t, err)
	rm := metricdata.ResourceMetrics{}
	err = reader.Collect(t.Context(), &rm)
	assert.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)

	for _, m := range rm.ScopeMetrics[0].Metrics {
		assert.NotEqual(t, goconv.MemoryGCCyclesObservable{}.Name(), m.Name)
		assert.NotEqual(t, goconv.CPUTimeObservable{}.Name(), m.Name)
	}
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
