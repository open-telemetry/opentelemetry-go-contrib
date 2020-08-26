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
package grpc

import (
	"context"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/otel/api/trace/tracetest"
	"go.opentelemetry.io/otel/semconv"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/golang/protobuf/proto" //nolint:staticcheck

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/label"
)

type SpanRecorder struct {
	mu    sync.RWMutex
	spans map[string]*tracetest.Span
}

func NewSpanRecorder() *SpanRecorder {
	return &SpanRecorder{spans: make(map[string]*tracetest.Span)}
}

func (sr *SpanRecorder) OnStart(span *tracetest.Span) {}

func (sr *SpanRecorder) OnEnd(span *tracetest.Span) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	sr.spans[span.Name()] = span
}

func (sr *SpanRecorder) Get(name string) (*tracetest.Span, bool) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	s, ok := sr.spans[name]
	return s, ok
}

type mockUICInvoker struct {
	ctx context.Context
}

func (mcuici *mockUICInvoker) invoker(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
	mcuici.ctx = ctx
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

	sr := NewSpanRecorder()
	tp := tracetest.NewProvider(tracetest.WithSpanRecorder(sr))
	tracer := tp.Tracer("grpc/client")
	unaryInterceptor := UnaryClientInterceptor(tracer)

	req := &mockProtoMessage{}
	reply := &mockProtoMessage{}
	uniInterceptorInvoker := &mockUICInvoker{}

	checks := []struct {
		method       string
		name         string
		expectedAttr map[label.Key]label.Value
		eventsAttr   []map[label.Key]label.Value
	}{
		{
			method: "/github.com.serviceName/bar",
			name:   "github.com.serviceName/bar",
			expectedAttr: map[label.Key]label.Value{
				semconv.RPCSystemKey:   label.StringValue("grpc"),
				semconv.RPCServiceKey:  label.StringValue("github.com.serviceName"),
				semconv.RPCMethodKey:   label.StringValue("bar"),
				semconv.NetPeerIPKey:   label.StringValue("fake"),
				semconv.NetPeerPortKey: label.StringValue("connection"),
			},
			eventsAttr: []map[label.Key]label.Value{
				{
					semconv.RPCMessageTypeKey:             label.StringValue("SENT"),
					semconv.RPCMessageIDKey:               label.IntValue(1),
					semconv.RPCMessageUncompressedSizeKey: label.IntValue(proto.Size(proto.Message(req))),
				},
				{
					semconv.RPCMessageTypeKey:             label.StringValue("RECEIVED"),
					semconv.RPCMessageIDKey:               label.IntValue(1),
					semconv.RPCMessageUncompressedSizeKey: label.IntValue(proto.Size(proto.Message(reply))),
				},
			},
		},
		{
			method: "/serviceName/bar",
			name:   "serviceName/bar",
			expectedAttr: map[label.Key]label.Value{
				semconv.RPCSystemKey:   label.StringValue("grpc"),
				semconv.RPCServiceKey:  label.StringValue("serviceName"),
				semconv.RPCMethodKey:   label.StringValue("bar"),
				semconv.NetPeerIPKey:   label.StringValue("fake"),
				semconv.NetPeerPortKey: label.StringValue("connection"),
			},
			eventsAttr: []map[label.Key]label.Value{
				{
					semconv.RPCMessageTypeKey:             label.StringValue("SENT"),
					semconv.RPCMessageIDKey:               label.IntValue(1),
					semconv.RPCMessageUncompressedSizeKey: label.IntValue(proto.Size(proto.Message(req))),
				},
				{
					semconv.RPCMessageTypeKey:             label.StringValue("RECEIVED"),
					semconv.RPCMessageIDKey:               label.IntValue(1),
					semconv.RPCMessageUncompressedSizeKey: label.IntValue(proto.Size(proto.Message(reply))),
				},
			},
		},
		{
			method: "serviceName/bar",
			name:   "serviceName/bar",
			expectedAttr: map[label.Key]label.Value{
				semconv.RPCSystemKey:   label.StringValue("grpc"),
				semconv.RPCServiceKey:  label.StringValue("serviceName"),
				semconv.RPCMethodKey:   label.StringValue("bar"),
				semconv.NetPeerIPKey:   label.StringValue("fake"),
				semconv.NetPeerPortKey: label.StringValue("connection"),
			},
			eventsAttr: []map[label.Key]label.Value{
				{
					semconv.RPCMessageTypeKey:             label.StringValue("SENT"),
					semconv.RPCMessageIDKey:               label.IntValue(1),
					semconv.RPCMessageUncompressedSizeKey: label.IntValue(proto.Size(proto.Message(req))),
				},
				{
					semconv.RPCMessageTypeKey:             label.StringValue("RECEIVED"),
					semconv.RPCMessageIDKey:               label.IntValue(1),
					semconv.RPCMessageUncompressedSizeKey: label.IntValue(proto.Size(proto.Message(reply))),
				},
			},
		},
		{
			method: "invalidName",
			name:   "invalidName",
			expectedAttr: map[label.Key]label.Value{
				semconv.RPCSystemKey:   label.StringValue("grpc"),
				semconv.NetPeerIPKey:   label.StringValue("fake"),
				semconv.NetPeerPortKey: label.StringValue("connection"),
			},
			eventsAttr: []map[label.Key]label.Value{
				{
					semconv.RPCMessageTypeKey:             label.StringValue("SENT"),
					semconv.RPCMessageIDKey:               label.IntValue(1),
					semconv.RPCMessageUncompressedSizeKey: label.IntValue(proto.Size(proto.Message(req))),
				},
				{
					semconv.RPCMessageTypeKey:             label.StringValue("RECEIVED"),
					semconv.RPCMessageIDKey:               label.IntValue(1),
					semconv.RPCMessageUncompressedSizeKey: label.IntValue(proto.Size(proto.Message(reply))),
				},
			},
		},
		{
			method: "/github.com.foo.serviceName_123/method",
			name:   "github.com.foo.serviceName_123/method",
			expectedAttr: map[label.Key]label.Value{
				semconv.RPCSystemKey:   label.StringValue("grpc"),
				semconv.RPCServiceKey:  label.StringValue("github.com.foo.serviceName_123"),
				semconv.RPCMethodKey:   label.StringValue("method"),
				semconv.NetPeerIPKey:   label.StringValue("fake"),
				semconv.NetPeerPortKey: label.StringValue("connection"),
			},
			eventsAttr: []map[label.Key]label.Value{
				{
					semconv.RPCMessageTypeKey:             label.StringValue("SENT"),
					semconv.RPCMessageIDKey:               label.IntValue(1),
					semconv.RPCMessageUncompressedSizeKey: label.IntValue(proto.Size(proto.Message(req))),
				},
				{
					semconv.RPCMessageTypeKey:             label.StringValue("RECEIVED"),
					semconv.RPCMessageIDKey:               label.IntValue(1),
					semconv.RPCMessageUncompressedSizeKey: label.IntValue(proto.Size(proto.Message(reply))),
				},
			},
		},
	}

	for _, check := range checks {
		if !assert.NoError(t, unaryInterceptor(context.Background(), check.method, req, reply, clientConn, uniInterceptorInvoker.invoker)) {
			continue
		}
		span, ok := sr.Get(check.name)
		if !assert.True(t, ok, "missing span %q", check.name) {
			continue
		}
		assert.Equal(t, check.expectedAttr, span.Attributes())
		assert.Equal(t, check.eventsAttr, eventAttrMap(span.Events()))
	}
}

func eventAttrMap(events []tracetest.Event) []map[label.Key]label.Value {
	maps := make([]map[label.Key]label.Value, len(events))
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

	// tracer
	sr := NewSpanRecorder()
	tp := tracetest.NewProvider(tracetest.WithSpanRecorder(sr))
	tracer := tp.Tracer("grpc/Server")
	streamCI := StreamClientInterceptor(tracer)

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
	_, ok := sr.Get(name)
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
	var span *tracetest.Span
	for retry := 0; retry < 5; retry++ {
		span, ok = sr.Get(name)
		if ok {
			break
		}
		time.Sleep(time.Second * 1)
	}
	require.True(t, ok, "missing span %s", name)

	expectedAttr := map[label.Key]label.Value{
		semconv.RPCSystemKey:   label.StringValue("grpc"),
		semconv.RPCServiceKey:  label.StringValue("github.com.serviceName"),
		semconv.RPCMethodKey:   label.StringValue("bar"),
		semconv.NetPeerIPKey:   label.StringValue("fake"),
		semconv.NetPeerPortKey: label.StringValue("connection"),
	}
	assert.Equal(t, expectedAttr, span.Attributes())

	events := span.Events()
	require.Len(t, events, 20)
	for i := 0; i < 20; i += 2 {
		msgID := i/2 + 1
		validate := func(eventName string, attrs map[label.Key]label.Value) {
			for k, v := range attrs {
				if k == semconv.RPCMessageTypeKey && v.AsString() != eventName {
					t.Errorf("invalid event on index: %d expecting %s event, receive %s event", i, eventName, v.AsString())
				}
				if k == semconv.RPCMessageIDKey && v != label.IntValue(msgID) {
					t.Errorf("invalid id for message event expected %d received %d", msgID, v.AsInt32())
				}
			}
		}
		validate("SENT", events[i].Attributes)
		validate("RECEIVED", events[i+1].Attributes)
	}

	// ensure CloseSend can be subsequently called
	_ = streamClient.CloseSend()
}

func TestServerInterceptorError(t *testing.T) {
	sr := NewSpanRecorder()
	tp := tracetest.NewProvider(tracetest.WithSpanRecorder(sr))
	tracer := tp.Tracer("grpc/Server")
	usi := UnaryServerInterceptor(tracer)
	deniedErr := status.Error(codes.PermissionDenied, "PERMISSION_DENIED_TEXT")
	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return nil, deniedErr
	}
	_, err := usi(context.Background(), &mockProtoMessage{}, &grpc.UnaryServerInfo{}, handler)
	require.Error(t, err)
	assert.Equal(t, err, deniedErr)

	span, ok := sr.Get("")
	if !ok {
		t.Fatalf("failed to export error span")
	}
	assert.Equal(t, span.StatusCode(), otelcodes.PermissionDenied)
	assert.Contains(t, deniedErr.Error(), span.StatusMessage())
	assert.Len(t, span.Events(), 2)
	assert.Equal(t, map[label.Key]label.Value{
		label.Key("message.type"):              label.StringValue("SENT"),
		label.Key("message.id"):                label.IntValue(1),
		label.Key("message.uncompressed_size"): label.IntValue(26),
	}, span.Events()[1].Attributes)
}

func TestParseFullMethod(t *testing.T) {
	tests := []struct {
		fullMethod string
		name       string
		attr       []label.KeyValue
	}{
		{
			fullMethod: "/grpc.test.EchoService/Echo",
			name:       "grpc.test.EchoService/Echo",
			attr: []label.KeyValue{
				semconv.RPCServiceKey.String("grpc.test.EchoService"),
				semconv.RPCMethodKey.String("Echo"),
			},
		}, {
			fullMethod: "/com.example.ExampleRmiService/exampleMethod",
			name:       "com.example.ExampleRmiService/exampleMethod",
			attr: []label.KeyValue{
				semconv.RPCServiceKey.String("com.example.ExampleRmiService"),
				semconv.RPCMethodKey.String("exampleMethod"),
			},
		}, {
			fullMethod: "/MyCalcService.Calculator/Add",
			name:       "MyCalcService.Calculator/Add",
			attr: []label.KeyValue{
				semconv.RPCServiceKey.String("MyCalcService.Calculator"),
				semconv.RPCMethodKey.String("Add"),
			},
		}, {
			fullMethod: "/MyServiceReference.ICalculator/Add",
			name:       "MyServiceReference.ICalculator/Add",
			attr: []label.KeyValue{
				semconv.RPCServiceKey.String("MyServiceReference.ICalculator"),
				semconv.RPCMethodKey.String("Add"),
			},
		}, {
			fullMethod: "/MyServiceWithNoPackage/theMethod",
			name:       "MyServiceWithNoPackage/theMethod",
			attr: []label.KeyValue{
				semconv.RPCServiceKey.String("MyServiceWithNoPackage"),
				semconv.RPCMethodKey.String("theMethod"),
			},
		}, {
			fullMethod: "/pkg.srv",
			name:       "pkg.srv",
			attr:       []label.KeyValue(nil),
		}, {
			fullMethod: "/pkg.srv/",
			name:       "pkg.srv/",
			attr: []label.KeyValue{
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
