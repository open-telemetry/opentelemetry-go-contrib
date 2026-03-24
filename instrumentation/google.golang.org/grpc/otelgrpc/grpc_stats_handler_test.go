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
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/semconv/v1.40.0/rpcconv"
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
			client := newGrpcTest(t, listener,
				[]grpc.DialOption{
					grpc.WithStatsHandler(otelgrpc.NewClientHandler(
						otelgrpc.WithTracerProvider(clientTP),
						otelgrpc.WithMeterProvider(clientMP),
						otelgrpc.WithMessageEvents(otelgrpc.ReceivedEvents, otelgrpc.SentEvents),
						otelgrpc.WithFilter(filters.ServiceName(tt.filterSvcName)),
						otelgrpc.WithSpanAttributes(testSpanAttr),
						otelgrpc.WithMetricAttributes(testMetricAttr)),
					),
				},
				[]grpc.ServerOption{
					grpc.StatsHandler(otelgrpc.NewServerHandler(
						otelgrpc.WithTracerProvider(serverTP),
						otelgrpc.WithMeterProvider(serverMP),
						otelgrpc.WithMessageEvents(otelgrpc.ReceivedEvents, otelgrpc.SentEvents),
						otelgrpc.WithFilter(filters.ServiceName(tt.filterSvcName)),
						otelgrpc.WithSpanAttributes(testSpanAttr),
						otelgrpc.WithMetricAttributes(testMetricAttr)),
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
					checkClientMetrics(t, clientMetricReader, listener.Addr().String())
				})

				t.Run("ServerSpans", func(t *testing.T) {
					checkServerSpans(t, serverSR, listener.Addr().String())
				})

				t.Run("ServerMetrics", func(t *testing.T) {
					checkServerMetrics(t, serverMetricReader)
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

func checkClientMetrics(t *testing.T, reader metric.Reader, addr string) {
	rm := metricdata.ResourceMetrics{}
	err := reader.Collect(t.Context(), &rm)
	assert.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	require.Len(t, rm.ScopeMetrics[0].Metrics, 1)

	host, p, err := net.SplitHostPort(addr)
	require.NoError(t, err)
	port, err := strconv.Atoi(p)
	require.NoError(t, err)
	expectedScopeMetric := metricdata.ScopeMetrics{
		Scope: instrumentation.Scope{
			Name:      "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc",
			Version:   otelgrpc.Version,
			SchemaURL: semconv.SchemaURL,
		},
		Metrics: []metricdata.Metrics{
			{
				Name:        rpcconv.ClientCallDuration{}.Name(),
				Description: rpcconv.ClientCallDuration{}.Description(),
				Unit:        rpcconv.ClientCallDuration{}.Unit(),
				Data: metricdata.Histogram[float64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.HistogramDataPoint[float64]{
						{
							Attributes: attribute.NewSet(
								semconv.RPCResponseStatusCode(codes.OK.String()),
								semconv.RPCMethod("grpc.testing.TestService/EmptyCall"),
								semconv.RPCSystemNameGRPC,
								semconv.ServerAddress(host),
								semconv.ServerPort(port),
								testMetricAttr),
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCResponseStatusCode(codes.OK.String()),
								semconv.RPCMethod("grpc.testing.TestService/UnaryCall"),
								semconv.RPCSystemNameGRPC,
								semconv.ServerAddress(host),
								semconv.ServerPort(port),
								testMetricAttr),
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCResponseStatusCode(codes.OK.String()),
								semconv.RPCMethod("grpc.testing.TestService/StreamingInputCall"),
								semconv.RPCSystemNameGRPC,
								semconv.ServerAddress(host),
								semconv.ServerPort(port),
								testMetricAttr),
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCResponseStatusCode(codes.OK.String()),
								semconv.RPCMethod("grpc.testing.TestService/StreamingOutputCall"),
								semconv.RPCSystemNameGRPC,
								semconv.ServerAddress(host),
								semconv.ServerPort(port),
								testMetricAttr),
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCResponseStatusCode(codes.OK.String()),
								semconv.RPCMethod("grpc.testing.TestService/FullDuplexCall"),
								semconv.RPCSystemNameGRPC,
								semconv.ServerAddress(host),
								semconv.ServerPort(port),
								testMetricAttr),
						},
					},
				},
			},
		},
	}
	metricdatatest.AssertEqual(t, expectedScopeMetric, rm.ScopeMetrics[0], metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue())
}

func checkServerMetrics(t *testing.T, reader metric.Reader) {
	var rm metricdata.ResourceMetrics
	require.Eventually(t, func() bool {
		rm = metricdata.ResourceMetrics{}
		if err := reader.Collect(t.Context(), &rm); err != nil {
			return false
		}
		if len(rm.ScopeMetrics) == 0 || len(rm.ScopeMetrics[0].Metrics) == 0 {
			return false
		}
		wantName := rpcconv.ServerCallDuration{}.Name()
		for _, m := range rm.ScopeMetrics[0].Metrics {
			if m.Name == wantName {
				data, ok := m.Data.(metricdata.Histogram[float64])
				return ok && len(data.DataPoints) == 5
			}
		}
		return false
	}, 1*time.Second, 10*time.Millisecond)

	require.Len(t, rm.ScopeMetrics, 1)
	require.Len(t, rm.ScopeMetrics[0].Metrics, 1)
	expectedScopeMetric := metricdata.ScopeMetrics{
		Scope: instrumentation.Scope{
			Name:      "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc",
			Version:   otelgrpc.Version,
			SchemaURL: semconv.SchemaURL,
		},
		Metrics: []metricdata.Metrics{
			{
				Name:        rpcconv.ServerCallDuration{}.Name(),
				Description: rpcconv.ServerCallDuration{}.Description(),
				Unit:        rpcconv.ServerCallDuration{}.Unit(),
				Data: metricdata.Histogram[float64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.HistogramDataPoint[float64]{
						{
							Attributes: attribute.NewSet(
								semconv.RPCResponseStatusCode(codes.OK.String()),
								semconv.RPCMethod("grpc.testing.TestService/EmptyCall"),
								semconv.RPCSystemNameGRPC,
								testMetricAttr),
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCResponseStatusCode(codes.OK.String()),
								semconv.RPCMethod("grpc.testing.TestService/UnaryCall"),
								semconv.RPCSystemNameGRPC,
								testMetricAttr),
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCResponseStatusCode(codes.OK.String()),
								semconv.RPCMethod("grpc.testing.TestService/StreamingInputCall"),
								semconv.RPCSystemNameGRPC,
								testMetricAttr),
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCResponseStatusCode(codes.OK.String()),
								semconv.RPCMethod("grpc.testing.TestService/StreamingOutputCall"),
								semconv.RPCSystemNameGRPC,
								testMetricAttr),
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCResponseStatusCode(codes.OK.String()),
								semconv.RPCMethod("grpc.testing.TestService/FullDuplexCall"),
								semconv.RPCSystemNameGRPC,
								testMetricAttr),
						},
					},
				},
			},
		},
	}
	metricdatatest.AssertEqual(t, expectedScopeMetric, rm.ScopeMetrics[0], metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue())
}

// Ensure there is no data race for the following scenario:
// Bidirectional streaming + client cancels context in the middle of streaming.
func TestStatsHandlerConcurrentSafeContextCancellation(t *testing.T) {
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err, "failed to open port")
	client := newGrpcTest(t, listener,
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
