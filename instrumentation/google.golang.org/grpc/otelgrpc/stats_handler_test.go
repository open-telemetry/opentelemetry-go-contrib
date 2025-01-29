// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgrpc

import (
	"context"
	"testing"

	sdltrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/stats"
)

func TestServerHandler_TagRPC(t *testing.T) {
	tests := []struct {
		name   string
		server stats.Handler
		ctx    context.Context
		info   *stats.RPCTagInfo
		exp    bool
	}{
		{
			name:   "start a span without filters",
			server: NewServerHandler(WithTracerProvider(sdltrace.NewTracerProvider())),
			ctx:    context.Background(),
			info: &stats.RPCTagInfo{
				FullMethodName: "/grpc.health.v1.Health/Check",
			},
			exp: true,
		},
		{
			name: "don't start a span with filter and match",
			server: NewServerHandler(WithTracerProvider(sdltrace.NewTracerProvider()), WithFilter(func(ri *stats.RPCTagInfo) bool {
				return ri.FullMethodName != "/grpc.health.v1.Health/Check"
			})),
			ctx: context.Background(),
			info: &stats.RPCTagInfo{
				FullMethodName: "/grpc.health.v1.Health/Check",
			},
			exp: false,
		},
		{
			name: "start a span with filter and no match",
			server: NewServerHandler(WithTracerProvider(sdltrace.NewTracerProvider()), WithFilter(func(ri *stats.RPCTagInfo) bool {
				return ri.FullMethodName != "/grpc.health.v1.Health/Check"
			})),
			ctx: context.Background(),
			info: &stats.RPCTagInfo{
				FullMethodName: "/app.v1.Service/Get",
			},
			exp: true,
		},
	}

	for _, test := range tests {
		t.Run(t.Name(), func(t *testing.T) {
			ctx := test.server.TagRPC(test.ctx, test.info)

			got := trace.SpanFromContext(ctx).IsRecording()

			if test.exp != got {
				t.Errorf("expected %t, got %t", test.exp, got)
			}
		})
	}
}

func TestClientHandler_TagRPC(t *testing.T) {
	tests := []struct {
		name   string
		client stats.Handler
		ctx    context.Context
		info   *stats.RPCTagInfo
		exp    bool
	}{
		{
			name:   "start a span without filters",
			client: NewClientHandler(WithTracerProvider(sdltrace.NewTracerProvider())),
			ctx:    context.Background(),
			info: &stats.RPCTagInfo{
				FullMethodName: "/grpc.health.v1.Health/Check",
			},
			exp: true,
		},
		{
			name: "don't start a span with filter and match",
			client: NewClientHandler(WithTracerProvider(sdltrace.NewTracerProvider()), WithFilter(func(ri *stats.RPCTagInfo) bool {
				return ri.FullMethodName != "/grpc.health.v1.Health/Check"
			})),
			ctx: context.Background(),
			info: &stats.RPCTagInfo{
				FullMethodName: "/grpc.health.v1.Health/Check",
			},
			exp: false,
		},
		{
			name: "start a span with filter and no match",
			client: NewClientHandler(WithTracerProvider(sdltrace.NewTracerProvider()), WithFilter(func(ri *stats.RPCTagInfo) bool {
				return ri.FullMethodName != "/grpc.health.v1.Health/Check"
			})),
			ctx: context.Background(),
			info: &stats.RPCTagInfo{
				FullMethodName: "/app.v1.Service/Get",
			},
			exp: true,
		},
	}

	for _, test := range tests {
		t.Run(t.Name(), func(t *testing.T) {
			ctx := test.client.TagRPC(test.ctx, test.info)

			got := trace.SpanFromContext(ctx).IsRecording()

			if test.exp != got {
				t.Errorf("expected %t, got %t", test.exp, got)
			}
		})
	}
}
