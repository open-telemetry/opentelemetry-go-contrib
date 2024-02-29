// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otellambda // import "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/lambdacontext"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	// ScopeName is the instrumentation scope name.
	ScopeName = "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
)

var errorLogger = log.New(log.Writer(), "OTel Lambda Error: ", 0)

type instrumentor struct {
	configuration config
	resAttrs      []attribute.KeyValue
	tracer        trace.Tracer
}

func newInstrumentor(opts ...Option) instrumentor {
	cfg := config{
		TracerProvider: otel.GetTracerProvider(),
		Flusher:        &noopFlusher{},
		EventToCarrier: emptyEventToCarrier,
		Propagator:     otel.GetTextMapPropagator(),
	}
	for _, opt := range opts {
		opt.apply(&cfg)
	}

	return instrumentor{
		configuration: cfg,
		tracer:        cfg.TracerProvider.Tracer(ScopeName, trace.WithInstrumentationVersion(Version())),
		resAttrs:      []attribute.KeyValue{},
	}
}

// Logic to start OTel Tracing.
func (i *instrumentor) tracingBegin(ctx context.Context, eventJSON []byte) (context.Context, trace.Span) {
	// Add trace id to context
	mc := i.configuration.EventToCarrier(eventJSON)
	ctx = i.configuration.Propagator.Extract(ctx, mc)

	var span trace.Span
	spanName := os.Getenv("AWS_LAMBDA_FUNCTION_NAME")

	var attributes []attribute.KeyValue
	lc, ok := lambdacontext.FromContext(ctx)
	if !ok {
		errorLogger.Println("failed to load lambda context from context, ensure tracing enabled in Lambda")
	}
	if lc != nil {
		ctxRequestID := lc.AwsRequestID
		attributes = append(attributes, semconv.FaaSInvocationID(ctxRequestID))

		// Some resource attrs added as span attrs because lambda
		// resource detectors are created before a lambda
		// invocation and therefore lack lambdacontext.
		// Create these attrs upon first invocation
		if len(i.resAttrs) == 0 {
			ctxFunctionArn := lc.InvokedFunctionArn
			attributes = append(attributes, semconv.AWSLambdaInvokedARN(ctxFunctionArn))
			arnParts := strings.Split(ctxFunctionArn, ":")
			if len(arnParts) >= 5 {
				attributes = append(attributes, semconv.CloudAccountID(arnParts[4]))
			}
		}
		attributes = append(attributes, i.resAttrs...)
	}

	ctx, span = i.tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindServer), trace.WithAttributes(attributes...))

	return ctx, span
}

// Logic to wrap up OTel Tracing.
func (i *instrumentor) tracingEnd(ctx context.Context, span trace.Span) {
	span.End()

	// force flush any tracing data since lambda may freeze
	err := i.configuration.Flusher.ForceFlush(ctx)
	if err != nil {
		errorLogger.Println("failed to force a flush, lambda may freeze before instrumentation exported: ", err)
	}
}
