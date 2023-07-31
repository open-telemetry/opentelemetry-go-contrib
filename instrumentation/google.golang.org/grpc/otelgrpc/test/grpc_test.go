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
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/interop"
	pb "google.golang.org/grpc/interop/grpc_testing"
	"google.golang.org/grpc/test/bufconn"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

const bufSize = 2048

func doCalls(cOpt []grpc.DialOption, sOpt []grpc.ServerOption) error {
	l := bufconn.Listen(bufSize)
	defer l.Close()

	s := grpc.NewServer(sOpt...)
	pb.RegisterTestServiceServer(s, interop.NewTestServer())
	go func() {
		if err := s.Serve(l); err != nil {
			panic(err)
		}
	}()
	defer s.Stop()

	ctx := context.Background()
	dial := func(context.Context, string) (net.Conn, error) { return l.Dial() }
	conn, err := grpc.DialContext(
		ctx,
		"bufnet",
		append([]grpc.DialOption{
			grpc.WithContextDialer(dial),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		}, cOpt...)...,
	)
	if err != nil {
		return err
	}
	defer conn.Close()
	client := pb.NewTestServiceClient(conn)

	interop.DoEmptyUnaryCall(client)
	interop.DoLargeUnaryCall(client)
	interop.DoClientStreaming(client)
	interop.DoServerStreaming(client)
	interop.DoPingPong(client)

	return nil
}

func TestInterceptors(t *testing.T) {
	clientUnarySR := tracetest.NewSpanRecorder()
	clientUnaryTP := trace.NewTracerProvider(trace.WithSpanProcessor(clientUnarySR))

	clientStreamSR := tracetest.NewSpanRecorder()
	clientStreamTP := trace.NewTracerProvider(trace.WithSpanProcessor(clientStreamSR))

	serverUnarySR := tracetest.NewSpanRecorder()
	serverUnaryTP := trace.NewTracerProvider(trace.WithSpanProcessor(serverUnarySR))
	serverUnaryMetricReader := metric.NewManualReader()
	serverUnaryMP := metric.NewMeterProvider(metric.WithReader(serverUnaryMetricReader))

	serverStreamSR := tracetest.NewSpanRecorder()
	serverStreamTP := trace.NewTracerProvider(trace.WithSpanProcessor(serverStreamSR))

	assert.NoError(t, doCalls(
		[]grpc.DialOption{
			grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor(
				otelgrpc.WithTracerProvider(clientUnaryTP),
				otelgrpc.WithMessageEvents(otelgrpc.ReceivedEvents, otelgrpc.SentEvents),
			)),
			grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor(
				otelgrpc.WithTracerProvider(clientStreamTP),
				otelgrpc.WithMessageEvents(otelgrpc.ReceivedEvents, otelgrpc.SentEvents),
			)),
		},
		[]grpc.ServerOption{
			grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor(
				otelgrpc.WithTracerProvider(serverUnaryTP),
				otelgrpc.WithMeterProvider(serverUnaryMP),
				otelgrpc.WithMessageEvents(otelgrpc.ReceivedEvents, otelgrpc.SentEvents),
			)),
			grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor(
				otelgrpc.WithTracerProvider(serverStreamTP),
				otelgrpc.WithMessageEvents(otelgrpc.ReceivedEvents, otelgrpc.SentEvents),
			)),
		},
	))

	t.Run("UnaryClientSpans", func(t *testing.T) {
		checkUnaryClientSpans(t, clientUnarySR.Ended())
	})

	t.Run("StreamClientSpans", func(t *testing.T) {
		checkStreamClientSpans(t, clientStreamSR.Ended())
	})

	t.Run("UnaryServerSpans", func(t *testing.T) {
		checkUnaryServerSpans(t, serverUnarySR.Ended())
		checkUnaryServerRecords(t, serverUnaryMetricReader)
	})

	t.Run("StreamServerSpans", func(t *testing.T) {
		checkStreamServerSpans(t, serverStreamSR.Ended())
	})
}

func checkUnaryClientSpans(t *testing.T, spans []trace.ReadOnlySpan) {
	require.Len(t, spans, 2)

	emptySpan := spans[0]
	assert.False(t, emptySpan.EndTime().IsZero())
	assert.Equal(t, "grpc.testing.TestService/EmptyCall", emptySpan.Name())
	assertEvents(t, []trace.Event{
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(1),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(1),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
			},
		},
	}, emptySpan.Events())
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethod("EmptyCall"),
		semconv.RPCService("grpc.testing.TestService"),
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
				// largeReqSize from "google.golang.org/grpc/interop" + 12 (overhead).
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(1),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
				// largeRespSize from "google.golang.org/grpc/interop" + 8 (overhead).
			},
		},
	}, largeSpan.Events())
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethod("UnaryCall"),
		semconv.RPCService("grpc.testing.TestService"),
		otelgrpc.RPCSystemGRPC,
		otelgrpc.GRPCStatusCodeKey.Int64(int64(codes.OK)),
	}, largeSpan.Attributes())
}

func checkStreamClientSpans(t *testing.T, spans []trace.ReadOnlySpan) {
	require.Len(t, spans, 3)

	streamInput := spans[0]
	assert.False(t, streamInput.EndTime().IsZero())
	assert.Equal(t, "grpc.testing.TestService/StreamingInputCall", streamInput.Name())
	// sizes from reqSizes in "google.golang.org/grpc/interop".
	assertEvents(t, []trace.Event{
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(1),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(2),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(3),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(4),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
			},
		},
		// client does not record an event for the server response.
	}, streamInput.Events())
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethod("StreamingInputCall"),
		semconv.RPCService("grpc.testing.TestService"),
		otelgrpc.RPCSystemGRPC,
		otelgrpc.GRPCStatusCodeKey.Int64(int64(codes.OK)),
	}, streamInput.Attributes())

	streamOutput := spans[1]
	assert.False(t, streamOutput.EndTime().IsZero())
	assert.Equal(t, "grpc.testing.TestService/StreamingOutputCall", streamOutput.Name())
	// sizes from respSizes in "google.golang.org/grpc/interop".
	assertEvents(t, []trace.Event{
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(1),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(1),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(2),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(3),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(4),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
			},
		},
	}, streamOutput.Events())
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethod("StreamingOutputCall"),
		semconv.RPCService("grpc.testing.TestService"),
		otelgrpc.RPCSystemGRPC,
		otelgrpc.GRPCStatusCodeKey.Int64(int64(codes.OK)),
	}, streamOutput.Attributes())

	pingPong := spans[2]
	assert.False(t, pingPong.EndTime().IsZero())
	assert.Equal(t, "grpc.testing.TestService/FullDuplexCall", pingPong.Name())
	assertEvents(t, []trace.Event{
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(1),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(1),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(2),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(2),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(3),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(3),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(4),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(4),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
			},
		},
	}, pingPong.Events())
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethod("FullDuplexCall"),
		semconv.RPCService("grpc.testing.TestService"),
		otelgrpc.RPCSystemGRPC,
		otelgrpc.GRPCStatusCodeKey.Int64(int64(codes.OK)),
	}, pingPong.Attributes())
}

func checkStreamServerSpans(t *testing.T, spans []trace.ReadOnlySpan) {
	require.Len(t, spans, 3)

	streamInput := spans[0]
	assert.False(t, streamInput.EndTime().IsZero())
	assert.Equal(t, "grpc.testing.TestService/StreamingInputCall", streamInput.Name())
	// sizes from reqSizes in "google.golang.org/grpc/interop".
	assertEvents(t, []trace.Event{
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(1),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(2),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(3),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(4),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(1),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
			},
		},
	}, streamInput.Events())
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethod("StreamingInputCall"),
		semconv.RPCService("grpc.testing.TestService"),
		otelgrpc.RPCSystemGRPC,
		otelgrpc.GRPCStatusCodeKey.Int64(int64(codes.OK)),
	}, streamInput.Attributes())

	streamOutput := spans[1]
	assert.False(t, streamOutput.EndTime().IsZero())
	assert.Equal(t, "grpc.testing.TestService/StreamingOutputCall", streamOutput.Name())
	// sizes from respSizes in "google.golang.org/grpc/interop".
	assertEvents(t, []trace.Event{
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(1),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(1),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(2),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(3),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(4),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
			},
		},
	}, streamOutput.Events())
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethod("StreamingOutputCall"),
		semconv.RPCService("grpc.testing.TestService"),
		otelgrpc.RPCSystemGRPC,
		otelgrpc.GRPCStatusCodeKey.Int64(int64(codes.OK)),
	}, streamOutput.Attributes())

	pingPong := spans[2]
	assert.False(t, pingPong.EndTime().IsZero())
	assert.Equal(t, "grpc.testing.TestService/FullDuplexCall", pingPong.Name())
	assertEvents(t, []trace.Event{
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(1),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(1),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(2),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(2),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(3),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(3),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(4),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(4),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
			},
		},
	}, pingPong.Events())
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethod("FullDuplexCall"),
		semconv.RPCService("grpc.testing.TestService"),
		otelgrpc.RPCSystemGRPC,
		otelgrpc.GRPCStatusCodeKey.Int64(int64(codes.OK)),
	}, pingPong.Attributes())
}

func checkUnaryServerSpans(t *testing.T, spans []trace.ReadOnlySpan) {
	require.Len(t, spans, 2)

	emptySpan := spans[0]
	assert.False(t, emptySpan.EndTime().IsZero())
	assert.Equal(t, "grpc.testing.TestService/EmptyCall", emptySpan.Name())
	assertEvents(t, []trace.Event{
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(1),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(1),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
			},
		},
	}, emptySpan.Events())
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethod("EmptyCall"),
		semconv.RPCService("grpc.testing.TestService"),
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
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
				// largeReqSize from "google.golang.org/grpc/interop" + 12 (overhead).
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(1),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
				// largeRespSize from "google.golang.org/grpc/interop" + 8 (overhead).
			},
		},
	}, largeSpan.Events())
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethod("UnaryCall"),
		semconv.RPCService("grpc.testing.TestService"),
		otelgrpc.RPCSystemGRPC,
		otelgrpc.GRPCStatusCodeKey.Int64(int64(codes.OK)),
	}, largeSpan.Attributes())
}

func assertEvents(t *testing.T, expected, actual []trace.Event) bool {
	if !assert.Len(t, actual, len(expected)) {
		return false
	}

	var failed bool
	for i, e := range expected {
		if !assert.Equal(t, e.Name, actual[i].Name, "names do not match") {
			failed = true
		}
		if !assert.ElementsMatch(t, e.Attributes, actual[i].Attributes, "attributes do not match: %s", e.Name) {
			failed = true
		}
	}

	return !failed
}

func checkUnaryServerRecords(t *testing.T, reader metric.Reader) {
	rm := metricdata.ResourceMetrics{}
	err := reader.Collect(context.Background(), &rm)
	assert.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	require.Len(t, rm.ScopeMetrics[0].Metrics, 1)
	require.IsType(t, rm.ScopeMetrics[0].Metrics[0].Data, metricdata.Histogram[int64]{})
	data := rm.ScopeMetrics[0].Metrics[0].Data.(metricdata.Histogram[int64])

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

func getRPCMethod(attrs []attribute.KeyValue) string {
	for _, kvs := range attrs {
		if kvs.Key == semconv.RPCMethodKey {
			return kvs.Value.AsString()
		}
	}
	return ""
}
