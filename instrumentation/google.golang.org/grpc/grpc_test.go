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
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/api/trace/tracetest"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/semconv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/interop"
	pb "google.golang.org/grpc/interop/grpc_testing"
	"google.golang.org/grpc/test/bufconn"
)

const (
	bufSize  = 2048
	instName = "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc"
)

func testUnaryCall(t *testing.T, cOpt []grpc.DialOption, sOpt []grpc.ServerOption) {
	l := bufconn.Listen(bufSize)
	defer l.Close()

	s := grpc.NewServer(sOpt...)
	pb.RegisterTestServiceServer(s, interop.NewTestServer())
	go func() {
		if err := s.Serve(l); err != nil {
			panic(err)
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

func TestUnaryInterceptors(t *testing.T) {
	clientSR := new(tracetest.StandardSpanRecorder)
	clientTracer := tracetest.NewProvider(tracetest.WithSpanRecorder(clientSR)).Tracer("TestUnaryClientInterceptor")

	serverSR := new(tracetest.StandardSpanRecorder)
	serverTracer := tracetest.NewProvider(tracetest.WithSpanRecorder(serverSR)).Tracer("TestUnaryServerInterceptor")

	testUnaryCall(
		t,
		[]grpc.DialOption{
			grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor(clientTracer)),
		},
		[]grpc.ServerOption{
			grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor(serverTracer)),
		},
	)

	t.Run("UnaryClientSpans", func(t *testing.T) {
		checkClientSpans(t, clientTracer, clientSR.Completed())
	})

	t.Run("UnaryServerSpans", func(t *testing.T) {
		checkServerSpans(t, serverTracer, serverSR.Completed())
	})
}

func checkClientSpans(t *testing.T, tracer trace.Tracer, spans []*tracetest.Span) {
	require.Len(t, spans, 2)

	emptySpan := spans[0]
	assert.True(t, emptySpan.Ended())
	assert.Equal(t, tracer, emptySpan.Tracer())
	assert.Equal(t, "grpc.testing.TestService/EmptyCall", emptySpan.Name())
	assert.Equal(t, []tracetest.Event{
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:               label.IntValue(1),
				semconv.RPCMessageTypeKey:             label.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(0),
			},
		},
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:               label.IntValue(1),
				semconv.RPCMessageTypeKey:             label.StringValue("RECEIVED"),
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(0),
			},
		},
	}, noTimestamp(emptySpan.Events()))
	assert.Equal(t, map[label.Key]label.Value{
		semconv.RPCMethodKey:      label.StringValue("EmptyCall"),
		semconv.RPCServiceKey:     label.StringValue("grpc.testing.TestService"),
		semconv.RPCSystemGRPC.Key: semconv.RPCSystemGRPC.Value,
	}, emptySpan.Attributes())

	largeSpan := spans[1]
	assert.True(t, largeSpan.Ended())
	assert.Equal(t, tracer, largeSpan.Tracer())
	assert.Equal(t, "grpc.testing.TestService/UnaryCall", largeSpan.Name())
	assert.Equal(t, []tracetest.Event{
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:   label.IntValue(1),
				semconv.RPCMessageTypeKey: label.StringValue("SENT"),
				// largeReqSize from "google.golang.org/grpc/interop" + 12 (overhead).
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(271840),
			},
		},
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:   label.IntValue(1),
				semconv.RPCMessageTypeKey: label.StringValue("RECEIVED"),
				// largeRespSize from "google.golang.org/grpc/interop" + 8 (overhead).
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(314167),
			},
		},
	}, noTimestamp(largeSpan.Events()))
	assert.Equal(t, map[label.Key]label.Value{
		semconv.RPCMethodKey:      label.StringValue("UnaryCall"),
		semconv.RPCServiceKey:     label.StringValue("grpc.testing.TestService"),
		semconv.RPCSystemGRPC.Key: semconv.RPCSystemGRPC.Value,
	}, largeSpan.Attributes())
}

func checkServerSpans(t *testing.T, tracer trace.Tracer, spans []*tracetest.Span) {
	require.Len(t, spans, 2)

	emptySpan := spans[0]
	assert.True(t, emptySpan.Ended())
	assert.Equal(t, tracer, emptySpan.Tracer())
	assert.Equal(t, "grpc.testing.TestService/EmptyCall", emptySpan.Name())
	assert.Equal(t, []tracetest.Event{
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:               label.IntValue(1),
				semconv.RPCMessageTypeKey:             label.StringValue("RECEIVED"),
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(0),
			},
		},
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:               label.IntValue(1),
				semconv.RPCMessageTypeKey:             label.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(0),
			},
		},
	}, noTimestamp(emptySpan.Events()))
	assert.Equal(t, map[label.Key]label.Value{
		semconv.RPCMethodKey:      label.StringValue("EmptyCall"),
		semconv.RPCServiceKey:     label.StringValue("grpc.testing.TestService"),
		semconv.RPCSystemGRPC.Key: semconv.RPCSystemGRPC.Value,
	}, emptySpan.Attributes())

	largeSpan := spans[1]
	assert.True(t, largeSpan.Ended())
	assert.Equal(t, tracer, largeSpan.Tracer())
	assert.Equal(t, "grpc.testing.TestService/UnaryCall", largeSpan.Name())
	assert.Equal(t, []tracetest.Event{
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:   label.IntValue(1),
				semconv.RPCMessageTypeKey: label.StringValue("RECEIVED"),
				// largeReqSize from "google.golang.org/grpc/interop" + 12 (overhead).
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(271840),
			},
		},
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:   label.IntValue(1),
				semconv.RPCMessageTypeKey: label.StringValue("SENT"),
				// largeRespSize from "google.golang.org/grpc/interop" + 8 (overhead).
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(314167),
			},
		},
	}, noTimestamp(largeSpan.Events()))
	assert.Equal(t, map[label.Key]label.Value{
		semconv.RPCMethodKey:      label.StringValue("UnaryCall"),
		semconv.RPCServiceKey:     label.StringValue("grpc.testing.TestService"),
		semconv.RPCSystemGRPC.Key: semconv.RPCSystemGRPC.Value,
	}, largeSpan.Attributes())
}

func noTimestamp(events []tracetest.Event) []tracetest.Event {
	out := make([]tracetest.Event, 0, len(events))
	for _, e := range events {
		out = append(out, tracetest.Event{
			Name:       e.Name,
			Attributes: e.Attributes,
		})
	}
	return out
}
