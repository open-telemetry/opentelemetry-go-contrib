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
	"google.golang.org/grpc"
	"google.golang.org/grpc/interop"
	pb "google.golang.org/grpc/interop/grpc_testing"
	"google.golang.org/grpc/test/bufconn"

	otelgrpc "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/api/trace/tracetest"
	"go.opentelemetry.io/otel/label"
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
	clientUnarySR := new(tracetest.StandardSpanRecorder)
	clientUnaryTracer := tracetest.NewProvider(tracetest.WithSpanRecorder(clientUnarySR)).Tracer("TestUnaryClientInterceptor")

	clientStreamSR := new(tracetest.StandardSpanRecorder)
	clientStreamTracer := tracetest.NewProvider(tracetest.WithSpanRecorder(clientStreamSR)).Tracer("TestStreamClientInterceptor")

	serverUnarySR := new(tracetest.StandardSpanRecorder)
	serverUnaryTracer := tracetest.NewProvider(tracetest.WithSpanRecorder(serverUnarySR)).Tracer("TestUnaryServerInterceptor")

	serverStreamSR := new(tracetest.StandardSpanRecorder)
	serverStreamTracer := tracetest.NewProvider(tracetest.WithSpanRecorder(serverStreamSR)).Tracer("TestStreamServerInterceptor")

	assert.NoError(t, doCalls(
		[]grpc.DialOption{
			grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor(clientUnaryTracer)),
			grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor(clientStreamTracer)),
		},
		[]grpc.ServerOption{
			grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor(serverUnaryTracer)),
			grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor(serverStreamTracer)),
		},
	))

	t.Run("UnaryClientSpans", func(t *testing.T) {
		checkUnaryClientSpans(t, clientUnaryTracer, clientUnarySR.Completed())
	})

	t.Run("StreamClientSpans", func(t *testing.T) {
		checkStreamClientSpans(t, clientStreamTracer, clientStreamSR.Completed())
	})

	t.Run("UnaryServerSpans", func(t *testing.T) {
		checkUnaryServerSpans(t, serverUnaryTracer, serverUnarySR.Completed())
	})

	t.Run("StreamServerSpans", func(t *testing.T) {
		checkStreamServerSpans(t, serverStreamTracer, serverStreamSR.Completed())
	})
}

func checkUnaryClientSpans(t *testing.T, tracer trace.Tracer, spans []*tracetest.Span) {
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

func checkStreamClientSpans(t *testing.T, tracer trace.Tracer, spans []*tracetest.Span) {
	require.Len(t, spans, 3)

	streamInput := spans[0]
	assert.True(t, streamInput.Ended())
	assert.Equal(t, tracer, streamInput.Tracer())
	assert.Equal(t, "grpc.testing.TestService/StreamingInputCall", streamInput.Name())
	// sizes from reqSizes in "google.golang.org/grpc/interop".
	assert.Equal(t, []tracetest.Event{
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:               label.IntValue(1),
				semconv.RPCMessageTypeKey:             label.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(27190),
			},
		},
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:               label.IntValue(2),
				semconv.RPCMessageTypeKey:             label.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(12),
			},
		},
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:               label.IntValue(3),
				semconv.RPCMessageTypeKey:             label.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(1834),
			},
		},
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:               label.IntValue(4),
				semconv.RPCMessageTypeKey:             label.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(45912),
			},
		},
		// client does not record an event for the server response.
	}, noTimestamp(streamInput.Events()))
	assert.Equal(t, map[label.Key]label.Value{
		semconv.RPCMethodKey:      label.StringValue("StreamingInputCall"),
		semconv.RPCServiceKey:     label.StringValue("grpc.testing.TestService"),
		semconv.RPCSystemGRPC.Key: semconv.RPCSystemGRPC.Value,
	}, streamInput.Attributes())

	streamOutput := spans[1]
	assert.True(t, streamOutput.Ended())
	assert.Equal(t, tracer, streamOutput.Tracer())
	assert.Equal(t, "grpc.testing.TestService/StreamingOutputCall", streamOutput.Name())
	// sizes from respSizes in "google.golang.org/grpc/interop".
	assert.Equal(t, []tracetest.Event{
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:               label.IntValue(1),
				semconv.RPCMessageTypeKey:             label.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(21),
			},
		},
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:               label.IntValue(1),
				semconv.RPCMessageTypeKey:             label.StringValue("RECEIVED"),
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(31423),
			},
		},
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:               label.IntValue(2),
				semconv.RPCMessageTypeKey:             label.StringValue("RECEIVED"),
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(13),
			},
		},
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:               label.IntValue(3),
				semconv.RPCMessageTypeKey:             label.StringValue("RECEIVED"),
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(2659),
			},
		},
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:               label.IntValue(4),
				semconv.RPCMessageTypeKey:             label.StringValue("RECEIVED"),
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(58987),
			},
		},
	}, noTimestamp(streamOutput.Events()))
	assert.Equal(t, map[label.Key]label.Value{
		semconv.RPCMethodKey:      label.StringValue("StreamingOutputCall"),
		semconv.RPCServiceKey:     label.StringValue("grpc.testing.TestService"),
		semconv.RPCSystemGRPC.Key: semconv.RPCSystemGRPC.Value,
	}, streamOutput.Attributes())

	pingPong := spans[2]
	assert.True(t, pingPong.Ended())
	assert.Equal(t, tracer, pingPong.Tracer())
	assert.Equal(t, "grpc.testing.TestService/FullDuplexCall", pingPong.Name())
	assert.Equal(t, []tracetest.Event{
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:               label.IntValue(1),
				semconv.RPCMessageTypeKey:             label.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(27196),
			},
		},
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:               label.IntValue(1),
				semconv.RPCMessageTypeKey:             label.StringValue("RECEIVED"),
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(31423),
			},
		},
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:               label.IntValue(2),
				semconv.RPCMessageTypeKey:             label.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(16),
			},
		},
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:               label.IntValue(2),
				semconv.RPCMessageTypeKey:             label.StringValue("RECEIVED"),
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(13),
			},
		},
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:               label.IntValue(3),
				semconv.RPCMessageTypeKey:             label.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(1839),
			},
		},
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:               label.IntValue(3),
				semconv.RPCMessageTypeKey:             label.StringValue("RECEIVED"),
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(2659),
			},
		},
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:               label.IntValue(4),
				semconv.RPCMessageTypeKey:             label.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(45918),
			},
		},
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:               label.IntValue(4),
				semconv.RPCMessageTypeKey:             label.StringValue("RECEIVED"),
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(58987),
			},
		},
	}, noTimestamp(pingPong.Events()))
	assert.Equal(t, map[label.Key]label.Value{
		semconv.RPCMethodKey:      label.StringValue("FullDuplexCall"),
		semconv.RPCServiceKey:     label.StringValue("grpc.testing.TestService"),
		semconv.RPCSystemGRPC.Key: semconv.RPCSystemGRPC.Value,
	}, pingPong.Attributes())
}

func checkStreamServerSpans(t *testing.T, tracer trace.Tracer, spans []*tracetest.Span) {
	require.Len(t, spans, 3)

	streamInput := spans[0]
	assert.True(t, streamInput.Ended())
	assert.Equal(t, tracer, streamInput.Tracer())
	assert.Equal(t, "grpc.testing.TestService/StreamingInputCall", streamInput.Name())
	// sizes from reqSizes in "google.golang.org/grpc/interop".
	assert.Equal(t, []tracetest.Event{
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:               label.IntValue(1),
				semconv.RPCMessageTypeKey:             label.StringValue("RECEIVED"),
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(27190),
			},
		},
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:               label.IntValue(2),
				semconv.RPCMessageTypeKey:             label.StringValue("RECEIVED"),
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(12),
			},
		},
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:               label.IntValue(3),
				semconv.RPCMessageTypeKey:             label.StringValue("RECEIVED"),
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(1834),
			},
		},
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:               label.IntValue(4),
				semconv.RPCMessageTypeKey:             label.StringValue("RECEIVED"),
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(45912),
			},
		},
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:               label.IntValue(1),
				semconv.RPCMessageTypeKey:             label.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(4),
			},
		},
	}, noTimestamp(streamInput.Events()))
	assert.Equal(t, map[label.Key]label.Value{
		semconv.RPCMethodKey:      label.StringValue("StreamingInputCall"),
		semconv.RPCServiceKey:     label.StringValue("grpc.testing.TestService"),
		semconv.RPCSystemGRPC.Key: semconv.RPCSystemGRPC.Value,
	}, streamInput.Attributes())

	streamOutput := spans[1]
	assert.True(t, streamOutput.Ended())
	assert.Equal(t, tracer, streamOutput.Tracer())
	assert.Equal(t, "grpc.testing.TestService/StreamingOutputCall", streamOutput.Name())
	// sizes from respSizes in "google.golang.org/grpc/interop".
	assert.Equal(t, []tracetest.Event{
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:               label.IntValue(1),
				semconv.RPCMessageTypeKey:             label.StringValue("RECEIVED"),
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(21),
			},
		},
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:               label.IntValue(1),
				semconv.RPCMessageTypeKey:             label.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(31423),
			},
		},
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:               label.IntValue(2),
				semconv.RPCMessageTypeKey:             label.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(13),
			},
		},
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:               label.IntValue(3),
				semconv.RPCMessageTypeKey:             label.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(2659),
			},
		},
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:               label.IntValue(4),
				semconv.RPCMessageTypeKey:             label.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(58987),
			},
		},
	}, noTimestamp(streamOutput.Events()))
	assert.Equal(t, map[label.Key]label.Value{
		semconv.RPCMethodKey:      label.StringValue("StreamingOutputCall"),
		semconv.RPCServiceKey:     label.StringValue("grpc.testing.TestService"),
		semconv.RPCSystemGRPC.Key: semconv.RPCSystemGRPC.Value,
	}, streamOutput.Attributes())

	pingPong := spans[2]
	assert.True(t, pingPong.Ended())
	assert.Equal(t, tracer, pingPong.Tracer())
	assert.Equal(t, "grpc.testing.TestService/FullDuplexCall", pingPong.Name())
	assert.Equal(t, []tracetest.Event{
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:               label.IntValue(1),
				semconv.RPCMessageTypeKey:             label.StringValue("RECEIVED"),
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(27196),
			},
		},
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:               label.IntValue(1),
				semconv.RPCMessageTypeKey:             label.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(31423),
			},
		},
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:               label.IntValue(2),
				semconv.RPCMessageTypeKey:             label.StringValue("RECEIVED"),
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(16),
			},
		},
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:               label.IntValue(2),
				semconv.RPCMessageTypeKey:             label.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(13),
			},
		},
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:               label.IntValue(3),
				semconv.RPCMessageTypeKey:             label.StringValue("RECEIVED"),
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(1839),
			},
		},
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:               label.IntValue(3),
				semconv.RPCMessageTypeKey:             label.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(2659),
			},
		},
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:               label.IntValue(4),
				semconv.RPCMessageTypeKey:             label.StringValue("RECEIVED"),
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(45918),
			},
		},
		{
			Name: "message",
			Attributes: map[label.Key]label.Value{
				semconv.RPCMessageIDKey:               label.IntValue(4),
				semconv.RPCMessageTypeKey:             label.StringValue("SENT"),
				semconv.RPCMessageUncompressedSizeKey: label.IntValue(58987),
			},
		},
	}, noTimestamp(pingPong.Events()))
	assert.Equal(t, map[label.Key]label.Value{
		semconv.RPCMethodKey:      label.StringValue("FullDuplexCall"),
		semconv.RPCServiceKey:     label.StringValue("grpc.testing.TestService"),
		semconv.RPCSystemGRPC.Key: semconv.RPCSystemGRPC.Value,
	}, pingPong.Attributes())
}

func checkUnaryServerSpans(t *testing.T, tracer trace.Tracer, spans []*tracetest.Span) {
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
