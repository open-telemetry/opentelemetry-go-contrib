// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"runtime"
	"runtime/metrics"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
	"go.opentelemetry.io/otel/semconv/v1.43.0/goconv"
)

func TestNewProducer(t *testing.T) {
	reader := metric.NewManualReader(metric.WithProducer(NewProducer()))
	_ = metric.NewMeterProvider(metric.WithReader(reader))
	rm := metricdata.ResourceMetrics{}
	err := reader.Collect(t.Context(), &rm)
	assert.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	require.Len(t, rm.ScopeMetrics[0].Metrics, 1)

	expectedScopeMetric := metricdata.ScopeMetrics{
		Scope: instrumentation.Scope{
			Name:    "go.opentelemetry.io/contrib/instrumentation/runtime",
			Version: Version,
		},
		Metrics: []metricdata.Metrics{
			{
				Name:        "go.schedule.duration",
				Description: "The time goroutines have spent in the scheduler in a runnable state before actually running.",
				Unit:        "s",
				Data: metricdata.Histogram[float64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.HistogramDataPoint[float64]{
						{},
					},
				},
			},
		},
	}
	metricdatatest.AssertEqual(t, expectedScopeMetric, rm.ScopeMetrics[0], metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue())
}

func TestNewProducerOptInGCPauses(t *testing.T) {
	t.Setenv("OTEL_GO_X_RUNTIME_METRICS_OPTIN", "go.memory.gc.pause.duration")
	runtime.GC()

	reader := metric.NewManualReader(metric.WithProducer(NewProducer()))
	_ = metric.NewMeterProvider(metric.WithReader(reader))
	rm := metricdata.ResourceMetrics{}
	err := reader.Collect(t.Context(), &rm)
	assert.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)

	expectedScopeMetric := metricdata.ScopeMetrics{
		Scope: instrumentation.Scope{
			Name:    "go.opentelemetry.io/contrib/instrumentation/runtime",
			Version: Version,
		},
		Metrics: []metricdata.Metrics{
			{
				Name:        goconv.ScheduleDuration{}.Name(),
				Description: goconv.ScheduleDuration{}.Description(),
				Unit:        goconv.ScheduleDuration{}.Unit(),
				Data: metricdata.Histogram[float64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints:  []metricdata.HistogramDataPoint[float64]{{}},
				},
			},
			{
				Name:        goconv.MemoryGCPauseDuration{}.Name(),
				Description: goconv.MemoryGCPauseDuration{}.Description(),
				Unit:        goconv.MemoryGCPauseDuration{}.Unit(),
				Data: metricdata.Histogram[float64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints:  []metricdata.HistogramDataPoint[float64]{{}},
				},
			},
		},
	}
	metricdatatest.AssertEqual(t, expectedScopeMetric, rm.ScopeMetrics[0], metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue())

	pauses := rm.ScopeMetrics[0].Metrics[1].Data.(metricdata.Histogram[float64])
	assert.Positive(t, pauses.DataPoints[0].Count)
}

func TestNewProducerIgnoresStartOptInMetrics(t *testing.T) {
	t.Setenv("OTEL_GO_X_RUNTIME_METRICS_OPTIN", "go.memory.gc.cycles,go.cpu.time")
	reader := metric.NewManualReader(metric.WithProducer(NewProducer()))
	_ = metric.NewMeterProvider(metric.WithReader(reader))
	rm := metricdata.ResourceMetrics{}
	err := reader.Collect(t.Context(), &rm)
	assert.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	require.Len(t, rm.ScopeMetrics[0].Metrics, 1)
	assert.Equal(t, goconv.ScheduleDuration{}.Name(), rm.ScopeMetrics[0].Metrics[0].Name)
}

func TestConvertRuntimeHistogram(t *testing.T) {
	rh := &metrics.Float64Histogram{
		Buckets: []float64{0, 1, 2, 3},
		Counts:  []uint64{2, 3, 4},
	}
	dps := convertRuntimeHistogram(rh, time.Now())
	require.Len(t, dps, 1)
	assert.Equal(t, float64(11), dps[0].Sum)
}

func TestProducerRetainedDataPointsAreStable(t *testing.T) {
	t.Setenv("OTEL_GO_X_RUNTIME_METRICS_OPTIN", "go.memory.gc.pause.duration")
	p := NewProducer()
	p.collector.minimumInterval = 0

	runtime.GC()
	first, err := p.Produce(t.Context())
	require.NoError(t, err)

	for range 10 {
		runtime.GC()
	}
	_, err = p.Produce(t.Context())
	require.NoError(t, err)

	for _, m := range first[0].Metrics {
		dp := m.Data.(metricdata.Histogram[float64]).DataPoints[0]
		var total uint64
		for _, c := range dp.BucketCounts {
			total += c
		}
		assert.Equalf(t, dp.Count, total, "%s bucket counts changed after a later collection", m.Name)
	}
}

func TestProducerConcurrentProduce(t *testing.T) {
	t.Setenv("OTEL_GO_X_RUNTIME_METRICS_OPTIN", "go.memory.gc.pause.duration")
	p := NewProducer()
	p.collector.minimumInterval = 0

	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			for range 50 {
				runtime.GC()
				_, err := p.Produce(t.Context())
				assert.NoError(t, err)
			}
		})
	}
	wg.Wait()
}

func TestProducerOptInGCPausesUnavailableKeepsScheduleDuration(t *testing.T) {
	t.Setenv("OTEL_GO_X_RUNTIME_METRICS_OPTIN", "go.memory.gc.pause.duration")
	p := NewProducer()
	p.collector.sampleMap[goGCPauses].Name = "/unsupported/gc/pauses:seconds"

	scopeMetrics, err := p.Produce(t.Context())
	require.NoError(t, err)
	require.Len(t, scopeMetrics, 1)
	require.Len(t, scopeMetrics[0].Metrics, 1)
	assert.Equal(t, goconv.ScheduleDuration{}.Name(), scopeMetrics[0].Metrics[0].Name)
}

func TestConvertRuntimeHistogramWithoutBuckets(t *testing.T) {
	for name, rh := range map[string]*metrics.Float64Histogram{
		"nil":          nil,
		"empty":        {},
		"single bound": {Buckets: []float64{0}},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Empty(t, convertRuntimeHistogram(rh, time.Now()))
		})
	}
}

func TestProducerScheduleDurationUnavailableFails(t *testing.T) {
	p := NewProducer()
	p.collector.sampleMap[goSchedLatencies].Name = "/unsupported/sched/latencies:seconds"

	scopeMetrics, err := p.Produce(t.Context())
	assert.EqualError(t, err, "unable to obtain go.schedule.duration metric from the runtime")
	assert.Nil(t, scopeMetrics)
}
