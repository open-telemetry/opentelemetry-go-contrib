// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"runtime"
	"runtime/metrics"
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
	t.Run("option", func(t *testing.T) {
		testNewProducerOptInGCPauses(t, WithOptInMetrics(MemoryGCPauseDuration))
	})
	t.Run("environment", func(t *testing.T) {
		t.Setenv("OTEL_GO_X_RUNTIME_METRICS_OPTIN", "go.memory.gc.pause.duration")
		testNewProducerOptInGCPauses(t)
	})
}

func testNewProducerOptInGCPauses(t *testing.T, opts ...ProducerOption) {
	runtime.GC()

	reader := metric.NewManualReader(metric.WithProducer(NewProducer(opts...)))
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
	reader := metric.NewManualReader(metric.WithProducer(NewProducer(WithOptInMetrics(MemoryGCCycles, CPUTime))))
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
