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

package grpc_test

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	otelgrpc "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc"
	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/standard"
	"go.opentelemetry.io/otel/api/trace/testtrace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/interop"
	pb "google.golang.org/grpc/interop/grpc_testing"
	"google.golang.org/grpc/test/bufconn"
)

func testUnaryCall(t *testing.T, cOpt []grpc.DialOption, sOpt []grpc.ServerOption) {
	l := bufconn.Listen(bufSize)
	defer l.Close()

	s := grpc.NewServer(sOpt...)
	pb.RegisterTestServiceServer(s, interop.NewTestServer())
	go func() {
		if err := s.Serve(l); err != nil {
			t.Fatal(err)
		}
	}()
	defer s.Stop()

	ctx := context.Background()
	dial := func(context.Context, string) (net.Conn, error) { return l.Dial() }
	conn, err := grpc.DialContext(
		ctx,
		"bufnet",
		append([]grpc.DialOption{
			grpc.WithContextDialer(dial),
			grpc.WithInsecure(),
		}, cOpt...)...,
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := pb.NewTestServiceClient(conn)

	interop.DoEmptyUnaryCall(client)
	interop.DoLargeUnaryCall(client)
}

func TestUnaryClientInterceptor(t *testing.T) {
	sr := new(testtrace.StandardSpanRecorder)
	tp := testtrace.NewProvider(testtrace.WithSpanRecorder(sr))
	tracer := tp.Tracer("TestUnaryClientInterceptor")

	testUnaryCall(t, nil, []grpc.ServerOption{
		grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor(tracer)),
	})

	spans := sr.Completed()
	require.Len(t, spans, 2)

	emptySpan := spans[0]
	assert.True(t, emptySpan.Ended())
	assert.Equal(t, tracer, emptySpan.Tracer())
	assert.Equal(t, "grpc.testing.TestService/EmptyCall", emptySpan.Name())
	assert.Equal(t, map[string]map[kv.Key]kv.Value{
		"message": map[kv.Key]kv.Value{
			standard.RPCMessageIDKey:               kv.IntValue(1),
			standard.RPCMessageTypeKey:             kv.StringValue("SENT"),
			standard.RPCMessageUncompressedSizeKey: kv.IntValue(0),
		},
	}, eventMap(emptySpan.Events()))
	assert.Equal(t, map[kv.Key]kv.Value{
		standard.RPCMethodKey:      kv.StringValue("EmptyCall"),
		standard.RPCServiceKey:     kv.StringValue("grpc.testing.TestService"),
		standard.RPCSystemGRPC.Key: standard.RPCSystemGRPC.Value,
	}, emptySpan.Attributes())

	largeSpan := spans[1]
	assert.True(t, largeSpan.Ended())
	assert.Equal(t, tracer, largeSpan.Tracer())
	assert.Equal(t, "grpc.testing.TestService/UnaryCall", largeSpan.Name())
	assert.Equal(t, map[string]map[kv.Key]kv.Value{
		"message": map[kv.Key]kv.Value{
			standard.RPCMessageIDKey:   kv.IntValue(1),
			standard.RPCMessageTypeKey: kv.StringValue("SENT"),
			// largeRespSize from "google.golang.org/grpc/interop" + 8 (overhead).
			standard.RPCMessageUncompressedSizeKey: kv.IntValue(314167),
		},
	}, eventMap(largeSpan.Events()))
	assert.Equal(t, map[kv.Key]kv.Value{
		standard.RPCMethodKey:      kv.StringValue("UnaryCall"),
		standard.RPCServiceKey:     kv.StringValue("grpc.testing.TestService"),
		standard.RPCSystemGRPC.Key: standard.RPCSystemGRPC.Value,
	}, largeSpan.Attributes())
}

func eventMap(events []testtrace.Event) map[string]map[kv.Key]kv.Value {
	m := make(map[string]map[kv.Key]kv.Value, len(events))
	for _, e := range events {
		// Assume no mutation.
		m[e.Name] = e.Attributes
	}
	return m
}
