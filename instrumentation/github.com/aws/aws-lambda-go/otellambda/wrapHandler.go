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

package otellambda

import (
	"context"

	"github.com/aws/aws-lambda-go/lambda"

	"go.opentelemetry.io/contrib"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type wrappedHandler struct {
	handler lambda.Handler
}

// Compile time check our Handler implements lambda.Handler
var _ lambda.Handler = wrappedHandler{}

// Invoke adds OTel span surrounding customer Handler invocation
func (h wrappedHandler) Invoke(ctx context.Context, payload []byte) ([]byte, error) {
	ctx, span := tracingBegin(ctx, payload)
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
		TracerProvider:                 otel.GetTracerProvider(),
		Flusher:                        &noopFlusher{},
		EventToTextMapCarrierConverter: noopEventToTextMapCarrierConverter,
		Propagator:                     otel.GetTextMapPropagator(),
	}
	for _, opt := range options {
		opt(&o)
	}
	configuration = o
	// Get a named tracer with package path as its name.
	tracer = configuration.TracerProvider.Tracer(tracerName, trace.WithInstrumentationVersion(contrib.SemVersion()))

	return wrappedHandler{handler: handler}
}
