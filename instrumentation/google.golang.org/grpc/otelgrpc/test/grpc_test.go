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
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"

	pb "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc/grpc_testing"
)

var wantInstrumentationScope = instrumentation.Scope{
	Name:      "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc",
	SchemaURL: "https://opentelemetry.io/schemas/1.17.0",
	Version:   otelgrpc.Version(),
}

// newGrpcTest creats a grpc server, starts it, and returns the client, closes everything down during test cleanup.
func newGrpcTest(t testing.TB, listener net.Listener, cOpt []grpc.DialOption, sOpt []grpc.ServerOption) pb.TestServiceClient {
	grpcServer := grpc.NewServer(sOpt...)
	pb.RegisterTestServiceServer(grpcServer, NewTestServer())
	errCh := make(chan error)
	go func() {
		errCh <- grpcServer.Serve(listener)
	}()
	t.Cleanup(func() {
		grpcServer.Stop()
		assert.NoError(t, <-errCh)
	})
	ctx := context.Background()

	cOpt = append(cOpt, grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials()))

	if l, ok := listener.(interface{ Dial() (net.Conn, error) }); ok {
		dial := func(context.Context, string) (net.Conn, error) { return l.Dial() }
		cOpt = append(cOpt, grpc.WithContextDialer(dial))
	}

	conn, err := grpc.DialContext(
		ctx,
		listener.Addr().String(),
		cOpt...,
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, conn.Close())
	})

	return pb.NewTestServiceClient(conn)
}

func doCalls(ctx context.Context, client pb.TestServiceClient) {
	DoEmptyUnaryCall(ctx, client)
	DoLargeUnaryCall(ctx, client)
	DoClientStreaming(ctx, client)
	DoServerStreaming(ctx, client)
	DoPingPong(ctx, client)
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

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "failed to open port")
	client := newGrpcTest(t, listener,
		[]grpc.DialOption{
			//nolint:staticcheck // Interceptors are deprecated and will be removed in the next release.
			grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor(
				otelgrpc.WithTracerProvider(clientUnaryTP),
				otelgrpc.WithMessageEvents(otelgrpc.ReceivedEvents, otelgrpc.SentEvents),
			)),
			//nolint:staticcheck // Interceptors are deprecated and will be removed in the next release.
			grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor(
				otelgrpc.WithTracerProvider(clientStreamTP),
				otelgrpc.WithMessageEvents(otelgrpc.ReceivedEvents, otelgrpc.SentEvents),
			)),
		},
		[]grpc.ServerOption{
			//nolint:staticcheck // Interceptors are deprecated and will be removed in the next release.
			grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor(
				otelgrpc.WithTracerProvider(serverUnaryTP),
				otelgrpc.WithMeterProvider(serverUnaryMP),
				otelgrpc.WithMessageEvents(otelgrpc.ReceivedEvents, otelgrpc.SentEvents),
			)),
			//nolint:staticcheck // Interceptors are deprecated and will be removed in the next release.
			grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor(
				otelgrpc.WithTracerProvider(serverStreamTP),
				otelgrpc.WithMessageEvents(otelgrpc.ReceivedEvents, otelgrpc.SentEvents),
			)),
		},
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	doCalls(ctx, client)

	t.Run("UnaryClientSpans", func(t *testing.T) {
		checkUnaryClientSpans(t, clientUnarySR.Ended(), listener.Addr().String())
	})

	t.Run("StreamClientSpans", func(t *testing.T) {
		checkStreamClientSpans(t, clientStreamSR.Ended(), listener.Addr().String())
	})

	t.Run("UnaryServerSpans", func(t *testing.T) {
		checkUnaryServerSpans(t, serverUnarySR.Ended())
		checkUnaryServerRecords(t, serverUnaryMetricReader)
	})

	t.Run("StreamServerSpans", func(t *testing.T) {
		checkStreamServerSpans(t, serverStreamSR.Ended())
	})
}

func checkUnaryClientSpans(t *testing.T, spans []trace.ReadOnlySpan, addr string) {
	require.Len(t, spans, 2)

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
		semconv.NetSockPeerAddr(host),
		semconv.NetSockPeerPort(port),
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
				// largeReqSize from "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc/interop" + 12 (overhead).
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(1),
				otelgrpc.RPCMessageTypeKey.String("RECEIVED"),
				// largeRespSize from "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc/interop" + 8 (overhead).
			},
		},
	}, largeSpan.Events())
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethod("UnaryCall"),
		semconv.RPCService("grpc.testing.TestService"),
		otelgrpc.RPCSystemGRPC,
		otelgrpc.GRPCStatusCodeKey.Int64(int64(codes.OK)),
		semconv.NetSockPeerAddr(host),
		semconv.NetSockPeerPort(port),
	}, largeSpan.Attributes())
}

func checkStreamClientSpans(t *testing.T, spans []trace.ReadOnlySpan, addr string) {
	require.Len(t, spans, 3)

	host, p, err := net.SplitHostPort(addr)
	require.NoError(t, err)
	port, err := strconv.Atoi(p)
	require.NoError(t, err)

	streamInput := spans[0]
	assert.False(t, streamInput.EndTime().IsZero())
	assert.Equal(t, "grpc.testing.TestService/StreamingInputCall", streamInput.Name())
	// sizes from reqSizes in "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc/interop".
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
		semconv.NetSockPeerAddr(host),
		semconv.NetSockPeerPort(port),
	}, streamInput.Attributes())

	streamOutput := spans[1]
	assert.False(t, streamOutput.EndTime().IsZero())
	assert.Equal(t, "grpc.testing.TestService/StreamingOutputCall", streamOutput.Name())
	// sizes from respSizes in "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc/interop".
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
		semconv.NetSockPeerAddr(host),
		semconv.NetSockPeerPort(port),
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
		semconv.NetSockPeerAddr(host),
		semconv.NetSockPeerPort(port),
	}, pingPong.Attributes())
}

func checkStreamServerSpans(t *testing.T, spans []trace.ReadOnlySpan) {
	require.Len(t, spans, 3)

	streamInput := spans[0]
	assert.False(t, streamInput.EndTime().IsZero())
	assert.Equal(t, "grpc.testing.TestService/StreamingInputCall", streamInput.Name())
	// sizes from reqSizes in "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc/interop".
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
	port, ok := findAttribute(streamInput.Attributes(), semconv.NetSockPeerPortKey)
	assert.True(t, ok)

	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethod("StreamingInputCall"),
		semconv.RPCService("grpc.testing.TestService"),
		otelgrpc.RPCSystemGRPC,
		otelgrpc.GRPCStatusCodeKey.Int64(int64(codes.OK)),
		semconv.NetSockPeerAddr("127.0.0.1"),
		port,
	}, streamInput.Attributes())

	streamOutput := spans[1]
	assert.False(t, streamOutput.EndTime().IsZero())
	assert.Equal(t, "grpc.testing.TestService/StreamingOutputCall", streamOutput.Name())
	// sizes from respSizes in "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc/interop".
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

	port, ok = findAttribute(streamOutput.Attributes(), semconv.NetSockPeerPortKey)
	assert.True(t, ok)

	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethod("StreamingOutputCall"),
		semconv.RPCService("grpc.testing.TestService"),
		otelgrpc.RPCSystemGRPC,
		otelgrpc.GRPCStatusCodeKey.Int64(int64(codes.OK)),
		semconv.NetSockPeerAddr("127.0.0.1"),
		port,
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
	port, ok = findAttribute(pingPong.Attributes(), semconv.NetSockPeerPortKey)
	assert.True(t, ok)
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethod("FullDuplexCall"),
		semconv.RPCService("grpc.testing.TestService"),
		otelgrpc.RPCSystemGRPC,
		otelgrpc.GRPCStatusCodeKey.Int64(int64(codes.OK)),
		semconv.NetSockPeerAddr("127.0.0.1"),
		port,
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

	port, ok := findAttribute(emptySpan.Attributes(), semconv.NetSockPeerPortKey)
	assert.True(t, ok)
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethod("EmptyCall"),
		semconv.RPCService("grpc.testing.TestService"),
		otelgrpc.RPCSystemGRPC,
		otelgrpc.GRPCStatusCodeKey.Int64(int64(codes.OK)),
		semconv.NetSockPeerAddr("127.0.0.1"),
		port,
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
				// largeReqSize from "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc/interop" + 12 (overhead).
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				otelgrpc.RPCMessageIDKey.Int(1),
				otelgrpc.RPCMessageTypeKey.String("SENT"),
				// largeRespSize from "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc/interop" + 8 (overhead).
			},
		},
	}, largeSpan.Events())

	port, ok = findAttribute(largeSpan.Attributes(), semconv.NetSockPeerPortKey)
	assert.True(t, ok)
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethod("UnaryCall"),
		semconv.RPCService("grpc.testing.TestService"),
		otelgrpc.RPCSystemGRPC,
		otelgrpc.GRPCStatusCodeKey.Int64(int64(codes.OK)),
		semconv.NetSockPeerAddr("127.0.0.1"),
		port,
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
								semconv.RPCMethod("EmptyCall"),
								semconv.RPCService("grpc.testing.TestService"),
								otelgrpc.RPCSystemGRPC,
								otelgrpc.GRPCStatusCodeKey.Int64(int64(codes.OK)),
							),
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("UnaryCall"),
								semconv.RPCService("grpc.testing.TestService"),
								otelgrpc.RPCSystemGRPC,
								otelgrpc.GRPCStatusCodeKey.Int64(int64(codes.OK)),
							),
						},
					},
				},
			},
		},
	}

	metricdatatest.AssertEqual(t, want, rm.ScopeMetrics[0], metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue())
}

func findAttribute(kvs []attribute.KeyValue, key attribute.Key) (attribute.KeyValue, bool) {
	for _, kv := range kvs {
		if kv.Key == key {
			return kv, true
		}
	}
	return attribute.KeyValue{}, false
}
