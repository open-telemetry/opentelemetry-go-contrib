// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package host_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
	"go.opentelemetry.io/otel/semconv/v1.39.0/processconv"
	"go.opentelemetry.io/otel/semconv/v1.39.0/systemconv"

	"go.opentelemetry.io/contrib/instrumentation/host"
)

func TestHostMetrics(t *testing.T) {
	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))
	err := host.Start(host.WithMeterProvider(mp))
	require.NoError(t, err)
	rm := metricdata.ResourceMetrics{}
	err = reader.Collect(t.Context(), &rm)
	require.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)

	want := metricdata.ScopeMetrics{
		Scope: instrumentation.Scope{
			Name:    host.ScopeName,
			Version: host.Version,
		},
		Metrics: []metricdata.Metrics{
			{
				Name:        processconv.CPUTime{}.Name(),
				Description: processconv.CPUTime{}.Description(),
				Unit:        processconv.CPUTime{}.Unit(),
				Data: metricdata.Sum[float64]{
					DataPoints: []metricdata.DataPoint[float64]{
						{Attributes: attribute.NewSet(
							processconv.CPUTime{}.AttrCPUMode(processconv.CPUModeUser),
						)},
						{Attributes: attribute.NewSet(
							processconv.CPUTime{}.AttrCPUMode(processconv.CPUModeSystem),
						)},
					},
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: true,
				},
			},
			{
				Name:        systemconv.CPUTime{}.Name(),
				Description: systemconv.CPUTime{}.Description(),
				Unit:        systemconv.CPUTime{}.Unit(),
				Data: metricdata.Sum[float64]{
					DataPoints: []metricdata.DataPoint[float64]{
						{Attributes: attribute.NewSet(
							systemconv.CPUTime{}.AttrCPUMode(systemconv.CPUModeUser),
						)},
						{Attributes: attribute.NewSet(
							systemconv.CPUTime{}.AttrCPUMode(systemconv.CPUModeSystem),
						)},
						{Attributes: attribute.NewSet(
							systemconv.CPUTime{}.AttrCPUMode(systemconv.CPUModeAttr("other")),
						)},
						{Attributes: attribute.NewSet(
							systemconv.CPUTime{}.AttrCPUMode(systemconv.CPUModeIdle),
						)},
					},
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: true,
				},
			},
			{
				Name:        systemconv.MemoryUsage{}.Name(),
				Description: systemconv.MemoryUsage{}.Description(),
				Unit:        systemconv.MemoryUsage{}.Unit(),
				Data: metricdata.Sum[int64]{
					DataPoints: []metricdata.DataPoint[int64]{
						{Attributes: attribute.NewSet(
							systemconv.MemoryUsage{}.AttrMemoryState(systemconv.MemoryStateUsed),
						)},
						{Attributes: attribute.NewSet(
							systemconv.MemoryUsage{}.AttrMemoryState(systemconv.MemoryStateFree),
						)},
					},
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: false,
				},
			},
			{
				Name:        systemconv.MemoryUtilization{}.Name(),
				Description: systemconv.MemoryUtilization{}.Description(),
				Unit:        systemconv.MemoryUtilization{}.Unit(),
				Data: metricdata.Gauge[float64]{
					DataPoints: []metricdata.DataPoint[float64]{
						{Attributes: attribute.NewSet(
							systemconv.MemoryUtilization{}.AttrMemoryState(systemconv.MemoryStateUsed),
						)},
						{Attributes: attribute.NewSet(
							systemconv.MemoryUtilization{}.AttrMemoryState(systemconv.MemoryStateFree),
						)},
					},
				},
			},
			{
				Name:        systemconv.NetworkIO{}.Name(),
				Description: systemconv.NetworkIO{}.Description(),
				Unit:        systemconv.NetworkIO{}.Unit(),
				Data: metricdata.Sum[int64]{
					DataPoints: []metricdata.DataPoint[int64]{
						{Attributes: attribute.NewSet(
							systemconv.NetworkIO{}.AttrNetworkIODirection(systemconv.NetworkIODirectionReceive),
						)},
						{Attributes: attribute.NewSet(
							systemconv.NetworkIO{}.AttrNetworkIODirection(systemconv.NetworkIODirectionTransmit),
						)},
					},
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: true,
				},
			},
		},
	}
	metricdatatest.AssertEqual(t, want, rm.ScopeMetrics[0], metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue())
}
