// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgrpc

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"google.golang.org/grpc/stats"
)

func newClientHandlerForAddrTest(t *testing.T) (*clientHandler, *tracetest.SpanRecorder) {
	t.Helper()
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	h := NewClientHandler(
		WithTracerProvider(tp),
		WithPropagators(propagation.TraceContext{}),
	).(*clientHandler)
	return h, sr
}

func runRPC(t *testing.T, h *clientHandler, ctx context.Context, remoteAddr net.Addr) {
	t.Helper()
	ctx = h.TagRPC(ctx, &stats.RPCTagInfo{FullMethodName: "pkg/Method"})
	h.HandleRPC(ctx, &stats.Begin{Client: true})
	if remoteAddr != nil {
		h.HandleRPC(ctx, &stats.OutHeader{Client: true, RemoteAddr: remoteAddr})
	}
	h.HandleRPC(ctx, &stats.End{Client: true, EndTime: time.Now()})
}

// TestClientHandlerFallsBackToRemoteAddr verifies that without a dial target in
// context (NewClientHandler callers), server.address is taken from the resolved
// RemoteAddr in OutHeader.
func TestClientHandlerFallsBackToRemoteAddr(t *testing.T) {
	h, sr := newClientHandlerForAddrTest(t)

	remoteAddr := &net.TCPAddr{IP: net.ParseIP("192.0.2.1"), Port: 9090}
	runRPC(t, h, t.Context(), remoteAddr)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	attrs := spans[0].Attributes()
	assert.Contains(t, attrs, semconv.ServerAddress("192.0.2.1"))
	assert.Contains(t, attrs, semconv.ServerPort(9090))
}

// TestClientHandlerUsesDialTargetWhenPresent verifies that when dialTargetContextKey
// is seeded (as NewClientOptions interceptors will do), server.address is taken
// from the dial target hostname on both success and failure paths.
func TestClientHandlerUsesDialTargetWhenPresent(t *testing.T) {
	remoteAddr := &net.TCPAddr{IP: net.ParseIP("192.0.2.1"), Port: 9090}

	tests := []struct {
		name        string
		dialTarget  string
		wantAddress string
		wantPort    int
	}{
		{
			name:        "dns canonical",
			dialTarget:  "dns:///myservice:443",
			wantAddress: "myservice",
			wantPort:    443,
		},
		{
			name:        "passthrough canonical",
			dialTarget:  "passthrough:///myservice:443",
			wantAddress: "myservice",
			wantPort:    443,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+"/failure/OutHeader absent", func(t *testing.T) {
			h, sr := newClientHandlerForAddrTest(t)
			ctx := context.WithValue(t.Context(), dialTargetContextKey{}, tt.dialTarget)

			runRPC(t, h, ctx, nil)

			spans := sr.Ended()
			require.Len(t, spans, 1)
			attrs := spans[0].Attributes()
			assert.Contains(t, attrs, semconv.ServerAddress(tt.wantAddress))
			assert.Contains(t, attrs, semconv.ServerPort(tt.wantPort))
		})

		t.Run(tt.name+"/success/OutHeader fires but hostname wins", func(t *testing.T) {
			h, sr := newClientHandlerForAddrTest(t)
			ctx := context.WithValue(t.Context(), dialTargetContextKey{}, tt.dialTarget)

			runRPC(t, h, ctx, remoteAddr)

			spans := sr.Ended()
			require.Len(t, spans, 1)
			attrs := spans[0].Attributes()
			assert.Contains(t, attrs, semconv.ServerAddress(tt.wantAddress))
			assert.Contains(t, attrs, semconv.ServerPort(tt.wantPort))
			assert.NotContains(t, attrs, semconv.ServerAddress("192.0.2.1"))
		})
	}
}
