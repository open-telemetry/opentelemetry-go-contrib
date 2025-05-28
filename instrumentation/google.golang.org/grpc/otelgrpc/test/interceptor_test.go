// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc/internal/test"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.32.0"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"
	grpc_codes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"

	"google.golang.org/grpc/interop/grpc_testing"
)

func getSpanFromRecorder(sr *tracetest.SpanRecorder, name string) (trace.ReadOnlySpan, bool) {
	for _, s := range sr.Ended() {
		if s.Name() == name {
			return s, true
		}
	}
	return nil, false
}

func ctxDialer() func(context.Context, string) (net.Conn, error) {
	l := bufconn.Listen(0)
	return func(ctx context.Context, _ string) (net.Conn, error) {
		return l.DialContext(ctx)
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
	Events               []otelgrpc.Event
}

func newMockClientStream(opts clientStreamOpts) *mockClientStream {
	var msgs []grpc_testing.SimpleResponse
	for i := 0; i < opts.NumRecvMsgs; i++ {
		msgs = append(msgs, grpc_testing.SimpleResponse{})
	}
	return &mockClientStream{msgs: msgs}
}

func createInterceptedStreamClient(t *testing.T, method string, opts clientStreamOpts) (grpc.ClientStream, *tracetest.SpanRecorder) {
	mockStream := newMockClientStream(opts)
	clientConn, err := grpc.NewClient("fake:8906",
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
	interceptorOpts := []otelgrpc.Option{
		otelgrpc.WithTracerProvider(tp),
		otelgrpc.WithSpanOptions(oteltrace.WithAttributes(attribute.Bool("custom", true))),
	}
	if len(opts.Events) > 0 {
		interceptorOpts = append(interceptorOpts, otelgrpc.WithMessageEvents(opts.Events...))
	}
	//nolint:staticcheck // Interceptors are deprecated and will be removed in the next release.
	streamCI := otelgrpc.StreamClientInterceptor(interceptorOpts...)

	streamClient, err := streamCI(
		context.Background(),
		&grpc.StreamDesc{ServerStreams: !opts.DisableServerStreams},
		clientConn,
		method,
		func(ctx context.Context,
			desc *grpc.StreamDesc,
			cc *grpc.ClientConn,
			method string,
			opts ...grpc.CallOption,
		) (grpc.ClientStream, error) {
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
	opts := clientStreamOpts{
		NumRecvMsgs: 10,
		Events:      []otelgrpc.Event{otelgrpc.SentEvents, otelgrpc.ReceivedEvents},
	}
	streamClient, sr := createInterceptedStreamClient(t, method, opts)
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

	// wait for span end that is called in separate go routine
	var span trace.ReadOnlySpan
	require.Eventually(t, func() bool {
		span, ok = getSpanFromRecorder(sr, name)
		return ok
	}, 5*time.Second, time.Second, "missing span %s", name)

	expectedAttr := []attribute.KeyValue{
		semconv.RPCSystemGRPC,
		semconv.RPCGRPCStatusCodeOk,
		semconv.RPCService("github.com.serviceName"),
		semconv.RPCMethod("bar"),
		semconv.ServerAddress("fake"),
		semconv.ServerPort(8906),
		attribute.Bool("custom", true),
	}
	assert.ElementsMatch(t, expectedAttr, span.Attributes())

	events := span.Events()
	require.Len(t, events, 20)
	for i := 0; i < 20; i += 2 {
		msgID := i/2 + 1
		validate := func(eventName string, attrs []attribute.KeyValue) {
			for _, kv := range attrs {
				k, v := kv.Key, kv.Value
				if k == semconv.RPCMessageTypeKey && v.AsString() != eventName {
					t.Errorf("invalid event on index: %d expecting %s event, receive %s event", i, eventName, v.AsString())
				}
				if k == semconv.RPCMessageIDKey && v != attribute.IntValue(msgID) {
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

func TestStreamClientInterceptorEvents(t *testing.T) {
	testCases := []struct {
		Name   string
		Events []otelgrpc.Event
	}{
		{Name: "With both events", Events: []otelgrpc.Event{otelgrpc.SentEvents, otelgrpc.ReceivedEvents}},
		{Name: "With only sent events", Events: []otelgrpc.Event{otelgrpc.SentEvents}},
		{Name: "With only received events", Events: []otelgrpc.Event{otelgrpc.ReceivedEvents}},
		{Name: "No events", Events: []otelgrpc.Event{}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			defer goleak.VerifyNone(t)

			method := "/github.com.serviceName/bar"
			name := "github.com.serviceName/bar"
			streamClient, sr := createInterceptedStreamClient(t, method, clientStreamOpts{NumRecvMsgs: 1, Events: testCase.Events})
			_, ok := getSpanFromRecorder(sr, name)
			require.False(t, ok, "span should not end while stream is open")

			req := &grpc_testing.SimpleRequest{}
			reply := &grpc_testing.SimpleResponse{}
			var eventsAttr []map[attribute.Key]attribute.Value

			// send and receive fake data
			_ = streamClient.SendMsg(req)
			_ = streamClient.RecvMsg(reply)
			for _, event := range testCase.Events {
				switch event {
				case otelgrpc.SentEvents:
					eventsAttr = append(eventsAttr,
						map[attribute.Key]attribute.Value{
							semconv.RPCMessageTypeKey: attribute.StringValue("SENT"),
							semconv.RPCMessageIDKey:   attribute.IntValue(1),
						},
					)
				case otelgrpc.ReceivedEvents:
					eventsAttr = append(eventsAttr,
						map[attribute.Key]attribute.Value{
							semconv.RPCMessageTypeKey: attribute.StringValue("RECEIVED"),
							semconv.RPCMessageIDKey:   attribute.IntValue(1),
						},
					)
				}
			}

			// The stream has been exhausted so next read should get a EOF and the stream should be considered closed.
			err := streamClient.RecvMsg(reply)
			require.Equal(t, io.EOF, err)

			// wait for span end that is called in separate go routine
			var span trace.ReadOnlySpan
			require.Eventually(t, func() bool {
				span, ok = getSpanFromRecorder(sr, name)
				return ok
			}, 5*time.Second, time.Second, "missing span %s", name)

			if len(testCase.Events) == 0 {
				assert.Empty(t, span.Events())
			} else {
				assert.Len(t, span.Events(), len(eventsAttr))
				assert.Equal(t, eventsAttr, eventAttrMap(span.Events()))
			}

			// ensure CloseSend can be subsequently called
			_ = streamClient.CloseSend()
		})
	}
}

func TestStreamClientInterceptorOnUnidirectionalClientServerStream(t *testing.T) {
	defer goleak.VerifyNone(t)

	method := "/github.com.serviceName/bar"
	name := "github.com.serviceName/bar"
	opts := clientStreamOpts{
		NumRecvMsgs:          1,
		DisableServerStreams: true,
		Events:               []otelgrpc.Event{otelgrpc.ReceivedEvents, otelgrpc.SentEvents},
	}
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
	require.NoError(t, err)

	// wait for span end that is called in separate go routine
	var span trace.ReadOnlySpan
	require.Eventually(t, func() bool {
		span, ok = getSpanFromRecorder(sr, name)
		return ok
	}, 5*time.Second, time.Second, "missing span %s", name)

	expectedAttr := []attribute.KeyValue{
		semconv.RPCSystemGRPC,
		semconv.RPCGRPCStatusCodeOk,
		semconv.RPCService("github.com.serviceName"),
		semconv.RPCMethod("bar"),
		semconv.ServerAddress("fake"),
		semconv.ServerPort(8906),
		attribute.Bool("custom", true),
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
				if k == semconv.RPCMessageTypeKey && v.AsString() != eventName {
					t.Errorf("invalid event on index: %d expecting %s event, receive %s event", i, eventName, v.AsString())
				}
				if k == semconv.RPCMessageIDKey && v != attribute.IntValue(msgID) {
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

	clientConn, err := grpc.NewClient("fake:8906",
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
	//nolint:staticcheck // Interceptors are deprecated and will be removed in the next release.
	streamCI := otelgrpc.StreamClientInterceptor(
		otelgrpc.WithTracerProvider(tp),
		otelgrpc.WithMessageEvents(otelgrpc.ReceivedEvents, otelgrpc.SentEvents),
	)

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
			opts ...grpc.CallOption,
		) (grpc.ClientStream, error) {
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

	clientConn, err := grpc.NewClient("fake:8906",
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
	//nolint:staticcheck // Interceptors are deprecated and will be removed in the next release.
	streamCI := otelgrpc.StreamClientInterceptor(
		otelgrpc.WithTracerProvider(tp),
		otelgrpc.WithMessageEvents(otelgrpc.ReceivedEvents, otelgrpc.SentEvents),
	)

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
			opts ...grpc.CallOption,
		) (grpc.ClientStream, error) {
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
		semconv.RPCGRPCStatusCodeUnknown,
		semconv.RPCService("github.com.serviceName"),
		semconv.RPCMethod("bar"),
		semconv.ServerAddress("fake"),
		semconv.ServerPort(8906),
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
		if a.Key == semconv.RPCGRPCStatusCodeKey {
			codeAttr = a
			break
		}
	}

	require.True(t, codeAttr.Valid(), "attributes contain gRPC status code")
	assert.Equal(t, attribute.Int64Value(int64(wantGrpcCode)), codeAttr.Value)
}

func BenchmarkStreamClientInterceptor(b *testing.B) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(b, err, "failed to open port")
	client := newGrpcTest(b, listener,
		[]grpc.DialOption{
			//nolint:staticcheck // Interceptors are deprecated and will be removed in the next release.
			grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor()),
		},
		[]grpc.ServerOption{},
	)

	b.ResetTimer()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for i := 0; i < b.N; i++ {
		test.DoClientStreaming(ctx, client)
	}
}
