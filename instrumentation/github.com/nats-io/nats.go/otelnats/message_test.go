package otelnats

import (
	"context"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func Test_NewMsg(t *testing.T) {
	t.Run("Should return new *nats.Msg if context is not a part of trace.", func(t *testing.T) {
		ctx := context.TODO()

		msg := NewMsg(ctx)

		assert.Equal(t, "", msg.Header.Get(HeaderTraceIdKey))
		assert.Equal(t, "", msg.Header.Get(HeaderSpanIdKey))
	})

	tracer := sdktrace.NewTracerProvider().Tracer("tracer")

	t.Run("Should return new *nats.Msg with spanId and traceId headers.", func(t *testing.T) {
		parent := context.TODO()

		ctx, span := tracer.Start(parent, "newSpan")
		defer span.End()

		msg := NewMsg(ctx)

		assert.NotEmpty(t, msg.Header.Get(HeaderTraceIdKey))
		assert.NotEmpty(t, msg.Header.Get(HeaderSpanIdKey))
	})

	t.Run("Should reconstruct spanID and traceID from message headers.", func(t *testing.T) {
		parent := context.TODO()

		spanCtx, span := tracer.Start(parent, "newSpan")
		defer span.End()

		msg := NewMsg(spanCtx)

		ctx := NewCtxFrom(msg)

		assert.Equal(t, msg.Header, NewMsg(ctx).Header)

		spanFromMsg := trace.SpanFromContext(ctx)
		defer spanFromMsg.End()

		assert.Equal(t, spanFromMsg.SpanContext().TraceID(), span.SpanContext().TraceID())
		assert.Equal(t, spanFromMsg.SpanContext().SpanID(), span.SpanContext().SpanID())
	})

	t.Run("Should create span from message.", func(t *testing.T) {
		parent := context.TODO()

		spanCtx, span := tracer.Start(parent, "newSpan")
		defer span.End()

		msg := NewMsg(spanCtx)

		spanFromMsg := NewSpanFrom(msg)
		defer spanFromMsg.End()

		assert.Equal(t, spanFromMsg.SpanContext().TraceID(), span.SpanContext().TraceID())
		assert.Equal(t, spanFromMsg.SpanContext().SpanID(), span.SpanContext().SpanID())
	})

	t.Run("Should inject trace and span ids.", func(t *testing.T) {
		parent := context.TODO()

		spanCtx, span := tracer.Start(parent, "newSpan")
		defer span.End()

		msg := &nats.Msg{}
		Inject(spanCtx, msg)

		spanFromMsg := NewSpanFrom(msg)
		defer spanFromMsg.End()

		assert.Equal(t, spanFromMsg.SpanContext().TraceID(), span.SpanContext().TraceID())
		assert.Equal(t, spanFromMsg.SpanContext().SpanID(), span.SpanContext().SpanID())
	})
}
