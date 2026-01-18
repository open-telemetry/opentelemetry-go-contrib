// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgrpc_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	otelcode "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.39.0"
	"go.opentelemetry.io/otel/semconv/v1.39.0/rpcconv"
	grpc_codes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

func getSpanFromRecorder(sr *tracetest.SpanRecorder, name string) (trace.ReadOnlySpan, bool) {
	for _, s := range sr.Ended() {
		if s.Name() == name {
			return s, true
		}
	}
	return nil, false
}

var serverChecks = []struct {
	grpcCode                  grpc_codes.Code
	wantSpanCode              otelcode.Code
	wantSpanStatusDescription string
}{
	{
		grpcCode:                  grpc_codes.OK,
		wantSpanCode:              otelcode.Unset,
		wantSpanStatusDescription: "",
	},
	{
		grpcCode:                  grpc_codes.Canceled,
		wantSpanCode:              otelcode.Unset,
		wantSpanStatusDescription: "",
	},
	{
		grpcCode:                  grpc_codes.Unknown,
		wantSpanCode:              otelcode.Error,
		wantSpanStatusDescription: grpc_codes.Unknown.String(),
	},
	{
		grpcCode:                  grpc_codes.InvalidArgument,
		wantSpanCode:              otelcode.Unset,
		wantSpanStatusDescription: "",
	},
	{
		grpcCode:                  grpc_codes.DeadlineExceeded,
		wantSpanCode:              otelcode.Error,
		wantSpanStatusDescription: grpc_codes.DeadlineExceeded.String(),
	},
	{
		grpcCode:                  grpc_codes.NotFound,
		wantSpanCode:              otelcode.Unset,
		wantSpanStatusDescription: "",
	},
	{
		grpcCode:                  grpc_codes.AlreadyExists,
		wantSpanCode:              otelcode.Unset,
		wantSpanStatusDescription: "",
	},
	{
		grpcCode:                  grpc_codes.PermissionDenied,
		wantSpanCode:              otelcode.Unset,
		wantSpanStatusDescription: "",
	},
	{
		grpcCode:                  grpc_codes.ResourceExhausted,
		wantSpanCode:              otelcode.Unset,
		wantSpanStatusDescription: "",
	},
	{
		grpcCode:                  grpc_codes.FailedPrecondition,
		wantSpanCode:              otelcode.Unset,
		wantSpanStatusDescription: "",
	},
	{
		grpcCode:                  grpc_codes.Aborted,
		wantSpanCode:              otelcode.Unset,
		wantSpanStatusDescription: "",
	},
	{
		grpcCode:                  grpc_codes.OutOfRange,
		wantSpanCode:              otelcode.Unset,
		wantSpanStatusDescription: "",
	},
	{
		grpcCode:                  grpc_codes.Unimplemented,
		wantSpanCode:              otelcode.Error,
		wantSpanStatusDescription: grpc_codes.Unimplemented.String(),
	},
	{
		grpcCode:                  grpc_codes.Internal,
		wantSpanCode:              otelcode.Error,
		wantSpanStatusDescription: grpc_codes.Internal.String(),
	},
	{
		grpcCode:                  grpc_codes.Unavailable,
		wantSpanCode:              otelcode.Error,
		wantSpanStatusDescription: grpc_codes.Unavailable.String(),
	},
	{
		grpcCode:                  grpc_codes.DataLoss,
		wantSpanCode:              otelcode.Error,
		wantSpanStatusDescription: grpc_codes.DataLoss.String(),
	},
	{
		grpcCode:                  grpc_codes.Unauthenticated,
		wantSpanCode:              otelcode.Unset,
		wantSpanStatusDescription: "",
	},
}

func TestStatsHandlerHandleRPCServerErrors(t *testing.T) {
	for _, check := range serverChecks {
		name := check.grpcCode.String()
		t.Run(name, func(t *testing.T) {
			t.Setenv("OTEL_METRICS_EXEMPLAR_FILTER", "always_off")
			sr := tracetest.NewSpanRecorder()
			tp := trace.NewTracerProvider(trace.WithSpanProcessor(sr))

			mr := metric.NewManualReader()
			mp := metric.NewMeterProvider(metric.WithReader(mr))

			serverHandler := otelgrpc.NewServerHandler(
				otelgrpc.WithTracerProvider(tp),
				otelgrpc.WithMeterProvider(mp),
				otelgrpc.WithMetricAttributes(testMetricAttr),
			)

			serviceName := "TestGrpcService"
			methodName := serviceName + "/" + name
			fullMethodName := "/" + methodName
			// call the server handler
			ctx := serverHandler.TagRPC(t.Context(), &stats.RPCTagInfo{
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

func assertServerSpan(t *testing.T, wantSpanCode otelcode.Code, wantSpanStatusDescription string, wantGrpcCode grpc_codes.Code, span trace.ReadOnlySpan) {
	// validate span status
	assert.Equal(t, wantSpanCode, span.Status().Code)
	assert.Equal(t, wantSpanStatusDescription, span.Status().Description)

	// validate grpc code span attribute
	var codeAttr attribute.KeyValue
	for _, a := range span.Attributes() {
		if a.Key == semconv.RPCGRPCStatusCodeKey {
			codeAttr = a
			break
		}
	}

	require.True(t, codeAttr.Valid(), "attributes contain gRPC status code")
	assert.Equal(t, attribute.Int64Value(int64(wantGrpcCode)), codeAttr.Value)
}

func assertStatsHandlerServerMetrics(t *testing.T, reader metric.Reader, serviceName, name string, code grpc_codes.Code) {
	want := metricdata.ScopeMetrics{
		Scope: wantInstrumentationScope,
		Metrics: []metricdata.Metrics{
			{
				Name:        rpcconv.ServerDuration{}.Name(),
				Description: rpcconv.ServerDuration{}.Description(),
				Unit:        rpcconv.ServerDuration{}.Unit(),
				Data: metricdata.Histogram[float64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.HistogramDataPoint[float64]{
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod(name),
								semconv.RPCService(serviceName),
								semconv.RPCSystemGRPC,
								semconv.RPCGRPCStatusCodeKey.Int64(int64(code)),
								testMetricAttr,
							),
						},
					},
				},
			},
			{
				Name:        rpcconv.ServerRequestsPerRPC{}.Name(),
				Description: rpcconv.ServerRequestsPerRPC{}.Description(),
				Unit:        rpcconv.ServerRequestsPerRPC{}.Unit(),
				Data: metricdata.Histogram[int64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.HistogramDataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod(name),
								semconv.RPCService(serviceName),
								semconv.RPCSystemGRPC,
								semconv.RPCGRPCStatusCodeKey.Int64(int64(code)),
								testMetricAttr,
							),
						},
					},
				},
			},
			{
				Name:        rpcconv.ServerResponsesPerRPC{}.Name(),
				Description: rpcconv.ServerResponsesPerRPC{}.Description(),
				Unit:        rpcconv.ServerResponsesPerRPC{}.Unit(),
				Data: metricdata.Histogram[int64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.HistogramDataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod(name),
								semconv.RPCService(serviceName),
								semconv.RPCSystemGRPC,
								semconv.RPCGRPCStatusCodeKey.Int64(int64(code)),
								testMetricAttr,
							),
						},
					},
				},
			},
		},
	}
	rm := metricdata.ResourceMetrics{}
	err := reader.Collect(t.Context(), &rm)
	assert.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	metricdatatest.AssertEqual(t, want, rm.ScopeMetrics[0], metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue())
}
