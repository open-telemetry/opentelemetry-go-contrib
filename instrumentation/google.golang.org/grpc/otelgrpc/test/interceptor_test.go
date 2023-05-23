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
	"errors"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc/internal"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"
	grpc_codes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/interop/grpc_testing"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func getSpanFromRecorder(sr *tracetest.SpanRecorder, name string) (trace.ReadOnlySpan, bool) {
	for _, s := range sr.Ended() {
		if s.Name() == name {
			return s, true
		}
	}
	return nil, false
}

type mockUICInvoker struct {
	ctx context.Context
}

func (mcuici *mockUICInvoker) invoker(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
	mcuici.ctx = ctx

	// if method contains error name, mock error return
	if strings.Contains(method, "error") {
		return status.Error(grpc_codes.Internal, "internal error")
	}

	return nil
}

func ctxDialer() func(context.Context, string) (net.Conn, error) {
	l := bufconn.Listen(0)
	return func(ctx context.Context, _ string) (net.Conn, error) {
		return l.DialContext(ctx)
	}
}

func TestUnaryClientInterceptor(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	clientConn, err := grpc.DialContext(ctx, "fake:8906",
		grpc.WithContextDialer(ctxDialer()),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("failed to create client connection: %v", err)
	}
	defer clientConn.Close()

	sr := tracetest.NewSpanRecorder()
	tp := trace.NewTracerProvider(trace.WithSpanProcessor(sr))
	unaryInterceptor := otelgrpc.UnaryClientInterceptor(otelgrpc.WithTracerProvider(tp))

	req := &grpc_testing.SimpleRequest{}
	reply := &grpc_testing.SimpleResponse{}
	uniInterceptorInvoker := &mockUICInvoker{}

	checks := []struct {
		method           string
		name             string
		expectedSpanCode codes.Code
		expectedAttr     []attribute.KeyValue
		eventsAttr       []map[attribute.Key]attribute.Value
		expectErr        bool
	}{
		{
			method: "/github.com.serviceName/bar",
			name:   "github.com.serviceName/bar",
			expectedAttr: []attribute.KeyValue{
				semconv.RPCSystemGRPC,
				semconv.RPCService("github.com.serviceName"),
				semconv.RPCMethod("bar"),
				otelgrpc.GRPCStatusCodeKey.Int64(0),
				semconv.NetPeerName("fake"),
				semconv.NetPeerPort(8906),
			},
			eventsAttr: []map[attribute.Key]attribute.Value{
				{
					otelgrpc.RPCMessageTypeKey: attribute.StringValue("SENT"),
					otelgrpc.RPCMessageIDKey:   attribute.IntValue(1),
				},
				{
					otelgrpc.RPCMessageTypeKey: attribute.StringValue("RECEIVED"),
					otelgrpc.RPCMessageIDKey:   attribute.IntValue(1),
				},
			},
		},
		{
			method: "/serviceName/bar",
			name:   "serviceName/bar",
			expectedAttr: []attribute.KeyValue{
				semconv.RPCSystemGRPC,
				semconv.RPCService("serviceName"),
				semconv.RPCMethod("bar"),
				otelgrpc.GRPCStatusCodeKey.Int64(0),
				semconv.NetPeerName("fake"),
				semconv.NetPeerPort(8906),
			},
			eventsAttr: []map[attribute.Key]attribute.Value{
				{
					otelgrpc.RPCMessageTypeKey: attribute.StringValue("SENT"),
					otelgrpc.RPCMessageIDKey:   attribute.IntValue(1),
				},
				{
					otelgrpc.RPCMessageTypeKey: attribute.StringValue("RECEIVED"),
					otelgrpc.RPCMessageIDKey:   attribute.IntValue(1),
				},
			},
		},
		{
			method: "serviceName/bar",
			name:   "serviceName/bar",
			expectedAttr: []attribute.KeyValue{
				semconv.RPCSystemGRPC,
				semconv.RPCService("serviceName"),
				semconv.RPCMethod("bar"),
				otelgrpc.GRPCStatusCodeKey.Int64(int64(grpc_codes.OK)),
				semconv.NetPeerName("fake"),
				semconv.NetPeerPort(8906),
			},
			eventsAttr: []map[attribute.Key]attribute.Value{
				{
					otelgrpc.RPCMessageTypeKey: attribute.StringValue("SENT"),
					otelgrpc.RPCMessageIDKey:   attribute.IntValue(1),
				},
				{
					otelgrpc.RPCMessageTypeKey: attribute.StringValue("RECEIVED"),
					otelgrpc.RPCMessageIDKey:   attribute.IntValue(1),
				},
			},
		},
		{
			method:           "serviceName/bar_error",
			name:             "serviceName/bar_error",
			expectedSpanCode: codes.Error,
			expectedAttr: []attribute.KeyValue{
				semconv.RPCSystemGRPC,
				semconv.RPCService("serviceName"),
				semconv.RPCMethod("bar_error"),
				otelgrpc.GRPCStatusCodeKey.Int64(int64(grpc_codes.Internal)),
				semconv.NetPeerName("fake"),
				semconv.NetPeerPort(8906),
			},
			eventsAttr: []map[attribute.Key]attribute.Value{
				{
					otelgrpc.RPCMessageTypeKey: attribute.StringValue("SENT"),
					otelgrpc.RPCMessageIDKey:   attribute.IntValue(1),
				},
				{
					otelgrpc.RPCMessageTypeKey: attribute.StringValue("RECEIVED"),
					otelgrpc.RPCMessageIDKey:   attribute.IntValue(1),
				},
			},
			expectErr: true,
		},
		{
			method: "invalidName",
			name:   "invalidName",
			expectedAttr: []attribute.KeyValue{
				semconv.RPCSystemGRPC,
				otelgrpc.GRPCStatusCodeKey.Int64(0),
				semconv.NetPeerName("fake"),
				semconv.NetPeerPort(8906),
			},
			eventsAttr: []map[attribute.Key]attribute.Value{
				{
					otelgrpc.RPCMessageTypeKey: attribute.StringValue("SENT"),
					otelgrpc.RPCMessageIDKey:   attribute.IntValue(1),
				},
				{
					otelgrpc.RPCMessageTypeKey: attribute.StringValue("RECEIVED"),
					otelgrpc.RPCMessageIDKey:   attribute.IntValue(1),
				},
			},
		},
		{
			method: "/github.com.foo.serviceName_123/method",
			name:   "github.com.foo.serviceName_123/method",
			expectedAttr: []attribute.KeyValue{
				semconv.RPCSystemGRPC,
				otelgrpc.GRPCStatusCodeKey.Int64(0),
				semconv.RPCService("github.com.foo.serviceName_123"),
				semconv.RPCMethod("method"),
				semconv.NetPeerName("fake"),
				semconv.NetPeerPort(8906),
			},
			eventsAttr: []map[attribute.Key]attribute.Value{
				{
					otelgrpc.RPCMessageTypeKey: attribute.StringValue("SENT"),
					otelgrpc.RPCMessageIDKey:   attribute.IntValue(1),
				},
				{
					otelgrpc.RPCMessageTypeKey: attribute.StringValue("RECEIVED"),
					otelgrpc.RPCMessageIDKey:   attribute.IntValue(1),
				},
			},
		},
	}

	for _, check := range checks {
		err := unaryInterceptor(context.Background(), check.method, req, reply, clientConn, uniInterceptorInvoker.invoker)
		if check.expectErr {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
		span, ok := getSpanFromRecorder(sr, check.name)
		if !assert.True(t, ok, "missing span %q", check.name) {
			continue
		}
		assert.Equal(t, check.expectedSpanCode, span.Status().Code)
		assert.ElementsMatch(t, check.expectedAttr, span.Attributes())
		assert.Equal(t, check.eventsAttr, eventAttrMap(span.Events()))
	}
}

func eventAttrMap(events []trace.Event) []map[attribute.Key]attribute.Value {
	maps := make([]map[attribute.Key]attribute.Value, len(events))
	for i, event := range events {
		maps[i] = make(map[attribute.Key]attribute.Value, len(event.Attributes))
		for _, a := range event.Attributes {
			maps[i][a.Key] = a.Value
		}
	}
	return maps
}

type mockClientStream struct {
	Desc *grpc.StreamDesc
	Ctx  context.Context
	msgs []grpc_testing.SimpleResponse
}

func (mockClientStream) SendMsg(m interface{}) error { return nil }
func (c *mockClientStream) RecvMsg(m interface{}) error {
	if len(c.msgs) == 0 {
		return io.EOF
	}
	c.msgs = c.msgs[1:]
	return nil
}
func (mockClientStream) CloseSend() error             { return nil }
func (c mockClientStream) Context() context.Context   { return c.Ctx }
func (mockClientStream) Header() (metadata.MD, error) { return nil, nil }
func (mockClientStream) Trailer() metadata.MD         { return nil }

type clientStreamOpts struct {
	NumRecvMsgs          int
	DisableServerStreams bool
}

func newMockClientStream(opts clientStreamOpts) *mockClientStream {
	var msgs []grpc_testing.SimpleResponse
	for i := 0; i < opts.NumRecvMsgs; i++ {
		msgs = append(msgs, grpc_testing.SimpleResponse{})
	}
	return &mockClientStream{msgs: msgs}
}

func createInterceptedStreamClient(t *testing.T, method string, opts clientStreamOpts) (grpc.ClientStream, *tracetest.SpanRecorder) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	mockStream := newMockClientStream(opts)
	clientConn, err := grpc.DialContext(ctx, "fake:8906",
		grpc.WithContextDialer(ctxDialer()),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("failed to create client connection: %v", err)
	}
	defer clientConn.Close()

	// tracer
	sr := tracetest.NewSpanRecorder()
	tp := trace.NewTracerProvider(trace.WithSpanProcessor(sr))
	streamCI := otelgrpc.StreamClientInterceptor(otelgrpc.WithTracerProvider(tp))

	streamClient, err := streamCI(
		context.Background(),
		&grpc.StreamDesc{ServerStreams: !opts.DisableServerStreams},
		clientConn,
		method,
		func(ctx context.Context,
			desc *grpc.StreamDesc,
			cc *grpc.ClientConn,
			method string,
			opts ...grpc.CallOption) (grpc.ClientStream, error) {
			mockStream.Desc = desc
			mockStream.Ctx = ctx
			return mockStream, nil
		},
	)
	require.NoError(t, err, "initialize grpc stream client")
	return streamClient, sr
}

func TestStreamClientInterceptorOnBIDIStream(t *testing.T) {
	defer goleak.VerifyNone(t)

	method := "/github.com.serviceName/bar"
	name := "github.com.serviceName/bar"
	streamClient, sr := createInterceptedStreamClient(t, method, clientStreamOpts{NumRecvMsgs: 10})
	_, ok := getSpanFromRecorder(sr, name)
	require.False(t, ok, "span should not end while stream is open")

	req := &grpc_testing.SimpleRequest{}
	reply := &grpc_testing.SimpleResponse{}

	// send and receive fake data
	for i := 0; i < 10; i++ {
		_ = streamClient.SendMsg(req)
		_ = streamClient.RecvMsg(reply)
	}

	// The stream has been exhausted so next read should get a EOF and the stream should be considered closed.
	err := streamClient.RecvMsg(reply)
	require.Equal(t, io.EOF, err)

	// added retry because span end is called in separate go routine
	var span trace.ReadOnlySpan
	for retry := 0; retry < 5; retry++ {
		span, ok = getSpanFromRecorder(sr, name)
		if ok {
			break
		}
		time.Sleep(time.Second * 1)
	}
	require.True(t, ok, "missing span %s", name)

	expectedAttr := []attribute.KeyValue{
		semconv.RPCSystemGRPC,
		otelgrpc.GRPCStatusCodeKey.Int64(int64(grpc_codes.OK)),
		semconv.RPCService("github.com.serviceName"),
		semconv.RPCMethod("bar"),
		semconv.NetPeerName("fake"),
		semconv.NetPeerPort(8906),
	}
	assert.ElementsMatch(t, expectedAttr, span.Attributes())

	events := span.Events()
	require.Len(t, events, 20)
	for i := 0; i < 20; i += 2 {
		msgID := i/2 + 1
		validate := func(eventName string, attrs []attribute.KeyValue) {
			for _, kv := range attrs {
				k, v := kv.Key, kv.Value
				if k == otelgrpc.RPCMessageTypeKey && v.AsString() != eventName {
					t.Errorf("invalid event on index: %d expecting %s event, receive %s event", i, eventName, v.AsString())
				}
				if k == otelgrpc.RPCMessageIDKey && v != attribute.IntValue(msgID) {
					t.Errorf("invalid id for message event expected %d received %d", msgID, v.AsInt64())
				}
			}
		}
		validate("SENT", events[i].Attributes)
		validate("RECEIVED", events[i+1].Attributes)
	}

	// ensure CloseSend can be subsequently called
	_ = streamClient.CloseSend()
}

func TestStreamClientInterceptorOnUnidirectionalClientServerStream(t *testing.T) {
	defer goleak.VerifyNone(t)

	method := "/github.com.serviceName/bar"
	name := "github.com.serviceName/bar"
	opts := clientStreamOpts{NumRecvMsgs: 1, DisableServerStreams: true}
	streamClient, sr := createInterceptedStreamClient(t, method, opts)
	_, ok := getSpanFromRecorder(sr, name)
	require.False(t, ok, "span should not end while stream is open")

	req := &grpc_testing.SimpleRequest{}
	reply := &grpc_testing.SimpleResponse{}

	// send fake data
	for i := 0; i < 10; i++ {
		_ = streamClient.SendMsg(req)
	}

	// A real user would call CloseAndRecv() on the generated client which would generate a sequence of CloseSend()
	// and RecvMsg() calls.
	_ = streamClient.CloseSend()
	err := streamClient.RecvMsg(reply)
	require.Nil(t, err)

	// added retry because span end is called in separate go routine
	var span trace.ReadOnlySpan
	for retry := 0; retry < 5; retry++ {
		span, ok = getSpanFromRecorder(sr, name)
		if ok {
			break
		}
		time.Sleep(time.Second * 1)
	}
	require.True(t, ok, "missing span %s", name)

	expectedAttr := []attribute.KeyValue{
		semconv.RPCSystemGRPC,
		otelgrpc.GRPCStatusCodeKey.Int64(int64(grpc_codes.OK)),
		semconv.RPCService("github.com.serviceName"),
		semconv.RPCMethod("bar"),
		semconv.NetPeerName("fake"),
		semconv.NetPeerPort(8906),
	}
	assert.ElementsMatch(t, expectedAttr, span.Attributes())

	// Note that there's no "RECEIVED" event generated for the server response. This is a bug.
	events := span.Events()
	require.Len(t, events, 10)
	for i := 0; i < 10; i++ {
		msgID := i + 1
		validate := func(eventName string, attrs []attribute.KeyValue) {
			for _, kv := range attrs {
				k, v := kv.Key, kv.Value
				if k == otelgrpc.RPCMessageTypeKey && v.AsString() != eventName {
					t.Errorf("invalid event on index: %d expecting %s event, receive %s event", i, eventName, v.AsString())
				}
				if k == otelgrpc.RPCMessageIDKey && v != attribute.IntValue(msgID) {
					t.Errorf("invalid id for message event expected %d received %d", msgID, v.AsInt64())
				}
			}
		}
		validate("SENT", events[i].Attributes)
	}
}

// TestStreamClientInterceptorCancelContext tests a cancel context situation.
// There should be no goleaks.
func TestStreamClientInterceptorCancelContext(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	clientConn, err := grpc.DialContext(ctx, "fake:8906",
		grpc.WithContextDialer(ctxDialer()),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("failed to create client connection: %v", err)
	}
	defer clientConn.Close()

	// tracer
	sr := tracetest.NewSpanRecorder()
	tp := trace.NewTracerProvider(trace.WithSpanProcessor(sr))
	streamCI := otelgrpc.StreamClientInterceptor(otelgrpc.WithTracerProvider(tp))

	var mockClStr *mockClientStream
	method := "/github.com.serviceName/bar"
	name := "github.com.serviceName/bar"

	// create a context with cancel
	cancelCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	streamClient, err := streamCI(
		cancelCtx,
		&grpc.StreamDesc{ServerStreams: true},
		clientConn,
		method,
		func(ctx context.Context,
			desc *grpc.StreamDesc,
			cc *grpc.ClientConn,
			method string,
			opts ...grpc.CallOption) (grpc.ClientStream, error) {
			mockClStr = &mockClientStream{Desc: desc, Ctx: ctx}
			return mockClStr, nil
		},
	)
	require.NoError(t, err, "initialize grpc stream client")
	_, ok := getSpanFromRecorder(sr, name)
	require.False(t, ok, "span should not ended while stream is open")

	req := &grpc_testing.SimpleRequest{}
	reply := &grpc_testing.SimpleResponse{}

	// send and receive fake data
	for i := 0; i < 10; i++ {
		_ = streamClient.SendMsg(req)
		_ = streamClient.RecvMsg(reply)
	}

	// close client stream
	_ = streamClient.CloseSend()
}

// TestStreamClientInterceptorWithNoClientConn tests a situation where clientConn is nil.
// There should be no NPEs.
func TestStreamClientInterceptorWithNoClientConn(t *testing.T) {
	defer goleak.VerifyNone(t)

	// tracer
	sr := tracetest.NewSpanRecorder()
	tp := trace.NewTracerProvider(trace.WithSpanProcessor(sr))
	streamCI := otelgrpc.StreamClientInterceptor(otelgrpc.WithTracerProvider(tp))

	var mockClStr *mockClientStream
	method := "/github.com.serviceName/bar"
	name := "github.com.serviceName/bar"

	// create a context with cancel
	cancelCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	streamClient, err := streamCI(
		cancelCtx,
		&grpc.StreamDesc{ServerStreams: true},
		nil, // no clientConn
		method,
		func(ctx context.Context,
			desc *grpc.StreamDesc,
			cc *grpc.ClientConn,
			method string,
			opts ...grpc.CallOption) (grpc.ClientStream, error) {
			mockClStr = &mockClientStream{Desc: desc, Ctx: ctx}
			return mockClStr, nil
		},
	)
	require.NoError(t, err, "initialize grpc stream client")
	_, ok := getSpanFromRecorder(sr, name)
	require.False(t, ok, "span should not ended while stream is open")

	req := &grpc_testing.SimpleRequest{}
	reply := &grpc_testing.SimpleResponse{}

	// send and receive fake data
	for i := 0; i < 10; i++ {
		_ = streamClient.SendMsg(req)
		_ = streamClient.RecvMsg(reply)
	}

	// close client stream
	_ = streamClient.CloseSend()
}

// TestStreamClientInterceptorWithError tests a situation that streamer returns an error.
func TestStreamClientInterceptorWithError(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	clientConn, err := grpc.DialContext(ctx, "fake:8906",
		grpc.WithContextDialer(ctxDialer()),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("failed to create client connection: %v", err)
	}
	defer clientConn.Close()

	// tracer
	sr := tracetest.NewSpanRecorder()
	tp := trace.NewTracerProvider(trace.WithSpanProcessor(sr))
	streamCI := otelgrpc.StreamClientInterceptor(otelgrpc.WithTracerProvider(tp))

	var mockClStr *mockClientStream
	method := "/github.com.serviceName/bar"
	name := "github.com.serviceName/bar"

	streamClient, err := streamCI(
		context.Background(),
		&grpc.StreamDesc{ServerStreams: true},
		clientConn,
		method,
		func(ctx context.Context,
			desc *grpc.StreamDesc,
			cc *grpc.ClientConn,
			method string,
			opts ...grpc.CallOption) (grpc.ClientStream, error) {
			mockClStr = &mockClientStream{Desc: desc, Ctx: ctx}
			return mockClStr, errors.New("test")
		},
	)
	require.Error(t, err, "initialize grpc stream client")
	assert.IsType(t, &mockClientStream{}, streamClient)

	span, ok := getSpanFromRecorder(sr, name)
	require.True(t, ok, "missing span %s", name)

	expectedAttr := []attribute.KeyValue{
		semconv.RPCSystemGRPC,
		otelgrpc.GRPCStatusCodeKey.Int64(int64(grpc_codes.Unknown)),
		semconv.RPCService("github.com.serviceName"),
		semconv.RPCMethod("bar"),
		semconv.NetPeerName("fake"),
		semconv.NetPeerPort(8906),
	}
	assert.ElementsMatch(t, expectedAttr, span.Attributes())
	assert.Equal(t, codes.Error, span.Status().Code)
}

var serverChecks = []struct {
	grpcCode                  grpc_codes.Code
	wantSpanCode              codes.Code
	wantSpanStatusDescription string
}{
	{
		grpcCode:                  grpc_codes.OK,
		wantSpanCode:              codes.Unset,
		wantSpanStatusDescription: "",
	},
	{
		grpcCode:                  grpc_codes.Canceled,
		wantSpanCode:              codes.Unset,
		wantSpanStatusDescription: "",
	},
	{
		grpcCode:                  grpc_codes.Unknown,
		wantSpanCode:              codes.Error,
		wantSpanStatusDescription: grpc_codes.Unknown.String(),
	},
	{
		grpcCode:                  grpc_codes.InvalidArgument,
		wantSpanCode:              codes.Unset,
		wantSpanStatusDescription: "",
	},
	{
		grpcCode:                  grpc_codes.DeadlineExceeded,
		wantSpanCode:              codes.Error,
		wantSpanStatusDescription: grpc_codes.DeadlineExceeded.String(),
	},
	{
		grpcCode:                  grpc_codes.NotFound,
		wantSpanCode:              codes.Unset,
		wantSpanStatusDescription: "",
	},
	{
		grpcCode:                  grpc_codes.AlreadyExists,
		wantSpanCode:              codes.Unset,
		wantSpanStatusDescription: "",
	},
	{
		grpcCode:                  grpc_codes.PermissionDenied,
		wantSpanCode:              codes.Unset,
		wantSpanStatusDescription: "",
	},
	{
		grpcCode:                  grpc_codes.ResourceExhausted,
		wantSpanCode:              codes.Unset,
		wantSpanStatusDescription: "",
	},
	{
		grpcCode:                  grpc_codes.FailedPrecondition,
		wantSpanCode:              codes.Unset,
		wantSpanStatusDescription: "",
	},
	{
		grpcCode:                  grpc_codes.Aborted,
		wantSpanCode:              codes.Unset,
		wantSpanStatusDescription: "",
	},
	{
		grpcCode:                  grpc_codes.OutOfRange,
		wantSpanCode:              codes.Unset,
		wantSpanStatusDescription: "",
	},
	{
		grpcCode:                  grpc_codes.Unimplemented,
		wantSpanCode:              codes.Error,
		wantSpanStatusDescription: grpc_codes.Unimplemented.String(),
	},
	{
		grpcCode:                  grpc_codes.Internal,
		wantSpanCode:              codes.Error,
		wantSpanStatusDescription: grpc_codes.Internal.String(),
	},
	{
		grpcCode:                  grpc_codes.Unavailable,
		wantSpanCode:              codes.Error,
		wantSpanStatusDescription: grpc_codes.Unavailable.String(),
	},
	{
		grpcCode:                  grpc_codes.DataLoss,
		wantSpanCode:              codes.Error,
		wantSpanStatusDescription: grpc_codes.DataLoss.String(),
	},
	{
		grpcCode:                  grpc_codes.Unauthenticated,
		wantSpanCode:              codes.Unset,
		wantSpanStatusDescription: "",
	},
}

func assertServerSpan(t *testing.T, wantSpanCode codes.Code, wantSpanStatusDescription string, wantGrpcCode grpc_codes.Code, span trace.ReadOnlySpan) {
	// validate span status
	assert.Equal(t, wantSpanCode, span.Status().Code)
	assert.Equal(t, wantSpanStatusDescription, span.Status().Description)

	// validate grpc code span attribute
	var codeAttr attribute.KeyValue
	for _, a := range span.Attributes() {
		if a.Key == otelgrpc.GRPCStatusCodeKey {
			codeAttr = a
			break
		}
	}

	require.True(t, codeAttr.Valid(), "attributes contain gRPC status code")
	assert.Equal(t, attribute.Int64Value(int64(wantGrpcCode)), codeAttr.Value)
}

// TestUnaryServerInterceptor tests the server interceptor for unary RPCs.
func TestUnaryServerInterceptor(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := trace.NewTracerProvider(trace.WithSpanProcessor(sr))
	usi := otelgrpc.UnaryServerInterceptor(otelgrpc.WithTracerProvider(tp))
	for _, check := range serverChecks {
		name := check.grpcCode.String()
		t.Run(name, func(t *testing.T) {
			// call the unary interceptor
			grpcErr := status.Error(check.grpcCode, check.grpcCode.String())
			handler := func(_ context.Context, _ interface{}) (interface{}, error) {
				return nil, grpcErr
			}
			_, err := usi(context.Background(), &grpc_testing.SimpleRequest{}, &grpc.UnaryServerInfo{FullMethod: name}, handler)
			assert.Equal(t, grpcErr, err)

			// validate span
			span, ok := getSpanFromRecorder(sr, name)
			require.True(t, ok, "missing span %s", name)
			assertServerSpan(t, check.wantSpanCode, check.wantSpanStatusDescription, check.grpcCode, span)

			// validate events and their attributes
			assert.Len(t, span.Events(), 2)
			assert.ElementsMatch(t, []attribute.KeyValue{
				attribute.Key("message.type").String("SENT"),
				attribute.Key("message.id").Int(1),
			}, span.Events()[1].Attributes)
		})
	}
}

type mockServerStream struct {
	grpc.ServerStream
}

func (m *mockServerStream) Context() context.Context { return context.Background() }

// TestStreamServerInterceptor tests the server interceptor for streaming RPCs.
func TestStreamServerInterceptor(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := trace.NewTracerProvider(trace.WithSpanProcessor(sr))
	usi := otelgrpc.StreamServerInterceptor(otelgrpc.WithTracerProvider(tp))
	for _, check := range serverChecks {
		name := check.grpcCode.String()
		t.Run(name, func(t *testing.T) {
			// call the stream interceptor
			grpcErr := status.Error(check.grpcCode, check.grpcCode.String())
			handler := func(_ interface{}, _ grpc.ServerStream) error {
				return grpcErr
			}
			err := usi(&grpc_testing.SimpleRequest{}, &mockServerStream{}, &grpc.StreamServerInfo{FullMethod: name}, handler)
			assert.Equal(t, grpcErr, err)

			// validate span
			span, ok := getSpanFromRecorder(sr, name)
			require.True(t, ok, "missing span %s", name)
			assertServerSpan(t, check.wantSpanCode, check.wantSpanStatusDescription, check.grpcCode, span)
		})
	}
}

func TestParseFullMethod(t *testing.T) {
	tests := []struct {
		fullMethod string
		name       string
		attr       []attribute.KeyValue
	}{
		{
			fullMethod: "/grpc.test.EchoService/Echo",
			name:       "grpc.test.EchoService/Echo",
			attr: []attribute.KeyValue{
				semconv.RPCService("grpc.test.EchoService"),
				semconv.RPCMethod("Echo"),
			},
		}, {
			fullMethod: "/com.example.ExampleRmiService/exampleMethod",
			name:       "com.example.ExampleRmiService/exampleMethod",
			attr: []attribute.KeyValue{
				semconv.RPCService("com.example.ExampleRmiService"),
				semconv.RPCMethod("exampleMethod"),
			},
		}, {
			fullMethod: "/MyCalcService.Calculator/Add",
			name:       "MyCalcService.Calculator/Add",
			attr: []attribute.KeyValue{
				semconv.RPCService("MyCalcService.Calculator"),
				semconv.RPCMethod("Add"),
			},
		}, {
			fullMethod: "/MyServiceReference.ICalculator/Add",
			name:       "MyServiceReference.ICalculator/Add",
			attr: []attribute.KeyValue{
				semconv.RPCService("MyServiceReference.ICalculator"),
				semconv.RPCMethod("Add"),
			},
		}, {
			fullMethod: "/MyServiceWithNoPackage/theMethod",
			name:       "MyServiceWithNoPackage/theMethod",
			attr: []attribute.KeyValue{
				semconv.RPCService("MyServiceWithNoPackage"),
				semconv.RPCMethod("theMethod"),
			},
		}, {
			fullMethod: "/pkg.srv",
			name:       "pkg.srv",
			attr:       []attribute.KeyValue(nil),
		}, {
			fullMethod: "/pkg.srv/",
			name:       "pkg.srv/",
			attr: []attribute.KeyValue{
				semconv.RPCService("pkg.srv"),
			},
		},
	}

	for _, test := range tests {
		n, a := internal.ParseFullMethod(test.fullMethod)
		assert.Equal(t, test.name, n)
		assert.Equal(t, test.attr, a)
	}
}
