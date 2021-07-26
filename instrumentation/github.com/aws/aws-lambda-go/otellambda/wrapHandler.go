package otellambda

import (
	"context"

	"github.com/aws/aws-lambda-go/lambda"

	"go.opentelemetry.io/otel"
)

type wrappedHandler struct {
	handler lambda.Handler
}

// Compile time check our Handler implements lambda.Handler
var _ lambda.Handler = wrappedHandler{}

// Invoke adds OTel span surrounding customer Handler invocation
func (h wrappedHandler) Invoke(ctx context.Context, payload []byte) ([]byte, error) {

	ctx, span := tracingBegin(ctx)
	defer tracingEnd(ctx, span)

	response, err := h.handler.Invoke(ctx, payload)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// WrapHandler Provides a Handler which wraps customer Handler with OTel Tracing
func WrapHandler(handler lambda.Handler, options ...InstrumentationOption) lambda.Handler {
	o := InstrumentationOptions{
		TracerProvider: otel.GetTracerProvider(),
		Flusher:        &noopFlusher{},
	}
	for _, opt := range options {
		opt(&o)
	}
	configuration = o

	return wrappedHandler{handler: handler}
}