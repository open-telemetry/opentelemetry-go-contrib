// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgrpc_test

import (
	"context"
	"errors"
	"io"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	oldrpcconv "go.opentelemetry.io/otel/semconv/v1.37.0/rpcconv" //nolint:depguard // Use of v1.37.0 is required for backward compatibility stability opt-in.
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/semconv/v1.43.0/rpcconv"
	oteltrace "go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	testpb "google.golang.org/grpc/interop/grpc_testing"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc/filters"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc/internal/test"
)

var (
	testSpanAttr   = attribute.String("test_span", "OK")
	testMetricAttr = attribute.String("test_metric", "OK")
)

func TestStatsHandler(t *testing.T) {
	tests := []struct {
		name           string
		filterSvcName  string
		expectRecorded bool
	}{
		{
			name:           "Recorded",
			filterSvcName:  "grpc.testing.TestService",
			expectRecorded: true,
		},
		{
			name:           "Dropped",
			filterSvcName:  "grpc.testing.OtherService",
			expectRecorded: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("OTEL_METRICS_EXEMPLAR_FILTER", "always_off")
			clientSR := tracetest.NewSpanRecorder()
			clientTP := trace.NewTracerProvider(trace.WithSpanProcessor(clientSR))
			clientMetricReader := metric.NewManualReader()
			clientMP := metric.NewMeterProvider(metric.WithReader(clientMetricReader))

			serverSR := tracetest.NewSpanRecorder()
			serverTP := trace.NewTracerProvider(trace.WithSpanProcessor(serverSR))
			serverMetricReader := metric.NewManualReader()
			serverMP := metric.NewMeterProvider(metric.WithReader(serverMetricReader))

			listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
			require.NoError(t, err, "failed to open port")
			client := newGrpcTest(
				t, listener,
				[]grpc.DialOption{
					grpc.WithStatsHandler(
						otelgrpc.NewClientHandler(
							otelgrpc.WithTracerProvider(clientTP),
							otelgrpc.WithMeterProvider(clientMP),
							otelgrpc.WithMessageEvents(otelgrpc.ReceivedEvents, otelgrpc.SentEvents),
							otelgrpc.WithFilter(filters.ServiceName(tt.filterSvcName)),
							otelgrpc.WithSpanAttributes(testSpanAttr),
							otelgrpc.WithMetricAttributes(testMetricAttr),
						),
					),
				},
				[]grpc.ServerOption{
					grpc.StatsHandler(
						otelgrpc.NewServerHandler(
							otelgrpc.WithTracerProvider(serverTP),
							otelgrpc.WithMeterProvider(serverMP),
							otelgrpc.WithMessageEvents(otelgrpc.ReceivedEvents, otelgrpc.SentEvents),
							otelgrpc.WithFilter(filters.ServiceName(tt.filterSvcName)),
							otelgrpc.WithSpanAttributes(testSpanAttr),
							otelgrpc.WithMetricAttributes(testMetricAttr),
						),
					),
				},
			)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			doCalls(ctx, client)

			if tt.expectRecorded {
				t.Run("ClientSpans", func(t *testing.T) {
					checkClientSpans(t, clientSR.Ended(), listener.Addr().String())
				})

				t.Run("ClientMetrics", func(t *testing.T) {
					checkClientMetrics(t, clientMetricReader, listener.Addr().String(), "")
				})

				t.Run("ServerSpans", func(t *testing.T) {
					checkServerSpans(t, serverSR, listener.Addr().String())
				})

				t.Run("ServerMetrics", func(t *testing.T) {
					checkServerMetrics(t, serverMetricReader, "")
				})
			} else {
				t.Run("ClientSpans", func(t *testing.T) {
					require.Empty(t, clientSR.Ended())
				})

				t.Run("ClientMetrics", func(t *testing.T) {
					rm := metricdata.ResourceMetrics{}
					err := clientMetricReader.Collect(t.Context(), &rm)
					assert.NoError(t, err)
					require.Empty(t, rm.ScopeMetrics)
				})

				t.Run("ServerSpans", func(t *testing.T) {
					require.Empty(t, serverSR.Ended())
				})

				t.Run("ServerMetrics", func(t *testing.T) {
					rm := metricdata.ResourceMetrics{}
					err := serverMetricReader.Collect(t.Context(), &rm)
					assert.NoError(t, err)
					require.Empty(t, rm.ScopeMetrics)
				})
			}
		})
	}
}

func checkClientSpans(t *testing.T, spans []trace.ReadOnlySpan, addr string) {
	require.Len(t, spans, 5)

	host, p, err := net.SplitHostPort(addr)
	require.NoError(t, err)
	port, err := strconv.Atoi(p)
	require.NoError(t, err)

	emptySpan := spans[0]
	assert.False(t, emptySpan.EndTime().IsZero())
	assert.Equal(t, "grpc.testing.TestService/EmptyCall", emptySpan.Name())
	assert.Empty(t, emptySpan.Events())
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethodKey.String("grpc.testing.TestService/EmptyCall"),
		semconv.RPCSystemNameGRPC,
		semconv.RPCResponseStatusCode(codes.OK.String()),
		semconv.ServerAddress(host),
		semconv.ServerPort(port),
		testSpanAttr,
	}, emptySpan.Attributes())

	largeSpan := spans[1]
	assert.False(t, largeSpan.EndTime().IsZero())
	assert.Equal(t, "grpc.testing.TestService/UnaryCall", largeSpan.Name())
	assert.Empty(t, largeSpan.Events())
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethodKey.String("grpc.testing.TestService/UnaryCall"),
		semconv.RPCSystemNameGRPC,
		semconv.RPCResponseStatusCode(codes.OK.String()),
		semconv.ServerAddress(host),
		semconv.ServerPort(port),
		testSpanAttr,
	}, largeSpan.Attributes())

	streamInput := spans[2]
	assert.False(t, streamInput.EndTime().IsZero())
	assert.Equal(t, "grpc.testing.TestService/StreamingInputCall", streamInput.Name())
	assert.Empty(t, streamInput.Events())
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethodKey.String("grpc.testing.TestService/StreamingInputCall"),
		semconv.RPCSystemNameGRPC,
		semconv.RPCResponseStatusCode(codes.OK.String()),
		semconv.ServerAddress(host),
		semconv.ServerPort(port),
		testSpanAttr,
	}, streamInput.Attributes())

	streamOutput := spans[3]
	assert.False(t, streamOutput.EndTime().IsZero())
	assert.Equal(t, "grpc.testing.TestService/StreamingOutputCall", streamOutput.Name())
	assert.Empty(t, streamOutput.Events())
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethodKey.String("grpc.testing.TestService/StreamingOutputCall"),
		semconv.RPCSystemNameGRPC,
		semconv.RPCResponseStatusCode(codes.OK.String()),
		semconv.ServerAddress(host),
		semconv.ServerPort(port),
		testSpanAttr,
	}, streamOutput.Attributes())

	pingPong := spans[4]
	assert.False(t, pingPong.EndTime().IsZero())
	assert.Equal(t, "grpc.testing.TestService/FullDuplexCall", pingPong.Name())
	assert.Empty(t, pingPong.Events())
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethodKey.String("grpc.testing.TestService/FullDuplexCall"),
		semconv.RPCSystemNameGRPC,
		semconv.RPCResponseStatusCode(codes.OK.String()),
		semconv.ServerAddress(host),
		semconv.ServerPort(port),
		testSpanAttr,
	}, pingPong.Attributes())
}

func checkServerSpans(t *testing.T, sr *tracetest.SpanRecorder, addr string) {
	host, p, err := net.SplitHostPort(addr)
	require.NoError(t, err)
	port, err := strconv.Atoi(p)
	require.NoError(t, err)

	var spans []trace.ReadOnlySpan
	require.Eventually(t, func() bool {
		spans = sr.Ended()
		return len(spans) == 5
	}, 1*time.Second, 10*time.Millisecond)

	spansByName := make(map[string]trace.ReadOnlySpan, len(spans))
	for _, s := range spans {
		spansByName[s.Name()] = s
	}

	for _, tc := range []struct {
		name string
	}{
		{"grpc.testing.TestService/EmptyCall"},
		{"grpc.testing.TestService/UnaryCall"},
		{"grpc.testing.TestService/StreamingInputCall"},
		{"grpc.testing.TestService/StreamingOutputCall"},
		{"grpc.testing.TestService/FullDuplexCall"},
	} {
		s, ok := spansByName[tc.name]
		if !assert.True(t, ok, "missing span %s", tc.name) {
			continue
		}
		assert.False(t, s.EndTime().IsZero())
		assert.Equal(t, tc.name, s.Name())
		assert.Empty(t, s.Events())
		assert.ElementsMatch(t, []attribute.KeyValue{
			semconv.RPCMethodKey.String(tc.name),
			semconv.RPCSystemNameGRPC,
			semconv.RPCResponseStatusCode(codes.OK.String()),
			semconv.ServerAddress(host),
			semconv.ServerPort(port),
			testSpanAttr,
		}, s.Attributes())
	}
}

// doFailingUnaryCall performs a unary RPC that returns a non-OK status
// (codes.Internal) and returns the resulting error.
func doFailingUnaryCall(ctx context.Context, tc testpb.TestServiceClient, args ...grpc.CallOption) error {
	req := &testpb.SimpleRequest{
		ResponseStatus: &testpb.EchoStatus{
			Code:    int32(codes.Internal),
			Message: "forced failure",
		},
	}
	_, err := tc.UnaryCall(ctx, req, args...)
	return err
}

// TestStatsHandlerErrorType verifies that error.type is recorded on the
// rpc.client.call.duration and rpc.server.call.duration metrics when the RPC
// fails with a non-OK status, and is absent on successful calls (per the RPC
// semantic conventions: Conditionally Required if and only if the operation
// failed). It covers all semconv stability opt-in modes: the legacy v1.37
// rpc.*.duration metrics must not carry error.type at all, so rpc/old and
// rpc/dup keep the compatibility telemetry shape.
func TestStatsHandlerErrorType(t *testing.T) {
	for _, tt := range []struct {
		name           string
		stabilityOptIn string
	}{
		{name: "new"},
		{name: "old", stabilityOptIn: "rpc/old"},
		{name: "dup", stabilityOptIn: "rpc/dup"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("OTEL_METRICS_EXEMPLAR_FILTER", "always_off")
			if tt.stabilityOptIn != "" {
				t.Setenv("OTEL_SEMCONV_STABILITY_OPT_IN", tt.stabilityOptIn)
			}
			clientMetricReader := metric.NewManualReader()
			clientMP := metric.NewMeterProvider(metric.WithReader(clientMetricReader))
			serverMetricReader := metric.NewManualReader()
			serverMP := metric.NewMeterProvider(metric.WithReader(serverMetricReader))

			listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
			require.NoError(t, err, "failed to open port")
			client := newGrpcTest(t, listener,
				[]grpc.DialOption{
					grpc.WithStatsHandler(otelgrpc.NewClientHandler(otelgrpc.WithMeterProvider(clientMP))),
				},
				[]grpc.ServerOption{
					grpc.StatsHandler(otelgrpc.NewServerHandler(otelgrpc.WithMeterProvider(serverMP))),
				},
			)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			// One successful and one failing call to the same method.
			_, err = client.UnaryCall(ctx, &testpb.SimpleRequest{})
			require.NoError(t, err)
			require.Error(t, doFailingUnaryCall(ctx, client))

			switch tt.stabilityOptIn {
			case "rpc/old":
				// Only the legacy v1.37 metrics exist; they must not carry error.type.
				checkLegacyNoErrorType(t, clientMetricReader, oldrpcconv.ClientDuration{}.Name())
				checkLegacyNoErrorType(t, serverMetricReader, oldrpcconv.ServerDuration{}.Name())
			case "rpc/dup":
				checkDurationErrorType(t, clientMetricReader, rpcconv.ClientCallDuration{}.Name())
				checkDurationErrorType(t, serverMetricReader, rpcconv.ServerCallDuration{}.Name())
				checkLegacyNoErrorType(t, clientMetricReader, oldrpcconv.ClientDuration{}.Name())
				checkLegacyNoErrorType(t, serverMetricReader, oldrpcconv.ServerDuration{}.Name())
			default:
				checkDurationErrorType(t, clientMetricReader, rpcconv.ClientCallDuration{}.Name())
				checkDurationErrorType(t, serverMetricReader, rpcconv.ServerCallDuration{}.Name())
			}
		})
	}
}

// checkDurationErrorType finds the data points for the failing and successful
// UnaryCall in the given duration metric and asserts error.type is only
// present on the failed one.
func checkDurationErrorType(t *testing.T, reader metric.Reader, metricName string) {
	t.Helper()
	const method = "grpc.testing.TestService/UnaryCall"

	var rm metricdata.ResourceMetrics
	var okDP, errDP *metricdata.HistogramDataPoint[float64]
	require.Eventually(t, func() bool {
		rm = metricdata.ResourceMetrics{}
		if err := reader.Collect(t.Context(), &rm); err != nil {
			return false
		}
		okDP, errDP = nil, nil
		for _, sm := range rm.ScopeMetrics {
			for _, m := range sm.Metrics {
				if m.Name != metricName {
					continue
				}
				hist, ok := m.Data.(metricdata.Histogram[float64])
				if !ok {
					continue
				}
				for i := range hist.DataPoints {
					dp := &hist.DataPoints[i]
					if v, ok := dp.Attributes.Value(attribute.Key("rpc.method")); !ok || v.AsString() != method {
						continue
					}
					if v, ok := dp.Attributes.Value(attribute.Key("error.type")); ok && v.AsString() == "INTERNAL" {
						errDP = dp
					} else {
						okDP = dp
					}
				}
			}
		}
		return okDP != nil && errDP != nil
	}, 3*time.Second, 10*time.Millisecond)

	require.NotNil(t, errDP, "%s: expected a failed-call data point with error.type=INTERNAL", metricName)
	require.NotNil(t, okDP, "%s: expected a successful-call data point without error.type", metricName)
	_, hasErrType := okDP.Attributes.Value(attribute.Key("error.type"))
	assert.False(t, hasErrType, "%s: successful call data point must not contain error.type", metricName)
	statusCode, ok := errDP.Attributes.Value(attribute.Key("rpc.response.status_code"))
	require.True(t, ok)
	assert.Equal(t, "INTERNAL", statusCode.AsString())
}

// checkLegacyNoErrorType verifies that the legacy v1.37 rpc.*.duration metrics
// do not carry error.type, even on failed calls. The failed data point is
// identified by rpc.response.status_code=INTERNAL since error.type itself
// cannot be used to distinguish it here.
func checkLegacyNoErrorType(t *testing.T, reader metric.Reader, metricName string) {
	t.Helper()
	// The stable modes use the full method path as the rpc.method value, while
	// the legacy rpc/old mode uses the short method name.
	const (
		fullMethod  = "grpc.testing.TestService/UnaryCall"
		shortMethod = "UnaryCall"
	)
	isUnaryCall := func(v string) bool {
		return v == fullMethod || v == shortMethod
	}

	var rm metricdata.ResourceMetrics
	var failedDP *metricdata.HistogramDataPoint[float64]
	require.Eventually(t, func() bool {
		rm = metricdata.ResourceMetrics{}
		if err := reader.Collect(t.Context(), &rm); err != nil {
			return false
		}
		failedDP = nil
		for _, sm := range rm.ScopeMetrics {
			for _, m := range sm.Metrics {
				if m.Name != metricName {
					continue
				}
				hist, ok := m.Data.(metricdata.Histogram[float64])
				if !ok {
					continue
				}
				for i := range hist.DataPoints {
					dp := &hist.DataPoints[i]
					if v, ok := dp.Attributes.Value(attribute.Key("rpc.method")); !ok || !isUnaryCall(v.AsString()) {
						continue
					}
					statusCode, ok := dp.Attributes.Value(attribute.Key("rpc.response.status_code"))
					if !ok || statusCode.AsString() != "INTERNAL" {
						continue
					}
					failedDP = dp
				}
			}
		}
		return failedDP != nil
	}, 3*time.Second, 10*time.Millisecond)

	require.NotNil(t, failedDP, "%s: expected a failed-call data point in the legacy metric", metricName)
	_, hasErrType := failedDP.Attributes.Value(attribute.Key("error.type"))
	assert.False(t, hasErrType, "%s: legacy metric must not carry error.type on failed calls", metricName)
}

func checkClientMetrics(t *testing.T, reader metric.Reader, addr, stabilityOptIn string) {
	rm := metricdata.ResourceMetrics{}
	err := reader.Collect(t.Context(), &rm)
	assert.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)

	host, p, err := net.SplitHostPort(addr)
	require.NoError(t, err)
	port, err := strconv.Atoi(p)
	require.NoError(t, err)

	var expectedMetrics []metricdata.Metrics

	switch stabilityOptIn {
	case "rpc/old":
		expectedMetrics = append(expectedMetrics, metricdata.Metrics{
			Name:        "rpc.client.duration",
			Description: "Measures the duration of outbound RPC.",
			Unit:        "ms",
			Data: metricdata.Histogram[float64]{
				Temporality: metricdata.CumulativeTemporality,
				DataPoints: []metricdata.HistogramDataPoint[float64]{
					{Attributes: attribute.NewSet(attribute.String("rpc.system", "grpc"), semconv.ServerAddress(host), semconv.ServerPort(port), attribute.String("rpc.service", "grpc.testing.TestService"), attribute.String("rpc.method", "EmptyCall"), semconv.RPCResponseStatusCode(codes.OK.String()), testMetricAttr)},
					{Attributes: attribute.NewSet(attribute.String("rpc.system", "grpc"), semconv.ServerAddress(host), semconv.ServerPort(port), attribute.String("rpc.service", "grpc.testing.TestService"), attribute.String("rpc.method", "UnaryCall"), semconv.RPCResponseStatusCode(codes.OK.String()), testMetricAttr)},
					{Attributes: attribute.NewSet(attribute.String("rpc.system", "grpc"), semconv.ServerAddress(host), semconv.ServerPort(port), attribute.String("rpc.service", "grpc.testing.TestService"), attribute.String("rpc.method", "StreamingInputCall"), semconv.RPCResponseStatusCode(codes.OK.String()), testMetricAttr)},
					{Attributes: attribute.NewSet(attribute.String("rpc.system", "grpc"), semconv.ServerAddress(host), semconv.ServerPort(port), attribute.String("rpc.service", "grpc.testing.TestService"), attribute.String("rpc.method", "StreamingOutputCall"), semconv.RPCResponseStatusCode(codes.OK.String()), testMetricAttr)},
					{Attributes: attribute.NewSet(attribute.String("rpc.system", "grpc"), semconv.ServerAddress(host), semconv.ServerPort(port), attribute.String("rpc.service", "grpc.testing.TestService"), attribute.String("rpc.method", "FullDuplexCall"), semconv.RPCResponseStatusCode(codes.OK.String()), testMetricAttr)},
				},
			},
		})
	case "":
		expectedMetrics = append(expectedMetrics, metricdata.Metrics{
			Name:        rpcconv.ClientCallDuration{}.Name(),
			Description: rpcconv.ClientCallDuration{}.Description(),
			Unit:        rpcconv.ClientCallDuration{}.Unit(),
			Data: metricdata.Histogram[float64]{
				Temporality: metricdata.CumulativeTemporality,
				DataPoints: []metricdata.HistogramDataPoint[float64]{
					{Attributes: attribute.NewSet(semconv.RPCResponseStatusCode(codes.OK.String()), semconv.RPCMethod("grpc.testing.TestService/EmptyCall"), semconv.RPCSystemNameGRPC, semconv.ServerAddress(host), semconv.ServerPort(port), testMetricAttr)},
					{Attributes: attribute.NewSet(semconv.RPCResponseStatusCode(codes.OK.String()), semconv.RPCMethod("grpc.testing.TestService/UnaryCall"), semconv.RPCSystemNameGRPC, semconv.ServerAddress(host), semconv.ServerPort(port), testMetricAttr)},
					{Attributes: attribute.NewSet(semconv.RPCResponseStatusCode(codes.OK.String()), semconv.RPCMethod("grpc.testing.TestService/StreamingInputCall"), semconv.RPCSystemNameGRPC, semconv.ServerAddress(host), semconv.ServerPort(port), testMetricAttr)},
					{Attributes: attribute.NewSet(semconv.RPCResponseStatusCode(codes.OK.String()), semconv.RPCMethod("grpc.testing.TestService/StreamingOutputCall"), semconv.RPCSystemNameGRPC, semconv.ServerAddress(host), semconv.ServerPort(port), testMetricAttr)},
					{Attributes: attribute.NewSet(semconv.RPCResponseStatusCode(codes.OK.String()), semconv.RPCMethod("grpc.testing.TestService/FullDuplexCall"), semconv.RPCSystemNameGRPC, semconv.ServerAddress(host), semconv.ServerPort(port), testMetricAttr)},
				},
			},
		})
	case "rpc/dup":
		combinedAttr := func(method string) attribute.Set {
			return attribute.NewSet(
				attribute.String("rpc.system", "grpc"),
				attribute.String("rpc.service", "grpc.testing.TestService"),
				semconv.RPCMethod("grpc.testing.TestService/"+method),
				semconv.RPCResponseStatusCode(codes.OK.String()),
				semconv.RPCSystemNameGRPC,
				semconv.ServerAddress(host),
				semconv.ServerPort(port),
				testMetricAttr,
			)
		}
		expectedMetrics = append(expectedMetrics, metricdata.Metrics{
			Name:        "rpc.client.duration",
			Description: "Measures the duration of outbound RPC.",
			Unit:        "ms",
			Data: metricdata.Histogram[float64]{
				Temporality: metricdata.CumulativeTemporality,
				DataPoints: []metricdata.HistogramDataPoint[float64]{
					{Attributes: combinedAttr("EmptyCall")},
					{Attributes: combinedAttr("UnaryCall")},
					{Attributes: combinedAttr("StreamingInputCall")},
					{Attributes: combinedAttr("StreamingOutputCall")},
					{Attributes: combinedAttr("FullDuplexCall")},
				},
			},
		}, metricdata.Metrics{
			Name:        rpcconv.ClientCallDuration{}.Name(),
			Description: rpcconv.ClientCallDuration{}.Description(),
			Unit:        rpcconv.ClientCallDuration{}.Unit(),
			Data: metricdata.Histogram[float64]{
				Temporality: metricdata.CumulativeTemporality,
				DataPoints: []metricdata.HistogramDataPoint[float64]{
					{Attributes: combinedAttr("EmptyCall")},
					{Attributes: combinedAttr("UnaryCall")},
					{Attributes: combinedAttr("StreamingInputCall")},
					{Attributes: combinedAttr("StreamingOutputCall")},
					{Attributes: combinedAttr("FullDuplexCall")},
				},
			},
		})
	}

	expectedScopeMetric := metricdata.ScopeMetrics{
		Scope: instrumentation.Scope{
			Name:      "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc",
			Version:   otelgrpc.Version,
			SchemaURL: semconv.SchemaURL,
		},
		Metrics: expectedMetrics,
	}

	metricdatatest.AssertEqual(t, expectedScopeMetric, rm.ScopeMetrics[0], metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue(), metricdatatest.IgnoreExemplars())
}

func checkServerMetrics(t *testing.T, reader metric.Reader, stabilityOptIn string) {
	var rm metricdata.ResourceMetrics
	require.Eventually(t, func() bool {
		rm = metricdata.ResourceMetrics{}
		if err := reader.Collect(t.Context(), &rm); err != nil {
			return false
		}
		if len(rm.ScopeMetrics) == 0 || len(rm.ScopeMetrics[0].Metrics) == 0 {
			return false
		}
		isOld := stabilityOptIn == "rpc/old" || stabilityOptIn == "rpc/dup"
		isNew := stabilityOptIn == "" || stabilityOptIn == "rpc/dup"

		var expectedCount int
		if isOld {
			expectedCount++
		}
		if isNew {
			expectedCount++
		}
		return len(rm.ScopeMetrics[0].Metrics) == expectedCount
	}, 1*time.Second, 10*time.Millisecond)

	require.Len(t, rm.ScopeMetrics, 1)

	var expectedMetrics []metricdata.Metrics

	switch stabilityOptIn {
	case "rpc/old":
		expectedMetrics = append(expectedMetrics, metricdata.Metrics{
			Name:        "rpc.server.duration",
			Description: "Measures the duration of inbound RPC.",
			Unit:        "ms",
			Data: metricdata.Histogram[float64]{
				Temporality: metricdata.CumulativeTemporality,
				DataPoints: []metricdata.HistogramDataPoint[float64]{
					{Attributes: attribute.NewSet(attribute.String("rpc.system", "grpc"), attribute.String("rpc.service", "grpc.testing.TestService"), attribute.String("rpc.method", "EmptyCall"), semconv.RPCResponseStatusCode(codes.OK.String()), testMetricAttr)},
					{Attributes: attribute.NewSet(attribute.String("rpc.system", "grpc"), attribute.String("rpc.service", "grpc.testing.TestService"), attribute.String("rpc.method", "UnaryCall"), semconv.RPCResponseStatusCode(codes.OK.String()), testMetricAttr)},
					{Attributes: attribute.NewSet(attribute.String("rpc.system", "grpc"), attribute.String("rpc.service", "grpc.testing.TestService"), attribute.String("rpc.method", "StreamingInputCall"), semconv.RPCResponseStatusCode(codes.OK.String()), testMetricAttr)},
					{Attributes: attribute.NewSet(attribute.String("rpc.system", "grpc"), attribute.String("rpc.service", "grpc.testing.TestService"), attribute.String("rpc.method", "StreamingOutputCall"), semconv.RPCResponseStatusCode(codes.OK.String()), testMetricAttr)},
					{Attributes: attribute.NewSet(attribute.String("rpc.system", "grpc"), attribute.String("rpc.service", "grpc.testing.TestService"), attribute.String("rpc.method", "FullDuplexCall"), semconv.RPCResponseStatusCode(codes.OK.String()), testMetricAttr)},
				},
			},
		})
	case "":
		expectedMetrics = append(expectedMetrics, metricdata.Metrics{
			Name:        rpcconv.ServerCallDuration{}.Name(),
			Description: rpcconv.ServerCallDuration{}.Description(),
			Unit:        rpcconv.ServerCallDuration{}.Unit(),
			Data: metricdata.Histogram[float64]{
				Temporality: metricdata.CumulativeTemporality,
				DataPoints: []metricdata.HistogramDataPoint[float64]{
					{Attributes: attribute.NewSet(semconv.RPCResponseStatusCode(codes.OK.String()), semconv.RPCMethod("grpc.testing.TestService/EmptyCall"), semconv.RPCSystemNameGRPC, testMetricAttr)},
					{Attributes: attribute.NewSet(semconv.RPCResponseStatusCode(codes.OK.String()), semconv.RPCMethod("grpc.testing.TestService/UnaryCall"), semconv.RPCSystemNameGRPC, testMetricAttr)},
					{Attributes: attribute.NewSet(semconv.RPCResponseStatusCode(codes.OK.String()), semconv.RPCMethod("grpc.testing.TestService/StreamingInputCall"), semconv.RPCSystemNameGRPC, testMetricAttr)},
					{Attributes: attribute.NewSet(semconv.RPCResponseStatusCode(codes.OK.String()), semconv.RPCMethod("grpc.testing.TestService/StreamingOutputCall"), semconv.RPCSystemNameGRPC, testMetricAttr)},
					{Attributes: attribute.NewSet(semconv.RPCResponseStatusCode(codes.OK.String()), semconv.RPCMethod("grpc.testing.TestService/FullDuplexCall"), semconv.RPCSystemNameGRPC, testMetricAttr)},
				},
			},
		})
	case "rpc/dup":
		combinedAttr := func(method string) attribute.Set {
			return attribute.NewSet(
				attribute.String("rpc.system", "grpc"),
				attribute.String("rpc.service", "grpc.testing.TestService"),
				semconv.RPCMethod("grpc.testing.TestService/"+method),
				semconv.RPCResponseStatusCode(codes.OK.String()),
				semconv.RPCSystemNameGRPC,
				testMetricAttr,
			)
		}
		expectedMetrics = append(expectedMetrics, metricdata.Metrics{
			Name:        "rpc.server.duration",
			Description: "Measures the duration of inbound RPC.",
			Unit:        "ms",
			Data: metricdata.Histogram[float64]{
				Temporality: metricdata.CumulativeTemporality,
				DataPoints: []metricdata.HistogramDataPoint[float64]{
					{Attributes: combinedAttr("EmptyCall")},
					{Attributes: combinedAttr("UnaryCall")},
					{Attributes: combinedAttr("StreamingInputCall")},
					{Attributes: combinedAttr("StreamingOutputCall")},
					{Attributes: combinedAttr("FullDuplexCall")},
				},
			},
		}, metricdata.Metrics{
			Name:        rpcconv.ServerCallDuration{}.Name(),
			Description: rpcconv.ServerCallDuration{}.Description(),
			Unit:        rpcconv.ServerCallDuration{}.Unit(),
			Data: metricdata.Histogram[float64]{
				Temporality: metricdata.CumulativeTemporality,
				DataPoints: []metricdata.HistogramDataPoint[float64]{
					{Attributes: combinedAttr("EmptyCall")},
					{Attributes: combinedAttr("UnaryCall")},
					{Attributes: combinedAttr("StreamingInputCall")},
					{Attributes: combinedAttr("StreamingOutputCall")},
					{Attributes: combinedAttr("FullDuplexCall")},
				},
			},
		})
	}

	expectedScopeMetric := metricdata.ScopeMetrics{
		Scope: instrumentation.Scope{
			Name:      "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc",
			Version:   otelgrpc.Version,
			SchemaURL: semconv.SchemaURL,
		},
		Metrics: expectedMetrics,
	}

	metricdatatest.AssertEqual(t, expectedScopeMetric, rm.ScopeMetrics[0], metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue(), metricdatatest.IgnoreExemplars())
}

// Ensure there is no data race for the following scenario:
// Bidirectional streaming + client cancels context in the middle of streaming.
func TestStatsHandlerConcurrentSafeContextCancellation(t *testing.T) {
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err, "failed to open port")
	client := newGrpcTest(
		t, listener,
		[]grpc.DialOption{
			grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		},
		[]grpc.ServerOption{
			grpc.StatsHandler(otelgrpc.NewServerHandler()),
		},
	)

	const n = 10
	for range n {
		ctx, cancel := context.WithCancel(t.Context())
		stream, err := client.FullDuplexCall(ctx)
		require.NoError(t, err)

		const messageCount = 10
		var wg sync.WaitGroup

		wg.Go(func() {
			for range messageCount {
				const reqSize = 1
				pl := test.ClientNewPayload(testpb.PayloadType_COMPRESSABLE, reqSize)
				respParam := []*testpb.ResponseParameters{
					{
						Size: reqSize,
					},
				}
				req := &testpb.StreamingOutputCallRequest{
					ResponseType:       testpb.PayloadType_COMPRESSABLE,
					ResponseParameters: respParam,
					Payload:            pl,
				}
				err := stream.Send(req)
				if errors.Is(err, io.EOF) || status.Code(err) == codes.Canceled { // possible due to context cancellation
					assert.ErrorIs(t, ctx.Err(), context.Canceled)
				} else {
					assert.NoError(t, err)
				}
			}
			assert.NoError(t, stream.CloseSend())
		})

		wg.Go(func() {
			for i := range messageCount {
				_, err := stream.Recv()
				if i > messageCount/2 {
					cancel()
				}
				// must continue to receive messages until server acknowledges the cancellation, to ensure no data race happens there too
				if status.Code(err) == codes.Canceled {
					return
				}
				assert.NoError(t, err)
			}
		})

		wg.Wait()
	}
}

func TestServerHandlerTagRPC(t *testing.T) {
	tests := []struct {
		name   string
		server stats.Handler
		ctx    context.Context
		info   *stats.RPCTagInfo
		exp    bool
	}{
		{
			name:   "start a span without filters",
			server: otelgrpc.NewServerHandler(otelgrpc.WithTracerProvider(trace.NewTracerProvider())),
			ctx:    t.Context(),
			info: &stats.RPCTagInfo{
				FullMethodName: "/grpc.health.v1.Health/Check",
			},
			exp: true,
		},
		{
			name: "don't start a span with filter and match",
			server: otelgrpc.NewServerHandler(otelgrpc.WithTracerProvider(trace.NewTracerProvider()), otelgrpc.WithFilter(func(ri *stats.RPCTagInfo) bool {
				return ri.FullMethodName != "/grpc.health.v1.Health/Check"
			})),
			ctx: t.Context(),
			info: &stats.RPCTagInfo{
				FullMethodName: "/grpc.health.v1.Health/Check",
			},
			exp: false,
		},
		{
			name: "start a span with filter and no match",
			server: otelgrpc.NewServerHandler(otelgrpc.WithTracerProvider(trace.NewTracerProvider()), otelgrpc.WithFilter(func(ri *stats.RPCTagInfo) bool {
				return ri.FullMethodName != "/grpc.health.v1.Health/Check"
			})),
			ctx: t.Context(),
			info: &stats.RPCTagInfo{
				FullMethodName: "/app.v1.Service/Get",
			},
			exp: true,
		},
	}

	for _, tt := range tests {
		t.Run(t.Name(), func(t *testing.T) {
			ctx := tt.server.TagRPC(tt.ctx, tt.info)

			got := oteltrace.SpanFromContext(ctx).IsRecording()

			if tt.exp != got {
				t.Errorf("expected %t, got %t", tt.exp, got)
			}
		})
	}
}

func TestClientHandlerTagRPC(t *testing.T) {
	tests := []struct {
		name   string
		client stats.Handler
		ctx    context.Context
		info   *stats.RPCTagInfo
		exp    bool
	}{
		{
			name:   "start a span without filters",
			client: otelgrpc.NewClientHandler(otelgrpc.WithTracerProvider(trace.NewTracerProvider())),
			ctx:    t.Context(),
			info: &stats.RPCTagInfo{
				FullMethodName: "/grpc.health.v1.Health/Check",
			},
			exp: true,
		},
		{
			name: "don't start a span with filter and match",
			client: otelgrpc.NewClientHandler(otelgrpc.WithTracerProvider(trace.NewTracerProvider()), otelgrpc.WithFilter(func(ri *stats.RPCTagInfo) bool {
				return ri.FullMethodName != "/grpc.health.v1.Health/Check"
			})),
			ctx: t.Context(),
			info: &stats.RPCTagInfo{
				FullMethodName: "/grpc.health.v1.Health/Check",
			},
			exp: false,
		},
		{
			name: "start a span with filter and no match",
			client: otelgrpc.NewClientHandler(otelgrpc.WithTracerProvider(trace.NewTracerProvider()), otelgrpc.WithFilter(func(ri *stats.RPCTagInfo) bool {
				return ri.FullMethodName != "/grpc.health.v1.Health/Check"
			})),
			ctx: t.Context(),
			info: &stats.RPCTagInfo{
				FullMethodName: "/app.v1.Service/Get",
			},
			exp: true,
		},
	}

	for _, tt := range tests {
		t.Run(t.Name(), func(t *testing.T) {
			ctx := tt.client.TagRPC(tt.ctx, tt.info)

			got := oteltrace.SpanFromContext(ctx).IsRecording()

			if tt.exp != got {
				t.Errorf("expected %t, got %t", tt.exp, got)
			}
		})
	}
}

func TestSpansSemconvOptIn(t *testing.T) {
	tests := []struct {
		name           string
		stabilityOptIn string
		wantAttrs      []attribute.KeyValue
		notWantAttrs   []string
	}{
		{
			name:           "default",
			stabilityOptIn: "",
			wantAttrs: []attribute.KeyValue{
				attribute.String("rpc.system.name", "grpc"),
				attribute.String("rpc.method", "pkg.Service/Method"),
			},
			notWantAttrs: []string{"rpc.service"},
		},

		{
			name:           "rpc_old",
			stabilityOptIn: "rpc/old",
			wantAttrs: []attribute.KeyValue{
				attribute.String("rpc.system", "grpc"),
				attribute.String("rpc.service", "pkg.Service"),
				attribute.String("rpc.method", "Method"),
			},
			notWantAttrs: []string{"rpc.system.name"},
		},
		{
			name:           "rpc_dup",
			stabilityOptIn: "rpc/dup",
			wantAttrs: []attribute.KeyValue{
				attribute.String("rpc.system", "grpc"),
				attribute.String("rpc.service", "pkg.Service"),
				attribute.String("rpc.method", "pkg.Service/Method"), // New wins
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.stabilityOptIn != "" {
				t.Setenv("OTEL_SEMCONV_STABILITY_OPT_IN", tt.stabilityOptIn)
			}

			sr := tracetest.NewSpanRecorder()
			tp := trace.NewTracerProvider(trace.WithSpanProcessor(sr))

			h := otelgrpc.NewServerHandler(otelgrpc.WithTracerProvider(tp))

			ctx := t.Context()
			info := &stats.RPCTagInfo{
				FullMethodName: "/pkg.Service/Method",
			}

			ctx = h.TagRPC(ctx, info)

			span := oteltrace.SpanFromContext(ctx)
			require.True(t, span.IsRecording())

			span.End()

			spans := sr.Ended()
			require.Len(t, spans, 1)

			attrs := spans[0].Attributes()
			for _, want := range tt.wantAttrs {
				assert.Contains(t, attrs, want)
			}

			for _, key := range tt.notWantAttrs {
				found := false
				for _, a := range attrs {
					if string(a.Key) == key {
						found = true
						break
					}
				}
				assert.False(t, found, "should not contain attribute %s", key)
			}
		})
	}
}

func TestMetricsSemconvOptIn(t *testing.T) {
	tests := []struct {
		name           string
		stabilityOptIn string
	}{
		{
			name:           "default",
			stabilityOptIn: "",
		},
		{
			name:           "rpc_old",
			stabilityOptIn: "rpc/old",
		},
		{
			name:           "rpc_dup",
			stabilityOptIn: "rpc/dup",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.stabilityOptIn != "" {
				t.Setenv("OTEL_SEMCONV_STABILITY_OPT_IN", tt.stabilityOptIn)
			} else {
				t.Setenv("OTEL_SEMCONV_STABILITY_OPT_IN", "")
			}

			clientMetricReader := metric.NewManualReader()
			clientMP := metric.NewMeterProvider(metric.WithReader(clientMetricReader))

			serverMetricReader := metric.NewManualReader()
			serverMP := metric.NewMeterProvider(metric.WithReader(serverMetricReader))

			listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
			require.NoError(t, err, "failed to open port")

			client := newGrpcTest(
				t, listener,
				[]grpc.DialOption{
					grpc.WithStatsHandler(
						otelgrpc.NewClientHandler(
							otelgrpc.WithMeterProvider(clientMP),
							otelgrpc.WithMetricAttributes(testMetricAttr),
						),
					),
				},
				[]grpc.ServerOption{
					grpc.StatsHandler(
						otelgrpc.NewServerHandler(
							otelgrpc.WithTracerProvider(trace.NewTracerProvider()), // ensure we create one for testing
							otelgrpc.WithMeterProvider(serverMP),
							otelgrpc.WithMetricAttributes(testMetricAttr),
						),
					),
				},
			)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			doCalls(ctx, client)

			t.Run("ClientMetrics", func(t *testing.T) {
				checkClientMetrics(t, clientMetricReader, listener.Addr().String(), tt.stabilityOptIn)
			})

			t.Run("ServerMetrics", func(t *testing.T) {
				checkServerMetrics(t, serverMetricReader, tt.stabilityOptIn)
			})
		})
	}
}
