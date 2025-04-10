// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgrpc

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/stats"

	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/sdk/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

var (
	noopTraceProvider = tracenoop.NewTracerProvider()
	noopMeterProvider = metricnoop.NewMeterProvider()
)

var rpcStats = []stats.RPCStats{
	&stats.Begin{
		BeginTime: time.Now(),
	},
	&stats.InPayload{
		Length:           1024,
		CompressedLength: 512,
	},
	&stats.OutPayload{
		Length:           1024,
		CompressedLength: 512,
	},
	&stats.OutTrailer{},
	&stats.OutHeader{},
	&stats.End{
		EndTime: time.Now().Add(10 * time.Second),
	},
	nil,
}

func BenchmarkServerHandler_HandleRPC_NoOp(b *testing.B) {
	handler := NewServerHandler(
		WithTracerProvider(noopTraceProvider),
		WithMeterProvider(noopMeterProvider),
		WithMessageEvents(ReceivedEvents, SentEvents),
	)
	ctx := context.Background()
	ctx = handler.TagRPC(ctx, &stats.RPCTagInfo{
		FullMethodName: "/package.service/method",
	})
	ctx = peer.NewContext(ctx, &peer.Peer{
		Addr: &net.TCPAddr{
			IP:   net.IPv4(127, 0, 0, 1),
			Port: 1234,
		},
	})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, rs := range rpcStats {
			handler.HandleRPC(ctx, rs)
		}
	}
}

func BenchmarkServerHandler_HandleRPC(b *testing.B) {
	handler := NewServerHandler(
		WithTracerProvider(trace.NewTracerProvider(
			trace.WithSampler(trace.AlwaysSample()),
		)),
		WithMeterProvider(noopMeterProvider),
		WithMessageEvents(ReceivedEvents, SentEvents),
	)
	ctx := context.Background()
	ctx = handler.TagRPC(ctx, &stats.RPCTagInfo{
		FullMethodName: "/package.service/method",
	})
	ctx = peer.NewContext(ctx, &peer.Peer{
		Addr: &net.TCPAddr{
			IP:   net.IPv4(127, 0, 0, 1),
			Port: 1234,
		},
	})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, rs := range rpcStats {
			handler.HandleRPC(ctx, rs)
		}
	}
}
