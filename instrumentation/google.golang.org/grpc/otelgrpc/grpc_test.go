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

package otelgrpc_test

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/interop"
	pb "google.golang.org/grpc/interop/grpc_testing"
	"google.golang.org/grpc/test/bufconn"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/oteltest"
	"go.opentelemetry.io/otel/semconv"
)

func doCalls(cOpt []grpc.DialOption, sOpt []grpc.ServerOption) error {
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
		return err
	}
	defer conn.Close()
	client := pb.NewTestServiceClient(conn)

	interop.DoEmptyUnaryCall(client)
	interop.DoLargeUnaryCall(client)
	interop.DoClientStreaming(client)
	interop.DoServerStreaming(client)
	interop.DoPingPong(client)

	return nil
}

func TestInterceptors(t *testing.T) {
	clientUnarySR := new(oteltest.SpanRecorder)
	clientUnaryTP := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(clientUnarySR))

	clientStreamSR := new(oteltest.SpanRecorder)
	clientStreamTP := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(clientStreamSR))

	serverUnarySR := new(oteltest.SpanRecorder)
	serverUnaryTP := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(serverUnarySR))

	serverStreamSR := new(oteltest.SpanRecorder)
	serverStreamTP := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(serverStreamSR))

	assert.NoError(t, doCalls(
		[]grpc.DialOption{
			grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor(otelgrpc.WithTracerProvider(clientUnaryTP))),
			grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor(otelgrpc.WithTracerProvider(clientStreamTP))),
		},
		[]grpc.ServerOption{
			grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor(otelgrpc.WithTracerProvider(serverUnaryTP))),
			grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor(otelgrpc.WithTracerProvider(serverStreamTP))),
		},
	))

	t.Run("UnaryClientSpans", func(t *testing.T) {
		checkUnaryClientSpans(t, clientUnarySR.Completed())
	})

	t.Run("StreamClientSpans", func(t *testing.T) {
		checkStreamClientSpans(t, clientStreamSR.Completed())
	})

	t.Run("UnaryServerSpans", func(t *testing.T) {
		checkUnaryServerSpans(t, serverUnarySR.Completed())
	})

	t.Run("StreamServerSpans", func(t *testing.T) {
		checkStreamServerSpans(t, serverStreamSR.Completed())
	})
}

func checkUnaryClientSpans(t *testing.T, spans []*oteltest.Span) {
	require.Len(t, spans, 2)

	emptySpan := spans[0]
	assert.True(t, emptySpan.Ended())
	assert.Equal(t, "grpc.testing.TestService/EmptyCall", emptySpan.Name())
	assert.Equal(t, []oteltest.Event{
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:               attribute.IntValue(1),
				semconv.RPCMessageTypeKey:             attribute.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(0),
			},
		},
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:               attribute.IntValue(1),
				semconv.RPCMessageTypeKey:             attribute.StringValue("RECEIVED"),
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(0),
			},
		},
	}, noTimestamp(emptySpan.Events()))
	assert.Equal(t, map[attribute.Key]attribute.Value{
		semconv.RPCMethodKey:       attribute.StringValue("EmptyCall"),
		semconv.RPCServiceKey:      attribute.StringValue("grpc.testing.TestService"),
		semconv.RPCSystemGRPC.Key:  semconv.RPCSystemGRPC.Value,
		otelgrpc.GRPCStatusCodeKey: attribute.Int64Value(int64(codes.OK)),
	}, emptySpan.Attributes())

	largeSpan := spans[1]
	assert.True(t, largeSpan.Ended())
	assert.Equal(t, "grpc.testing.TestService/UnaryCall", largeSpan.Name())
	assert.Equal(t, []oteltest.Event{
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:   attribute.IntValue(1),
				semconv.RPCMessageTypeKey: attribute.StringValue("SENT"),
				// largeReqSize from "google.golang.org/grpc/interop" + 12 (overhead).
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(271840),
			},
		},
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:   attribute.IntValue(1),
				semconv.RPCMessageTypeKey: attribute.StringValue("RECEIVED"),
				// largeRespSize from "google.golang.org/grpc/interop" + 8 (overhead).
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(314167),
			},
		},
	}, noTimestamp(largeSpan.Events()))
	assert.Equal(t, map[attribute.Key]attribute.Value{
		semconv.RPCMethodKey:       attribute.StringValue("UnaryCall"),
		semconv.RPCServiceKey:      attribute.StringValue("grpc.testing.TestService"),
		semconv.RPCSystemGRPC.Key:  semconv.RPCSystemGRPC.Value,
		otelgrpc.GRPCStatusCodeKey: attribute.Int64Value(int64(codes.OK)),
	}, largeSpan.Attributes())
}

func checkStreamClientSpans(t *testing.T, spans []*oteltest.Span) {
	require.Len(t, spans, 3)

	streamInput := spans[0]
	assert.True(t, streamInput.Ended())
	assert.Equal(t, "grpc.testing.TestService/StreamingInputCall", streamInput.Name())
	// sizes from reqSizes in "google.golang.org/grpc/interop".
	assert.Equal(t, []oteltest.Event{
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:               attribute.IntValue(1),
				semconv.RPCMessageTypeKey:             attribute.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(27190),
			},
		},
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:               attribute.IntValue(2),
				semconv.RPCMessageTypeKey:             attribute.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(12),
			},
		},
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:               attribute.IntValue(3),
				semconv.RPCMessageTypeKey:             attribute.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(1834),
			},
		},
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:               attribute.IntValue(4),
				semconv.RPCMessageTypeKey:             attribute.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(45912),
			},
		},
		// client does not record an event for the server response.
	}, noTimestamp(streamInput.Events()))
	assert.Equal(t, map[attribute.Key]attribute.Value{
		semconv.RPCMethodKey:       attribute.StringValue("StreamingInputCall"),
		semconv.RPCServiceKey:      attribute.StringValue("grpc.testing.TestService"),
		semconv.RPCSystemGRPC.Key:  semconv.RPCSystemGRPC.Value,
		otelgrpc.GRPCStatusCodeKey: attribute.Int64Value(int64(codes.OK)),
	}, streamInput.Attributes())

	streamOutput := spans[1]
	assert.True(t, streamOutput.Ended())
	assert.Equal(t, "grpc.testing.TestService/StreamingOutputCall", streamOutput.Name())
	// sizes from respSizes in "google.golang.org/grpc/interop".
	assert.Equal(t, []oteltest.Event{
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:               attribute.IntValue(1),
				semconv.RPCMessageTypeKey:             attribute.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(21),
			},
		},
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:               attribute.IntValue(1),
				semconv.RPCMessageTypeKey:             attribute.StringValue("RECEIVED"),
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(31423),
			},
		},
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:               attribute.IntValue(2),
				semconv.RPCMessageTypeKey:             attribute.StringValue("RECEIVED"),
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(13),
			},
		},
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:               attribute.IntValue(3),
				semconv.RPCMessageTypeKey:             attribute.StringValue("RECEIVED"),
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(2659),
			},
		},
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:               attribute.IntValue(4),
				semconv.RPCMessageTypeKey:             attribute.StringValue("RECEIVED"),
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(58987),
			},
		},
	}, noTimestamp(streamOutput.Events()))
	assert.Equal(t, map[attribute.Key]attribute.Value{
		semconv.RPCMethodKey:       attribute.StringValue("StreamingOutputCall"),
		semconv.RPCServiceKey:      attribute.StringValue("grpc.testing.TestService"),
		semconv.RPCSystemGRPC.Key:  semconv.RPCSystemGRPC.Value,
		otelgrpc.GRPCStatusCodeKey: attribute.Int64Value(int64(codes.OK)),
	}, streamOutput.Attributes())

	pingPong := spans[2]
	assert.True(t, pingPong.Ended())
	assert.Equal(t, "grpc.testing.TestService/FullDuplexCall", pingPong.Name())
	assert.Equal(t, []oteltest.Event{
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:               attribute.IntValue(1),
				semconv.RPCMessageTypeKey:             attribute.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(27196),
			},
		},
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:               attribute.IntValue(1),
				semconv.RPCMessageTypeKey:             attribute.StringValue("RECEIVED"),
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(31423),
			},
		},
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:               attribute.IntValue(2),
				semconv.RPCMessageTypeKey:             attribute.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(16),
			},
		},
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:               attribute.IntValue(2),
				semconv.RPCMessageTypeKey:             attribute.StringValue("RECEIVED"),
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(13),
			},
		},
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:               attribute.IntValue(3),
				semconv.RPCMessageTypeKey:             attribute.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(1839),
			},
		},
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:               attribute.IntValue(3),
				semconv.RPCMessageTypeKey:             attribute.StringValue("RECEIVED"),
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(2659),
			},
		},
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:               attribute.IntValue(4),
				semconv.RPCMessageTypeKey:             attribute.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(45918),
			},
		},
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:               attribute.IntValue(4),
				semconv.RPCMessageTypeKey:             attribute.StringValue("RECEIVED"),
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(58987),
			},
		},
	}, noTimestamp(pingPong.Events()))
	assert.Equal(t, map[attribute.Key]attribute.Value{
		semconv.RPCMethodKey:       attribute.StringValue("FullDuplexCall"),
		semconv.RPCServiceKey:      attribute.StringValue("grpc.testing.TestService"),
		semconv.RPCSystemGRPC.Key:  semconv.RPCSystemGRPC.Value,
		otelgrpc.GRPCStatusCodeKey: attribute.Int64Value(int64(codes.OK)),
	}, pingPong.Attributes())
}

func checkStreamServerSpans(t *testing.T, spans []*oteltest.Span) {
	require.Len(t, spans, 3)

	streamInput := spans[0]
	assert.True(t, streamInput.Ended())
	assert.Equal(t, "grpc.testing.TestService/StreamingInputCall", streamInput.Name())
	// sizes from reqSizes in "google.golang.org/grpc/interop".
	assert.Equal(t, []oteltest.Event{
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:               attribute.IntValue(1),
				semconv.RPCMessageTypeKey:             attribute.StringValue("RECEIVED"),
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(27190),
			},
		},
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:               attribute.IntValue(2),
				semconv.RPCMessageTypeKey:             attribute.StringValue("RECEIVED"),
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(12),
			},
		},
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:               attribute.IntValue(3),
				semconv.RPCMessageTypeKey:             attribute.StringValue("RECEIVED"),
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(1834),
			},
		},
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:               attribute.IntValue(4),
				semconv.RPCMessageTypeKey:             attribute.StringValue("RECEIVED"),
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(45912),
			},
		},
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:               attribute.IntValue(1),
				semconv.RPCMessageTypeKey:             attribute.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(4),
			},
		},
	}, noTimestamp(streamInput.Events()))
	assert.Equal(t, map[attribute.Key]attribute.Value{
		semconv.RPCMethodKey:       attribute.StringValue("StreamingInputCall"),
		semconv.RPCServiceKey:      attribute.StringValue("grpc.testing.TestService"),
		semconv.RPCSystemGRPC.Key:  semconv.RPCSystemGRPC.Value,
		otelgrpc.GRPCStatusCodeKey: attribute.Int64Value(int64(codes.OK)),
	}, streamInput.Attributes())

	streamOutput := spans[1]
	assert.True(t, streamOutput.Ended())
	assert.Equal(t, "grpc.testing.TestService/StreamingOutputCall", streamOutput.Name())
	// sizes from respSizes in "google.golang.org/grpc/interop".
	assert.Equal(t, []oteltest.Event{
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:               attribute.IntValue(1),
				semconv.RPCMessageTypeKey:             attribute.StringValue("RECEIVED"),
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(21),
			},
		},
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:               attribute.IntValue(1),
				semconv.RPCMessageTypeKey:             attribute.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(31423),
			},
		},
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:               attribute.IntValue(2),
				semconv.RPCMessageTypeKey:             attribute.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(13),
			},
		},
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:               attribute.IntValue(3),
				semconv.RPCMessageTypeKey:             attribute.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(2659),
			},
		},
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:               attribute.IntValue(4),
				semconv.RPCMessageTypeKey:             attribute.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(58987),
			},
		},
	}, noTimestamp(streamOutput.Events()))
	assert.Equal(t, map[attribute.Key]attribute.Value{
		semconv.RPCMethodKey:       attribute.StringValue("StreamingOutputCall"),
		semconv.RPCServiceKey:      attribute.StringValue("grpc.testing.TestService"),
		semconv.RPCSystemGRPC.Key:  semconv.RPCSystemGRPC.Value,
		otelgrpc.GRPCStatusCodeKey: attribute.Int64Value(int64(codes.OK)),
	}, streamOutput.Attributes())

	pingPong := spans[2]
	assert.True(t, pingPong.Ended())
	assert.Equal(t, "grpc.testing.TestService/FullDuplexCall", pingPong.Name())
	assert.Equal(t, []oteltest.Event{
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:               attribute.IntValue(1),
				semconv.RPCMessageTypeKey:             attribute.StringValue("RECEIVED"),
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(27196),
			},
		},
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:               attribute.IntValue(1),
				semconv.RPCMessageTypeKey:             attribute.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(31423),
			},
		},
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:               attribute.IntValue(2),
				semconv.RPCMessageTypeKey:             attribute.StringValue("RECEIVED"),
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(16),
			},
		},
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:               attribute.IntValue(2),
				semconv.RPCMessageTypeKey:             attribute.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(13),
			},
		},
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:               attribute.IntValue(3),
				semconv.RPCMessageTypeKey:             attribute.StringValue("RECEIVED"),
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(1839),
			},
		},
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:               attribute.IntValue(3),
				semconv.RPCMessageTypeKey:             attribute.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(2659),
			},
		},
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:               attribute.IntValue(4),
				semconv.RPCMessageTypeKey:             attribute.StringValue("RECEIVED"),
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(45918),
			},
		},
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:               attribute.IntValue(4),
				semconv.RPCMessageTypeKey:             attribute.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(58987),
			},
		},
	}, noTimestamp(pingPong.Events()))
	assert.Equal(t, map[attribute.Key]attribute.Value{
		semconv.RPCMethodKey:       attribute.StringValue("FullDuplexCall"),
		semconv.RPCServiceKey:      attribute.StringValue("grpc.testing.TestService"),
		semconv.RPCSystemGRPC.Key:  semconv.RPCSystemGRPC.Value,
		otelgrpc.GRPCStatusCodeKey: attribute.Int64Value(int64(codes.OK)),
	}, pingPong.Attributes())
}

func checkUnaryServerSpans(t *testing.T, spans []*oteltest.Span) {
	require.Len(t, spans, 2)

	emptySpan := spans[0]
	assert.True(t, emptySpan.Ended())
	assert.Equal(t, "grpc.testing.TestService/EmptyCall", emptySpan.Name())
	assert.Equal(t, []oteltest.Event{
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:               attribute.IntValue(1),
				semconv.RPCMessageTypeKey:             attribute.StringValue("RECEIVED"),
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(0),
			},
		},
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:               attribute.IntValue(1),
				semconv.RPCMessageTypeKey:             attribute.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(0),
			},
		},
	}, noTimestamp(emptySpan.Events()))
	assert.Equal(t, map[attribute.Key]attribute.Value{
		semconv.RPCMethodKey:       attribute.StringValue("EmptyCall"),
		semconv.RPCServiceKey:      attribute.StringValue("grpc.testing.TestService"),
		semconv.RPCSystemGRPC.Key:  semconv.RPCSystemGRPC.Value,
		otelgrpc.GRPCStatusCodeKey: attribute.Int64Value(int64(codes.OK)),
	}, emptySpan.Attributes())

	largeSpan := spans[1]
	assert.True(t, largeSpan.Ended())
	assert.Equal(t, "grpc.testing.TestService/UnaryCall", largeSpan.Name())
	assert.Equal(t, []oteltest.Event{
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:   attribute.IntValue(1),
				semconv.RPCMessageTypeKey: attribute.StringValue("RECEIVED"),
				// largeReqSize from "google.golang.org/grpc/interop" + 12 (overhead).
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(271840),
			},
		},
		{
			Name: "message",
			Attributes: map[attribute.Key]attribute.Value{
				semconv.RPCMessageIDKey:   attribute.IntValue(1),
				semconv.RPCMessageTypeKey: attribute.StringValue("SENT"),
				// largeRespSize from "google.golang.org/grpc/interop" + 8 (overhead).
				semconv.RPCMessageUncompressedSizeKey: attribute.IntValue(314167),
			},
		},
	}, noTimestamp(largeSpan.Events()))
	assert.Equal(t, map[attribute.Key]attribute.Value{
		semconv.RPCMethodKey:       attribute.StringValue("UnaryCall"),
		semconv.RPCServiceKey:      attribute.StringValue("grpc.testing.TestService"),
		semconv.RPCSystemGRPC.Key:  semconv.RPCSystemGRPC.Value,
		otelgrpc.GRPCStatusCodeKey: attribute.Int64Value(int64(codes.OK)),
	}, largeSpan.Attributes())
}

func noTimestamp(events []oteltest.Event) []oteltest.Event {
	out := make([]oteltest.Event, 0, len(events))
	for _, e := range events {
		out = append(out, oteltest.Event{
			Name:       e.Name,
			Attributes: e.Attributes,
		})
	}
	return out
}
