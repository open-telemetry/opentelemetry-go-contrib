// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgrpc_test

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	testpb "google.golang.org/grpc/interop/grpc_testing"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/stats"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

// testLabelerServer is a test server that implements the test service.
type testLabelerServer struct {
	testpb.UnimplementedTestServiceServer
}

// EmptyCall is a test method that returns an empty response.
func (*testLabelerServer) EmptyCall(_ context.Context, _ *testpb.Empty) (*testpb.Empty, error) {
	return &testpb.Empty{}, nil
}

// UnaryCall is a test method that returns a simple response.
func (*testLabelerServer) UnaryCall(_ context.Context, _ *testpb.SimpleRequest) (*testpb.SimpleResponse, error) {
	return &testpb.SimpleResponse{}, nil
}

// StreamingInputCall is a test method that implements a client-side streaming RPC.
func (*testLabelerServer) StreamingInputCall(stream testpb.TestService_StreamingInputCallServer) error {
	for {
		_, err := stream.Recv()
		if err != nil {
			switch {
			case errors.Is(err, io.EOF):
				return stream.SendAndClose(&testpb.StreamingInputCallResponse{})
			default:
				return err
			}
		}
	}
}

// StreamingOutputCall is a test method that implements a server-side streaming RPC.
func (*testLabelerServer) StreamingOutputCall(req *testpb.StreamingOutputCallRequest, stream testpb.TestService_StreamingOutputCallServer) error {
	for _, param := range req.ResponseParameters {
		payload := &testpb.Payload{
			Type: testpb.PayloadType_COMPRESSABLE,
			Body: make([]byte, param.Size),
		}
		if err := stream.Send(&testpb.StreamingOutputCallResponse{Payload: payload}); err != nil {
			return err
		}
	}

	return nil
}

const (
	serverLabelingDirection = iota
	clientLabelingDirection
)

// TestMetricAttributesFn_ServerSide tests that labels are added to server-side metrics for unary RPCs.
func TestMetricAttributesFn_ServerSide(t *testing.T) {
	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))

	lis, server := startTestServerWithOptions(t, mp, otelgrpc.WithMetricAttributesFn(func(ctx context.Context) []attribute.KeyValue {
		md, ok := metadata.FromIncomingContext(ctx)
		var origin string
		if ok {
			originVals := md.Get("x-origin")
			if len(originVals) > 0 {
				origin = originVals[0]
			}
		}

		return []attribute.KeyValue{
			attribute.String("origin", origin),
			attribute.String("tier", "premium"),
		}
	}))
	defer server.Stop()

	ctx := metadata.NewOutgoingContext(t.Context(), metadata.Pairs("x-origin", "dynamic-origin"))
	client := createTestClient(t, lis.Addr().String(), nil, nil)
	_, err := client.EmptyCall(ctx, &testpb.Empty{})
	require.NoError(t, err)

	assertAllMetricsHaveLabels(t, reader, serverLabelingDirection, map[string]string{
		"origin": "dynamic-origin",
		"tier":   "premium",
	})
}

// TestMetricAttributesFn_ServerSideStreaming tests that labels are added to server-side metrics for server-side streaming RPCs.
func TestMetricAttributesFn_ServerSideStreaming(t *testing.T) {
	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))

	lis, server := startTestServerWithOptions(t, mp, otelgrpc.WithMetricAttributesFn(func(ctx context.Context) []attribute.KeyValue {
		md, ok := metadata.FromIncomingContext(ctx)
		var origin string
		if ok {
			originVals := md.Get("x-origin")
			if len(originVals) > 0 {
				origin = originVals[0]
			}
		}
		return []attribute.KeyValue{
			attribute.String("origin", origin),
			attribute.String("tier", "streaming"),
		}
	}))
	defer server.Stop()

	ctx := metadata.NewOutgoingContext(t.Context(), metadata.Pairs("x-origin", "dynamic-stream-origin"))
	client := createTestClient(t, lis.Addr().String(), nil, nil)

	stream, err := client.StreamingOutputCall(ctx, &testpb.StreamingOutputCallRequest{
		ResponseParameters: []*testpb.ResponseParameters{
			{Size: 1}, {Size: 2},
		},
	})
	require.NoError(t, err)

	var count int
	for {
		_, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		count++
	}
	require.Equal(t, 2, count)

	assertAllMetricsHaveLabels(t, reader, serverLabelingDirection, map[string]string{
		"origin": "dynamic-stream-origin",
		"tier":   "streaming",
	})
}

// TestMetricAttributesFn_ServerSide_Baggage tests that baggage can be used on the server-side to populate context values for MetricAttributesFn.
func TestMetricAttributesFn_ServerSide_Baggage(t *testing.T) {
	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))

	metricFunc := func(ctx context.Context) []attribute.KeyValue {
		bag := baggage.FromContext(ctx)
		if tier := bag.Member("tenant.tier"); tier.Value() != "" {
			return []attribute.KeyValue{
				attribute.String("tenant.tier", tier.Value()),
			}
		}
		return []attribute.KeyValue{
			attribute.String("tenant.tier", "NOT_FOUND"),
		}
	}

	lis, server := startTestServerWithOptions(t, mp,
		otelgrpc.WithMetricAttributesFn(metricFunc),
		otelgrpc.WithPropagators(propagation.NewCompositeTextMapPropagator(
			propagation.Baggage{},
		)),
	)
	defer server.Stop()

	conn, err := grpc.NewClient(
		lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler(
			otelgrpc.WithPropagators(propagation.NewCompositeTextMapPropagator(
				propagation.Baggage{},
			)),
		)),
	)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	client := testpb.NewTestServiceClient(conn)

	member, err := baggage.NewMember("tenant.tier", "premium")
	require.NoError(t, err)
	bag, err := baggage.New(member)
	require.NoError(t, err)
	ctx := baggage.ContextWithBaggage(t.Context(), bag)

	_, err = client.EmptyCall(ctx, &testpb.Empty{})
	require.NoError(t, err)

	assertAllMetricsHaveLabels(t, reader, serverLabelingDirection, map[string]string{
		"tenant.tier": "premium",
	})
}

type enrichedServerHandler struct {
	stats.Handler
}

type tenantTierKeyType struct{}

var tenantTierKey tenantTierKeyType

// TagRPC overrides the TagRPC method of the stats handler to add tenant tier to context.
func (h *enrichedServerHandler) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	ctx = context.WithValue(ctx, tenantTierKey, "premium")
	return h.Handler.TagRPC(ctx, info)
}

// TestMetricAttributesFn_ServerSide_WithWrappedHandler tests that a wrapped stats handler
// can populate context values for MetricAttributesFn.
func TestMetricAttributesFn_ServerSide_WithWrappedHandler(t *testing.T) {
	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))

	metricFunc := func(ctx context.Context) []attribute.KeyValue {
		if tier, ok := ctx.Value(tenantTierKey).(string); ok {
			return []attribute.KeyValue{
				attribute.String("tenant.tier", tier),
			}
		}
		return []attribute.KeyValue{
			attribute.String("tenant.tier", "NOT_FOUND"),
		}
	}

	lis, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)

	wrappedHandler := &enrichedServerHandler{
		Handler: otelgrpc.NewServerHandler(
			otelgrpc.WithMeterProvider(mp),
			otelgrpc.WithMetricAttributesFn(metricFunc),
		),
	}

	server := grpc.NewServer(
		grpc.StatsHandler(wrappedHandler),
	)
	testpb.RegisterTestServiceServer(server, &testLabelerServer{})

	go func() {
		if err := server.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			t.Errorf("server failed: %v", err)
		}
	}()
	defer server.Stop()

	client := createTestClient(t, lis.Addr().String(), nil, nil)
	_, err = client.EmptyCall(t.Context(), &testpb.Empty{})
	require.NoError(t, err)

	assertAllMetricsHaveLabels(t, reader, serverLabelingDirection, map[string]string{
		"tenant.tier": "premium",
	})
}

// TestMetricAttributesFn_ClientSide tests that labels are added to client-side metrics for unary RPCs.
func TestMetricAttributesFn_ClientSide(t *testing.T) {
	serverLis, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)

	server := grpc.NewServer()
	testpb.RegisterTestServiceServer(server, &testLabelerServer{})

	go func() {
		if err := server.Serve(serverLis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			t.Errorf("server failed: %v", err)
		}
	}()
	defer server.Stop()

	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))

	type rpcServiceKeyType struct{}
	var rpcServiceKey rpcServiceKeyType

	dynamicServiceName := "orders-service"
	var interceptor grpc.UnaryClientInterceptor = func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx = context.WithValue(ctx, rpcServiceKey, dynamicServiceName)
		return invoker(ctx, method, req, reply, cc, opts...)
	}

	metricFunc := func(ctx context.Context) []attribute.KeyValue {
		if svc, ok := ctx.Value(rpcServiceKey).(string); ok {
			return []attribute.KeyValue{
				attribute.String("rpc.service", svc),
				attribute.String("client.version", "v1.2.3"),
			}
		}

		return []attribute.KeyValue{
			attribute.String("client.version", "v1.2.3"),
		}
	}

	client := createTestClient(t, serverLis.Addr().String(), mp, metricFunc, interceptor)

	_, err = client.EmptyCall(t.Context(), &testpb.Empty{})
	require.NoError(t, err)

	assertAllMetricsHaveLabels(t, reader, clientLabelingDirection, map[string]string{
		"client.version": "v1.2.3",
		"rpc.service":    dynamicServiceName,
	})
}

// TestMetricAttributesFn_ClientSideStreaming tests that labels are added to client-side metrics for client-side streaming RPCs.
func TestMetricAttributesFn_ClientSideStreaming(t *testing.T) {
	serverLis, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)

	server := grpc.NewServer()
	testpb.RegisterTestServiceServer(server, &testLabelerServer{})
	go func() {
		if err := server.Serve(serverLis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			t.Errorf("server failed: %v", err)
		}
	}()
	defer server.Stop()

	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))

	type rpcServiceKeyType struct{}
	var rpcServiceKey rpcServiceKeyType
	dynamicServiceName := "orders-service"

	var interceptor grpc.StreamClientInterceptor = func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		ctx = context.WithValue(ctx, rpcServiceKey, dynamicServiceName)
		return streamer(ctx, desc, cc, method, opts...)
	}

	metricFunc := func(ctx context.Context) []attribute.KeyValue {
		if svc, ok := ctx.Value(rpcServiceKey).(string); ok {
			return []attribute.KeyValue{
				attribute.String("rpc.service", svc),
				attribute.String("client.version", "v2.0.0"),
			}
		}
		return []attribute.KeyValue{
			attribute.String("client.version", "v2.0.0"),
		}
	}

	client := createTestClient(t, serverLis.Addr().String(), mp, metricFunc, interceptor)

	stream, err := client.StreamingInputCall(t.Context())
	require.NoError(t, err)

	for range 3 {
		err := stream.Send(&testpb.StreamingInputCallRequest{
			Payload: &testpb.Payload{Body: []byte("hello")},
		})
		require.NoError(t, err)
	}

	_, err = stream.CloseAndRecv()
	require.NoError(t, err)

	assertAllMetricsHaveLabels(t, reader, clientLabelingDirection, map[string]string{
		"client.version": "v2.0.0",
		"rpc.service":    dynamicServiceName,
	})
}

// TestMetricAttributesFn_ClientSide_Baggage tests that baggage can be used on the client-side to populate context values for MetricAttributesFn.
func TestMetricAttributesFn_ClientSide_Baggage(t *testing.T) {
	serverLis, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)

	server := grpc.NewServer()
	testpb.RegisterTestServiceServer(server, &testLabelerServer{})

	go func() {
		if err := server.Serve(serverLis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			t.Errorf("server failed: %v", err)
		}
	}()
	defer server.Stop()

	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))

	metricFunc := func(ctx context.Context) []attribute.KeyValue {
		bag := baggage.FromContext(ctx)
		if env := bag.Member("environment"); env.Value() != "" {
			return []attribute.KeyValue{
				attribute.String("environment", env.Value()),
			}
		}
		return []attribute.KeyValue{
			attribute.String("environment", "NOT_FOUND"),
		}
	}

	client := createTestClient(t, serverLis.Addr().String(), mp, metricFunc)

	member, err := baggage.NewMember("environment", "staging")
	require.NoError(t, err)
	bag, err := baggage.New(member)
	require.NoError(t, err)
	ctx := baggage.ContextWithBaggage(t.Context(), bag)

	_, err = client.EmptyCall(ctx, &testpb.Empty{})
	require.NoError(t, err)

	assertAllMetricsHaveLabels(t, reader, clientLabelingDirection, map[string]string{
		"environment": "staging",
	})
}

type enrichedClientHandler struct {
	stats.Handler
}

type rpcServiceKeyType struct{}

var rpcServiceKey rpcServiceKeyType

// TagRPC overrides the TagRPC method of the stats handler to add service name to context.
func (h *enrichedClientHandler) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	ctx = context.WithValue(ctx, rpcServiceKey, "orders-service-wrapped")
	return h.Handler.TagRPC(ctx, info)
}

// TestMetricAttributesFn_ClientSide_WithWrappedHandler tests that a wrapped client stats handler
// can populate context values for MetricAttributesFn.
func TestMetricAttributesFn_ClientSide_WithWrappedHandler(t *testing.T) {
	serverLis, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)

	server := grpc.NewServer()
	testpb.RegisterTestServiceServer(server, &testLabelerServer{})

	go func() {
		if err := server.Serve(serverLis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			t.Errorf("server failed: %v", err)
		}
	}()
	defer server.Stop()

	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))

	metricFunc := func(ctx context.Context) []attribute.KeyValue {
		if svc, ok := ctx.Value(rpcServiceKey).(string); ok {
			return []attribute.KeyValue{
				attribute.String("rpc.service", svc),
			}
		}
		return []attribute.KeyValue{
			attribute.String("rpc.service", "NOT_FOUND"),
		}
	}

	wrappedHandler := &enrichedClientHandler{
		Handler: otelgrpc.NewClientHandler(
			otelgrpc.WithMeterProvider(mp),
			otelgrpc.WithMetricAttributesFn(metricFunc),
		),
	}

	conn, err := grpc.NewClient(
		serverLis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(wrappedHandler),
	)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	client := testpb.NewTestServiceClient(conn)
	_, err = client.EmptyCall(t.Context(), &testpb.Empty{})
	require.NoError(t, err)

	assertAllMetricsHaveLabels(t, reader, clientLabelingDirection, map[string]string{
		"rpc.service": "orders-service-wrapped",
	})
}

// TestMetricAttributesFn_ClientAndServerIndependent tests that labels are separated between the client- and the server-side metrics.
func TestMetricAttributesFn_ClientAndServerIndependent(t *testing.T) {
	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))

	// Dowstream server (the main server acts as a client to this server)
	downstreamLis, downstreamServer := startTestServerWithOptions(t, nil)
	defer downstreamServer.Stop()

	// Main server setup
	lis, server := startTestServerWithOptions(t, mp, otelgrpc.WithMetricAttributesFn(func(_ context.Context) []attribute.KeyValue {
		return []attribute.KeyValue{
			attribute.String("origin", "test-origin"),
			attribute.String("tier", "premium"),
		}
	}))
	defer server.Stop()

	metricFunc := func(_ context.Context) []attribute.KeyValue {
		return []attribute.KeyValue{
			attribute.String("client.version", "v1.1.1"),
			attribute.String("client.env", "staging"),
		}
	}
	downstreamClient := createTestClient(t, downstreamLis.Addr().String(), mp, metricFunc)

	// Client for the main server, triggering the flow (client -> server -> downstreamServer)
	var interceptor grpc.UnaryClientInterceptor = func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if method == "/grpc.testing.TestService/EmptyCall" {
			_, _ = downstreamClient.EmptyCall(ctx, &testpb.Empty{})
		}

		return invoker(ctx, method, req, reply, cc, opts...)
	}
	client := createTestClient(t, lis.Addr().String(), nil, nil, interceptor)

	_, err := client.EmptyCall(t.Context(), &testpb.Empty{})
	require.NoError(t, err)

	assertAllMetricsHaveLabels(t, reader, serverLabelingDirection, map[string]string{
		"origin": "test-origin",
		"tier":   "premium",
	})

	assertAllMetricsHaveLabels(t, reader, clientLabelingDirection, map[string]string{
		"client.version": "v1.1.1",
		"client.env":     "staging",
	})

	assertAllMetricsDoNotHaveLabels(t, reader, serverLabelingDirection, map[string]string{
		"client.version": "v1.1.1",
		"client.env":     "staging",
	})

	assertAllMetricsDoNotHaveLabels(t, reader, clientLabelingDirection, map[string]string{
		"origin": "test-origin",
		"tier":   "premium",
	})
}

func startTestServerWithOptions(t *testing.T, mp *metric.MeterProvider, opts ...otelgrpc.Option) (net.Listener, *grpc.Server) {
	t.Helper()

	lis, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)

	var allOpts []otelgrpc.Option
	if mp != nil {
		allOpts = append([]otelgrpc.Option{otelgrpc.WithMeterProvider(mp)}, opts...)
	}
	server := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler(allOpts...)),
	)
	testpb.RegisterTestServiceServer(server, &testLabelerServer{})

	go func() {
		if err := server.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			t.Errorf("server failed: %v", err)
		}
	}()

	return lis, server
}

func createTestClient(t *testing.T, addr string, mp *metric.MeterProvider, metricFunc func(ctx context.Context) []attribute.KeyValue, interceptors ...any) testpb.TestServiceClient {
	t.Helper()

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	var unaryInterceptors []grpc.UnaryClientInterceptor
	var streamInterceptors []grpc.StreamClientInterceptor

	for _, ic := range interceptors {
		switch v := ic.(type) {
		case grpc.UnaryClientInterceptor:
			unaryInterceptors = append(unaryInterceptors, v)
		case grpc.StreamClientInterceptor:
			streamInterceptors = append(streamInterceptors, v)
		default:
			t.Fatalf("unsupported interceptor type: %T", v)
		}
	}

	if len(unaryInterceptors) > 0 {
		opts = append(opts, grpc.WithChainUnaryInterceptor(unaryInterceptors...))
	}
	if len(streamInterceptors) > 0 {
		opts = append(opts, grpc.WithChainStreamInterceptor(streamInterceptors...))
	}

	if mp != nil && metricFunc != nil {
		opts = append(opts,
			grpc.WithStatsHandler(
				otelgrpc.NewClientHandler(
					otelgrpc.WithMeterProvider(mp),
					otelgrpc.WithMetricAttributesFn(metricFunc),
				),
			),
		)
	}

	conn, err := grpc.NewClient(addr, opts...)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	return testpb.NewTestServiceClient(conn)
}

type dpWithAttrs struct {
	metricName string
	attrs      map[string]string
}

func assertAllMetricsHaveLabels(t *testing.T, reader metric.Reader, direction int, expectedLabels map[string]string) {
	t.Helper()

	rm := metricdata.ResourceMetrics{}
	err := reader.Collect(t.Context(), &rm)
	require.NoError(t, err)

	datapoints := collectDataPointsByMetric(rm, direction)
	assert.NotEmpty(t, datapoints, "no metrics instrumented")

	for _, dp := range datapoints {
		for key, val := range expectedLabels {
			attr, ok := dp.attrs[key]
			t.Logf("metric %q has label %q", dp.metricName, attr)
			assert.Truef(t, ok, "metric %q missing label %q", dp.metricName, key)
			if ok {
				assert.Equalf(t, val, attr, "metric %q has incorrect value for label %q: %s", dp.metricName, key, attr)
			}
		}
	}
}

func assertAllMetricsDoNotHaveLabels(t *testing.T, reader metric.Reader, direction int, notExpectedLabels map[string]string) {
	t.Helper()

	rm := metricdata.ResourceMetrics{}
	err := reader.Collect(t.Context(), &rm)
	require.NoError(t, err)

	datapoints := collectDataPointsByMetric(rm, direction)

	for _, dp := range datapoints {
		for key := range notExpectedLabels {
			_, ok := dp.attrs[key]
			assert.Falsef(t, ok, "metric %q should NOT have label %q", dp.metricName, key)
		}
	}
}

func collectDataPointsByMetric(rm metricdata.ResourceMetrics, direction int) []dpWithAttrs {
	var result []dpWithAttrs

	var prefix string
	switch direction {
	case serverLabelingDirection:
		prefix = "rpc.server."
	case clientLabelingDirection:
		prefix = "rpc.client."
	}

	for _, scope := range rm.ScopeMetrics {
		for _, m := range scope.Metrics {
			if !strings.HasPrefix(m.Name, prefix) {
				continue
			}

			switch data := m.Data.(type) {
			case metricdata.Histogram[float64]:
				for _, dp := range data.DataPoints {
					result = append(result, dpWithAttrs{
						metricName: m.Name,
						attrs:      extractAttributes(dp.Attributes),
					})
				}
			case metricdata.Histogram[int64]:
				for _, dp := range data.DataPoints {
					result = append(result, dpWithAttrs{
						metricName: m.Name,
						attrs:      extractAttributes(dp.Attributes),
					})
				}
			}
		}
	}
	return result
}

func extractAttributes(attrSet attribute.Set) map[string]string {
	m := make(map[string]string)
	iter := attrSet.Iter()
	for iter.Next() {
		kv := iter.Attribute()
		m[string(kv.Key)] = kv.Value.AsString()
	}
	return m
}
