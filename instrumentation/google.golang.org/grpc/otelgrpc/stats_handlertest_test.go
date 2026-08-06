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
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/semconv/v1.43.0/rpcconv"
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
	wantRPCResponseStatusCode string
}{
	{
		grpcCode:                  grpc_codes.OK,
		wantSpanCode:              otelcode.Unset,
		wantSpanStatusDescription: "",
		wantRPCResponseStatusCode: "OK",
	},
	{
		grpcCode:                  grpc_codes.Canceled,
		wantSpanCode:              otelcode.Unset,
		wantSpanStatusDescription: "",
		wantRPCResponseStatusCode: "CANCELLED",
	},
	{
		grpcCode:                  grpc_codes.Unknown,
		wantSpanCode:              otelcode.Error,
		wantSpanStatusDescription: grpc_codes.Unknown.String(),
		wantRPCResponseStatusCode: "UNKNOWN",
	},
	{
		grpcCode:                  grpc_codes.InvalidArgument,
		wantSpanCode:              otelcode.Unset,
		wantSpanStatusDescription: "",
		wantRPCResponseStatusCode: "INVALID_ARGUMENT",
	},
	{
		grpcCode:                  grpc_codes.DeadlineExceeded,
		wantSpanCode:              otelcode.Error,
		wantSpanStatusDescription: grpc_codes.DeadlineExceeded.String(),
		wantRPCResponseStatusCode: "DEADLINE_EXCEEDED",
	},
	{
		grpcCode:                  grpc_codes.NotFound,
		wantSpanCode:              otelcode.Unset,
		wantSpanStatusDescription: "",
		wantRPCResponseStatusCode: "NOT_FOUND",
	},
	{
		grpcCode:                  grpc_codes.AlreadyExists,
		wantSpanCode:              otelcode.Unset,
		wantSpanStatusDescription: "",
		wantRPCResponseStatusCode: "ALREADY_EXISTS",
	},
	{
		grpcCode:                  grpc_codes.PermissionDenied,
		wantSpanCode:              otelcode.Unset,
		wantSpanStatusDescription: "",
		wantRPCResponseStatusCode: "PERMISSION_DENIED",
	},
	{
		grpcCode:                  grpc_codes.ResourceExhausted,
		wantSpanCode:              otelcode.Unset,
		wantSpanStatusDescription: "",
		wantRPCResponseStatusCode: "RESOURCE_EXHAUSTED",
	},
	{
		grpcCode:                  grpc_codes.FailedPrecondition,
		wantSpanCode:              otelcode.Unset,
		wantSpanStatusDescription: "",
		wantRPCResponseStatusCode: "FAILED_PRECONDITION",
	},
	{
		grpcCode:                  grpc_codes.Aborted,
		wantSpanCode:              otelcode.Unset,
		wantSpanStatusDescription: "",
		wantRPCResponseStatusCode: "ABORTED",
	},
	{
		grpcCode:                  grpc_codes.OutOfRange,
		wantSpanCode:              otelcode.Unset,
		wantSpanStatusDescription: "",
		wantRPCResponseStatusCode: "OUT_OF_RANGE",
	},
	{
		grpcCode:                  grpc_codes.Unimplemented,
		wantSpanCode:              otelcode.Error,
		wantSpanStatusDescription: grpc_codes.Unimplemented.String(),
		wantRPCResponseStatusCode: "UNIMPLEMENTED",
	},
	{
		grpcCode:                  grpc_codes.Internal,
		wantSpanCode:              otelcode.Error,
		wantSpanStatusDescription: grpc_codes.Internal.String(),
		wantRPCResponseStatusCode: "INTERNAL",
	},
	{
		grpcCode:                  grpc_codes.Unavailable,
		wantSpanCode:              otelcode.Error,
		wantSpanStatusDescription: grpc_codes.Unavailable.String(),
		wantRPCResponseStatusCode: "UNAVAILABLE",
	},
	{
		grpcCode:                  grpc_codes.DataLoss,
		wantSpanCode:              otelcode.Error,
		wantSpanStatusDescription: grpc_codes.DataLoss.String(),
		wantRPCResponseStatusCode: "DATA_LOSS",
	},
	{
		grpcCode:                  grpc_codes.Unauthenticated,
		wantSpanCode:              otelcode.Unset,
		wantSpanStatusDescription: "",
		wantRPCResponseStatusCode: "UNAUTHENTICATED",
	},
	{
		grpcCode:                  grpc_codes.Code(9999),
		wantSpanCode:              otelcode.Unset,
		wantSpanStatusDescription: "",
		wantRPCResponseStatusCode: "CODE(9999)",
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
			assertServerSpan(t, check.wantSpanCode, check.wantSpanStatusDescription, check.wantRPCResponseStatusCode, span)

			// validate metric
			assertStatsHandlerServerMetrics(t, mr, serviceName, name, check.wantRPCResponseStatusCode)
		})
	}
}

func assertServerSpan(t *testing.T, wantSpanCode otelcode.Code, wantSpanStatusDescription, wantGrpcCode string, span trace.ReadOnlySpan) {
	// validate span status
	assert.Equal(t, wantSpanCode, span.Status().Code)
	assert.Equal(t, wantSpanStatusDescription, span.Status().Description)

	// validate grpc code span attribute
	var codeAttr attribute.KeyValue
	for _, a := range span.Attributes() {
		if a.Key == semconv.RPCResponseStatusCodeKey {
			codeAttr = a
			break
		}
	}

	require.True(t, codeAttr.Valid(), "attributes contain gRPC status code")
	assert.Equal(t, attribute.StringValue(wantGrpcCode), codeAttr.Value)
}

func TestStatsHandlerHandleRPCClientErrors(t *testing.T) {
	for _, check := range serverChecks {
		name := check.grpcCode.String()
		t.Run(name, func(t *testing.T) {
			t.Setenv("OTEL_METRICS_EXEMPLAR_FILTER", "always_off")
			sr := tracetest.NewSpanRecorder()
			tp := trace.NewTracerProvider(trace.WithSpanProcessor(sr))

			mr := metric.NewManualReader()
			mp := metric.NewMeterProvider(metric.WithReader(mr))

			clientHandler := otelgrpc.NewClientHandler(
				otelgrpc.WithTracerProvider(tp),
				otelgrpc.WithMeterProvider(mp),
				otelgrpc.WithMetricAttributes(testMetricAttr),
			)

			serviceName := "TestGrpcService"
			methodName := serviceName + "/" + name
			fullMethodName := "/" + methodName
			// call the client handler
			ctx := clientHandler.TagRPC(t.Context(), &stats.RPCTagInfo{
				FullMethodName: fullMethodName,
			})

			grpcErr := status.Error(check.grpcCode, check.grpcCode.String())
			clientHandler.HandleRPC(ctx, &stats.End{
				Error: grpcErr,
			})

			// validate span: clients treat every non-OK status as an error
			// (a nil End error, i.e. a successful RPC, leaves the span Unset)
			span, ok := getSpanFromRecorder(sr, methodName)
			require.True(t, ok, "missing span %s", methodName)
			wantSpanCode := otelcode.Unset
			if check.grpcCode != grpc_codes.OK {
				wantSpanCode = otelcode.Error
			}
			assert.Equal(t, wantSpanCode, span.Status().Code)

			// validate metric
			assertStatsHandlerClientMetrics(t, mr, serviceName, name, check.wantRPCResponseStatusCode)
		})
	}
}

func assertStatsHandlerClientMetrics(t *testing.T, reader metric.Reader, serviceName, name, code string) {
	attrs := []attribute.KeyValue{
		semconv.RPCMethod(serviceName + "/" + name),
		semconv.RPCSystemNameGRPC,
		semconv.RPCResponseStatusCode(code),
		testMetricAttr,
	}
	// error.type is Conditionally Required on the duration metric if and only
	// if the RPC failed, and SHOULD be the canonical status code name. The
	// client-side failure classification treats every non-OK status as an
	// error, unlike the server side (see assertStatsHandlerServerMetrics).
	if code != "OK" {
		attrs = append(attrs, semconv.ErrorTypeKey.String(code))
	}
	want := metricdata.ScopeMetrics{
		Scope: wantInstrumentationScope,
		Metrics: []metricdata.Metrics{
			{
				Name:        rpcconv.ClientCallDuration{}.Name(),
				Description: rpcconv.ClientCallDuration{}.Description(),
				Unit:        rpcconv.ClientCallDuration{}.Unit(),
				Data: metricdata.Histogram[float64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.HistogramDataPoint[float64]{
						{
							Attributes: attribute.NewSet(attrs...),
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

func assertStatsHandlerServerMetrics(t *testing.T, reader metric.Reader, serviceName, name, code string) {
	attrs := []attribute.KeyValue{
		semconv.RPCMethod(serviceName + "/" + name),
		semconv.RPCSystemNameGRPC,
		semconv.RPCResponseStatusCode(code),
		testMetricAttr,
	}
	// error.type is Conditionally Required on the duration metric if and only
	// if the RPC failed, and SHOULD be the canonical status code name. The
	// server-side failure classification treats only Unknown, DeadlineExceeded,
	// Unimplemented, Internal, Unavailable and DataLoss as errors; e.g. a
	// client cancellation (CANCELLED) is not a server error.
	switch code {
	case "UNKNOWN", "DEADLINE_EXCEEDED", "UNIMPLEMENTED", "INTERNAL", "UNAVAILABLE", "DATA_LOSS":
		attrs = append(attrs, semconv.ErrorTypeKey.String(code))
	}
	want := metricdata.ScopeMetrics{
		Scope: wantInstrumentationScope,
		Metrics: []metricdata.Metrics{
			{
				Name:        rpcconv.ServerCallDuration{}.Name(),
				Description: rpcconv.ServerCallDuration{}.Description(),
				Unit:        rpcconv.ServerCallDuration{}.Unit(),
				Data: metricdata.Histogram[float64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.HistogramDataPoint[float64]{
						{
							Attributes: attribute.NewSet(attrs...),
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
