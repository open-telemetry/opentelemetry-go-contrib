// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"context"
	"net"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc/internal/test"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.32.0"

	pb "google.golang.org/grpc/interop/grpc_testing"
)

var wantInstrumentationScope = instrumentation.Scope{
	Name:      "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc",
	SchemaURL: semconv.SchemaURL,
	Version:   otelgrpc.Version(),
}

// newGrpcTest creates a grpc server, starts it, and returns the client, closes everything down during test cleanup.
func newGrpcTest(t testing.TB, listener net.Listener, cOpt []grpc.DialOption, sOpt []grpc.ServerOption) pb.TestServiceClient {
	grpcServer := grpc.NewServer(sOpt...)
	pb.RegisterTestServiceServer(grpcServer, test.NewTestServer())
	errCh := make(chan error)
	go func() {
		errCh <- grpcServer.Serve(listener)
	}()
	t.Cleanup(func() {
		grpcServer.Stop()
		assert.NoError(t, <-errCh)
	})

	cOpt = append(cOpt, grpc.WithTransportCredentials(insecure.NewCredentials()))

	dialAddr := listener.Addr().String()

	if l, ok := listener.(interface{ Dial() (net.Conn, error) }); ok {
		dial := func(context.Context, string) (net.Conn, error) { return l.Dial() }
		cOpt = append(cOpt, grpc.WithContextDialer(dial))
		dialAddr = "passthrough:" + dialAddr
	}

	conn, err := grpc.NewClient(
		dialAddr,
		cOpt...,
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, conn.Close())
	})

	return pb.NewTestServiceClient(conn)
}

func doCalls(ctx context.Context, client pb.TestServiceClient) {
	test.DoEmptyUnaryCall(ctx, client)
	test.DoLargeUnaryCall(ctx, client)
	test.DoClientStreaming(ctx, client)
	test.DoServerStreaming(ctx, client)
	test.DoPingPong(ctx, client)
}

func TestInterceptors(t *testing.T) {
	t.Setenv("OTEL_METRICS_EXEMPLAR_FILTER", "always_off")

	clientStreamSR := tracetest.NewSpanRecorder()
	clientStreamTP := trace.NewTracerProvider(trace.WithSpanProcessor(clientStreamSR))

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "failed to open port")
	client := newGrpcTest(t, listener,
		[]grpc.DialOption{
			//nolint:staticcheck // Interceptors are deprecated and will be removed in the next release.
			grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor(
				otelgrpc.WithTracerProvider(clientStreamTP),
				otelgrpc.WithMessageEvents(otelgrpc.ReceivedEvents, otelgrpc.SentEvents),
			)),
		},
		[]grpc.ServerOption{},
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	doCalls(ctx, client)

	t.Run("StreamClientSpans", func(t *testing.T) {
		checkStreamClientSpans(t, clientStreamSR.Ended(), listener.Addr().String())
	})
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
	// sizes from reqSizes in "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc/test".
	assertEvents(t, []trace.Event{
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(1),
				semconv.RPCMessageTypeKey.String("SENT"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(2),
				semconv.RPCMessageTypeKey.String("SENT"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(3),
				semconv.RPCMessageTypeKey.String("SENT"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(4),
				semconv.RPCMessageTypeKey.String("SENT"),
			},
		},
		// client does not record an event for the server response.
	}, streamInput.Events())
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethod("StreamingInputCall"),
		semconv.RPCService("grpc.testing.TestService"),
		semconv.RPCSystemGRPC,
		semconv.RPCGRPCStatusCodeOk,
		semconv.ServerAddress(host),
		semconv.ServerPort(port),
	}, streamInput.Attributes())

	streamOutput := spans[1]
	assert.False(t, streamOutput.EndTime().IsZero())
	assert.Equal(t, "grpc.testing.TestService/StreamingOutputCall", streamOutput.Name())
	// sizes from respSizes in "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc/test".
	assertEvents(t, []trace.Event{
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(1),
				semconv.RPCMessageTypeKey.String("SENT"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(1),
				semconv.RPCMessageTypeKey.String("RECEIVED"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(2),
				semconv.RPCMessageTypeKey.String("RECEIVED"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(3),
				semconv.RPCMessageTypeKey.String("RECEIVED"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(4),
				semconv.RPCMessageTypeKey.String("RECEIVED"),
			},
		},
	}, streamOutput.Events())
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethod("StreamingOutputCall"),
		semconv.RPCService("grpc.testing.TestService"),
		semconv.RPCSystemGRPC,
		semconv.RPCGRPCStatusCodeOk,
		semconv.ServerAddress(host),
		semconv.ServerPort(port),
	}, streamOutput.Attributes())

	pingPong := spans[2]
	assert.False(t, pingPong.EndTime().IsZero())
	assert.Equal(t, "grpc.testing.TestService/FullDuplexCall", pingPong.Name())
	assertEvents(t, []trace.Event{
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(1),
				semconv.RPCMessageTypeKey.String("SENT"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(1),
				semconv.RPCMessageTypeKey.String("RECEIVED"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(2),
				semconv.RPCMessageTypeKey.String("SENT"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(2),
				semconv.RPCMessageTypeKey.String("RECEIVED"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(3),
				semconv.RPCMessageTypeKey.String("SENT"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(3),
				semconv.RPCMessageTypeKey.String("RECEIVED"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(4),
				semconv.RPCMessageTypeKey.String("SENT"),
			},
		},
		{
			Name: "message",
			Attributes: []attribute.KeyValue{
				semconv.RPCMessageIDKey.Int(4),
				semconv.RPCMessageTypeKey.String("RECEIVED"),
			},
		},
	}, pingPong.Events())
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.RPCMethod("FullDuplexCall"),
		semconv.RPCService("grpc.testing.TestService"),
		semconv.RPCSystemGRPC,
		semconv.RPCGRPCStatusCodeOk,
		semconv.ServerAddress(host),
		semconv.ServerPort(port),
	}, pingPong.Attributes())
}

func assertEvents(t *testing.T, expected, actual []trace.Event) bool { //nolint:unparam
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

func findAttribute(kvs []attribute.KeyValue, key attribute.Key) (attribute.KeyValue, bool) { //nolint:unparam
	for _, kv := range kvs {
		if kv.Key == key {
			return kv, true
		}
	}
	return attribute.KeyValue{}, false
}
