// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgrpc_test

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "google.golang.org/grpc/interop/grpc_testing"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc/internal/test"
)

var wantInstrumentationScope = instrumentation.Scope{
	Name:      "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc",
	SchemaURL: semconv.SchemaURL,
	Version:   otelgrpc.Version,
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
