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
	"log"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/lambdacontext"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	tracerName = "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
)

var errorLogger = log.New(log.Writer(), "OTel Lambda Error: ", 0)

type Flusher interface {
	ForceFlush(context.Context) error
}

type noopFlusher struct{}

func (*noopFlusher) ForceFlush(context.Context) error { return nil }

// Compile time check our noopFlusher implements FLusher
var _ Flusher = &noopFlusher{}

type EventToTextMapCarrierConverter func([]byte) propagation.TextMapCarrier

func noopEventToTextMapCarrierConverter([]byte) propagation.TextMapCarrier {
	return propagation.HeaderCarrier{}
}

type InstrumentationOption func(o *InstrumentationOptions)

type InstrumentationOptions struct {
	// TracerProvider is the TracerProvider which will be used
	// to create instrumentation spans
	// The default value of TracerProvider the global otel TracerProvider
	// returned by otel.GetTracerProvider()
	TracerProvider trace.TracerProvider

	// Flusher is the mechanism used to flush any unexported spans
	// each Lambda Invocation to avoid spans being unexported for long
	// when periods of time if Lambda freezes the execution environment
	// The default value of Flusher is a noop Flusher, using this
	// default can result in long data delays in asynchronous settings
	Flusher Flusher

	// eventToTextMapCarrierConverter is the mechanism used to retrieve the TraceID
	// from the event or environment and generate a TextMapCarrier which
	// can then be used by a Propagator to extract the TraceID into our context
	// The default value of eventToTextMapCarrierConverter returns an empty
	// HeaderCarrier, using this default will cause all spans to be not be traced
	EventToTextMapCarrierConverter EventToTextMapCarrierConverter

	// Propagator is the Propagator which will be used
	// to extrract Trace info into the context
	// The default value of Propagator the global otel Propagator
	// returned by otel.GetTextMapPropagator()
	Propagator propagation.TextMapPropagator
}

var configuration InstrumentationOptions
var resourceAttributesToAddAsSpanAttributes []attribute.KeyValue
var tracer trace.Tracer

// Logic to start OTel Tracing
func tracingBegin(ctx context.Context, eventJSON []byte) (context.Context, trace.Span) {
	// Add trace id to context
	mc := configuration.EventToTextMapCarrierConverter(eventJSON)
	ctx = configuration.Propagator.Extract(ctx, mc)

	var span trace.Span
	spanName := os.Getenv("AWS_LAMBDA_FUNCTION_NAME")

	var attributes []attribute.KeyValue
	lc, ok := lambdacontext.FromContext(ctx)
	if !ok {
		errorLogger.Println("failed to load lambda context from context, ensure tracing enabled in Lambda")
	}
	if lc != nil {
		ctxRequestID := lc.AwsRequestID
		attributes = append(attributes, semconv.FaaSExecutionKey.String(ctxRequestID))

		// Some resource attrs added as span attrs because lambda
		// resource detectors are created before a lambda
		// invocation and therefore lack lambdacontext.
		// Create these attrs upon first invocation
		if resourceAttributesToAddAsSpanAttributes == nil {
			ctxFunctionArn := lc.InvokedFunctionArn
			attributes = append(attributes, semconv.FaaSIDKey.String(ctxFunctionArn))
			arnParts := strings.Split(ctxFunctionArn, ":")
			if len(arnParts) >= 5 {
				attributes = append(attributes, semconv.CloudAccountIDKey.String(arnParts[4]))
			}
		}
		attributes = append(attributes, resourceAttributesToAddAsSpanAttributes...)
	}

	ctx, span = tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindServer), trace.WithAttributes(attributes...))

	return ctx, span
}

// Logic to wrap up OTel Tracing
func tracingEnd(ctx context.Context, span trace.Span) {
	span.End()

	// force flush any tracing data since lambda may freeze
	err := configuration.Flusher.ForceFlush(ctx)
	if err != nil {
		errorLogger.Println("failed to force a flush, lambda may freeze before instrumentation exported: ", err)
	}
}

func WithTracerProvider(tracerProvider trace.TracerProvider) InstrumentationOption {
	return func(o *InstrumentationOptions) {
		o.TracerProvider = tracerProvider
	}
}

func WithFlusher(flusher Flusher) InstrumentationOption {
	return func(o *InstrumentationOptions) {
		o.Flusher = flusher
	}
}

func WithEventToTextMapCarrierConverter(eventToTextMapCarrierConverter EventToTextMapCarrierConverter) InstrumentationOption {
	return func(o *InstrumentationOptions) {
		o.EventToTextMapCarrierConverter = eventToTextMapCarrierConverter
	}
}

func WithPropagator(propagator propagation.TextMapPropagator) InstrumentationOption {
	return func(o *InstrumentationOptions) {
		o.Propagator = propagator
	}
}
