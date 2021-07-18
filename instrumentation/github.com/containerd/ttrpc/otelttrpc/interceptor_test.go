// Copyright The OpenTelemetry Authors
// Copyright The containerd Authors
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
package otelttrpc

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"

	"github.com/containerd/ttrpc"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/oteltest"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/semconv"

	//nolint:staticcheck
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	grpc_codes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newTestServerListener(t testing.TB) (string, net.Listener) {
	addr := "\x00" + t.Name()
	listener, err := net.Listen("unix", addr)
	if err != nil {
		t.Fatal(err)
	}

	return addr, listener
}

func mustServer(t testing.TB) func(server *ttrpc.Server, err error) *ttrpc.Server {
	return func(server *ttrpc.Server, err error) *ttrpc.Server {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}

		return server
	}
}

// testingServer is what would be implemented by the user of this package.
type testingServer struct{}

func (s *testingServer) Test(ctx context.Context, req *testPayload) (*testPayload, error) {
	tp := &testPayload{Foo: strings.Repeat(req.Foo, 2)}

	return tp, nil
}

func newTestClient(t testing.TB, addr string, tp *oteltest.TracerProvider) (*ttrpc.Client, func()) {
	conn, err := net.Dial("unix", addr)
	if err != nil {
		t.Fatal(err)
	}
	client := ttrpc.NewClient(conn, ttrpc.WithUnaryClientInterceptor(UnaryClientInterceptor(WithTracerProvider(tp))))

	return client, func() {
		conn.Close()
		client.Close()
	}
}

type serviceClient struct {
	client *ttrpc.Client
}

func (tc *serviceClient) Test(ctx context.Context, req *testPayload) (*testPayload, error) {
	var tp testPayload
	return &tp, tc.client.Call(ctx, serviceName, "Test", req, &tp)
}

func newServiceClient(client *ttrpc.Client) *serviceClient {
	return &serviceClient{
		client: client,
	}
}

type testPayload struct {
	Foo      string `protobuf:"bytes,1,opt,name=foo,proto3"`
	Deadline int64  `protobuf:"varint,2,opt,name=deadline,proto3"`
	Metadata string `protobuf:"bytes,3,opt,name=metadata,proto3"`
}

func (r *testPayload) Reset()         { *r = testPayload{} }
func (r *testPayload) String() string { return fmt.Sprintf("%+#v", r) }
func (r *testPayload) ProtoMessage()  {}

// testingService is our prototype service definition for use in testing the full model.
//
// Typically, this is generated. We define it here to ensure that that package
// primitive has what is required for generated code.
type testingService interface {
	Test(ctx context.Context, req *testPayload) (*testPayload, error)
}

const serviceName = "testService"

// registerTestingService mocks more of what is generated code. Unlike grpc, we
// register with a closure so that the descriptor is allocated only on
// registration.
func registerTestingService(srv *ttrpc.Server, svc testingService) {
	srv.Register(serviceName, map[string]ttrpc.Method{
		"Test": func(ctx context.Context, unmarshal func(interface{}) error) (interface{}, error) {
			var req testPayload
			if err := unmarshal(&req); err != nil {
				return nil, err
			}
			return svc.Test(ctx, &req)
		},
	})
}
func TestClientCallServer(t *testing.T) {
	var (
		ctx = context.Background()

		sr = NewSpanRecorder()
		tp = oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))

		server          = mustServer(t)(ttrpc.NewServer(ttrpc.WithUnaryServerInterceptor(UnaryServerInterceptor(WithTracerProvider(tp)))))
		testImpl        = &testingServer{}
		addr, listener  = newTestServerListener(t)
		client, cleanup = newTestClient(t, addr, tp)
		svcClient       = newServiceClient(client)
		payload         = &testPayload{
			Foo: "bar",
		}
	)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	defer listener.Close()
	defer cleanup()

	registerTestingService(server, testImpl)

	go server.Serve(ctx, listener) //nolint
	defer server.Shutdown(ctx)     //nolint

	ctx = ttrpc.WithMetadata(ctx, ttrpc.MD{"foo": []string{"bar"}})

	_, err := svcClient.Test(ctx, payload)
	if err != nil {
		t.Fatal(err)
	}
}

type SpanRecorder struct {
	mu    sync.RWMutex
	spans map[string]*oteltest.Span
}

func NewSpanRecorder() *SpanRecorder {
	return &SpanRecorder{spans: make(map[string]*oteltest.Span)}
}

func (sr *SpanRecorder) OnStart(span *oteltest.Span) {}

func (sr *SpanRecorder) OnEnd(span *oteltest.Span) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	sr.spans[span.Name()] = span
}

func (sr *SpanRecorder) Get(name string) (*oteltest.Span, bool) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	s, ok := sr.spans[name]
	return s, ok
}

func invoker(_ctx context.Context, req *ttrpc.Request, _resp *ttrpc.Response) error {

	// if method contains error name, mock error return
	if strings.Contains(req.Method, "error") {
		return status.Error(grpc_codes.Internal, "internal error")
	}

	return nil
}

func TestUnaryClientInterceptor(t *testing.T) {
	sr := NewSpanRecorder()
	tp := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))
	unaryInterceptor := UnaryClientInterceptor(WithTracerProvider(tp))

	const bodySize = 36
	reqBody := make([]byte, bodySize)

	checks := []struct {
		service          string
		method           string
		name             string
		expectedSpanCode codes.Code
		expectedAttr     map[label.Key]label.Value
		expectErr        bool
	}{
		{
			service: "github.com.serviceName",
			method:  "bar",
			name:    "github.com.serviceName/bar",
			expectedAttr: map[label.Key]label.Value{
				semconv.RPCSystemKey:                        label.StringValue("grpc"),
				semconv.RPCServiceKey:                       label.StringValue("github.com.serviceName"),
				semconv.RPCMethodKey:                        label.StringValue("bar"),
				TTRPCStatusCodeKey:                          label.Uint32Value(0),
				semconv.MessagingProtocolKey:                label.StringValue(protocolName),
				semconv.MessagingMessagePayloadSizeBytesKey: label.IntValue(len(reqBody)),
			},
		},
		{
			service: "serviceName",
			method:  "bar",
			name:    "serviceName/bar",
			expectedAttr: map[label.Key]label.Value{
				semconv.RPCSystemKey:                        label.StringValue("grpc"),
				semconv.RPCServiceKey:                       label.StringValue("serviceName"),
				semconv.RPCMethodKey:                        label.StringValue("bar"),
				TTRPCStatusCodeKey:                          label.Uint32Value(0),
				semconv.MessagingProtocolKey:                label.StringValue(protocolName),
				semconv.MessagingMessagePayloadSizeBytesKey: label.IntValue(len(reqBody)),
			},
		},
		{
			service: "serviceName",
			method:  "bar",
			name:    "serviceName/bar",
			expectedAttr: map[label.Key]label.Value{
				semconv.RPCSystemKey:                        label.StringValue("grpc"),
				semconv.RPCServiceKey:                       label.StringValue("serviceName"),
				semconv.RPCMethodKey:                        label.StringValue("bar"),
				TTRPCStatusCodeKey:                          label.Uint32Value(uint32(grpc_codes.OK)),
				semconv.MessagingProtocolKey:                label.StringValue(protocolName),
				semconv.MessagingMessagePayloadSizeBytesKey: label.IntValue(len(reqBody)),
			},
		},
		{
			service:          "serviceName",
			method:           "bar_error",
			name:             "serviceName/bar_error",
			expectedSpanCode: codes.Error,
			expectedAttr: map[label.Key]label.Value{
				semconv.RPCSystemKey:                        label.StringValue("grpc"),
				semconv.RPCServiceKey:                       label.StringValue("serviceName"),
				semconv.RPCMethodKey:                        label.StringValue("bar_error"),
				TTRPCStatusCodeKey:                          label.Uint32Value(uint32(grpc_codes.Internal)),
				semconv.MessagingProtocolKey:                label.StringValue(protocolName),
				semconv.MessagingMessagePayloadSizeBytesKey: label.IntValue(len(reqBody)),
			},

			expectErr: true,
		},
		{
			service: "github.com.foo.serviceName_123",
			method:  "method",
			name:    "github.com.foo.serviceName_123/method",
			expectedAttr: map[label.Key]label.Value{
				semconv.RPCSystemKey:                        label.StringValue("grpc"),
				TTRPCStatusCodeKey:                          label.Uint32Value(0),
				semconv.RPCServiceKey:                       label.StringValue("github.com.foo.serviceName_123"),
				semconv.RPCMethodKey:                        label.StringValue("method"),
				semconv.MessagingProtocolKey:                label.StringValue(protocolName),
				semconv.MessagingMessagePayloadSizeBytesKey: label.IntValue(len(reqBody)),
			},
		},
	}

	for _, check := range checks {
		req := ttrpc.Request{
			Service: check.service,
			Method:  check.method,
			Payload: reqBody,
		}
		resp := ttrpc.Response{}
		u := ttrpc.UnaryClientInfo{
			FullMethod: check.method,
		}
		err := unaryInterceptor(context.Background(), &req, &resp, &u, invoker)
		if check.expectErr {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
		span, ok := sr.Get(check.name)
		if !assert.True(t, ok, "missing span %q", check.name) {
			continue
		}
		assert.Equal(t, check.expectedSpanCode, span.StatusCode())
		assert.Equal(t, check.expectedAttr, span.Attributes())
	}
}

func TestServerInterceptorError(t *testing.T) {
	sr := NewSpanRecorder()
	tp := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))
	usi := UnaryServerInterceptor(WithTracerProvider(tp))
	deniedErr := status.Error(grpc_codes.PermissionDenied, "PERMISSION_DENIED_TEXT")
	handler := func(ctx context.Context, unmarshal func(interface{}) error) (interface{}, error) {
		return nil, deniedErr
	}

	info := ttrpc.UnaryServerInfo{
		FullMethod: "/service/method",
	}

	um := func(interface{}) error {
		return nil
	}

	_, err := usi(context.Background(), um, &info, handler)
	require.Error(t, err)
	assert.Equal(t, err, deniedErr)

	span, ok := sr.Get("service/method")
	if !ok {
		t.Fatalf("failed to export error span")
	}
	assert.Equal(t, codes.Error, span.StatusCode())
	assert.Contains(t, deniedErr.Error(), span.StatusMessage())
	codeAttr, ok := span.Attributes()[TTRPCStatusCodeKey]
	assert.True(t, ok, "attributes contain gRPC status code")
	assert.Equal(t, label.Uint32Value(uint32(grpc_codes.PermissionDenied)), codeAttr)

}

func TestServerSpanInfo(t *testing.T) {
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
				semconv.RPCSystemGRPC,
				semconv.MessagingProtocolKey.String(protocolName),
			},
		}, {
			fullMethod: "/com.example.ExampleRmiService/exampleMethod",
			name:       "com.example.ExampleRmiService/exampleMethod",
			attr: []label.KeyValue{
				semconv.RPCServiceKey.String("com.example.ExampleRmiService"),
				semconv.RPCMethodKey.String("exampleMethod"),
				semconv.RPCSystemGRPC,
				semconv.MessagingProtocolKey.String(protocolName),
			},
		}, {
			fullMethod: "/MyCalcService.Calculator/Add",
			name:       "MyCalcService.Calculator/Add",
			attr: []label.KeyValue{
				semconv.RPCServiceKey.String("MyCalcService.Calculator"),
				semconv.RPCMethodKey.String("Add"),
				semconv.RPCSystemGRPC,
				semconv.MessagingProtocolKey.String(protocolName),
			},
		}, {
			fullMethod: "/MyServiceReference.ICalculator/Add",
			name:       "MyServiceReference.ICalculator/Add",
			attr: []label.KeyValue{
				semconv.RPCServiceKey.String("MyServiceReference.ICalculator"),
				semconv.RPCMethodKey.String("Add"),
				semconv.RPCSystemGRPC,
				semconv.MessagingProtocolKey.String(protocolName),
			},
		}, {
			fullMethod: "/MyServiceWithNoPackage/theMethod",
			name:       "MyServiceWithNoPackage/theMethod",
			attr: []label.KeyValue{
				semconv.RPCServiceKey.String("MyServiceWithNoPackage"),
				semconv.RPCMethodKey.String("theMethod"),
				semconv.RPCSystemGRPC,
				semconv.MessagingProtocolKey.String(protocolName),
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
				semconv.RPCMethodKey.String(""),
				semconv.RPCSystemGRPC,
				semconv.MessagingProtocolKey.String(protocolName),
			},
		},
	}

	for _, test := range tests {
		n, a := serverSpanInfo(test.fullMethod)
		assert.Equal(t, test.name, n)
		assert.Equal(t, test.attr, a)
	}
}
