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

	"go.opentelemetry.io/otel/sdk/trace"

	metricnoop "go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

func benchmarkStatsHandlerHandleRPC(b *testing.B, ctx context.Context, handler stats.Handler, stat stats.RPCStats) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.HandleRPC(ctx, stat)
	}
}

func benchmarkServerHandlerHandleRPC(b *testing.B, stat stats.RPCStats) {
	handler := NewServerHandler(
		WithTracerProvider(trace.NewTracerProvider(
			trace.WithSampler(trace.AlwaysSample()),
		)),
		WithMeterProvider(metricnoop.NewMeterProvider()),
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
	benchmarkStatsHandlerHandleRPC(b, ctx, handler, stat)
}

func BenchmarkServerHandler_HandleRPC_Begin(b *testing.B) {
	benchmarkServerHandlerHandleRPC(b, &stats.Begin{
		BeginTime: time.Now(),
	})
}

func BenchmarkServerHandler_HandleRPC_InPayload(b *testing.B) {
	benchmarkServerHandlerHandleRPC(b, &stats.InPayload{
		Length:           1024,
		CompressedLength: 512,
	})
}

func BenchmarkServerHandler_HandleRPC_OutPayload(b *testing.B) {
	benchmarkServerHandlerHandleRPC(b, &stats.OutPayload{
		Length:           1024,
		CompressedLength: 512,
	})
}

func BenchmarkServerHandler_HandleRPC_OutTrailer(b *testing.B) {
	benchmarkServerHandlerHandleRPC(b, &stats.OutTrailer{})
}

func BenchmarkServerHandler_HandleRPC_OutHeader(b *testing.B) {
	benchmarkServerHandlerHandleRPC(b, &stats.OutHeader{})
}

func BenchmarkServerHandler_HandleRPC_End(b *testing.B) {
	benchmarkServerHandlerHandleRPC(b, &stats.End{
		EndTime: time.Now(),
	})
}

func BenchmarkServerHandler_HandleRPC_Nil(b *testing.B) {
	benchmarkServerHandlerHandleRPC(b, nil)
}

func benchmarkServerHandlerHandleRPCNoOp(b *testing.B, stat stats.RPCStats) {
	handler := NewServerHandler(
		WithTracerProvider(tracenoop.NewTracerProvider()),
		WithMeterProvider(metricnoop.NewMeterProvider()),
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

	benchmarkStatsHandlerHandleRPC(b, ctx, handler, stat)
}

func BenchmarkServerHandler_HandleRPC_NoOp_Begin(b *testing.B) {
	benchmarkServerHandlerHandleRPCNoOp(b, &stats.Begin{
		BeginTime: time.Now(),
	})
}

func BenchmarkServerHandler_HandleRPC_NoOp_InPayload(b *testing.B) {
	benchmarkServerHandlerHandleRPCNoOp(b, &stats.InPayload{
		Length:           1024,
		CompressedLength: 512,
	})
}

func BenchmarkServerHandler_HandleRPC_NoOp_OutPayload(b *testing.B) {
	benchmarkServerHandlerHandleRPCNoOp(b, &stats.OutPayload{
		Length:           1024,
		CompressedLength: 512,
	})
}

func BenchmarkServerHandler_HandleRPC_NoOp_OutTrailer(b *testing.B) {
	benchmarkServerHandlerHandleRPCNoOp(b, &stats.OutTrailer{})
}

func BenchmarkServerHandler_HandleRPC_NoOp_OutHeader(b *testing.B) {
	benchmarkServerHandlerHandleRPCNoOp(b, &stats.OutHeader{})
}

func BenchmarkServerHandler_HandleRPC_NoOp_End(b *testing.B) {
	benchmarkServerHandlerHandleRPCNoOp(b, &stats.End{
		EndTime: time.Now(),
	})
}

func BenchmarkServerHandler_HandleRPC_NoOp_Nil(b *testing.B) {
	benchmarkServerHandlerHandleRPCNoOp(b, nil)
}
