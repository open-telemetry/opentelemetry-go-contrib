// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgrpc

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"google.golang.org/grpc/stats"
)

func TestStatsHandlerMetricSemanticConventions(t *testing.T) {
	for _, tt := range []struct {
		name            string
		newHandler      func(...Option) stats.Handler
		wantName        string
		wantDescription string
	}{
		{
			name:            "client",
			newHandler:      NewClientHandler,
			wantName:        "rpc.client.call.duration",
			wantDescription: "Measures the duration of an outgoing Remote Procedure Call (RPC).",
		},
		{
			name:            "server",
			newHandler:      NewServerHandler,
			wantName:        "rpc.server.call.duration",
			wantDescription: "Measures the duration of an incoming Remote Procedure Call (RPC).",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			reader := sdkmetric.NewManualReader()
			provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
			t.Cleanup(func() { assert.NoError(t, provider.Shutdown(context.WithoutCancel(t.Context()))) })
			handler := tt.newHandler(WithMeterProvider(provider))
			ctx := handler.TagRPC(t.Context(), &stats.RPCTagInfo{FullMethodName: "/package.service/method"})
			begin := time.Unix(0, 0)
			handler.HandleRPC(ctx, &stats.End{BeginTime: begin, EndTime: begin.Add(time.Second)})

			var metrics metricdata.ResourceMetrics
			require.NoError(t, reader.Collect(t.Context(), &metrics))
			require.Len(t, metrics.ScopeMetrics, 1, "one instrumentation scope should record the RPC duration")
			want := metricdata.ScopeMetrics{
				Scope: instrumentation.Scope{
					Name:      ScopeName,
					Version:   Version,
					SchemaURL: "https://opentelemetry.io/schemas/1.45.0",
				},
				Metrics: []metricdata.Metrics{{
					Name:        tt.wantName,
					Description: tt.wantDescription,
					Unit:        "s",
					Data:        wantGRPCDurationHistogram(t),
				}},
			}
			metricdatatest.AssertEqual(t, want, metrics.ScopeMetrics[0],
				metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
		})
	}
}

func wantGRPCDurationHistogram(t *testing.T) metricdata.Histogram[float64] {
	t.Helper()

	bounds := []float64{
		0, 0.00001, 0.00005, 0.0001, 0.0003, 0.0006, 0.0008,
		0.001, 0.002, 0.003, 0.004, 0.005, 0.006, 0.008, 0.01,
		0.013, 0.016, 0.02, 0.025, 0.03, 0.04, 0.05, 0.065,
		0.08, 0.1, 0.13, 0.16, 0.2, 0.25, 0.3, 0.4, 0.5,
		0.65, 0.8, 1, 2, 5, 10, 20, 50, 100,
	}
	oneSecondBucket := slices.Index(bounds, 1)
	require.NotEqual(t, -1, oneSecondBucket, "gRPC advice must include the one-second boundary")
	bucketCounts := make([]uint64, len(bounds)+1)
	bucketCounts[oneSecondBucket] = 1

	return metricdata.Histogram[float64]{
		Temporality: metricdata.CumulativeTemporality,
		DataPoints: []metricdata.HistogramDataPoint[float64]{{
			Attributes: attribute.NewSet(
				attribute.String("rpc.method", "package.service/method"),
				attribute.String("rpc.system.name", "grpc"),
				attribute.String("rpc.status_code", "OK"),
			),
			Count:        1,
			Sum:          1,
			Min:          metricdata.NewExtrema(float64(1)),
			Max:          metricdata.NewExtrema(float64(1)),
			Bounds:       bounds,
			BucketCounts: bucketCounts,
		}},
	}
}

func TestStatsHandlerSpanSemanticConventions(t *testing.T) {
	for _, tt := range []struct {
		name       string
		newHandler func(...Option) stats.Handler
	}{
		{name: "client", newHandler: NewClientHandler},
		{name: "server", newHandler: NewServerHandler},
	} {
		t.Run(tt.name, func(t *testing.T) {
			recorder := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
			t.Cleanup(func() { assert.NoError(t, provider.Shutdown(context.WithoutCancel(t.Context()))) })
			handler := tt.newHandler(WithTracerProvider(provider))
			ctx := handler.TagRPC(t.Context(), &stats.RPCTagInfo{FullMethodName: "/package.service/method"})
			handler.HandleRPC(ctx, &stats.End{})

			spans := recorder.Ended()
			require.Len(t, spans, 1, "one span should end with the RPC")
			assert.ElementsMatch(t, []attribute.KeyValue{
				attribute.String("rpc.method", "package.service/method"),
				attribute.String("rpc.system.name", "grpc"),
				attribute.String("rpc.status_code", "OK"),
			}, spans[0].Attributes())
		})
	}
}
