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
	"fmt"
	"net"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/interop"
	pb "google.golang.org/grpc/interop/grpc_testing"

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

var wantInstrumentationScope = instrumentation.Scope{
	Name:      "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc",
	SchemaURL: "https://opentelemetry.io/schemas/1.17.0",
	Version:   otelgrpc.Version(),
}

const bufSize = 2048

// newGrpcTest creats a grpc server, starts it, and executes all the calls, closes everything down.
func newGrpcTest(listener net.Listener, cOpt []grpc.DialOption, sOpt []grpc.ServerOption) error {
	grpcServer := grpc.NewServer(sOpt...)
	pb.RegisterTestServiceServer(grpcServer, interop.NewTestServer())
	errCh := make(chan error)
	go func() {
		errCh <- grpcServer.Serve(listener)
	}()
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
	if err != nil {
		return err
	}
	client := pb.NewTestServiceClient(conn)

	doCalls(client)

	conn.Close()
	grpcServer.Stop()

	return <-errCh
}

func doCalls(client pb.TestServiceClient) {
	interop.DoEmptyUnaryCall(client)
	interop.DoLargeUnaryCall(client)
	interop.DoClientStreaming(client)
	interop.DoServerStreaming(client)
	interop.DoPingPong(client)
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
	err = newGrpcTest(listener, []grpc.DialOption{
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
	)
	require.NoError(t, err)

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
		semconv.NetSockPeerAddr(host),
		semconv.NetSockPeerPort(port),
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

	// TODO: Remove these #4322
	address, ok := findScopeMetricAttribute(rm.ScopeMetrics[0], semconv.NetSockPeerAddrKey)
	assert.True(t, ok)
	port, ok := findScopeMetricAttribute(rm.ScopeMetrics[0], semconv.NetSockPeerPortKey)
	assert.True(t, ok)

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
								address,
								port,
							),
						},
						{
							Attributes: attribute.NewSet(
								semconv.RPCMethod("UnaryCall"),
								semconv.RPCService("grpc.testing.TestService"),
								otelgrpc.RPCSystemGRPC,
								otelgrpc.GRPCStatusCodeKey.Int64(int64(codes.OK)),
								address,
								port,
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

func findScopeMetricAttribute(sm metricdata.ScopeMetrics, key attribute.Key) (attribute.KeyValue, bool) {
	for _, m := range sm.Metrics {
		// This only needs to cover data types used by the instrumentation.
		switch d := m.Data.(type) {
		case metricdata.Histogram[int64]:
			for _, dp := range d.DataPoints {
				if kv, ok := findAttribute(dp.Attributes.ToSlice(), key); ok {
					return kv, true
				}
			}
		case metricdata.Histogram[float64]:
			for _, dp := range d.DataPoints {
				if kv, ok := findAttribute(dp.Attributes.ToSlice(), key); ok {
					return kv, true
				}
			}
		default:
			panic(fmt.Sprintf("unexpected data type %T - name %s", d, m.Name))
		}
	}
	return attribute.KeyValue{}, false
}
