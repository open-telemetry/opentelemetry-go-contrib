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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.39.0"
	"go.opentelemetry.io/otel/semconv/v1.39.0/rpcconv"
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

			listener, err := net.Listen("tcp", "127.0.0.1:0")
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
					checkClientMetrics(t, clientMetricReader)
				})

				t.Run("ServerSpans", func(t *testing.T) {
					checkServerSpans(t, serverSR.Ended())
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
	assertEvents(t, []trace.Event{
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(1),
				semconv.RPCMessageTypeKey.String("SENT"),
				semconv.RPCMessageCompressedSizeKey.Int(0),
				semconv.RPCMessageUncompressedSizeKey.Int(0),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(1),
				semconv.RPCMessageTypeKey.String("RECEIVED"),
				semconv.RPCMessageCompressedSizeKey.Int(0),
				semconv.RPCMessageUncompressedSizeKey.Int(0),
			},
		},
	}, emptySpan.Events())
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
	assertEvents(t, []trace.Event{
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(1),
				semconv.RPCMessageTypeKey.String("SENT"),
				semconv.RPCMessageCompressedSizeKey.Int(271840),
				semconv.RPCMessageUncompressedSizeKey.Int(271840),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(1),
				semconv.RPCMessageTypeKey.String("RECEIVED"),
				semconv.RPCMessageCompressedSizeKey.Int(314167),
				semconv.RPCMessageUncompressedSizeKey.Int(314167),
			},
		},
	}, largeSpan.Events())
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
	assertEvents(t, []trace.Event{
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(1),
				semconv.RPCMessageTypeKey.String("SENT"),
				semconv.RPCMessageCompressedSizeKey.Int(27190),
				semconv.RPCMessageUncompressedSizeKey.Int(27190),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(2),
				semconv.RPCMessageTypeKey.String("SENT"),
				semconv.RPCMessageCompressedSizeKey.Int(12),
				semconv.RPCMessageUncompressedSizeKey.Int(12),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(3),
				semconv.RPCMessageTypeKey.String("SENT"),
				semconv.RPCMessageCompressedSizeKey.Int(1834),
				semconv.RPCMessageUncompressedSizeKey.Int(1834),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(4),
				semconv.RPCMessageTypeKey.String("SENT"),
				semconv.RPCMessageCompressedSizeKey.Int(45912),
				semconv.RPCMessageUncompressedSizeKey.Int(45912),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(1),
				semconv.RPCMessageTypeKey.String("RECEIVED"),
				semconv.RPCMessageCompressedSizeKey.Int(4),
				semconv.RPCMessageUncompressedSizeKey.Int(4),
			},
		},
		// client does not record an event for the server response.
	}, streamInput.Events())
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
	assertEvents(t, []trace.Event{
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(1),
				semconv.RPCMessageTypeKey.String("SENT"),
				semconv.RPCMessageCompressedSizeKey.Int(21),
				semconv.RPCMessageUncompressedSizeKey.Int(21),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(1),
				semconv.RPCMessageTypeKey.String("RECEIVED"),
				semconv.RPCMessageCompressedSizeKey.Int(31423),
				semconv.RPCMessageUncompressedSizeKey.Int(31423),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(2),
				semconv.RPCMessageTypeKey.String("RECEIVED"),
				semconv.RPCMessageCompressedSizeKey.Int(13),
				semconv.RPCMessageUncompressedSizeKey.Int(13),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(3),
				semconv.RPCMessageTypeKey.String("RECEIVED"),
				semconv.RPCMessageCompressedSizeKey.Int(2659),
				semconv.RPCMessageUncompressedSizeKey.Int(2659),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(4),
				semconv.RPCMessageTypeKey.String("RECEIVED"),
				semconv.RPCMessageCompressedSizeKey.Int(58987),
				semconv.RPCMessageUncompressedSizeKey.Int(58987),
			},
		},
	}, streamOutput.Events())
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
	assertEvents(t, []trace.Event{
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(1),
				semconv.RPCMessageTypeKey.String("SENT"),
				semconv.RPCMessageCompressedSizeKey.Int(27196),
				semconv.RPCMessageUncompressedSizeKey.Int(27196),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(1),
				semconv.RPCMessageTypeKey.String("RECEIVED"),
				semconv.RPCMessageCompressedSizeKey.Int(31423),
				semconv.RPCMessageUncompressedSizeKey.Int(31423),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(2),
				semconv.RPCMessageTypeKey.String("SENT"),
				semconv.RPCMessageCompressedSizeKey.Int(16),
				semconv.RPCMessageUncompressedSizeKey.Int(16),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(2),
				semconv.RPCMessageTypeKey.String("RECEIVED"),
				semconv.RPCMessageCompressedSizeKey.Int(13),
				semconv.RPCMessageUncompressedSizeKey.Int(13),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(3),
				semconv.RPCMessageTypeKey.String("SENT"),
				semconv.RPCMessageCompressedSizeKey.Int(1839),
				semconv.RPCMessageUncompressedSizeKey.Int(1839),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(3),
				semconv.RPCMessageTypeKey.String("RECEIVED"),
				semconv.RPCMessageCompressedSizeKey.Int(2659),
				semconv.RPCMessageUncompressedSizeKey.Int(2659),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(4),
				semconv.RPCMessageTypeKey.String("SENT"),
				semconv.RPCMessageCompressedSizeKey.Int(45918),
				semconv.RPCMessageUncompressedSizeKey.Int(45918),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(4),
				semconv.RPCMessageTypeKey.String("RECEIVED"),
				semconv.RPCMessageCompressedSizeKey.Int(58987),
				semconv.RPCMessageUncompressedSizeKey.Int(58987),
			},
		},
	}, pingPong.Events())
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethodKey.String("grpc.testing.TestService/FullDuplexCall"),
		semconv.RPCSystemNameGRPC,
		semconv.RPCResponseStatusCode(codes.OK.String()),
		semconv.ServerAddress(host),
		semconv.ServerPort(port),
		testSpanAttr,
	}, pingPong.Attributes())
}

func checkServerSpans(t *testing.T, spans []trace.ReadOnlySpan) {
	require.Len(t, spans, 5)

	emptySpan := spans[0]
	assert.False(t, emptySpan.EndTime().IsZero())
	assert.Equal(t, "grpc.testing.TestService/EmptyCall", emptySpan.Name())
	assertEvents(t, []trace.Event{
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(1),
				semconv.RPCMessageTypeKey.String("RECEIVED"),
				semconv.RPCMessageCompressedSizeKey.Int(0),
				semconv.RPCMessageUncompressedSizeKey.Int(0),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(1),
				semconv.RPCMessageTypeKey.String("SENT"),
				semconv.RPCMessageCompressedSizeKey.Int(0),
				semconv.RPCMessageUncompressedSizeKey.Int(0),
			},
		},
	}, emptySpan.Events())
	port, ok := findAttribute(emptySpan.Attributes(), semconv.ServerPortKey)
	assert.True(t, ok)
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethodKey.String("grpc.testing.TestService/EmptyCall"),
		semconv.RPCSystemNameGRPC,
		semconv.RPCResponseStatusCode(codes.OK.String()),
		semconv.ServerAddress("127.0.0.1"),
		port,
		testSpanAttr,
	}, emptySpan.Attributes())

	largeSpan := spans[1]
	assert.False(t, largeSpan.EndTime().IsZero())
	assert.Equal(t, "grpc.testing.TestService/UnaryCall", largeSpan.Name())
	assertEvents(t, []trace.Event{
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageTypeKey.String("RECEIVED"),
				semconv.RPCMessageIDKey.Int(1),
				semconv.RPCMessageCompressedSizeKey.Int(271840),
				semconv.RPCMessageUncompressedSizeKey.Int(271840),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageTypeKey.String("SENT"),
				semconv.RPCMessageIDKey.Int(1),
				semconv.RPCMessageCompressedSizeKey.Int(314167),
				semconv.RPCMessageUncompressedSizeKey.Int(314167),
			},
		},
	}, largeSpan.Events())
	port, ok = findAttribute(largeSpan.Attributes(), semconv.ServerPortKey)
	assert.True(t, ok)
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethodKey.String("grpc.testing.TestService/UnaryCall"),
		semconv.RPCSystemNameGRPC,
		semconv.RPCResponseStatusCode(codes.OK.String()),
		semconv.ServerAddress("127.0.0.1"),
		port,
		testSpanAttr,
	}, largeSpan.Attributes())

	streamInput := spans[2]
	assert.False(t, streamInput.EndTime().IsZero())
	assert.Equal(t, "grpc.testing.TestService/StreamingInputCall", streamInput.Name())
	assertEvents(t, []trace.Event{
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(1),
				semconv.RPCMessageTypeKey.String("RECEIVED"),
				semconv.RPCMessageCompressedSizeKey.Int(27190),
				semconv.RPCMessageUncompressedSizeKey.Int(27190),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(2),
				semconv.RPCMessageTypeKey.String("RECEIVED"),
				semconv.RPCMessageCompressedSizeKey.Int(12),
				semconv.RPCMessageUncompressedSizeKey.Int(12),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(3),
				semconv.RPCMessageTypeKey.String("RECEIVED"),
				semconv.RPCMessageCompressedSizeKey.Int(1834),
				semconv.RPCMessageUncompressedSizeKey.Int(1834),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(4),
				semconv.RPCMessageTypeKey.String("RECEIVED"),
				semconv.RPCMessageCompressedSizeKey.Int(45912),
				semconv.RPCMessageUncompressedSizeKey.Int(45912),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(1),
				semconv.RPCMessageTypeKey.String("SENT"),
				semconv.RPCMessageCompressedSizeKey.Int(4),
				semconv.RPCMessageUncompressedSizeKey.Int(4),
			},
		},
		// client does not record an event for the server response.
	}, streamInput.Events())
	port, ok = findAttribute(streamInput.Attributes(), semconv.ServerPortKey)
	assert.True(t, ok)
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethodKey.String("grpc.testing.TestService/StreamingInputCall"),
		semconv.RPCSystemNameGRPC,
		semconv.RPCResponseStatusCode(codes.OK.String()),
		semconv.ServerAddress("127.0.0.1"),
		port,
		testSpanAttr,
	}, streamInput.Attributes())

	streamOutput := spans[3]
	assert.False(t, streamOutput.EndTime().IsZero())
	assert.Equal(t, "grpc.testing.TestService/StreamingOutputCall", streamOutput.Name())
	assertEvents(t, []trace.Event{
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(1),
				semconv.RPCMessageTypeKey.String("RECEIVED"),
				semconv.RPCMessageCompressedSizeKey.Int(21),
				semconv.RPCMessageUncompressedSizeKey.Int(21),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(1),
				semconv.RPCMessageTypeKey.String("SENT"),
				semconv.RPCMessageCompressedSizeKey.Int(31423),
				semconv.RPCMessageUncompressedSizeKey.Int(31423),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(2),
				semconv.RPCMessageTypeKey.String("SENT"),
				semconv.RPCMessageCompressedSizeKey.Int(13),
				semconv.RPCMessageUncompressedSizeKey.Int(13),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(3),
				semconv.RPCMessageTypeKey.String("SENT"),
				semconv.RPCMessageCompressedSizeKey.Int(2659),
				semconv.RPCMessageUncompressedSizeKey.Int(2659),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(4),
				semconv.RPCMessageTypeKey.String("SENT"),
				semconv.RPCMessageCompressedSizeKey.Int(58987),
				semconv.RPCMessageUncompressedSizeKey.Int(58987),
			},
		},
	}, streamOutput.Events())
	port, ok = findAttribute(streamOutput.Attributes(), semconv.ServerPortKey)
	assert.True(t, ok)
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethodKey.String("grpc.testing.TestService/StreamingOutputCall"),
		semconv.RPCSystemNameGRPC,
		semconv.RPCResponseStatusCode(codes.OK.String()),
		semconv.ServerAddress("127.0.0.1"),
		port,
		testSpanAttr,
	}, streamOutput.Attributes())

	pingPong := spans[4]
	assert.False(t, pingPong.EndTime().IsZero())
	assert.Equal(t, "grpc.testing.TestService/FullDuplexCall", pingPong.Name())
	assertEvents(t, []trace.Event{
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(1),
				semconv.RPCMessageTypeKey.String("RECEIVED"),
				semconv.RPCMessageCompressedSizeKey.Int(27196),
				semconv.RPCMessageUncompressedSizeKey.Int(27196),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(1),
				semconv.RPCMessageTypeKey.String("SENT"),
				semconv.RPCMessageCompressedSizeKey.Int(31423),
				semconv.RPCMessageUncompressedSizeKey.Int(31423),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(2),
				semconv.RPCMessageTypeKey.String("RECEIVED"),
				semconv.RPCMessageCompressedSizeKey.Int(16),
				semconv.RPCMessageUncompressedSizeKey.Int(16),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(2),
				semconv.RPCMessageTypeKey.String("SENT"),
				semconv.RPCMessageCompressedSizeKey.Int(13),
				semconv.RPCMessageUncompressedSizeKey.Int(13),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(3),
				semconv.RPCMessageTypeKey.String("RECEIVED"),
				semconv.RPCMessageCompressedSizeKey.Int(1839),
				semconv.RPCMessageUncompressedSizeKey.Int(1839),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(3),
				semconv.RPCMessageTypeKey.String("SENT"),
				semconv.RPCMessageCompressedSizeKey.Int(2659),
				semconv.RPCMessageUncompressedSizeKey.Int(2659),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(4),
				semconv.RPCMessageTypeKey.String("RECEIVED"),
				semconv.RPCMessageCompressedSizeKey.Int(45918),
				semconv.RPCMessageUncompressedSizeKey.Int(45918),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(4),
				semconv.RPCMessageTypeKey.String("SENT"),
				semconv.RPCMessageCompressedSizeKey.Int(58987),
				semconv.RPCMessageUncompressedSizeKey.Int(58987),
			},
		},
	}, pingPong.Events())
	port, ok = findAttribute(pingPong.Attributes(), semconv.ServerPortKey)
	assert.True(t, ok)
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethodKey.String("grpc.testing.TestService/FullDuplexCall"),
		semconv.RPCSystemNameGRPC,
		semconv.RPCResponseStatusCode(codes.OK.String()),
		semconv.ServerAddress("127.0.0.1"),
		port,
		testSpanAttr,
	}, pingPong.Attributes())
}

func checkClientMetrics(t *testing.T, reader metric.Reader) {
	rm := metricdata.ResourceMetrics{}
	err := reader.Collect(t.Context(), &rm)
	assert.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	require.Len(t, rm.ScopeMetrics[0].Metrics, 3)
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
			{
				Name:        rpcconv.ClientRequestSize{}.Name(),
				Description: rpcconv.ClientRequestSize{}.Description(),
				Unit:        rpcconv.ClientRequestSize{}.Unit(),
				Data: metricdata.Histogram[int64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.HistogramDataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("grpc.testing.TestService/EmptyCall"),
								semconv.RPCSystemNameGRPC,
								testMetricAttr),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Max:          metricdata.NewExtrema(int64(0)),
							Min:          metricdata.NewExtrema(int64(0)),
							Count:        1,
							Sum:          0,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("grpc.testing.TestService/UnaryCall"),
								semconv.RPCSystemNameGRPC,
								testMetricAttr),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
							Max:          metricdata.NewExtrema(int64(271840)),
							Min:          metricdata.NewExtrema(int64(271840)),
							Count:        1,
							Sum:          271840,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("grpc.testing.TestService/StreamingInputCall"),
								semconv.RPCSystemNameGRPC,
								testMetricAttr),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 2},
							Max:          metricdata.NewExtrema(int64(45912)),
							Min:          metricdata.NewExtrema(int64(12)),
							Count:        4,
							Sum:          74948,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("grpc.testing.TestService/StreamingOutputCall"),
								semconv.RPCSystemNameGRPC,
								testMetricAttr),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Max:          metricdata.NewExtrema(int64(21)),
							Min:          metricdata.NewExtrema(int64(21)),
							Count:        1,
							Sum:          21,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("grpc.testing.TestService/FullDuplexCall"),
								semconv.RPCSystemNameGRPC,
								testMetricAttr),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 2},
							Max:          metricdata.NewExtrema(int64(45918)),
							Min:          metricdata.NewExtrema(int64(16)),
							Count:        4,
							Sum:          74969,
						},
					},
				},
			},
			{
				Name:        rpcconv.ClientResponseSize{}.Name(),
				Description: rpcconv.ClientResponseSize{}.Description(),
				Unit:        rpcconv.ClientResponseSize{}.Unit(),
				Data: metricdata.Histogram[int64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.HistogramDataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("grpc.testing.TestService/EmptyCall"),
								semconv.RPCSystemNameGRPC,
								testMetricAttr),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Max:          metricdata.NewExtrema(int64(0)),
							Min:          metricdata.NewExtrema(int64(0)),
							Count:        1,
							Sum:          0,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("grpc.testing.TestService/UnaryCall"),
								semconv.RPCSystemNameGRPC,
								testMetricAttr),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
							Max:          metricdata.NewExtrema(int64(314167)),
							Min:          metricdata.NewExtrema(int64(314167)),
							Count:        1,
							Sum:          314167,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("grpc.testing.TestService/StreamingInputCall"),
								semconv.RPCSystemNameGRPC,
								testMetricAttr),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Max:          metricdata.NewExtrema(int64(4)),
							Min:          metricdata.NewExtrema(int64(4)),
							Count:        1,
							Sum:          4,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("grpc.testing.TestService/StreamingOutputCall"),
								semconv.RPCSystemNameGRPC,
								testMetricAttr),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 2},
							Max:          metricdata.NewExtrema(int64(58987)),
							Min:          metricdata.NewExtrema(int64(13)),
							Count:        4,
							Sum:          93082,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("grpc.testing.TestService/FullDuplexCall"),
								semconv.RPCSystemNameGRPC,
								testMetricAttr),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 2},
							Max:          metricdata.NewExtrema(int64(58987)),
							Min:          metricdata.NewExtrema(int64(13)),
							Count:        4,
							Sum:          93082,
						},
					},
				},
			},
		},
	}
	metricdatatest.AssertEqual(t, expectedScopeMetric, rm.ScopeMetrics[0], metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue())
}

func checkServerMetrics(t *testing.T, reader metric.Reader) {
	rm := metricdata.ResourceMetrics{}
	err := reader.Collect(t.Context(), &rm)
	assert.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	require.Len(t, rm.ScopeMetrics[0].Metrics, 3)
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
			{
				Name:        rpcconv.ServerRequestSize{}.Name(),
				Description: rpcconv.ServerRequestSize{}.Description(),
				Unit:        rpcconv.ServerRequestSize{}.Unit(),
				Data: metricdata.Histogram[int64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.HistogramDataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("grpc.testing.TestService/EmptyCall"),
								semconv.RPCSystemNameGRPC,
								testMetricAttr),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Max:          metricdata.NewExtrema(int64(0)),
							Min:          metricdata.NewExtrema(int64(0)),
							Count:        1,
							Sum:          0,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("grpc.testing.TestService/UnaryCall"),
								semconv.RPCSystemNameGRPC,
								testMetricAttr),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
							Max:          metricdata.NewExtrema(int64(271840)),
							Min:          metricdata.NewExtrema(int64(271840)),
							Count:        1,
							Sum:          271840,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("grpc.testing.TestService/StreamingInputCall"),
								semconv.RPCSystemNameGRPC,
								testMetricAttr),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 2},
							Max:          metricdata.NewExtrema(int64(45912)),
							Min:          metricdata.NewExtrema(int64(12)),
							Count:        4,
							Sum:          74948,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("grpc.testing.TestService/StreamingOutputCall"),
								semconv.RPCSystemNameGRPC,
								testMetricAttr),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Max:          metricdata.NewExtrema(int64(21)),
							Min:          metricdata.NewExtrema(int64(21)),
							Count:        1,
							Sum:          21,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("grpc.testing.TestService/FullDuplexCall"),
								semconv.RPCSystemNameGRPC,
								testMetricAttr),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 2},
							Max:          metricdata.NewExtrema(int64(45918)),
							Min:          metricdata.NewExtrema(int64(16)),
							Count:        4,
							Sum:          74969,
						},
					},
				},
			},
			{
				Name:        rpcconv.ServerResponseSize{}.Name(),
				Description: rpcconv.ServerResponseSize{}.Description(),
				Unit:        rpcconv.ServerResponseSize{}.Unit(),
				Data: metricdata.Histogram[int64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.HistogramDataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("grpc.testing.TestService/EmptyCall"),
								semconv.RPCSystemNameGRPC,
								testMetricAttr),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Max:          metricdata.NewExtrema(int64(0)),
							Min:          metricdata.NewExtrema(int64(0)),
							Count:        1,
							Sum:          0,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("grpc.testing.TestService/UnaryCall"),
								semconv.RPCSystemNameGRPC,
								testMetricAttr),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
							Max:          metricdata.NewExtrema(int64(314167)),
							Min:          metricdata.NewExtrema(int64(314167)),
							Count:        1,
							Sum:          314167,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("grpc.testing.TestService/StreamingInputCall"),
								semconv.RPCSystemNameGRPC,
								testMetricAttr),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Max:          metricdata.NewExtrema(int64(4)),
							Min:          metricdata.NewExtrema(int64(4)),
							Count:        1,
							Sum:          4,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("grpc.testing.TestService/StreamingOutputCall"),
								semconv.RPCSystemNameGRPC,
								testMetricAttr),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 2},
							Max:          metricdata.NewExtrema(int64(58987)),
							Min:          metricdata.NewExtrema(int64(13)),
							Count:        4,
							Sum:          93082,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("grpc.testing.TestService/FullDuplexCall"),
								semconv.RPCSystemNameGRPC,
								testMetricAttr),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 2},
							Max:          metricdata.NewExtrema(int64(58987)),
							Min:          metricdata.NewExtrema(int64(13)),
							Count:        4,
							Sum:          93082,
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
	listener, err := net.Listen("tcp", "127.0.0.1:0")
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

		wg.Add(1)
		go func() {
			defer wg.Done()
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
				if errors.Is(err, io.EOF) { // possible due to context cancellation
					assert.ErrorIs(t, ctx.Err(), context.Canceled)
				} else {
					assert.NoError(t, err)
				}
			}
			assert.NoError(t, stream.CloseSend())
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
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
		}()

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
