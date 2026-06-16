// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgrpc // import "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

import (
"context"
"strconv"
"sync/atomic"
"time"

"go.opentelemetry.io/otel"
"go.opentelemetry.io/otel/attribute"
"go.opentelemetry.io/otel/codes"
"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
oldrpcconv "go.opentelemetry.io/otel/semconv/v1.37.0/rpcconv" //nolint:depguard // Use of v1.37.0 is required for backward compatibility stability opt-in.
semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
"go.opentelemetry.io/otel/semconv/v1.41.0/rpcconv"
"go.opentelemetry.io/otel/trace"

grpc_codes "google.golang.org/grpc/codes"
"google.golang.org/grpc/stats"
"google.golang.org/grpc/status"

"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc/internal"
)

type gRPCContextKey struct{}

type gRPCContext struct {
	metricAttrs []attribute.KeyValue
	record      bool

	// sentCompressedBytes accumulates the wire bytes sent across all OutPayload
	// events during the RPC, recorded as rpc.{client,server}.call.sent_compressed_length at End.
	sentCompressedBytes atomic.Int64
	// rcvdCompressedBytes accumulates the wire bytes received across all InPayload
	// events during the RPC, recorded as rpc.{client,server}.call.rcvd_compressed_length at End.
	rcvdCompressedBytes atomic.Int64
}

type serverHandler struct {
	*config

	tracer trace.Tracer

	duration    rpcconv.ServerCallDuration
	oldDuration oldrpcconv.ServerDuration

	// callStarted counts server calls started (rpc.server.call.started).
	callStarted metric.Int64Counter
	// sentCompressedLength records total compressed bytes sent per server call
	// (rpc.server.call.sent_compressed_length).
	sentCompressedLength metric.Int64Histogram
	// rcvdCompressedLength records total compressed bytes received per server call
	// (rpc.server.call.rcvd_compressed_length).
	rcvdCompressedLength metric.Int64Histogram
}

// NewServerHandler creates a stats.Handler for a gRPC server.
func NewServerHandler(opts ...Option) stats.Handler {
	c := newConfig(opts)
	if c.SpanKind == trace.SpanKindUnspecified {
		c.SpanKind = trace.SpanKindServer
	}

	h := &serverHandler{config: c}

	h.tracer = c.TracerProvider.Tracer(
ScopeName,
trace.WithInstrumentationVersion(Version),
)

	meter := c.MeterProvider.Meter(
ScopeName,
metric.WithInstrumentationVersion(Version),
metric.WithSchemaURL(semconv.SchemaURL),
)

	var err error
	if c.semconvMode == semconvModeOld || c.semconvMode == semconvModeDup {
		oldDur, err := oldrpcconv.NewServerDuration(meter)
		if err != nil {
			otel.Handle(err)
		} else {
			h.oldDuration = oldDur
		}
	}

	if c.semconvMode == semconvModeNew || c.semconvMode == semconvModeDup {
		h.duration, err = rpcconv.NewServerCallDuration(
meter,
metric.WithExplicitBucketBoundaries(
0.005, 0.01, 0.025, 0.05, 0.075, 0.1,
0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10,
),
)
		if err != nil {
			otel.Handle(err)
		}
	}

	h.callStarted, err = meter.Int64Counter(
"rpc.server.call.started",
metric.WithDescription("The number of server calls started."),
metric.WithUnit("{call}"),
)
	if err != nil {
		otel.Handle(err)
	}
		h.callStarted = noop.Int64Counter{}

	h.sentCompressedLength, err = meter.Int64Histogram(
"rpc.server.call.sent_compressed_length",
metric.WithDescription("Compressed bytes sent across all response messages per RPC."),
metric.WithUnit("By"),
metric.WithExplicitBucketBoundaries(
0, 1024, 2048, 4096, 16384, 65536, 262144,
1048576, 4194304, 16777216, 67108864,
),
)
	if err != nil {
		otel.Handle(err)
	}
		h.sentCompressedLength = noop.Int64Histogram{}

	h.rcvdCompressedLength, err = meter.Int64Histogram(
"rpc.server.call.rcvd_compressed_length",
metric.WithDescription("Compressed bytes received across all request messages per RPC."),
metric.WithUnit("By"),
metric.WithExplicitBucketBoundaries(
0, 1024, 2048, 4096, 16384, 65536, 262144,
1048576, 4194304, 16777216, 67108864,
),
)
	if err != nil {
		otel.Handle(err)
	}
		h.rcvdCompressedLength = noop.Int64Histogram{}

	return h
}

// TagConn can attach some information to the given context.
func (*serverHandler) TagConn(ctx context.Context, _ *stats.ConnTagInfo) context.Context {
	return ctx
}

// HandleConn processes the Conn stats.
func (*serverHandler) HandleConn(context.Context, stats.ConnStats) {
}

// TagRPC can attach some information to the given context.
func (h *serverHandler) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	ctx = extract(ctx, h.Propagators)

	var name string
	var attrs []attribute.KeyValue

	switch h.semconvMode {
	case semconvModeOld:
		name, attrs = internal.ParseFullMethodOld(info.FullMethodName)
	case semconvModeDup:
		var attrsNew, attrsOld []attribute.KeyValue
		name, attrsNew = internal.ParseFullMethod(info.FullMethodName)
		_, attrsOld = internal.ParseFullMethodOld(info.FullMethodName)
		attrs = append(append([]attribute.KeyValue{}, attrsOld...), attrsNew...)
		attrs = append(attrs, semconv.RPCSystemNameGRPC)
	default: // semconvModeNew
		name, attrs = internal.ParseFullMethod(info.FullMethodName)
		attrs = append(attrs, semconv.RPCSystemNameGRPC)
	}

	record := true
	if h.Filter != nil {
		record = h.Filter(info)
	}

	if record {
		spanAttributes := make([]attribute.KeyValue, 0, len(attrs)+len(h.SpanAttributes))
		spanAttributes = append(append(spanAttributes, attrs...), h.SpanAttributes...)
		opts := []trace.SpanStartOption{
			trace.WithSpanKind(h.SpanKind),
			trace.WithAttributes(spanAttributes...),
		}
		if h.PublicEndpoint || (h.PublicEndpointFn != nil && h.PublicEndpointFn(ctx, info)) {
			opts = append(opts, trace.WithNewRoot())
			if s := trace.SpanContextFromContext(ctx); s.IsValid() && s.IsRemote() {
				opts = append(opts, trace.WithLinks(trace.Link{SpanContext: s}))
			}
		}
		ctx, _ = h.tracer.Start(
trace.ContextWithRemoteSpanContext(ctx, trace.SpanContextFromContext(ctx)),
name,
opts...,
)
	}

	gctx := gRPCContext{
		metricAttrs: append(attrs, h.MetricAttributes...),
		record:      record,
	}

	if h.MetricAttributesFn != nil {
		extraAttrs := h.MetricAttributesFn(ctx)
		gctx.metricAttrs = append(gctx.metricAttrs, extraAttrs...)
	}

	return context.WithValue(ctx, gRPCContextKey{}, &gctx)
}

// HandleRPC processes the RPC stats.
func (h *serverHandler) HandleRPC(ctx context.Context, rs stats.RPCStats) {
	var dur metric.Float64Histogram
	if h.semconvMode == semconvModeNew || h.semconvMode == semconvModeDup {
		dur = h.duration.Inst()
	}
	var oldDur metric.Float64Histogram
	if h.semconvMode == semconvModeOld || h.semconvMode == semconvModeDup {
		oldDur = h.oldDuration.Inst()
	}
	h.handleRPC(
ctx,
rs,
dur,
oldDur,
h.callStarted,
nil, // no per-attempt duration on server side
h.sentCompressedLength,
h.rcvdCompressedLength,
serverStatus,
)
}

type clientHandler struct {
	*config

	tracer trace.Tracer

	duration    rpcconv.ClientCallDuration
	oldDuration oldrpcconv.ClientDuration

	// attemptStarted counts client call attempts started (rpc.client.attempt.started).
	attemptStarted metric.Int64Counter
	// attemptDuration records per-attempt latency (rpc.client.attempt.duration).
	attemptDuration metric.Float64Histogram
	// sentCompressedLength records total compressed bytes sent per attempt
	// (rpc.client.attempt.sent_compressed_length).
	sentCompressedLength metric.Int64Histogram
	// rcvdCompressedLength records total compressed bytes received per attempt
	// (rpc.client.attempt.rcvd_compressed_length).
	rcvdCompressedLength metric.Int64Histogram
}

// NewClientHandler creates a stats.Handler for a gRPC client.
func NewClientHandler(opts ...Option) stats.Handler {
	c := newConfig(opts)
	if c.SpanKind == trace.SpanKindUnspecified {
		c.SpanKind = trace.SpanKindClient
	}

	h := &clientHandler{config: c}

	h.tracer = c.TracerProvider.Tracer(
ScopeName,
trace.WithInstrumentationVersion(Version),
)

	meter := c.MeterProvider.Meter(
ScopeName,
metric.WithInstrumentationVersion(Version),
metric.WithSchemaURL(semconv.SchemaURL),
)

	var err error
	if c.semconvMode == semconvModeOld || c.semconvMode == semconvModeDup {
		oldDur, err := oldrpcconv.NewClientDuration(meter)
		if err != nil {
			otel.Handle(err)
		} else {
			h.oldDuration = oldDur
		}
	}

	if c.semconvMode == semconvModeNew || c.semconvMode == semconvModeDup {
		h.duration, err = rpcconv.NewClientCallDuration(
meter,
metric.WithExplicitBucketBoundaries(
0.005, 0.01, 0.025, 0.05, 0.075, 0.1,
0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10,
),
)
		if err != nil {
			otel.Handle(err)
		}
	}

	h.attemptStarted, err = meter.Int64Counter(
"rpc.client.attempt.started",
metric.WithDescription("The number of client call attempts started."),
metric.WithUnit("{attempt}"),
)
	if err != nil {
		otel.Handle(err)
	}
		h.attemptStarted = noop.Int64Counter{}

	h.attemptDuration, err = meter.Float64Histogram(
"rpc.client.attempt.duration",
metric.WithDescription("Measures the duration of an individual attempt of an outgoing RPC."),
metric.WithUnit("s"),
metric.WithExplicitBucketBoundaries(
0.005, 0.01, 0.025, 0.05, 0.075, 0.1,
0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10,
),
)
	if err != nil {
		otel.Handle(err)
	}
		h.attemptDuration = noop.Float64Histogram{}

	h.sentCompressedLength, err = meter.Int64Histogram(
"rpc.client.attempt.sent_compressed_length",
metric.WithDescription("Compressed bytes sent across all request messages per RPC attempt."),
metric.WithUnit("By"),
metric.WithExplicitBucketBoundaries(
0, 1024, 2048, 4096, 16384, 65536, 262144,
1048576, 4194304, 16777216, 67108864,
),
)
	if err != nil {
		otel.Handle(err)
	}
		h.sentCompressedLength = noop.Int64Histogram{}

	h.rcvdCompressedLength, err = meter.Int64Histogram(
"rpc.client.attempt.rcvd_compressed_length",
metric.WithDescription("Compressed bytes received across all response messages per RPC attempt."),
metric.WithUnit("By"),
metric.WithExplicitBucketBoundaries(
0, 1024, 2048, 4096, 16384, 65536, 262144,
1048576, 4194304, 16777216, 67108864,
),
)
	if err != nil {
		otel.Handle(err)
	}
		h.rcvdCompressedLength = noop.Int64Histogram{}

	return h
}

// TagRPC can attach some information to the given context.
func (h *clientHandler) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	var name string
	var attrs []attribute.KeyValue

	switch h.semconvMode {
	case semconvModeOld:
		name, attrs = internal.ParseFullMethodOld(info.FullMethodName)
	case semconvModeDup:
		var attrsNew, attrsOld []attribute.KeyValue
		name, attrsNew = internal.ParseFullMethod(info.FullMethodName)
		_, attrsOld = internal.ParseFullMethodOld(info.FullMethodName)
		attrs = append(append([]attribute.KeyValue{}, attrsOld...), attrsNew...)
		attrs = append(attrs, semconv.RPCSystemNameGRPC)
	default: // semconvModeNew
		name, attrs = internal.ParseFullMethod(info.FullMethodName)
		attrs = append(attrs, semconv.RPCSystemNameGRPC)
	}

	record := true
	if h.Filter != nil {
		record = h.Filter(info)
	}

	if record {
		spanAttributes := make([]attribute.KeyValue, 0, len(attrs)+len(h.SpanAttributes))
		spanAttributes = append(append(spanAttributes, attrs...), h.SpanAttributes...)
		ctx, _ = h.tracer.Start(
ctx,
name,
trace.WithSpanKind(h.SpanKind),
trace.WithAttributes(spanAttributes...),
)
	}

	gctx := gRPCContext{
		metricAttrs: append(attrs, h.MetricAttributes...),
		record:      record,
	}

	if h.MetricAttributesFn != nil {
		extraAttrs := h.MetricAttributesFn(ctx)
		gctx.metricAttrs = append(gctx.metricAttrs, extraAttrs...)
	}

	return inject(context.WithValue(ctx, gRPCContextKey{}, &gctx), h.Propagators)
}

// HandleRPC processes the RPC stats.
func (h *clientHandler) HandleRPC(ctx context.Context, rs stats.RPCStats) {
	var dur metric.Float64Histogram
	if h.semconvMode == semconvModeNew || h.semconvMode == semconvModeDup {
		dur = h.duration.Inst()
	}
	var oldDur metric.Float64Histogram
	if h.semconvMode == semconvModeOld || h.semconvMode == semconvModeDup {
		oldDur = h.oldDuration.Inst()
	}
	h.handleRPC(
ctx,
rs,
dur,
oldDur,
h.attemptStarted,
h.attemptDuration,
h.sentCompressedLength,
h.rcvdCompressedLength,
func(s *status.Status) (codes.Code, string) {
return codes.Error, s.Message()
		},
	)
}

// TagConn can attach some information to the given context.
func (*clientHandler) TagConn(ctx context.Context, _ *stats.ConnTagInfo) context.Context {
	return ctx
}

// HandleConn processes the Conn stats.
func (*clientHandler) HandleConn(context.Context, stats.ConnStats) {
	// no-op
}

func (*config) handleRPC(
ctx context.Context,
rs stats.RPCStats,
duration metric.Float64Histogram,
oldDuration metric.Float64Histogram,
startedCounter metric.Int64Counter,
attemptDuration metric.Float64Histogram,
sentCompressedLength metric.Int64Histogram,
rcvdCompressedLength metric.Int64Histogram,
recordStatus func(*status.Status) (codes.Code, string),
) {
	gctx, _ := ctx.Value(gRPCContextKey{}).(*gRPCContext)
	if gctx != nil && !gctx.record {
		return
	}

	span := trace.SpanFromContext(ctx)

	switch rs := rs.(type) {
	case *stats.Begin:
		// Record a new call/attempt started.
		if startedCounter != nil {
			var metricAttrs []attribute.KeyValue
			if gctx != nil {
				metricAttrs = gctx.metricAttrs
			}
			startedCounter.Add(ctx, 1, metric.WithAttributeSet(attribute.NewSet(metricAttrs...)))
		}
	case *stats.InPayload:
		// Accumulate compressed bytes received; recorded in total at End.
		if gctx != nil {
			gctx.rcvdCompressedBytes.Add(int64(rs.WireLength))
		}
	case *stats.InHeader:
		if !rs.Client && rs.LocalAddr != nil {
			if span.IsRecording() {
				span.SetAttributes(serverAddrAttrs(rs.LocalAddr.String())...)
			}
			// TODO: add server.address and server.port to metrics once the API supports opt-in attributes.
		}
	case *stats.OutPayload:
		// Accumulate compressed bytes sent; recorded in total at End.
		if gctx != nil {
			gctx.sentCompressedBytes.Add(int64(rs.WireLength))
		}
	case *stats.OutTrailer:
	case *stats.OutHeader:
		if rs.Client && rs.RemoteAddr != nil && (span.IsRecording() || gctx != nil) {
			attrs := serverAddrAttrs(rs.RemoteAddr.String())
			if span.IsRecording() {
				span.SetAttributes(attrs...)
			}
			if gctx != nil {
				gctx.metricAttrs = append(gctx.metricAttrs, attrs...)
			}
		}
	case *stats.End:
		var rpcStatusAttr attribute.KeyValue

		var s *status.Status
		if rs.Error != nil {
			s, _ = status.FromError(rs.Error)
			rpcStatusAttr = semconv.RPCResponseStatusCode(canonicalString(s.Code()))
		} else {
			rpcStatusAttr = semconv.RPCResponseStatusCode(canonicalString(grpc_codes.OK))
		}
		if span.IsRecording() {
			if s != nil {
				c, m := recordStatus(s)
				span.SetStatus(c, m)
			}
			span.SetAttributes(rpcStatusAttr)
			span.End()
		}

		var durationEnabled bool
		var oldDurationEnabled bool

		if duration != nil {
			durationEnabled = duration.Enabled(ctx)
		}
		if oldDuration != nil {
			oldDurationEnabled = oldDuration.Enabled(ctx)
		}

		var metricAttrs []attribute.KeyValue
		if gctx != nil {
			metricAttrs = make([]attribute.KeyValue, 0, len(gctx.metricAttrs)+1)
			metricAttrs = append(metricAttrs, gctx.metricAttrs...)
		}
		metricAttrs = append(metricAttrs, rpcStatusAttr)

		// Allocate option slices once and reuse.
		recordOpts := []metric.RecordOption{metric.WithAttributeSet(attribute.NewSet(metricAttrs...))}
		addOpts := []metric.AddOption{metric.WithAttributeSet(attribute.NewSet(metricAttrs...))}
		_ = addOpts // used below for int64 histograms via Record

		// Use floating point division for higher precision.
		// Measure right before Record() to capture as much elapsed time as possible.
		elapsedTime := float64(rs.EndTime.Sub(rs.BeginTime)) / float64(time.Second)

		if durationEnabled {
			duration.Record(ctx, elapsedTime, recordOpts...)
		}
		if oldDurationEnabled {
			oldDuration.Record(ctx, elapsedTime*1000.0, recordOpts...)
		}

		// Record per-attempt duration (client side only; nil on server side).
		if attemptDuration != nil && attemptDuration.Enabled(ctx) {
			attemptDuration.Record(ctx, elapsedTime, recordOpts...)
		}

		// Record accumulated compressed message sizes.
		if gctx != nil {
			if sentCompressedLength != nil && sentCompressedLength.Enabled(ctx) {
				sentCompressedLength.Record(ctx, gctx.sentCompressedBytes.Load(), recordOpts...)
			}
			if rcvdCompressedLength != nil && rcvdCompressedLength.Enabled(ctx) {
				rcvdCompressedLength.Record(ctx, gctx.rcvdCompressedBytes.Load(), recordOpts...)
			}
		}

	default:
		return
	}
}

func canonicalString(code grpc_codes.Code) string {
	switch code {
	case grpc_codes.OK:
		return "OK"
	case grpc_codes.Canceled:
		return "CANCELLED"
	case grpc_codes.Unknown:
		return "UNKNOWN"
	case grpc_codes.InvalidArgument:
		return "INVALID_ARGUMENT"
	case grpc_codes.DeadlineExceeded:
		return "DEADLINE_EXCEEDED"
	case grpc_codes.NotFound:
		return "NOT_FOUND"
	case grpc_codes.AlreadyExists:
		return "ALREADY_EXISTS"
	case grpc_codes.PermissionDenied:
		return "PERMISSION_DENIED"
	case grpc_codes.ResourceExhausted:
		return "RESOURCE_EXHAUSTED"
	case grpc_codes.FailedPrecondition:
		return "FAILED_PRECONDITION"
	case grpc_codes.Aborted:
		return "ABORTED"
	case grpc_codes.OutOfRange:
		return "OUT_OF_RANGE"
	case grpc_codes.Unimplemented:
		return "UNIMPLEMENTED"
	case grpc_codes.Internal:
		return "INTERNAL"
	case grpc_codes.Unavailable:
		return "UNAVAILABLE"
	case grpc_codes.DataLoss:
		return "DATA_LOSS"
	case grpc_codes.Unauthenticated:
		return "UNAUTHENTICATED"
	default:
		return "CODE(" + strconv.FormatInt(int64(code), 10) + ")"
	}
}
