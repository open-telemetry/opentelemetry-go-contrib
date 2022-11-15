package otelnats

import (
	"context"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/trace"
)

const (
	HeaderTraceIdKey = "otel.parent.TraceID"
	HeaderSpanIdKey  = "otel.parent.SpanID"
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
