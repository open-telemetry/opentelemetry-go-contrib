// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	grpc_codes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const (
	serviceName = "TestGrpcService"
)

func TestStatsHandlerHandleRPCServerErrors(t *testing.T) {
	for _, check := range serverChecks {
		name := check.grpcCode.String()
		t.Run(name, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			tp := trace.NewTracerProvider(trace.WithSpanProcessor(sr))

			mr := metric.NewManualReader()
			mp := metric.NewMeterProvider(metric.WithReader(mr))

			serverHandler := otelgrpc.NewServerHandler(
				otelgrpc.WithTracerProvider(tp),
				otelgrpc.WithMeterProvider(mp),
			)

			methodName := serviceName + "/" + name
			fullMethodName := "/" + methodName
			// call the server handler
			ctx := serverHandler.TagRPC(context.Background(), &stats.RPCTagInfo{
				FullMethodName: fullMethodName,
			})

			grpcErr := status.Error(check.grpcCode, check.grpcCode.String())
			serverHandler.HandleRPC(ctx, &stats.End{
				Error: grpcErr,
			})

			// validate span
			span, ok := getSpanFromRecorder(sr, methodName)
			require.True(t, ok, "missing span %s", methodName)
			assertServerSpan(t, check.wantSpanCode, check.wantSpanStatusDescription, check.grpcCode, span)

			// validate metric
			assertStatsHandlerServerMetrics(t, mr, serviceName, name, check.grpcCode)
		})
	}
}

func TestStatsHandlerServerWithSpanOptions(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := trace.NewTracerProvider(trace.WithSpanProcessor(sr))
	serverHandler := otelgrpc.NewServerHandler(
		otelgrpc.WithTracerProvider(tp),
		otelgrpc.WithSpanOptions(oteltrace.WithAttributes(attribute.Bool("custom", true))),
	)

	methodName := serviceName + "/" + "test"
	fullMethodName := "/" + methodName
	// call the server handler
	ctx := serverHandler.TagRPC(context.Background(), &stats.RPCTagInfo{
		FullMethodName: fullMethodName,
	})

	serverHandler.HandleRPC(ctx, &stats.End{})

	expected := []attribute.KeyValue{
		semconv.RPCSystemGRPC,
		semconv.RPCService("TestGrpcService"),
		semconv.RPCMethod("test"),
		otelgrpc.GRPCStatusCodeKey.Int64(0),
		attribute.Bool("custom", true),
	}
	span, ok := getSpanFromRecorder(sr, methodName)
	require.True(t, ok, "missing span %q", methodName)
	assert.ElementsMatch(t, expected, span.Attributes())
}

func TestStatsHandlerClientWithSpanOptions(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := trace.NewTracerProvider(trace.WithSpanProcessor(sr))
	clientHandler := otelgrpc.NewClientHandler(
		otelgrpc.WithTracerProvider(tp),
		otelgrpc.WithSpanOptions(oteltrace.WithAttributes(attribute.Bool("custom", true))),
	)

	methodName := serviceName + "/" + "test"
	fullMethodName := "/" + methodName
	// call the client handler
	ctx := clientHandler.TagRPC(context.Background(), &stats.RPCTagInfo{
		FullMethodName: fullMethodName,
	})

	clientHandler.HandleRPC(ctx, &stats.End{})

	expected := []attribute.KeyValue{
		semconv.RPCSystemGRPC,
		semconv.RPCService("TestGrpcService"),
		semconv.RPCMethod("test"),
		otelgrpc.GRPCStatusCodeKey.Int64(0),
		attribute.Bool("custom", true),
	}
	span, ok := getSpanFromRecorder(sr, methodName)
	require.True(t, ok, "missing span %q", methodName)
	assert.ElementsMatch(t, expected, span.Attributes())
}

func assertStatsHandlerServerMetrics(t *testing.T, reader metric.Reader, serviceName, name string, code grpc_codes.Code) {
	want := metricdata.ScopeMetrics{
		Scope: wantInstrumentationScope,
		Metrics: []metricdata.Metrics{
			{
				Name:        "rpc.server.duration",
				Description: "Measures the duration of inbound RPC.",
				Unit:        "ms",
				Data: metricdata.Histogram[float64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.HistogramDataPoint[float64]{
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod(name),
								semconv.RPCService(serviceName),
								otelgrpc.RPCSystemGRPC,
								otelgrpc.GRPCStatusCodeKey.Int64(int64(code)),
							),
						},
					},
				},
			},
			{
				Name:        "rpc.server.requests_per_rpc",
				Description: "Measures the number of messages received per RPC. Should be 1 for all non-streaming RPCs.",
				Unit:        "{count}",
				Data: metricdata.Histogram[int64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.HistogramDataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod(name),
								semconv.RPCService(serviceName),
								otelgrpc.RPCSystemGRPC,
								otelgrpc.GRPCStatusCodeKey.Int64(int64(code)),
							),
						},
					},
				},
			},
			{
				Name:        "rpc.server.responses_per_rpc",
				Description: "Measures the number of messages received per RPC. Should be 1 for all non-streaming RPCs.",
				Unit:        "{count}",
				Data: metricdata.Histogram[int64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.HistogramDataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod(name),
								semconv.RPCService(serviceName),
								otelgrpc.RPCSystemGRPC,
								otelgrpc.GRPCStatusCodeKey.Int64(int64(code)),
							),
						},
					},
				},
			},
		},
	}
	rm := metricdata.ResourceMetrics{}
	err := reader.Collect(context.Background(), &rm)
	assert.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	metricdatatest.AssertEqual(t, want, rm.ScopeMetrics[0], metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue())
}
