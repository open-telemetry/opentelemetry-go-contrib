// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otellambda // import "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"

import (
	"context"

	"github.com/aws/aws-lambda-go/lambda"
)

// wrappedHandler is a struct which holds an instrumentor
// as well as the user's original lambda.Handler and is
// able to instrument invocations of the user's lambda.Handler.
type wrappedHandler struct {
	instrumentor instrumentor
	handler      lambda.Handler
}

// Compile time check our Handler implements lambda.Handler.
var _ lambda.Handler = wrappedHandler{}

// Invoke adds OTel span surrounding customer Handler invocation.
func (h wrappedHandler) Invoke(ctx context.Context, payload []byte) ([]byte, error) {
	ctx, span := h.instrumentor.tracingBegin(ctx, payload)
	defer h.instrumentor.tracingEnd(ctx, span)

	response, err := h.handler.Invoke(ctx, payload)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// WrapHandler Provides a Handler which wraps customer Handler with OTel Tracing.
func WrapHandler(handler lambda.Handler, options ...Option) lambda.Handler {
	return wrappedHandler{instrumentor: newInstrumentor(options...), handler: handler}
}
