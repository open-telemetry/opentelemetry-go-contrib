// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgrpc

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"google.golang.org/grpc/stats"
)

func TestStatsHandlerSemanticConventions(t *testing.T) {
	for _, tt := range []struct {
		name       string
		newHandler func(...Option) stats.Handler
		metricName string
	}{
		{"client", NewClientHandler, "rpc.client.call.duration"},
		{"server", NewServerHandler, "rpc.server.call.duration"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			reader := sdkmetric.NewManualReader()
			meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
			t.Cleanup(func() { assert.NoError(t, meterProvider.Shutdown(context.WithoutCancel(t.Context()))) })
			spanRecorder := tracetest.NewSpanRecorder()
			tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
			t.Cleanup(func() { assert.NoError(t, tracerProvider.Shutdown(context.WithoutCancel(t.Context()))) })
			handler := tt.newHandler(
				WithMeterProvider(meterProvider),
				WithTracerProvider(tracerProvider),
			)
			ctx := handler.TagRPC(t.Context(), &stats.RPCTagInfo{FullMethodName: "/package.service/method"})
			begin := time.Unix(0, 0)
			handler.HandleRPC(ctx, &stats.End{BeginTime: begin, EndTime: begin.Add(time.Second)})

			var metrics metricdata.ResourceMetrics
			require.NoError(t, reader.Collect(t.Context(), &metrics))
			require.Len(t, metrics.ScopeMetrics, 1)
			scope := metrics.ScopeMetrics[0]
			assert.Equal(t, "https://opentelemetry.io/schemas/1.45.0", scope.Scope.SchemaURL)
			require.Len(t, scope.Metrics, 1)
			assert.Equal(t, tt.metricName, scope.Metrics[0].Name)
			assert.Equal(t, "s", scope.Metrics[0].Unit)
			histogram, ok := scope.Metrics[0].Data.(metricdata.Histogram[float64])
			require.True(t, ok)
			require.Len(t, histogram.DataPoints, 1)
			point := histogram.DataPoints[0]
			assert.Equal(t, []float64{
				0, 0.00001, 0.00005, 0.0001, 0.0003, 0.0006, 0.0008,
				0.001, 0.002, 0.003, 0.004, 0.005, 0.006, 0.008, 0.01,
				0.013, 0.016, 0.02, 0.025, 0.03, 0.04, 0.05, 0.065,
				0.08, 0.1, 0.13, 0.16, 0.2, 0.25, 0.3, 0.4, 0.5,
				0.65, 0.8, 1, 2, 5, 10, 20, 50, 100,
			}, point.Bounds)
			assert.Equal(t, uint64(1), point.Count)
			assert.Equal(t, float64(1), point.Sum)
			status, ok := point.Attributes.Value("rpc.status_code")
			assert.True(t, ok)
			assert.Equal(t, "OK", status.AsString())
			_, ok = point.Attributes.Value("rpc.response.status_code")
			assert.False(t, ok)

			spans := spanRecorder.Ended()
			require.Len(t, spans, 1)
			spanAttributes := attribute.NewSet(spans[0].Attributes()...)
			status, ok = spanAttributes.Value("rpc.status_code")
			assert.True(t, ok)
			assert.Equal(t, "OK", status.AsString())
			_, ok = spanAttributes.Value("rpc.response.status_code")
			assert.False(t, ok)
		})
	}
}
