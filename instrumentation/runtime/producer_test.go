// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package runtime // import "go.opentelemetry.io/contrib/instrumentation/runtime"

import (
	"testing"

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

func TestNewProducerOptIn(t *testing.T) {
	t.Setenv("OTEL_GO_X_RUNTIME_METRICS_OPTIN", "go.memory.gc.pause.duration")

	reader := metric.NewManualReader(metric.WithProducer(NewProducer()))
	_ = metric.NewMeterProvider(metric.WithReader(reader))
	rm := metricdata.ResourceMetrics{}
	err := reader.Collect(t.Context(), &rm)
	assert.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)

	// We expect 1 default metric + 1 opt-in metric = 2 metrics.
	require.Len(t, rm.ScopeMetrics[0].Metrics, 2)

	found := false
	for _, m := range rm.ScopeMetrics[0].Metrics {
		if m.Name == "go.memory.gc.pause.duration" {
			found = true
			assert.Equal(t, "s", m.Unit)
			assert.Equal(t, "Distribution of individual GC-related stop-the-world pause latencies.", m.Description)
			break
		}
	}
	assert.True(t, found, "go.memory.gc.pause.duration metric not found")
}
