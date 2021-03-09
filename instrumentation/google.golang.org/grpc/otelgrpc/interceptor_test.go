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
package otelgrpc

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/oteltest"
	"go.opentelemetry.io/otel/semconv"

	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"
	grpc_codes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func NewSpanRecorder() *oteltest.SpanRecorder {
	return &oteltest.SpanRecorder{}
}

func getSpanFromRecorder(sr *oteltest.SpanRecorder, name string) (*oteltest.Span, bool) {
	for _, s := range sr.Completed() {
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

type mockProtoMessage struct{}

func (mm *mockProtoMessage) Reset() {
}

func (mm *mockProtoMessage) String() string {
	return "mock"
}

func (mm *mockProtoMessage) ProtoMessage() {
}

func TestUnaryClientInterceptor(t *testing.T) {
	clientConn, err := grpc.Dial("fake:connection", grpc.WithInsecure())
	if err != nil {
		t.Fatalf("failed to create client connection: %v", err)
	}
	defer clientConn.Close()

	sr := NewSpanRecorder()
	tp := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))
	unaryInterceptor := UnaryClientInterceptor(WithTracerProvider(tp))

	req := &mockProtoMessage{}
	reply := &mockProtoMessage{}
	uniInterceptorInvoker := &mockUICInvoker{}

	checks := []struct {
		method           string
		name             string
		expectedSpanCode codes.Code
		expectedAttr     map[attribute.Key]attribute.Value
		eventsAttr       []map[attribute.Key]attribute.Value
		expectErr        bool
	}{
		{
			method: "/github.com.serviceName/bar",
			name:   "github.com.serviceName/bar",
			expectedAttr: map[attribute.Key]attribute.Value{
				semconv.RPCSystemKey:   attribute.StringValue("grpc"),
				semconv.RPCServiceKey:  attribute.StringValue("github.com.serviceName"),
				semconv.RPCMethodKey:   attribute.StringValue("bar"),
				GRPCStatusCodeKey:      attribute.Int64Value(0),
				semconv.NetPeerIPKey:   attribute.StringValue("fake"),
				semconv.NetPeerPortKey: attribute.StringValue("connection"),
			},
			eventsAttr: []map[attribute.Key]attribute.Value{
				{
					semconv.RPCMessageTypeKey:             attribute.StringValue("SENT"),
					semconv.RPCMessageIDKey:               attribute.IntValue(1),
					semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(proto.Size(proto.Message(req))),
				},
				{
					semconv.RPCMessageTypeKey:             attribute.StringValue("RECEIVED"),
					semconv.RPCMessageIDKey:               attribute.IntValue(1),
					semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(proto.Size(proto.Message(reply))),
				},
			},
		},
		{
			method: "/serviceName/bar",
			name:   "serviceName/bar",
			expectedAttr: map[attribute.Key]attribute.Value{
				semconv.RPCSystemKey:   attribute.StringValue("grpc"),
				semconv.RPCServiceKey:  attribute.StringValue("serviceName"),
				semconv.RPCMethodKey:   attribute.StringValue("bar"),
				GRPCStatusCodeKey:      attribute.Int64Value(0),
				semconv.NetPeerIPKey:   attribute.StringValue("fake"),
				semconv.NetPeerPortKey: attribute.StringValue("connection"),
			},
			eventsAttr: []map[attribute.Key]attribute.Value{
				{
					semconv.RPCMessageTypeKey:             attribute.StringValue("SENT"),
					semconv.RPCMessageIDKey:               attribute.IntValue(1),
					semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(proto.Size(proto.Message(req))),
				},
				{
					semconv.RPCMessageTypeKey:             attribute.StringValue("RECEIVED"),
					semconv.RPCMessageIDKey:               attribute.IntValue(1),
					semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(proto.Size(proto.Message(reply))),
				},
			},
		},
		{
			method: "serviceName/bar",
			name:   "serviceName/bar",
			expectedAttr: map[attribute.Key]attribute.Value{
				semconv.RPCSystemKey:   attribute.StringValue("grpc"),
				semconv.RPCServiceKey:  attribute.StringValue("serviceName"),
				semconv.RPCMethodKey:   attribute.StringValue("bar"),
				GRPCStatusCodeKey:      attribute.Int64Value(int64(grpc_codes.OK)),
				semconv.NetPeerIPKey:   attribute.StringValue("fake"),
				semconv.NetPeerPortKey: attribute.StringValue("connection"),
			},
			eventsAttr: []map[attribute.Key]attribute.Value{
				{
					semconv.RPCMessageTypeKey:             attribute.StringValue("SENT"),
					semconv.RPCMessageIDKey:               attribute.IntValue(1),
					semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(proto.Size(proto.Message(req))),
				},
				{
					semconv.RPCMessageTypeKey:             attribute.StringValue("RECEIVED"),
					semconv.RPCMessageIDKey:               attribute.IntValue(1),
					semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(proto.Size(proto.Message(reply))),
				},
			},
		},
		{
			method:           "serviceName/bar_error",
			name:             "serviceName/bar_error",
			expectedSpanCode: codes.Error,
			expectedAttr: map[attribute.Key]attribute.Value{
				semconv.RPCSystemKey:   attribute.StringValue("grpc"),
				semconv.RPCServiceKey:  attribute.StringValue("serviceName"),
				semconv.RPCMethodKey:   attribute.StringValue("bar_error"),
				GRPCStatusCodeKey:      attribute.Int64Value(int64(grpc_codes.Internal)),
				semconv.NetPeerIPKey:   attribute.StringValue("fake"),
				semconv.NetPeerPortKey: attribute.StringValue("connection"),
			},
			eventsAttr: []map[attribute.Key]attribute.Value{
				{
					semconv.RPCMessageTypeKey:             attribute.StringValue("SENT"),
					semconv.RPCMessageIDKey:               attribute.IntValue(1),
					semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(proto.Size(proto.Message(req))),
				},
				{
					semconv.RPCMessageTypeKey:             attribute.StringValue("RECEIVED"),
					semconv.RPCMessageIDKey:               attribute.IntValue(1),
					semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(proto.Size(proto.Message(reply))),
				},
			},
			expectErr: true,
		},
		{
			method: "invalidName",
			name:   "invalidName",
			expectedAttr: map[attribute.Key]attribute.Value{
				semconv.RPCSystemKey:   attribute.StringValue("grpc"),
				GRPCStatusCodeKey:      attribute.Int64Value(0),
				semconv.NetPeerIPKey:   attribute.StringValue("fake"),
				semconv.NetPeerPortKey: attribute.StringValue("connection"),
			},
			eventsAttr: []map[attribute.Key]attribute.Value{
				{
					semconv.RPCMessageTypeKey:             attribute.StringValue("SENT"),
					semconv.RPCMessageIDKey:               attribute.IntValue(1),
					semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(proto.Size(proto.Message(req))),
				},
				{
					semconv.RPCMessageTypeKey:             attribute.StringValue("RECEIVED"),
					semconv.RPCMessageIDKey:               attribute.IntValue(1),
					semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(proto.Size(proto.Message(reply))),
				},
			},
		},
		{
			method: "/github.com.foo.serviceName_123/method",
			name:   "github.com.foo.serviceName_123/method",
			expectedAttr: map[attribute.Key]attribute.Value{
				semconv.RPCSystemKey:   attribute.StringValue("grpc"),
				GRPCStatusCodeKey:      attribute.Int64Value(0),
				semconv.RPCServiceKey:  attribute.StringValue("github.com.foo.serviceName_123"),
				semconv.RPCMethodKey:   attribute.StringValue("method"),
				semconv.NetPeerIPKey:   attribute.StringValue("fake"),
				semconv.NetPeerPortKey: attribute.StringValue("connection"),
			},
			eventsAttr: []map[attribute.Key]attribute.Value{
				{
					semconv.RPCMessageTypeKey:             attribute.StringValue("SENT"),
					semconv.RPCMessageIDKey:               attribute.IntValue(1),
					semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(proto.Size(proto.Message(req))),
				},
				{
					semconv.RPCMessageTypeKey:             attribute.StringValue("RECEIVED"),
					semconv.RPCMessageIDKey:               attribute.IntValue(1),
					semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(proto.Size(proto.Message(reply))),
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
		assert.Equal(t, check.expectedSpanCode, span.StatusCode())
		assert.Equal(t, check.expectedAttr, span.Attributes())
		assert.Equal(t, check.eventsAttr, eventAttrMap(span.Events()))
	}
}

func eventAttrMap(events []oteltest.Event) []map[attribute.Key]attribute.Value {
	maps := make([]map[attribute.Key]attribute.Value, len(events))
	for i, event := range events {
		maps[i] = event.Attributes
	}
	return maps
}

type mockClientStream struct {
	Desc *grpc.StreamDesc
	Ctx  context.Context
}

func (mockClientStream) SendMsg(m interface{}) error  { return nil }
func (mockClientStream) RecvMsg(m interface{}) error  { return nil }
func (mockClientStream) CloseSend() error             { return nil }
func (c mockClientStream) Context() context.Context   { return c.Ctx }
func (mockClientStream) Header() (metadata.MD, error) { return nil, nil }
func (mockClientStream) Trailer() metadata.MD         { return nil }

func TestStreamClientInterceptor(t *testing.T) {
	clientConn, err := grpc.Dial("fake:connection", grpc.WithInsecure())
	if err != nil {
		t.Fatalf("failed to create client connection: %v", err)
	}
	defer clientConn.Close()

	// tracer
	sr := NewSpanRecorder()
	tp := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))
	streamCI := StreamClientInterceptor(WithTracerProvider(tp))

	var mockClStr mockClientStream
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
			mockClStr = mockClientStream{Desc: desc, Ctx: ctx}
			return mockClStr, nil
		},
	)
	require.NoError(t, err, "initialize grpc stream client")
	_, ok := getSpanFromRecorder(sr, name)
	require.False(t, ok, "span should ended while stream is open")

	req := &mockProtoMessage{}
	reply := &mockProtoMessage{}

	// send and receive fake data
	for i := 0; i < 10; i++ {
		_ = streamClient.SendMsg(req)
		_ = streamClient.RecvMsg(reply)
	}

	// close client and server stream
	_ = streamClient.CloseSend()
	mockClStr.Desc.ServerStreams = false
	_ = streamClient.RecvMsg(reply)

	// added retry because span end is called in separate go routine
	var span *oteltest.Span
	for retry := 0; retry < 5; retry++ {
		span, ok = getSpanFromRecorder(sr, name)
		if ok {
			break
		}
		time.Sleep(time.Second * 1)
	}
	require.True(t, ok, "missing span %s", name)

	expectedAttr := map[attribute.Key]attribute.Value{
		semconv.RPCSystemKey:   attribute.StringValue("grpc"),
		GRPCStatusCodeKey:      attribute.Int64Value(int64(grpc_codes.OK)),
		semconv.RPCServiceKey:  attribute.StringValue("github.com.serviceName"),
		semconv.RPCMethodKey:   attribute.StringValue("bar"),
		semconv.NetPeerIPKey:   attribute.StringValue("fake"),
		semconv.NetPeerPortKey: attribute.StringValue("connection"),
	}
	assert.Equal(t, expectedAttr, span.Attributes())

	events := span.Events()
	require.Len(t, events, 20)
	for i := 0; i < 20; i += 2 {
		msgID := i/2 + 1
		validate := func(eventName string, attrs map[attribute.Key]attribute.Value) {
			for k, v := range attrs {
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

// TestStreamClientInterceptorWithError tests a situation that streamer returns an error.
func TestStreamClientInterceptorWithError(t *testing.T) {
	defer goleak.VerifyNone(t)

	clientConn, err := grpc.Dial("fake:connection", grpc.WithInsecure())
	if err != nil {
		t.Fatalf("failed to create client connection: %v", err)
	}
	defer clientConn.Close()

	// tracer
	sr := NewSpanRecorder()
	tp := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))
	streamCI := StreamClientInterceptor(WithTracerProvider(tp))

	var mockClStr mockClientStream
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
			mockClStr = mockClientStream{Desc: desc, Ctx: ctx}
			return mockClStr, errors.New("test")
		},
	)
	require.Error(t, err, "initialize grpc stream client")
	assert.IsType(t, mockClientStream{}, streamClient)

	span, ok := getSpanFromRecorder(sr, name)
	require.True(t, ok, "missing span %s", name)

	expectedAttr := map[attribute.Key]attribute.Value{
		semconv.RPCSystemKey:   attribute.StringValue("grpc"),
		GRPCStatusCodeKey:      attribute.Int64Value(int64(grpc_codes.Unknown)),
		semconv.RPCServiceKey:  attribute.StringValue("github.com.serviceName"),
		semconv.RPCMethodKey:   attribute.StringValue("bar"),
		semconv.NetPeerIPKey:   attribute.StringValue("fake"),
		semconv.NetPeerPortKey: attribute.StringValue("connection"),
	}
	assert.Equal(t, expectedAttr, span.Attributes())
	assert.Equal(t, codes.Error, span.StatusCode())
}

func TestServerInterceptorError(t *testing.T) {
	sr := NewSpanRecorder()
	tp := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))
	usi := UnaryServerInterceptor(WithTracerProvider(tp))
	deniedErr := status.Error(grpc_codes.PermissionDenied, "PERMISSION_DENIED_TEXT")
	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return nil, deniedErr
	}
	_, err := usi(context.Background(), &mockProtoMessage{}, &grpc.UnaryServerInfo{}, handler)
	require.Error(t, err)
	assert.Equal(t, err, deniedErr)

	span, ok := getSpanFromRecorder(sr, "")
	if !ok {
		t.Fatalf("failed to export error span")
	}
	assert.Equal(t, codes.Error, span.StatusCode())
	assert.Contains(t, deniedErr.Error(), span.StatusMessage())
	codeAttr, ok := span.Attributes()[GRPCStatusCodeKey]
	assert.True(t, ok, "attributes contain gRPC status code")
	assert.Equal(t, attribute.Int64Value(int64(grpc_codes.PermissionDenied)), codeAttr)
	assert.Len(t, span.Events(), 2)
	assert.Equal(t, map[attribute.Key]attribute.Value{
		attribute.Key("message.type"):              attribute.StringValue("SENT"),
		attribute.Key("message.id"):                attribute.IntValue(1),
		attribute.Key("message.uncompressed_size"): attribute.IntValue(26),
	}, span.Events()[1].Attributes)
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
				semconv.RPCServiceKey.String("grpc.test.EchoService"),
				semconv.RPCMethodKey.String("Echo"),
			},
		}, {
			fullMethod: "/com.example.ExampleRmiService/exampleMethod",
			name:       "com.example.ExampleRmiService/exampleMethod",
			attr: []attribute.KeyValue{
				semconv.RPCServiceKey.String("com.example.ExampleRmiService"),
				semconv.RPCMethodKey.String("exampleMethod"),
			},
		}, {
			fullMethod: "/MyCalcService.Calculator/Add",
			name:       "MyCalcService.Calculator/Add",
			attr: []attribute.KeyValue{
				semconv.RPCServiceKey.String("MyCalcService.Calculator"),
				semconv.RPCMethodKey.String("Add"),
			},
		}, {
			fullMethod: "/MyServiceReference.ICalculator/Add",
			name:       "MyServiceReference.ICalculator/Add",
			attr: []attribute.KeyValue{
				semconv.RPCServiceKey.String("MyServiceReference.ICalculator"),
				semconv.RPCMethodKey.String("Add"),
			},
		}, {
			fullMethod: "/MyServiceWithNoPackage/theMethod",
			name:       "MyServiceWithNoPackage/theMethod",
			attr: []attribute.KeyValue{
				semconv.RPCServiceKey.String("MyServiceWithNoPackage"),
				semconv.RPCMethodKey.String("theMethod"),
			},
		}, {
			fullMethod: "/pkg.srv",
			name:       "pkg.srv",
			attr:       []attribute.KeyValue(nil),
		}, {
			fullMethod: "/pkg.srv/",
			name:       "pkg.srv/",
			attr: []attribute.KeyValue{
				semconv.RPCServiceKey.String("pkg.srv"),
			},
		},
	}

	for _, test := range tests {
		n, a := parseFullMethod(test.fullMethod)
		assert.Equal(t, test.name, n)
		assert.Equal(t, test.attr, a)
	}
}
