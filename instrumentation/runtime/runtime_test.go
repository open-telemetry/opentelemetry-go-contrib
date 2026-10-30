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
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
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
			Name:      "go.opentelemetry.io/contrib/instrumentation/runtime",
			Version:   Version,
			SchemaURL: semconv.SchemaURL,
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
	t.Setenv("OTEL_GO_X_RUNTIME_METRICS_OPTIN", "go.memory.gc.cycles, go.cpu.time")
	goruntime.GC()

	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))
	err := Start(WithMeterProvider(mp))
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

	var totalCPUTime float64
	for _, dp := range metrics[cpuTime.Name()].Data.(metricdata.Sum[float64]).DataPoints {
		state, _ := dp.Attributes.Value("go.cpu.state")
		assert.GreaterOrEqualf(t, dp.Value, float64(0), "go.cpu.state %q", state.AsString())
		totalCPUTime += dp.Value
	}
	assert.Positive(t, totalCPUTime)
}

func TestRuntimeIgnoresProducerOptInMetrics(t *testing.T) {
	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))
	t.Setenv("OTEL_GO_X_RUNTIME_METRICS_OPTIN", "go.memory.gc.pause.duration")
	err := Start(WithMeterProvider(mp))
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

func TestRuntimeDetailedMemoryAttributes(t *testing.T) {
	memoryUsed := goconv.MemoryUsed{}
	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(
		metric.WithReader(reader),
		metric.WithView(metric.NewView(
			metric.Instrument{Name: memoryUsed.Name()},
			metric.Stream{AttributeFilter: func(_ attribute.KeyValue) bool { return true }},
		)),
	)
	err := Start(WithMeterProvider(mp))
	require.NoError(t, err)
	rm := metricdata.ResourceMetrics{}
	err = reader.Collect(t.Context(), &rm)
	require.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)

	var got metricdata.Metrics
	var found bool
	for _, m := range rm.ScopeMetrics[0].Metrics {
		if m.Name == memoryUsed.Name() {
			got = m
			found = true
			break
		}
	}
	require.True(t, found, "go.memory.used metric not found")

	expected := []metricdata.DataPoint[int64]{
		{Attributes: attribute.NewSet(memoryUsed.AttrMemoryType(goconv.MemoryTypeStack), memoryUsed.AttrMemoryDetailedType("heap/stacks"))},
		{Attributes: attribute.NewSet(memoryUsed.AttrMemoryType(goconv.MemoryTypeOther), memoryUsed.AttrMemoryDetailedType("heap/free"))},
		{Attributes: attribute.NewSet(memoryUsed.AttrMemoryType(goconv.MemoryTypeOther), memoryUsed.AttrMemoryDetailedType("heap/objects"))},
		{Attributes: attribute.NewSet(memoryUsed.AttrMemoryType(goconv.MemoryTypeOther), memoryUsed.AttrMemoryDetailedType("heap/unused"))},
		{Attributes: attribute.NewSet(memoryUsed.AttrMemoryType(goconv.MemoryTypeOther), memoryUsed.AttrMemoryDetailedType("metadata/mcache/free"))},
		{Attributes: attribute.NewSet(memoryUsed.AttrMemoryType(goconv.MemoryTypeOther), memoryUsed.AttrMemoryDetailedType("metadata/mcache/inuse"))},
		{Attributes: attribute.NewSet(memoryUsed.AttrMemoryType(goconv.MemoryTypeOther), memoryUsed.AttrMemoryDetailedType("metadata/mspan/free"))},
		{Attributes: attribute.NewSet(memoryUsed.AttrMemoryType(goconv.MemoryTypeOther), memoryUsed.AttrMemoryDetailedType("metadata/mspan/inuse"))},
		{Attributes: attribute.NewSet(memoryUsed.AttrMemoryType(goconv.MemoryTypeOther), memoryUsed.AttrMemoryDetailedType("metadata/other"))},
		{Attributes: attribute.NewSet(memoryUsed.AttrMemoryType(goconv.MemoryTypeOther), memoryUsed.AttrMemoryDetailedType("os-stacks"))},
		{Attributes: attribute.NewSet(memoryUsed.AttrMemoryType(goconv.MemoryTypeOther), memoryUsed.AttrMemoryDetailedType("other"))},
		{Attributes: attribute.NewSet(memoryUsed.AttrMemoryType(goconv.MemoryTypeOther), memoryUsed.AttrMemoryDetailedType("profiling/buckets"))},
	}
	metricdatatest.AssertEqual(t, metricdata.Metrics{
		Name:        memoryUsed.Name(),
		Description: memoryUsed.Description(),
		Unit:        memoryUsed.Unit(),
		Data: metricdata.Sum[int64]{
			Temporality: metricdata.CumulativeTemporality,
			IsMonotonic: false,
			DataPoints:  expected,
		},
	}, got, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue())

	var totalMemory int64
	for _, dp := range got.Data.(metricdata.Sum[int64]).DataPoints {
		detailedType, _ := dp.Attributes.Value("go.memory.detailed_type")
		assert.GreaterOrEqualf(t, dp.Value, int64(0), "go.memory.detailed_type %q", detailedType.AsString())
		totalMemory += dp.Value
	}
	assert.Positive(t, totalMemory)
}

func TestRuntimeDetailedCPUAttributes(t *testing.T) {
	t.Setenv("OTEL_GO_X_RUNTIME_METRICS_OPTIN", "go.cpu.time")
	goruntime.GC()

	cpuTime := goconv.CPUTimeObservable{}
	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(
		metric.WithReader(reader),
		metric.WithView(metric.NewView(
			metric.Instrument{Name: cpuTime.Name()},
			metric.Stream{AttributeFilter: func(_ attribute.KeyValue) bool { return true }},
		)),
	)
	err := Start(WithMeterProvider(mp))
	require.NoError(t, err)
	rm := metricdata.ResourceMetrics{}
	err = reader.Collect(t.Context(), &rm)
	require.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)

	var got metricdata.Metrics
	var found bool
	for _, m := range rm.ScopeMetrics[0].Metrics {
		if m.Name == cpuTime.Name() {
			got = m
			found = true
			break
		}
	}
	require.True(t, found, "go.cpu.time metric not found")

	expected := []metricdata.DataPoint[float64]{
		{Attributes: attribute.NewSet(cpuTime.AttrCPUState(goconv.CPUStateUser), cpuTime.AttrCPUDetailedState("user"))},
		{Attributes: attribute.NewSet(cpuTime.AttrCPUState(goconv.CPUStateGC), cpuTime.AttrCPUDetailedState("gc/mark/assist"))},
		{Attributes: attribute.NewSet(cpuTime.AttrCPUState(goconv.CPUStateGC), cpuTime.AttrCPUDetailedState("gc/mark/dedicated"))},
		{Attributes: attribute.NewSet(cpuTime.AttrCPUState(goconv.CPUStateGC), cpuTime.AttrCPUDetailedState("gc/mark/idle"))},
		{Attributes: attribute.NewSet(cpuTime.AttrCPUState(goconv.CPUStateGC), cpuTime.AttrCPUDetailedState("gc/pause"))},
		{Attributes: attribute.NewSet(cpuTime.AttrCPUState(goconv.CPUStateScavenge), cpuTime.AttrCPUDetailedState("scavenge/assist"))},
		{Attributes: attribute.NewSet(cpuTime.AttrCPUState(goconv.CPUStateScavenge), cpuTime.AttrCPUDetailedState("scavenge/background"))},
		{Attributes: attribute.NewSet(cpuTime.AttrCPUState(goconv.CPUStateIdle), cpuTime.AttrCPUDetailedState("idle"))},
	}
	metricdatatest.AssertEqual(t, metricdata.Metrics{
		Name:        cpuTime.Name(),
		Description: cpuTime.Description(),
		Unit:        cpuTime.Unit(),
		Data: metricdata.Sum[float64]{
			Temporality: metricdata.CumulativeTemporality,
			IsMonotonic: true,
			DataPoints:  expected,
		},
	}, got, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue())

	var totalCPUTime float64
	for _, dp := range got.Data.(metricdata.Sum[float64]).DataPoints {
		detailedState, _ := dp.Attributes.Value("go.cpu.detailed_state")
		assert.GreaterOrEqualf(t, dp.Value, float64(0), "go.cpu.detailed_state %q", detailedState.AsString())
		totalCPUTime += dp.Value
	}
	assert.Positive(t, totalCPUTime)
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
