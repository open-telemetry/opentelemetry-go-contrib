// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"context"
	"net"
	"strings"
	"testing"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	grpc_codes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
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
	clientConn, err := grpc.NewClient("fake:8906",
		grpc.WithContextDialer(ctxDialer()),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("failed to create client connection: %v", err)
	}
	defer clientConn.Close()

	sr := tracetest.NewSpanRecorder()
	tp := trace.NewTracerProvider(trace.WithSpanProcessor(sr))
	//nolint:staticcheck // Interceptors are deprecated and will be removed in the next release.
	unaryInterceptor := otelgrpc.UnaryClientInterceptor(
		otelgrpc.WithTracerProvider(tp),
		otelgrpc.WithMessageEvents(otelgrpc.ReceivedEvents, otelgrpc.SentEvents),
		otelgrpc.WithSpanOptions(oteltrace.WithAttributes(attribute.Bool("custom", true))),
	)
	//nolint:staticcheck // Interceptors are deprecated and will be removed in the next release.
	unaryInterceptorOnlySentEvents := otelgrpc.UnaryClientInterceptor(
		otelgrpc.WithTracerProvider(tp),
		otelgrpc.WithMessageEvents(otelgrpc.SentEvents),
	)
	//nolint:staticcheck // Interceptors are deprecated and will be removed in the next release.
	unaryInterceptorOnlyReceivedEvents := otelgrpc.UnaryClientInterceptor(
		otelgrpc.WithTracerProvider(tp),
		otelgrpc.WithMessageEvents(otelgrpc.ReceivedEvents),
	)
	//nolint:staticcheck // Interceptors are deprecated and will be removed in the next release.
	unaryInterceptorNoEvents := otelgrpc.UnaryClientInterceptor(
		otelgrpc.WithTracerProvider(tp),
	)

	req := &grpc_testing.SimpleRequest{}
	reply := &grpc_testing.SimpleResponse{}
	uniInterceptorInvoker := &mockUICInvoker{}

	checks := []struct {
		method           string
		name             string
		interceptor      grpc.UnaryClientInterceptor
		expectedSpanCode codes.Code
		expectedAttr     []attribute.KeyValue
		eventsAttr       []map[attribute.Key]attribute.Value
		expectErr        bool
	}{
		{
			method:      "/github.com.serviceName/bar",
			name:        "github.com.serviceName/bar",
			interceptor: unaryInterceptor,
			expectedAttr: []attribute.KeyValue{
				semconv.RPCSystemGRPC,
				semconv.RPCService("github.com.serviceName"),
				semconv.RPCMethod("bar"),
				otelgrpc.GRPCStatusCodeKey.Int64(0),
				semconv.NetPeerName("fake"),
				semconv.NetPeerPort(8906),
				attribute.Bool("custom", true),
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
			method:      "/serviceName/bar",
			name:        "serviceName/bar",
			interceptor: unaryInterceptor,
			expectedAttr: []attribute.KeyValue{
				semconv.RPCSystemGRPC,
				semconv.RPCService("serviceName"),
				semconv.RPCMethod("bar"),
				otelgrpc.GRPCStatusCodeKey.Int64(0),
				semconv.NetPeerName("fake"),
				semconv.NetPeerPort(8906),
				attribute.Bool("custom", true),
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
			method:      "/serviceName/bar_onlysentevents",
			name:        "serviceName/bar_onlysentevents",
			interceptor: unaryInterceptorOnlySentEvents,
			expectedAttr: []attribute.KeyValue{
				semconv.RPCSystemGRPC,
				semconv.RPCService("serviceName"),
				semconv.RPCMethod("bar_onlysentevents"),
				otelgrpc.GRPCStatusCodeKey.Int64(0),
				semconv.NetPeerName("fake"),
				semconv.NetPeerPort(8906),
			},
			eventsAttr: []map[attribute.Key]attribute.Value{
				{
					otelgrpc.RPCMessageTypeKey: attribute.StringValue("SENT"),
					otelgrpc.RPCMessageIDKey:   attribute.IntValue(1),
				},
			},
		},
		{
			method:      "/serviceName/bar_onlyreceivedevents",
			name:        "serviceName/bar_onlyreceivedevents",
			interceptor: unaryInterceptorOnlyReceivedEvents,
			expectedAttr: []attribute.KeyValue{
				semconv.RPCSystemGRPC,
				semconv.RPCService("serviceName"),
				semconv.RPCMethod("bar_onlyreceivedevents"),
				otelgrpc.GRPCStatusCodeKey.Int64(0),
				semconv.NetPeerName("fake"),
				semconv.NetPeerPort(8906),
			},
			eventsAttr: []map[attribute.Key]attribute.Value{
				{
					otelgrpc.RPCMessageTypeKey: attribute.StringValue("RECEIVED"),
					otelgrpc.RPCMessageIDKey:   attribute.IntValue(1),
				},
			},
		},
		{
			method:      "/serviceName/bar_noevents",
			name:        "serviceName/bar_noevents",
			interceptor: unaryInterceptorNoEvents,
			expectedAttr: []attribute.KeyValue{
				semconv.RPCSystemGRPC,
				semconv.RPCService("serviceName"),
				semconv.RPCMethod("bar_noevents"),
				otelgrpc.GRPCStatusCodeKey.Int64(0),
				semconv.NetPeerName("fake"),
				semconv.NetPeerPort(8906),
			},
			eventsAttr: []map[attribute.Key]attribute.Value{},
		},
		{
			method:      "/serviceName/bar",
			name:        "serviceName/bar",
			interceptor: unaryInterceptor,
			expectedAttr: []attribute.KeyValue{
				semconv.RPCSystemGRPC,
				semconv.RPCService("serviceName"),
				semconv.RPCMethod("bar"),
				otelgrpc.GRPCStatusCodeKey.Int64(int64(grpc_codes.OK)),
				semconv.NetPeerName("fake"),
				semconv.NetPeerPort(8906),
				attribute.Bool("custom", true),
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
			method:           "/serviceName/bar_error",
			name:             "serviceName/bar_error",
			interceptor:      unaryInterceptor,
			expectedSpanCode: codes.Error,
			expectedAttr: []attribute.KeyValue{
				semconv.RPCSystemGRPC,
				semconv.RPCService("serviceName"),
				semconv.RPCMethod("bar_error"),
				otelgrpc.GRPCStatusCodeKey.Int64(int64(grpc_codes.Internal)),
				semconv.NetPeerName("fake"),
				semconv.NetPeerPort(8906),
				attribute.Bool("custom", true),
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
			method:      "invalidName",
			name:        "invalidName",
			interceptor: unaryInterceptor,
			expectedAttr: []attribute.KeyValue{
				semconv.RPCSystemGRPC,
				otelgrpc.GRPCStatusCodeKey.Int64(0),
				semconv.NetPeerName("fake"),
				semconv.NetPeerPort(8906),
				attribute.Bool("custom", true),
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
			method:      "/github.com.foo.serviceName_123/method",
			name:        "github.com.foo.serviceName_123/method",
			interceptor: unaryInterceptor,
			expectedAttr: []attribute.KeyValue{
				semconv.RPCSystemGRPC,
				otelgrpc.GRPCStatusCodeKey.Int64(0),
				semconv.RPCService("github.com.foo.serviceName_123"),
				semconv.RPCMethod("method"),
				semconv.NetPeerName("fake"),
				semconv.NetPeerPort(8906),
				attribute.Bool("custom", true),
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
		err := check.interceptor(context.Background(), check.method, req, reply, clientConn, uniInterceptorInvoker.invoker)
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

// serverChecks can't be removed in #7105. Should be removed when issue #7106 is fixed.
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
	for _, check := range serverChecks {
		name := check.grpcCode.String()
		t.Run(name, func(t *testing.T) {
			t.Setenv("OTEL_METRICS_EXEMPLAR_FILTER", "always_off")
			sr := tracetest.NewSpanRecorder()
			tp := trace.NewTracerProvider(trace.WithSpanProcessor(sr))

			mr := metric.NewManualReader()
			mp := metric.NewMeterProvider(metric.WithReader(mr))

			//nolint:staticcheck // Interceptors are deprecated and will be removed in the next release.
			usi := otelgrpc.UnaryServerInterceptor(
				otelgrpc.WithTracerProvider(tp),
				otelgrpc.WithMeterProvider(mp),
			)

			serviceName := "TestGrpcService"
			methodName := serviceName + "/" + name
			fullMethodName := "/" + methodName
			// call the unary interceptor
			grpcErr := status.Error(check.grpcCode, check.grpcCode.String())
			handler := func(_ context.Context, _ interface{}) (interface{}, error) {
				return nil, grpcErr
			}
			_, err := usi(context.Background(), &grpc_testing.SimpleRequest{}, &grpc.UnaryServerInfo{FullMethod: fullMethodName}, handler)
			assert.Equal(t, grpcErr, err)

			// validate span
			span, ok := getSpanFromRecorder(sr, methodName)
			require.True(t, ok, "missing span %s", methodName)
			assertServerSpan(t, check.wantSpanCode, check.wantSpanStatusDescription, check.grpcCode, span)

			// validate metric
			assertServerMetrics(t, mr, serviceName, name, check.grpcCode)
		})
	}
}

func TestUnaryServerInterceptorEvents(t *testing.T) {
	testCases := []struct {
		Name   string
		Events []otelgrpc.Event
	}{
		{
			Name:   "No events",
			Events: []otelgrpc.Event{},
		},
		{
			Name:   "With only received events",
			Events: []otelgrpc.Event{otelgrpc.ReceivedEvents},
		},
		{
			Name:   "With only sent events",
			Events: []otelgrpc.Event{otelgrpc.SentEvents},
		},
		{
			Name:   "With both events",
			Events: []otelgrpc.Event{otelgrpc.ReceivedEvents, otelgrpc.SentEvents},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			tp := trace.NewTracerProvider(trace.WithSpanProcessor(sr))
			opts := []otelgrpc.Option{
				otelgrpc.WithTracerProvider(tp),
			}
			if len(testCase.Events) > 0 {
				opts = append(opts, otelgrpc.WithMessageEvents(testCase.Events...))
			}
			//nolint:staticcheck // Interceptors are deprecated and will be removed in the next release.
			usi := otelgrpc.UnaryServerInterceptor(opts...)
			grpcCode := grpc_codes.OK
			name := grpcCode.String()
			// call the unary interceptor
			grpcErr := status.Error(grpcCode, name)
			handler := func(_ context.Context, _ interface{}) (interface{}, error) {
				return nil, grpcErr
			}
			_, err := usi(context.Background(), &grpc_testing.SimpleRequest{}, &grpc.UnaryServerInfo{FullMethod: name}, handler)
			assert.Equal(t, grpcErr, err)

			// validate span
			span, ok := getSpanFromRecorder(sr, name)
			require.True(t, ok, "missing span %s", name)

			// validate events and their attributes
			if len(testCase.Events) == 0 {
				assert.Empty(t, span.Events())
			} else {
				assert.Len(t, span.Events(), len(testCase.Events))
				for i, event := range testCase.Events {
					switch event {
					case otelgrpc.ReceivedEvents:
						assert.ElementsMatch(t, []attribute.KeyValue{
							attribute.Key("message.type").String("RECEIVED"),
							attribute.Key("message.id").Int(1),
						}, span.Events()[i].Attributes)
					case otelgrpc.SentEvents:
						assert.ElementsMatch(t, []attribute.KeyValue{
							attribute.Key("message.type").String("SENT"),
							attribute.Key("message.id").Int(1),
						}, span.Events()[i].Attributes)
					}
				}
			}
		})
	}
}

type mockServerStream struct {
	grpc.ServerStream
}

func (m *mockServerStream) Context() context.Context { return context.Background() }

func (m *mockServerStream) SendMsg(_ interface{}) error {
	return nil
}

func (m *mockServerStream) RecvMsg(_ interface{}) error {
	return nil
}

// TestStreamServerInterceptor tests the server interceptor for streaming RPCs.
func TestStreamServerInterceptor(t *testing.T) {
	for _, check := range serverChecks {
		name := check.grpcCode.String()
		t.Run(name, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			tp := trace.NewTracerProvider(trace.WithSpanProcessor(sr))

			//nolint:staticcheck // Interceptors are deprecated and will be removed in the next release.
			usi := otelgrpc.StreamServerInterceptor(
				otelgrpc.WithTracerProvider(tp),
			)

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

func TestStreamServerInterceptorEvents(t *testing.T) {
	testCases := []struct {
		Name   string
		Events []otelgrpc.Event
	}{
		{Name: "With events", Events: []otelgrpc.Event{otelgrpc.ReceivedEvents, otelgrpc.SentEvents}},
		{Name: "With only sent events", Events: []otelgrpc.Event{otelgrpc.SentEvents}},
		{Name: "With only received events", Events: []otelgrpc.Event{otelgrpc.ReceivedEvents}},
		{Name: "No events", Events: []otelgrpc.Event{}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			tp := trace.NewTracerProvider(trace.WithSpanProcessor(sr))
			opts := []otelgrpc.Option{
				otelgrpc.WithTracerProvider(tp),
			}
			if len(testCase.Events) > 0 {
				opts = append(opts, otelgrpc.WithMessageEvents(testCase.Events...))
			}
			//nolint:staticcheck // Interceptors are deprecated and will be removed in the next release.
			usi := otelgrpc.StreamServerInterceptor(opts...)
			stream := &mockServerStream{}

			grpcCode := grpc_codes.OK
			name := grpcCode.String()
			// call the stream interceptor
			grpcErr := status.Error(grpcCode, name)
			handler := func(_ interface{}, handlerStream grpc.ServerStream) error {
				var msg grpc_testing.SimpleRequest
				err := handlerStream.RecvMsg(&msg)
				require.NoError(t, err)
				err = handlerStream.SendMsg(&msg)
				require.NoError(t, err)
				return grpcErr
			}

			err := usi(&grpc_testing.SimpleRequest{}, stream, &grpc.StreamServerInfo{FullMethod: name}, handler)
			require.Equal(t, grpcErr, err)

			// validate span
			span, ok := getSpanFromRecorder(sr, name)
			require.True(t, ok, "missing span %s", name)

			if len(testCase.Events) == 0 {
				assert.Empty(t, span.Events())
			} else {
				var eventsAttr []map[attribute.Key]attribute.Value
				for _, event := range testCase.Events {
					switch event {
					case otelgrpc.SentEvents:
						eventsAttr = append(eventsAttr, map[attribute.Key]attribute.Value{
							otelgrpc.RPCMessageTypeKey: attribute.StringValue("SENT"),
							otelgrpc.RPCMessageIDKey:   attribute.IntValue(1),
						})
					case otelgrpc.ReceivedEvents:
						eventsAttr = append(eventsAttr, map[attribute.Key]attribute.Value{
							otelgrpc.RPCMessageTypeKey: attribute.StringValue("RECEIVED"),
							otelgrpc.RPCMessageIDKey:   attribute.IntValue(1),
						})
					}
				}
				assert.Len(t, span.Events(), len(eventsAttr))
				assert.Equal(t, eventsAttr, eventAttrMap(span.Events()))
			}
		})
	}
}

func assertServerMetrics(t *testing.T, reader metric.Reader, serviceName, name string, code grpc_codes.Code) {
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
								semconv.RPCMethod(name),
								semconv.RPCService(serviceName),
								otelgrpc.RPCSystemGRPC,
								otelgrpc.GRPCStatusCodeKey.Int64(int64(code)),
							),
						},
					},
				},
			},
		},
	}
	rm := metricdata.ResourceMetrics{}
	err := reader.Collect(context.Background(), &rm)
	assert.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	metricdatatest.AssertEqual(t, want, rm.ScopeMetrics[0], metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue())
}
