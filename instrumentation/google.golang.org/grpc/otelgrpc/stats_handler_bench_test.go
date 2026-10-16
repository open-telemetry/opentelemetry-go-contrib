// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgrpc

import (
	"context"
	"net"
	"testing"
	"time"

	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"
)

func benchmarkStatsHandlerHandleRPC(b *testing.B, ctx context.Context, handler stats.Handler, stat stats.RPCStats) {
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		handler.HandleRPC(ctx, stat)
	}
}

func benchmarkServerHandlerHandleRPC(b *testing.B, stat stats.RPCStats) {
	handler := NewServerHandler(
		WithTracerProvider(trace.NewTracerProvider(
			trace.WithSampler(trace.AlwaysSample()),
		)),
		WithMeterProvider(metric.NewMeterProvider()),
		WithMessageEvents(ReceivedEvents, SentEvents),
	)
	ctx := b.Context()
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
	ctx := b.Context()
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

func BenchmarkServerHandler_TagRPCNoOp(b *testing.B) {
	handler := NewServerHandler(
		WithTracerProvider(tracenoop.NewTracerProvider()),
		WithMeterProvider(metricnoop.NewMeterProvider()),
		WithMessageEvents(ReceivedEvents, SentEvents),
	)
	ctx := b.Context()

	tagInfo := &stats.RPCTagInfo{
		FullMethodName: "/package.service/method",
	}

	b.ReportAllocs()
	for b.Loop() {
		_ = handler.TagRPC(ctx, tagInfo)
	}
}

func BenchmarkClientHandler_TagRPCNoOp(b *testing.B) {
	handler := NewClientHandler(
		WithTracerProvider(tracenoop.NewTracerProvider()),
		WithMeterProvider(metricnoop.NewMeterProvider()),
		WithMessageEvents(ReceivedEvents, SentEvents),
	)
	ctx := b.Context()

	tagInfo := &stats.RPCTagInfo{
		FullMethodName: "/package.service/method",
	}

	b.ReportAllocs()
	for b.Loop() {
		_ = handler.TagRPC(ctx, tagInfo)
	}
}

// benchmarkServerHandlerHandleRPCEnd runs the End path with a reader-enabled
// meter provider so the duration histogram is actually recorded. The
// existing End benchmarks use a meter provider without a reader, which
// disables the instruments and cannot measure the record hot path.
func benchmarkServerHandlerHandleRPCEnd(b *testing.B, endErr error) {
	handler := NewServerHandler(
		WithTracerProvider(trace.NewTracerProvider(trace.WithSampler(trace.AlwaysSample()))),
		WithMeterProvider(metric.NewMeterProvider(metric.WithReader(metric.NewManualReader()))),
	)
	ctx := b.Context()
	ctx = handler.TagRPC(ctx, &stats.RPCTagInfo{
		FullMethodName: "/package.service/method",
	})
	ctx = peer.NewContext(ctx, &peer.Peer{
		Addr: &net.TCPAddr{
			IP:   net.IPv4(127, 0, 0, 1),
			Port: 1234,
		},
	})
	benchmarkStatsHandlerHandleRPC(b, ctx, handler, &stats.End{
		EndTime: time.Now(),
		Error:   endErr,
	})
}

// BenchmarkServerHandler_HandleRPC_End_WithReader measures a successful RPC
// completion with the duration histogram enabled.
func BenchmarkServerHandler_HandleRPC_End_WithReader(b *testing.B) {
	benchmarkServerHandlerHandleRPCEnd(b, nil)
}

// BenchmarkServerHandler_HandleRPC_End_Error_WithReader measures a failed RPC
// completion (the error.type attribute path) with the duration histogram
// enabled.
func BenchmarkServerHandler_HandleRPC_End_Error_WithReader(b *testing.B) {
	benchmarkServerHandlerHandleRPCEnd(b, status.Error(codes.Internal, "internal"))
}

// BenchmarkServerHandler_HandleRPC_End_Error_OldMode_WithReader measures a
// failed RPC completion in rpc/old mode: only the legacy histogram is
// enabled, so the stable error.type attribute must not be built.
func BenchmarkServerHandler_HandleRPC_End_Error_OldMode_WithReader(b *testing.B) {
	b.Setenv("OTEL_SEMCONV_STABILITY_OPT_IN", "rpc/old")
	benchmarkServerHandlerHandleRPCEnd(b, status.Error(codes.Internal, "internal"))
}
