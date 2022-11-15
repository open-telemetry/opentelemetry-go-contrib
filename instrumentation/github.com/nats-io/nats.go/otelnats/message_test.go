package otelnats

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func Test_NewMsg(t *testing.T) {
	t.Run("Should return new *nats.Msg if context is not a part of trace.", func(t *testing.T) {
		ctx := context.TODO()

		msg := NewMsg(ctx)

		assert.Equal(t, "", msg.Header.Get(HeaderTraceIdKey))
		assert.Equal(t, "", msg.Header.Get(HeaderSpanIdKey))
	})

	t.Run("Should return new *nats.Msg with spanId and traceId headers.", func(t *testing.T) {
		parent := context.TODO()

		tracer := sdktrace.NewTracerProvider().Tracer("tracer")

		ctx, span := tracer.Start(parent, "newTrace")
		defer span.End()

		msg := NewMsg(ctx)

		assert.NotEmpty(t, msg.Header.Get(HeaderTraceIdKey))
		assert.NotEmpty(t, msg.Header.Get(HeaderSpanIdKey))
	})
}
