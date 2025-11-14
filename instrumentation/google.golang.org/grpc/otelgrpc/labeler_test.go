// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgrpc_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	testpb "google.golang.org/grpc/interop/grpc_testing"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

// testLabelerServer is a test server that implements the test service.
type testLabelerServer struct {
	testpb.UnimplementedTestServiceServer
}

// EmptyCall is a test method that returns an empty response.
func (s *testLabelerServer) EmptyCall(ctx context.Context, req *testpb.Empty) (*testpb.Empty, error) {
	return &testpb.Empty{}, nil
}

// UnaryCall is a test method that returns a simple response.
func (s *testLabelerServer) UnaryCall(ctx context.Context, req *testpb.SimpleRequest) (*testpb.SimpleResponse, error) {
	return &testpb.SimpleResponse{}, nil
}

// TestLabeler_ServerSide_SingleInterceptor tests that a single interceptor can add attributes to the metrics.
func TestLabeler_ServerSide_SingleInterceptor(t *testing.T) {
	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))

	serverInterceptor := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		labeler, ok := otelgrpc.LabelerFromContext(ctx, otelgrpc.ServerLabelerDirection)
		require.True(t, ok, "labeler should be available in interceptor")

		labeler.Add(
			attribute.String("custom.user_id", "user123"),
			attribute.String("custom.tier", "premium"),
		)

		return handler(ctx, req)
	}

	lis, server := startTestServerWithInterceptors(t, mp, serverInterceptor)
	defer server.Stop()

	client := createTestClient(t, lis.Addr().String())
	_, err := client.EmptyCall(t.Context(), &testpb.Empty{})
	require.NoError(t, err)

	assertMetricAttributes(t, reader, map[string]any{
		"custom.user_id": "user123",
		"custom.tier":    "premium",
	})
}

// TestLabeler_ServerSide_MultipleInterceptors tests that multiple interceptors can add attributes to the metrics.
func TestLabeler_ServerSide_MultipleInterceptors(t *testing.T) {
	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))

	firstInterceptor := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		labeler, _ := otelgrpc.LabelerFromContext(ctx, otelgrpc.ServerLabelerDirection)
		labeler.Add(attribute.String("firstAttribute", "firstValue"))
		return handler(ctx, req)
	}

	secondInterceptor := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		labeler, _ := otelgrpc.LabelerFromContext(ctx, otelgrpc.ServerLabelerDirection)
		labeler.Add(attribute.String("secondAttribute", "secondValue"))
		return handler(ctx, req)
	}

	thirdInterceptor := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		labeler, _ := otelgrpc.LabelerFromContext(ctx, otelgrpc.ServerLabelerDirection)
		labeler.Add(attribute.Bool("thirdAttribute", true))
		return handler(ctx, req)
	}

	lis, server := startTestServerWithInterceptors(t, mp, firstInterceptor, secondInterceptor, thirdInterceptor)
	defer server.Stop()

	client := createTestClient(t, lis.Addr().String())
	_, err := client.EmptyCall(t.Context(), &testpb.Empty{})
	require.NoError(t, err)

	assertMetricAttributes(t, reader, map[string]any{
		"firstAttribute":  "firstValue",
		"secondAttribute": "secondValue",
		"thirdAttribute":  true,
	})
}

// TestLabeler_ClientSide tests that a client can add attributes to the metrics.
func TestLabeler_ClientSide(t *testing.T) {
	// Setup server without instrumentation
	serverLis, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)

	server := grpc.NewServer()
	testpb.RegisterTestServiceServer(server, &testLabelerServer{})

	go server.Serve(serverLis)
	defer server.Stop()

	// Setup client with metrics
	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))

	clientInterceptor := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		labeler := &otelgrpc.Labeler{}
		labeler.Add(
			attribute.String("client.version", "v1.2.3"),
			attribute.String("client.env", "test"),
		)
		ctx = otelgrpc.ContextWithLabeler(ctx, labeler, otelgrpc.ClientLabelerDirection)
		return invoker(ctx, method, req, reply, cc, opts...)
	}

	conn, err := grpc.NewClient(
		serverLis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler(otelgrpc.WithMeterProvider(mp))),
		grpc.WithUnaryInterceptor(clientInterceptor),
	)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	client := testpb.NewTestServiceClient(conn)
	_, err = client.EmptyCall(t.Context(), &testpb.Empty{})
	require.NoError(t, err)

	assertMetricAttributes(t, reader, map[string]any{
		"client.version": "v1.2.3",
		"client.env":     "test",
	})
}

// TestLabeler_WithRequestData tests that a server can add attributes to the metrics based on request data.
func TestLabeler_WithRequestData(t *testing.T) {
	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))

	dataInterceptor := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		labeler, _ := otelgrpc.LabelerFromContext(ctx, otelgrpc.ServerLabelerDirection)

		if simpleReq, ok := req.(*testpb.SimpleRequest); ok {
			if simpleReq.GetResponseSize() > 1000 {
				labeler.Add(attribute.String("request.size", "large"))
			} else {
				labeler.Add(attribute.String("request.size", "small"))
			}
		}

		return handler(ctx, req)
	}

	lis, server := startTestServerWithInterceptors(t, mp, dataInterceptor)
	defer server.Stop()

	client := createTestClient(t, lis.Addr().String())

	_, err := client.UnaryCall(t.Context(), &testpb.SimpleRequest{
		ResponseSize: 3000,
	})
	require.NoError(t, err)

	assertMetricAttributes(t, reader, map[string]any{
		"request.size": "large",
	})
}

// TestLabeler_AddBeforeAndAfterHandler tests that a server can add attributes to the metrics before and after the handler is executed.
func TestLabeler_AddBeforeAndAfterHandler(t *testing.T) {
	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))

	wrappingInterceptor := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		labeler, _ := otelgrpc.LabelerFromContext(ctx, otelgrpc.ServerLabelerDirection)

		// Before handler
		labeler.Add(attribute.String("phase", "before"))
		labeler.Add(attribute.Int64("timestamp.before", time.Now().Unix()))

		// Execute handler
		resp, err := handler(ctx, req)

		// After handler
		labeler.Add(attribute.String("phase.after", "complete"))
		if err != nil {
			labeler.Add(attribute.Bool("has_error", true))
		} else {
			labeler.Add(attribute.Bool("has_error", false))
		}

		return resp, err
	}

	lis, server := startTestServerWithInterceptors(t, mp, wrappingInterceptor)
	defer server.Stop()

	client := createTestClient(t, lis.Addr().String())
	_, err := client.EmptyCall(t.Context(), &testpb.Empty{})
	require.NoError(t, err)

	assertMetricAttributes(t, reader, map[string]any{
		"phase":       "before",
		"phase.after": "complete",
		"has_error":   false,
	})
}

// TestLabeler_CombinedWithStaticAttributes tests that a server can add attributes to the metrics combined with static attributes.
func TestLabeler_CombinedWithStaticAttributes(t *testing.T) {
	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))

	interceptor := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		labeler, _ := otelgrpc.LabelerFromContext(ctx, otelgrpc.ServerLabelerDirection)
		labeler.Add(attribute.String("dynamic", "from_labeler"))
		return handler(ctx, req)
	}

	lis, server := startTestServerWithOptions(t, mp, interceptor,
		otelgrpc.WithMetricAttributes(
			attribute.String("static", "configured"),
			attribute.String("service", "test"),
		),
	)
	defer server.Stop()

	client := createTestClient(t, lis.Addr().String())
	_, err := client.EmptyCall(t.Context(), &testpb.Empty{})
	require.NoError(t, err)

	assertMetricAttributes(t, reader, map[string]any{
		"static":  "configured",
		"dynamic": "from_labeler",
		"service": "test",
	})
}

// TestLabeler_ClientAndServerIndependent tests that client and server labelers are completely independent.
func TestLabeler_ClientAndServerIndependent(t *testing.T) {
	serverReader := metric.NewManualReader()
	serverMP := metric.NewMeterProvider(metric.WithReader(serverReader))

	serverInterceptor := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		labeler, ok := otelgrpc.LabelerFromContext(ctx, otelgrpc.ServerLabelerDirection)
		require.True(t, ok, "server labeler should be available")
		labeler.Add(attribute.String("side", "server"))
		labeler.Add(attribute.String("server.region", "us-east"))
		return handler(ctx, req)
	}

	lis, server := startTestServerWithInterceptors(t, serverMP, serverInterceptor)
	defer server.Stop()

	clientReader := metric.NewManualReader()
	clientMP := metric.NewMeterProvider(metric.WithReader(clientReader))

	clientInterceptor := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		labeler := &otelgrpc.Labeler{}
		labeler.Add(attribute.String("side", "client"))
		labeler.Add(attribute.String("client.version", "v2.0"))
		ctx = otelgrpc.ContextWithLabeler(ctx, labeler, otelgrpc.ClientLabelerDirection)
		return invoker(ctx, method, req, reply, cc, opts...)
	}

	conn, err := grpc.NewClient(
		lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler(otelgrpc.WithMeterProvider(clientMP))),
		grpc.WithUnaryInterceptor(clientInterceptor),
	)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	client := testpb.NewTestServiceClient(conn)
	_, err = client.EmptyCall(t.Context(), &testpb.Empty{})
	require.NoError(t, err)

	assertMetricAttributes(t, serverReader, map[string]any{
		"side":          "server",
		"server.region": "us-east",
	})

	assertMetricAttributes(t, clientReader, map[string]any{
		"side":           "client",
		"client.version": "v2.0",
	})
}

func assertMetricAttributes(t *testing.T, reader metric.Reader, expected map[string]any) {
	t.Helper()

	rm := metricdata.ResourceMetrics{}
	err := reader.Collect(t.Context(), &rm)
	require.NoError(t, err, "failed to collect metrics")
	require.NotEmpty(t, rm.ScopeMetrics, "no metrics recorded")

	allAttrs := make(map[string]any)
	for _, scopeMetric := range rm.ScopeMetrics {
		for _, m := range scopeMetric.Metrics {
			switch data := m.Data.(type) {
			case metricdata.Histogram[float64]:
				for _, dp := range data.DataPoints {
					extractAttributes(dp.Attributes, allAttrs)
				}
			case metricdata.Histogram[int64]:
				for _, dp := range data.DataPoints {
					extractAttributes(dp.Attributes, allAttrs)
				}
			}
		}
	}

	for key, expectedVal := range expected {
		actualVal, exists := allAttrs[key]
		assert.True(t, exists, "attribute %s should exist", key)
		if exists {
			assert.Equal(t, expectedVal, actualVal, "attribute %s has wrong value", key)
		}
	}
}

func extractAttributes(attrSet attribute.Set, dest map[string]any) {
	iter := attrSet.Iter()
	for iter.Next() {
		kv := iter.Attribute()
		dest[string(kv.Key)] = kv.Value.AsInterface()
	}
}

func startTestServerWithInterceptors(t *testing.T, mp *metric.MeterProvider, interceptors ...grpc.UnaryServerInterceptor) (net.Listener, *grpc.Server) {
	t.Helper()

	lis, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)

	server := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler(otelgrpc.WithMeterProvider(mp))),
		grpc.ChainUnaryInterceptor(interceptors...),
	)
	testpb.RegisterTestServiceServer(server, &testLabelerServer{})

	go server.Serve(lis)
	return lis, server
}

func startTestServerWithOptions(t *testing.T, mp *metric.MeterProvider, interceptor grpc.UnaryServerInterceptor, opts ...otelgrpc.Option) (net.Listener, *grpc.Server) {
	t.Helper()

	lis, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)

	allOpts := append([]otelgrpc.Option{otelgrpc.WithMeterProvider(mp)}, opts...)
	server := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler(allOpts...)),
		grpc.UnaryInterceptor(interceptor),
	)
	testpb.RegisterTestServiceServer(server, &testLabelerServer{})

	go server.Serve(lis)
	return lis, server
}

func createTestClient(t *testing.T, addr string) testpb.TestServiceClient {
	t.Helper()

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	return testpb.NewTestServiceClient(conn)
}
