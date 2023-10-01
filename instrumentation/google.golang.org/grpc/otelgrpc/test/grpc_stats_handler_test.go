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

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
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

	assert.NoError(t, doCalls(
		[]grpc.DialOption{
			grpc.WithStatsHandler(otelgrpc.NewClientHandler(otelgrpc.WithTracerProvider(clientTP), otelgrpc.WithMeterProvider(clientMP))),
		},
		[]grpc.ServerOption{
			grpc.StatsHandler(otelgrpc.NewServerHandler(otelgrpc.WithTracerProvider(serverTP), otelgrpc.WithMeterProvider(serverMP))),
		},
	))

	t.Run("Client", func(t *testing.T) {
		checkClientSpans(t, clientSR.Ended())
		checkClientRecords(t, clientMetricReader)
	})

	t.Run("Server", func(t *testing.T) {
		checkServerSpans(t, serverSR.Ended())
		checkServerRecords(t, serverMetricReader)
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

func checkServerRecords(t *testing.T, reader metric.Reader) {
	rm := metricdata.ResourceMetrics{}
	err := reader.Collect(context.Background(), &rm)
	assert.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	require.Len(t, rm.ScopeMetrics[0].Metrics, 5)
	for _, m := range rm.ScopeMetrics[0].Metrics {
		require.IsType(t, m.Data, metricdata.Histogram[int64]{})
		data := m.Data.(metricdata.Histogram[int64])
		for _, dpt := range data.DataPoints {
			attr := dpt.Attributes.ToSlice()
			method := getRPCMethod(attr)
			assert.NotEmpty(t, method)
			assert.ElementsMatch(t, []attribute.KeyValue{
				semconv.RPCMethod(method),
				semconv.RPCService("grpc.testing.TestService"),
				otelgrpc.RPCSystemGRPC,
				otelgrpc.GRPCStatusCodeKey.Int64(int64(codes.OK)),
			}, attr)
		}
	}
}

func checkClientRecords(t *testing.T, reader metric.Reader) {
	rm := metricdata.ResourceMetrics{}
	err := reader.Collect(context.Background(), &rm)
	assert.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	require.Len(t, rm.ScopeMetrics[0].Metrics, 5)
	for _, m := range rm.ScopeMetrics[0].Metrics {
		require.IsType(t, m.Data, metricdata.Histogram[int64]{})
		data := m.Data.(metricdata.Histogram[int64])
		for _, dpt := range data.DataPoints {
			attr := dpt.Attributes.ToSlice()
			method := getRPCMethod(attr)
			assert.NotEmpty(t, method)
			assert.ElementsMatch(t, []attribute.KeyValue{
				semconv.RPCMethod(method),
				semconv.RPCService("grpc.testing.TestService"),
				otelgrpc.RPCSystemGRPC,
				otelgrpc.GRPCStatusCodeKey.Int64(int64(codes.OK)),
			}, attr)
		}
	}
}
