// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/test/bufconn"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"

	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

func TestStatsHandler(t *testing.T) {
	clientSR := tracetest.NewSpanRecorder()
	clientTP := trace.NewTracerProvider(trace.WithSpanProcessor(clientSR))
	clientMetricReader := metric.NewManualReader()
	clientMP := metric.NewMeterProvider(metric.WithReader(clientMetricReader))

	serverSR := tracetest.NewSpanRecorder()
	serverTP := trace.NewTracerProvider(trace.WithSpanProcessor(serverSR))
	serverMetricReader := metric.NewManualReader()
	serverMP := metric.NewMeterProvider(metric.WithReader(serverMetricReader))

	listener := bufconn.Listen(bufSize)
	defer listener.Close()
	err := newGrpcTest(
		listener,
		[]grpc.DialOption{
			grpc.WithStatsHandler(otelgrpc.NewClientHandler(otelgrpc.WithTracerProvider(clientTP), otelgrpc.WithMeterProvider(clientMP))),
		},
		[]grpc.ServerOption{
			grpc.StatsHandler(otelgrpc.NewServerHandler(otelgrpc.WithTracerProvider(serverTP), otelgrpc.WithMeterProvider(serverMP))),
		},
	)
	require.NoError(t, err)

	t.Run("ClientSpans", func(t *testing.T) {
		checkClientSpans(t, clientSR.Ended())
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
}

func checkClientSpans(t *testing.T, spans []trace.ReadOnlySpan) {
	require.Len(t, spans, 5)

	emptySpan := spans[0]
	assert.False(t, emptySpan.EndTime().IsZero())
	assert.Equal(t, "grpc.testing.TestService/EmptyCall", emptySpan.Name())
	assertEvents(t, []trace.Event{
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(1),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(0),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(0),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(1),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(0),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(0),
			},
		},
	}, emptySpan.Events())
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethodKey.String("EmptyCall"),
		semconv.RPCServiceKey.String("grpc.testing.TestService"),
		otelgrpc.RPCSystemGRPC,
		otelgrpc.GRPCStatusCodeKey.Int64(int64(codes.OK)),
	}, emptySpan.Attributes())

	largeSpan := spans[1]
	assert.False(t, largeSpan.EndTime().IsZero())
	assert.Equal(t, "grpc.testing.TestService/UnaryCall", largeSpan.Name())
	assertEvents(t, []trace.Event{
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(1),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(271840),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(271840),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(1),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(314167),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(314167),
			},
		},
	}, largeSpan.Events())
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethodKey.String("UnaryCall"),
		semconv.RPCServiceKey.String("grpc.testing.TestService"),
		otelgrpc.RPCSystemGRPC,
		otelgrpc.GRPCStatusCodeKey.Int64(int64(codes.OK)),
	}, largeSpan.Attributes())

	streamInput := spans[2]
	assert.False(t, streamInput.EndTime().IsZero())
	assert.Equal(t, "grpc.testing.TestService/StreamingInputCall", streamInput.Name())
	assertEvents(t, []trace.Event{
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(1),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(27190),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(27190),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(2),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(12),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(12),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(3),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(1834),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(1834),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(4),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(45912),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(45912),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(1),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(4),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(4),
			},
		},
		// client does not record an event for the server response.
	}, streamInput.Events())
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethodKey.String("StreamingInputCall"),
		semconv.RPCServiceKey.String("grpc.testing.TestService"),
		otelgrpc.RPCSystemGRPC,
		otelgrpc.GRPCStatusCodeKey.Int64(int64(codes.OK)),
	}, streamInput.Attributes())

	streamOutput := spans[3]
	assert.False(t, streamOutput.EndTime().IsZero())
	assert.Equal(t, "grpc.testing.TestService/StreamingOutputCall", streamOutput.Name())
	assertEvents(t, []trace.Event{
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(1),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(21),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(21),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(1),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(31423),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(31423),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(2),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(13),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(13),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(3),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(2659),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(2659),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(4),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(58987),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(58987),
			},
		},
	}, streamOutput.Events())
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethodKey.String("StreamingOutputCall"),
		semconv.RPCServiceKey.String("grpc.testing.TestService"),
		otelgrpc.RPCSystemGRPC,
		otelgrpc.GRPCStatusCodeKey.Int64(int64(codes.OK)),
	}, streamOutput.Attributes())

	pingPong := spans[4]
	assert.False(t, pingPong.EndTime().IsZero())
	assert.Equal(t, "grpc.testing.TestService/FullDuplexCall", pingPong.Name())
	assertEvents(t, []trace.Event{
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(1),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(27196),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(27196),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(1),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(31423),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(31423),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(2),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(16),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(16),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(2),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(13),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(13),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(3),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(1839),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(1839),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(3),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(2659),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(2659),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(4),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(45918),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(45918),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(4),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(58987),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(58987),
			},
		},
	}, pingPong.Events())
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethodKey.String("FullDuplexCall"),
		semconv.RPCServiceKey.String("grpc.testing.TestService"),
		otelgrpc.RPCSystemGRPC,
		otelgrpc.GRPCStatusCodeKey.Int64(int64(codes.OK)),
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
				otelgrpc.RPCMessageIDKey.Int(1),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(0),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(0),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(1),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(0),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(0),
			},
		},
	}, emptySpan.Events())
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethodKey.String("EmptyCall"),
		semconv.RPCServiceKey.String("grpc.testing.TestService"),
		otelgrpc.RPCSystemGRPC,
		otelgrpc.GRPCStatusCodeKey.Int64(int64(codes.OK)),
	}, emptySpan.Attributes())

	largeSpan := spans[1]
	assert.False(t, largeSpan.EndTime().IsZero())
	assert.Equal(t, "grpc.testing.TestService/UnaryCall", largeSpan.Name())
	assertEvents(t, []trace.Event{
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
				otelgrpc.RPCMessageIDKey.Int(1),
				otelgrpc.RPCMessageCompressedSizeKey.Int(271840),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(271840),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageTypeKey.String("SENT"),
				otelgrpc.RPCMessageIDKey.Int(1),
				otelgrpc.RPCMessageCompressedSizeKey.Int(314167),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(314167),
			},
		},
	}, largeSpan.Events())
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethodKey.String("UnaryCall"),
		semconv.RPCServiceKey.String("grpc.testing.TestService"),
		otelgrpc.RPCSystemGRPC,
		otelgrpc.GRPCStatusCodeKey.Int64(int64(codes.OK)),
	}, largeSpan.Attributes())

	streamInput := spans[2]
	assert.False(t, streamInput.EndTime().IsZero())
	assert.Equal(t, "grpc.testing.TestService/StreamingInputCall", streamInput.Name())
	assertEvents(t, []trace.Event{
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(1),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(27190),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(27190),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(2),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(12),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(12),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(3),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(1834),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(1834),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(4),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(45912),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(45912),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(1),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(4),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(4),
			},
		},
		// client does not record an event for the server response.
	}, streamInput.Events())
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethodKey.String("StreamingInputCall"),
		semconv.RPCServiceKey.String("grpc.testing.TestService"),
		otelgrpc.RPCSystemGRPC,
		otelgrpc.GRPCStatusCodeKey.Int64(int64(codes.OK)),
	}, streamInput.Attributes())

	streamOutput := spans[3]
	assert.False(t, streamOutput.EndTime().IsZero())
	assert.Equal(t, "grpc.testing.TestService/StreamingOutputCall", streamOutput.Name())
	assertEvents(t, []trace.Event{
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(1),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(21),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(21),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(1),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(31423),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(31423),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(2),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(13),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(13),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(3),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(2659),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(2659),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(4),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(58987),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(58987),
			},
		},
	}, streamOutput.Events())
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethodKey.String("StreamingOutputCall"),
		semconv.RPCServiceKey.String("grpc.testing.TestService"),
		otelgrpc.RPCSystemGRPC,
		otelgrpc.GRPCStatusCodeKey.Int64(int64(codes.OK)),
	}, streamOutput.Attributes())

	pingPong := spans[4]
	assert.False(t, pingPong.EndTime().IsZero())
	assert.Equal(t, "grpc.testing.TestService/FullDuplexCall", pingPong.Name())
	assertEvents(t, []trace.Event{
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(1),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(27196),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(27196),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(1),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(31423),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(31423),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(2),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(16),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(16),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(2),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(13),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(13),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(3),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(1839),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(1839),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(3),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(2659),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(2659),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(4),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(45918),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(45918),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(4),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
				otelgrpc.RPCMessageCompressedSizeKey.Int(58987),
				otelgrpc.RPCMessageUncompressedSizeKey.Int(58987),
			},
		},
	}, pingPong.Events())
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethodKey.String("FullDuplexCall"),
		semconv.RPCServiceKey.String("grpc.testing.TestService"),
		otelgrpc.RPCSystemGRPC,
		otelgrpc.GRPCStatusCodeKey.Int64(int64(codes.OK)),
	}, pingPong.Attributes())
}

func checkClientMetrics(t *testing.T, reader metric.Reader) {
	rm := metricdata.ResourceMetrics{}
	err := reader.Collect(context.Background(), &rm)
	assert.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	require.Len(t, rm.ScopeMetrics[0].Metrics, 5)
	expectedScopeMetric := metricdata.ScopeMetrics{
		Scope: instrumentation.Scope{
			Name:      "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc",
			Version:   otelgrpc.Version(),
			SchemaURL: "https://opentelemetry.io/schemas/1.17.0",
		},
		Metrics: []metricdata.Metrics{
			{
				Name:        "rpc.client.duration",
				Description: "Measures the duration of inbound RPC.",
				Unit:        "ms",
				Data: metricdata.Histogram[float64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.HistogramDataPoint[float64]{
						{
							Attributes: attribute.NewSet(
								semconv.RPCGRPCStatusCodeOk,
								semconv.RPCMethod("EmptyCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCGRPCStatusCodeOk,
								semconv.RPCMethod("UnaryCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCGRPCStatusCodeOk,
								semconv.RPCMethod("StreamingInputCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCGRPCStatusCodeOk,
								semconv.RPCMethod("StreamingOutputCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCGRPCStatusCodeOk,
								semconv.RPCMethod("FullDuplexCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
						},
					},
				},
			},
			{
				Name:        "rpc.client.request.size",
				Description: "Measures size of RPC request messages (uncompressed).",
				Unit:        "By",
				Data: metricdata.Histogram[int64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.HistogramDataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("EmptyCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Max:          metricdata.NewExtrema(int64(0)),
							Min:          metricdata.NewExtrema(int64(0)),
							Count:        1,
							Sum:          0,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("UnaryCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
							Max:          metricdata.NewExtrema(int64(314167)),
							Min:          metricdata.NewExtrema(int64(314167)),
							Count:        1,
							Sum:          314167,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("StreamingInputCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Max:          metricdata.NewExtrema(int64(4)),
							Min:          metricdata.NewExtrema(int64(4)),
							Count:        1,
							Sum:          4,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("StreamingOutputCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 2},
							Max:          metricdata.NewExtrema(int64(58987)),
							Min:          metricdata.NewExtrema(int64(13)),
							Count:        4,
							Sum:          93082,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("FullDuplexCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
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
			{
				Name:        "rpc.client.response.size",
				Description: "Measures size of RPC response messages (uncompressed).",
				Unit:        "By",
				Data: metricdata.Histogram[int64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.HistogramDataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("EmptyCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Max:          metricdata.NewExtrema(int64(0)),
							Min:          metricdata.NewExtrema(int64(0)),
							Count:        1,
							Sum:          0,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("UnaryCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
							Max:          metricdata.NewExtrema(int64(271840)),
							Min:          metricdata.NewExtrema(int64(271840)),
							Count:        1,
							Sum:          271840,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("StreamingInputCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 2},
							Max:          metricdata.NewExtrema(int64(45912)),
							Min:          metricdata.NewExtrema(int64(12)),
							Count:        4,
							Sum:          74948,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("StreamingOutputCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Max:          metricdata.NewExtrema(int64(21)),
							Min:          metricdata.NewExtrema(int64(21)),
							Count:        1,
							Sum:          21,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("FullDuplexCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
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
				Name:        "rpc.client.requests_per_rpc",
				Description: "Measures the number of messages received per RPC. Should be 1 for all non-streaming RPCs.",
				Unit:        "{count}",
				Data: metricdata.Histogram[int64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.HistogramDataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								semconv.RPCGRPCStatusCodeOk,
								semconv.RPCMethod("EmptyCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Max:          metricdata.NewExtrema(int64(1)),
							Min:          metricdata.NewExtrema(int64(1)),
							Count:        1,
							Sum:          1,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCGRPCStatusCodeOk,
								semconv.RPCMethod("UnaryCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Max:          metricdata.NewExtrema(int64(1)),
							Min:          metricdata.NewExtrema(int64(1)),
							Count:        1,
							Sum:          1,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCGRPCStatusCodeOk,
								semconv.RPCMethod("StreamingInputCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Max:          metricdata.NewExtrema(int64(1)),
							Min:          metricdata.NewExtrema(int64(1)),
							Count:        1,
							Sum:          1,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCGRPCStatusCodeOk,
								semconv.RPCMethod("StreamingOutputCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Max:          metricdata.NewExtrema(int64(4)),
							Min:          metricdata.NewExtrema(int64(4)),
							Count:        1,
							Sum:          4,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCGRPCStatusCodeOk,
								semconv.RPCMethod("FullDuplexCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Max:          metricdata.NewExtrema(int64(4)),
							Min:          metricdata.NewExtrema(int64(4)),
							Count:        1,
							Sum:          4,
						},
					},
				},
			},
			{
				Name:        "rpc.client.responses_per_rpc",
				Description: "Measures the number of messages received per RPC. Should be 1 for all non-streaming RPCs.",
				Unit:        "{count}",
				Data: metricdata.Histogram[int64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.HistogramDataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								semconv.RPCGRPCStatusCodeOk,
								semconv.RPCMethod("EmptyCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Max:          metricdata.NewExtrema(int64(1)),
							Min:          metricdata.NewExtrema(int64(1)),
							Count:        1,
							Sum:          1,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCGRPCStatusCodeOk,
								semconv.RPCMethod("UnaryCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Max:          metricdata.NewExtrema(int64(1)),
							Min:          metricdata.NewExtrema(int64(1)),
							Count:        1,
							Sum:          1,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCGRPCStatusCodeOk,
								semconv.RPCMethod("StreamingInputCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Max:          metricdata.NewExtrema(int64(4)),
							Min:          metricdata.NewExtrema(int64(4)),
							Count:        1,
							Sum:          4,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCGRPCStatusCodeOk,
								semconv.RPCMethod("StreamingOutputCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Max:          metricdata.NewExtrema(int64(1)),
							Min:          metricdata.NewExtrema(int64(1)),
							Count:        1,
							Sum:          1,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCGRPCStatusCodeOk,
								semconv.RPCMethod("FullDuplexCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Max:          metricdata.NewExtrema(int64(4)),
							Min:          metricdata.NewExtrema(int64(4)),
							Count:        1,
							Sum:          4,
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
	err := reader.Collect(context.Background(), &rm)
	assert.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	require.Len(t, rm.ScopeMetrics[0].Metrics, 5)
	expectedScopeMetric := metricdata.ScopeMetrics{
		Scope: instrumentation.Scope{
			Name:      "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc",
			Version:   otelgrpc.Version(),
			SchemaURL: "https://opentelemetry.io/schemas/1.17.0",
		},
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
								semconv.RPCGRPCStatusCodeOk,
								semconv.RPCMethod("EmptyCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCGRPCStatusCodeOk,
								semconv.RPCMethod("UnaryCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCGRPCStatusCodeOk,
								semconv.RPCMethod("StreamingInputCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCGRPCStatusCodeOk,
								semconv.RPCMethod("StreamingOutputCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCGRPCStatusCodeOk,
								semconv.RPCMethod("FullDuplexCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
						},
					},
				},
			},
			{
				Name:        "rpc.server.request.size",
				Description: "Measures size of RPC request messages (uncompressed).",
				Unit:        "By",
				Data: metricdata.Histogram[int64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.HistogramDataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("EmptyCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Max:          metricdata.NewExtrema(int64(0)),
							Min:          metricdata.NewExtrema(int64(0)),
							Count:        1,
							Sum:          0,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("UnaryCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
							Max:          metricdata.NewExtrema(int64(271840)),
							Min:          metricdata.NewExtrema(int64(271840)),
							Count:        1,
							Sum:          271840,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("StreamingInputCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 2},
							Max:          metricdata.NewExtrema(int64(45912)),
							Min:          metricdata.NewExtrema(int64(12)),
							Count:        4,
							Sum:          74948,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("StreamingOutputCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Max:          metricdata.NewExtrema(int64(21)),
							Min:          metricdata.NewExtrema(int64(21)),
							Count:        1,
							Sum:          21,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("FullDuplexCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
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
				Name:        "rpc.server.response.size",
				Description: "Measures size of RPC response messages (uncompressed).",
				Unit:        "By",
				Data: metricdata.Histogram[int64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.HistogramDataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("EmptyCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Max:          metricdata.NewExtrema(int64(0)),
							Min:          metricdata.NewExtrema(int64(0)),
							Count:        1,
							Sum:          0,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("UnaryCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
							Max:          metricdata.NewExtrema(int64(314167)),
							Min:          metricdata.NewExtrema(int64(314167)),
							Count:        1,
							Sum:          314167,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("StreamingInputCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Max:          metricdata.NewExtrema(int64(4)),
							Min:          metricdata.NewExtrema(int64(4)),
							Count:        1,
							Sum:          4,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("StreamingOutputCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 2},
							Max:          metricdata.NewExtrema(int64(58987)),
							Min:          metricdata.NewExtrema(int64(13)),
							Count:        4,
							Sum:          93082,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("FullDuplexCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
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
			{
				Name:        "rpc.server.requests_per_rpc",
				Description: "Measures the number of messages received per RPC. Should be 1 for all non-streaming RPCs.",
				Unit:        "{count}",
				Data: metricdata.Histogram[int64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.HistogramDataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								semconv.RPCGRPCStatusCodeOk,
								semconv.RPCMethod("EmptyCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Max:          metricdata.NewExtrema(int64(1)),
							Min:          metricdata.NewExtrema(int64(1)),
							Count:        1,
							Sum:          1,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCGRPCStatusCodeOk,
								semconv.RPCMethod("UnaryCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Max:          metricdata.NewExtrema(int64(1)),
							Min:          metricdata.NewExtrema(int64(1)),
							Count:        1,
							Sum:          1,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCGRPCStatusCodeOk,
								semconv.RPCMethod("StreamingInputCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Max:          metricdata.NewExtrema(int64(4)),
							Min:          metricdata.NewExtrema(int64(4)),
							Count:        1,
							Sum:          4,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCGRPCStatusCodeOk,
								semconv.RPCMethod("StreamingOutputCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Max:          metricdata.NewExtrema(int64(1)),
							Min:          metricdata.NewExtrema(int64(1)),
							Count:        1,
							Sum:          1,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCGRPCStatusCodeOk,
								semconv.RPCMethod("FullDuplexCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Max:          metricdata.NewExtrema(int64(4)),
							Min:          metricdata.NewExtrema(int64(4)),
							Count:        1,
							Sum:          4,
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
								semconv.RPCGRPCStatusCodeOk,
								semconv.RPCMethod("EmptyCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Max:          metricdata.NewExtrema(int64(1)),
							Min:          metricdata.NewExtrema(int64(1)),
							Count:        1,
							Sum:          1,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCGRPCStatusCodeOk,
								semconv.RPCMethod("UnaryCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Max:          metricdata.NewExtrema(int64(1)),
							Min:          metricdata.NewExtrema(int64(1)),
							Count:        1,
							Sum:          1,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCGRPCStatusCodeOk,
								semconv.RPCMethod("StreamingInputCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Max:          metricdata.NewExtrema(int64(1)),
							Min:          metricdata.NewExtrema(int64(1)),
							Count:        1,
							Sum:          1,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCGRPCStatusCodeOk,
								semconv.RPCMethod("StreamingOutputCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Max:          metricdata.NewExtrema(int64(4)),
							Min:          metricdata.NewExtrema(int64(4)),
							Count:        1,
							Sum:          4,
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCGRPCStatusCodeOk,
								semconv.RPCMethod("FullDuplexCall"),
								semconv.RPCService("grpc.testing.TestService"),
								semconv.RPCSystemGRPC),
							Bounds:       []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							BucketCounts: []uint64{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							Max:          metricdata.NewExtrema(int64(4)),
							Min:          metricdata.NewExtrema(int64(4)),
							Count:        1,
							Sum:          4,
						},
					},
				},
			},
		},
	}

	metricdatatest.AssertEqual(t, expectedScopeMetric, rm.ScopeMetrics[0], metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue())
}
