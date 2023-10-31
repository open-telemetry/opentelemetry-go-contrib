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

package otelgrpc // import "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

import (
	"context"
	"sync/atomic"

	grpc_codes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc/internal"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

type gRPCContextKey struct{}

type gRPCContext struct {
	messagesReceived int64
	messagesSent     int64
	metricAttrs      []attribute.KeyValue
}

type handler struct {
	tracer             trace.Tracer
	meter              metric.Meter
	rpcDuration        metric.Float64Histogram
	rpcRequestSize     metric.Int64Histogram
	rpcResponseSize    metric.Int64Histogram
	rpcRequestsPerRPC  metric.Int64Histogram
	rpcResponsesPerRPC metric.Int64Histogram
}

type serverHandler struct {
	*config
	r *handler
}

// NewServerHandler creates a stats.Handler for gRPC server.
func NewServerHandler(opts ...Option) stats.Handler {
	h := &serverHandler{
		config: newConfig(opts),
		r:      &handler{},
	}

	h.r.tracer = h.config.TracerProvider.Tracer(
		instrumentationName,
		trace.WithInstrumentationVersion(SemVersion()),
	)
	h.r.meter = h.config.MeterProvider.Meter(
		instrumentationName,
		metric.WithInstrumentationVersion(Version()),
		metric.WithSchemaURL(semconv.SchemaURL),
	)
	var err error
	h.r.rpcDuration, err = h.r.meter.Float64Histogram("rpc.server.duration",
		metric.WithDescription("Measures the duration of inbound RPC."),
		metric.WithUnit("ms"))
	if err != nil {
		otel.Handle(err)
	}

	h.r.rpcRequestSize, err = h.r.meter.Int64Histogram("rpc.server.request.size",
		metric.WithDescription("Measures size of RPC request messages (uncompressed)."),
		metric.WithUnit("By"))
	if err != nil {
		otel.Handle(err)
	}

	h.r.rpcResponseSize, err = h.r.meter.Int64Histogram("rpc.server.response.size",
		metric.WithDescription("Measures size of RPC response messages (uncompressed)."),
		metric.WithUnit("By"))
	if err != nil {
		otel.Handle(err)
	}

	h.r.rpcRequestsPerRPC, err = h.r.meter.Int64Histogram("rpc.server.requests_per_rpc",
		metric.WithDescription("Measures the number of messages received per RPC. Should be 1 for all non-streaming RPCs."),
		metric.WithUnit("{count}"))
	if err != nil {
		otel.Handle(err)
	}

	h.r.rpcResponsesPerRPC, err = h.r.meter.Int64Histogram("rpc.server.responses_per_rpc",
		metric.WithDescription("Measures the number of messages received per RPC. Should be 1 for all non-streaming RPCs."),
		metric.WithUnit("{count}"))
	if err != nil {
		otel.Handle(err)
	}

	return h
}

// TagConn can attach some information to the given context.
func (h *serverHandler) TagConn(ctx context.Context, info *stats.ConnTagInfo) context.Context {
	span := trace.SpanFromContext(ctx)
	attrs := peerAttr(peerFromCtx(ctx))
	span.SetAttributes(attrs...)

	return ctx
}

// HandleConn processes the Conn stats.
func (h *serverHandler) HandleConn(ctx context.Context, info stats.ConnStats) {
}

// TagRPC can attach some information to the given context.
func (h *serverHandler) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	ctx = extract(ctx, h.config.Propagators)

	name, attrs := internal.ParseFullMethod(info.FullMethodName)
	attrs = append(attrs, RPCSystemGRPC)
	ctx, _ = h.r.tracer.Start(
		trace.ContextWithRemoteSpanContext(ctx, trace.SpanContextFromContext(ctx)),
		name,
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(attrs...),
	)

	gctx := gRPCContext{
		metricAttrs: attrs,
	}
	return context.WithValue(ctx, gRPCContextKey{}, &gctx)
}

// HandleRPC processes the RPC stats.
func (h *serverHandler) HandleRPC(ctx context.Context, rs stats.RPCStats) {
	h.r.handleRPC(ctx, rs)
}

type clientHandler struct {
	*config
	r *handler
}

// NewClientHandler creates a stats.Handler for gRPC client.
func NewClientHandler(opts ...Option) stats.Handler {
	h := &clientHandler{
		config: newConfig(opts),
		r:      &handler{},
	}

	h.r.tracer = h.config.TracerProvider.Tracer(
		instrumentationName,
		trace.WithInstrumentationVersion(SemVersion()),
	)

	h.r.meter = h.config.MeterProvider.Meter(
		instrumentationName,
		metric.WithInstrumentationVersion(Version()),
		metric.WithSchemaURL(semconv.SchemaURL),
	)
	var err error
	h.r.rpcDuration, err = h.r.meter.Float64Histogram("rpc.client.duration",
		metric.WithDescription("Measures the duration of inbound RPC."),
		metric.WithUnit("ms"))
	if err != nil {
		otel.Handle(err)
	}

	h.r.rpcRequestSize, err = h.r.meter.Int64Histogram("rpc.client.request.size",
		metric.WithDescription("Measures size of RPC request messages (uncompressed)."),
		metric.WithUnit("By"))
	if err != nil {
		otel.Handle(err)
	}

	h.r.rpcResponseSize, err = h.r.meter.Int64Histogram("rpc.client.response.size",
		metric.WithDescription("Measures size of RPC response messages (uncompressed)."),
		metric.WithUnit("By"))
	if err != nil {
		otel.Handle(err)
	}

	h.r.rpcRequestsPerRPC, err = h.r.meter.Int64Histogram("rpc.client.requests_per_rpc",
		metric.WithDescription("Measures the number of messages received per RPC. Should be 1 for all non-streaming RPCs."),
		metric.WithUnit("{count}"))
	if err != nil {
		otel.Handle(err)
	}

	h.r.rpcResponsesPerRPC, err = h.r.meter.Int64Histogram("rpc.client.responses_per_rpc",
		metric.WithDescription("Measures the number of messages received per RPC. Should be 1 for all non-streaming RPCs."),
		metric.WithUnit("{count}"))
	if err != nil {
		otel.Handle(err)
	}

	return h
}

// TagRPC can attach some information to the given context.
func (h *clientHandler) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	name, attrs := internal.ParseFullMethod(info.FullMethodName)
	attrs = append(attrs, RPCSystemGRPC)
	ctx, _ = h.r.tracer.Start(
		ctx,
		name,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attrs...),
	)

	gctx := gRPCContext{
		metricAttrs: attrs,
	}

	return inject(context.WithValue(ctx, gRPCContextKey{}, &gctx), h.config.Propagators)
}

// HandleRPC processes the RPC stats.
func (h *clientHandler) HandleRPC(ctx context.Context, rs stats.RPCStats) {
	h.r.handleRPC(ctx, rs)
}

// TagConn can attach some information to the given context.
func (h *clientHandler) TagConn(ctx context.Context, cti *stats.ConnTagInfo) context.Context {
	span := trace.SpanFromContext(ctx)
	attrs := peerAttr(cti.RemoteAddr.String())
	span.SetAttributes(attrs...)
	return ctx
}

// HandleConn processes the Conn stats.
func (h *clientHandler) HandleConn(context.Context, stats.ConnStats) {
	// no-op
}

func (r *handler) handleRPC(ctx context.Context, rs stats.RPCStats) {
	span := trace.SpanFromContext(ctx)
	gctx, _ := ctx.Value(gRPCContextKey{}).(*gRPCContext)
	var messageId int64
	metricAttrs := make([]attribute.KeyValue, 0, len(gctx.metricAttrs)+1)
	metricAttrs = append(metricAttrs, gctx.metricAttrs...)

	switch rs := rs.(type) {
	case *stats.Begin:
	case *stats.InPayload:
		if gctx != nil {
			messageId = atomic.AddInt64(&gctx.messagesReceived, 1)
			r.rpcRequestSize.Record(ctx, int64(rs.Length), metric.WithAttributes(metricAttrs...))
		}

		span.AddEvent("message",
			trace.WithAttributes(
				semconv.MessageTypeReceived,
				semconv.MessageIDKey.Int64(messageId),
				semconv.MessageCompressedSizeKey.Int(rs.CompressedLength),
				semconv.MessageUncompressedSizeKey.Int(rs.Length),
			),
		)
	case *stats.OutPayload:
		if gctx != nil {
			messageId = atomic.AddInt64(&gctx.messagesSent, 1)
			r.rpcResponseSize.Record(ctx, int64(rs.Length), metric.WithAttributes(metricAttrs...))
		}

		span.AddEvent("message",
			trace.WithAttributes(
				semconv.MessageTypeSent,
				semconv.MessageIDKey.Int64(messageId),
				semconv.MessageCompressedSizeKey.Int(rs.CompressedLength),
				semconv.MessageUncompressedSizeKey.Int(rs.Length),
			),
		)
	case *stats.OutTrailer:
	case *stats.End:
		var rpcStatusAttr attribute.KeyValue

		if rs.Error != nil {
			s, _ := status.FromError(rs.Error)
			span.SetStatus(codes.Error, s.Message())
			rpcStatusAttr = semconv.RPCGRPCStatusCodeKey.Int(int(s.Code()))
		} else {
			rpcStatusAttr = semconv.RPCGRPCStatusCodeKey.Int(int(grpc_codes.OK))
		}
		span.SetAttributes(rpcStatusAttr)
		span.End()

		metricAttrs = append(metricAttrs, rpcStatusAttr)
		r.rpcDuration.Record(context.TODO(), float64(rs.EndTime.Sub(rs.BeginTime)), metric.WithAttributes(metricAttrs...))
		r.rpcRequestsPerRPC.Record(context.TODO(), gctx.messagesReceived, metric.WithAttributes(metricAttrs...))
		r.rpcResponsesPerRPC.Record(context.TODO(), gctx.messagesSent, metric.WithAttributes(metricAttrs...))

	default:
		return
	}
}
