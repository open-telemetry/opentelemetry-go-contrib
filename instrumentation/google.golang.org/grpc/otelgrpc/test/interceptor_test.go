// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"context"
	"testing"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	grpc_codes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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
