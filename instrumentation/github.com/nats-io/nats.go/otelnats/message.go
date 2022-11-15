package otelnats

import (
	"context"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/trace"
)

const (
	// HeaderTraceIdKey represents otel parent trace header key that will be used to propagate TraceID in nats.Msg.
	HeaderTraceIdKey = "otel.parent.TraceID"
	// HeaderTraceIdKey represents otel parent trace header key that will be used to propagate SpanID in nats.Msg.
	HeaderSpanIdKey = "otel.parent.SpanID"
)

// NewMsg will create new *nats.Msg with nats.Header initialized with TraceID and SpanID from the context.
// If either TraceID or SpanID are not present in the context new *nats.Msg will be created with empty header.
func NewMsg(ctx context.Context) (msg *nats.Msg) {
	span := trace.SpanFromContext(ctx)
	spanCtx := span.SpanContext()

	header := nats.Header{}

	if spanCtx.HasTraceID() {
		header.Set(HeaderTraceIdKey, spanCtx.TraceID().String())
	}
	if spanCtx.HasSpanID() {
		header.Set(HeaderSpanIdKey, spanCtx.SpanID().String())
	}

	msg = &nats.Msg{
		Header: header,
	}
	return
}

// NewCtxFrom will return new context.Context with initialized traceID and spanID from nats.Msg and context.Background().
func NewCtxFrom(msg *nats.Msg) (ctx context.Context) {
	return CtxFrom(context.Background(), msg)
}

// CtxFrom will return new context.Context with initialized traceID and spanID from nats.Msg and parent context.Context.
// If either TraceID or SpanID are not present in the *nats.Msg parent context will be returned.
func CtxFrom(parent context.Context, msg *nats.Msg) (ctx context.Context) {
	traceID, err := trace.TraceIDFromHex(msg.Header.Get(HeaderTraceIdKey))
	if err != nil {
		ctx = parent
		return
	}
	spanID, err := trace.SpanIDFromHex(msg.Header.Get(HeaderSpanIdKey))
	if err != nil {
		ctx = parent
		return
	}
	spanCtxCfg := trace.SpanContextConfig{
		TraceID: traceID,
		SpanID:  spanID,
		Remote:  true,
	}
	spanCtx := trace.NewSpanContext(spanCtxCfg)
	ctx = trace.ContextWithSpanContext(parent, spanCtx)
	return
}

// SpanFrom will return new span from exisiting context initialized with traceID and spanID from *nats.Msg.
func SpanFrom(parent context.Context, msg *nats.Msg) (span trace.Span) {
	span = trace.SpanFromContext(CtxFrom(parent, msg))
	return
}

// SpanFrom will return new span from context.Background() initialized with traceID and spanID from *nats.Msg.
func NewSpanFrom(msg *nats.Msg) (span trace.Span) {
	span = trace.SpanFromContext(NewCtxFrom(msg))
	return
}
