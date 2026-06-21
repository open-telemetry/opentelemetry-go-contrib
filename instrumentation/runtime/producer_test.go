// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package runtime // import "go.opentelemetry.io/contrib/instrumentation/runtime"

import (
	"math"
	"runtime/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
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

func TestConvertRuntimeHistogram(t *testing.T) {
	hist := &metrics.Float64Histogram{
		Counts:  []uint64{10, 20, 30, 40},
		Buckets: []float64{0, 1, 2, 3, math.Inf(1)},
	}
	// Buckets after conversion: (-∞,1], (1,2], (2,3], (3,+∞)
	// Expected sum: 1*20 + 2*30 + 3*40 = 200
	dp := convertRuntimeHistogram(hist, time.Now())
	require.Len(t, dp, 1)
	assert.Equal(t, uint64(100), dp[0].Count)
	assert.Equal(t, float64(200), dp[0].Sum)
	assert.Equal(t, []float64{1, 2, 3}, dp[0].Bounds)
	assert.Equal(t, []uint64{10, 20, 30, 40}, dp[0].BucketCounts)
}
