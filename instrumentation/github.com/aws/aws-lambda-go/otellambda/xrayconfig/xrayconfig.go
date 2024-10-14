// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xrayconfig // import "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda/xrayconfig"

import (
	"context"
	"os"

	lambdadetector "go.opentelemetry.io/contrib/detectors/aws/lambda"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
	"go.opentelemetry.io/contrib/propagators/aws/xray"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace" //nolint:depguard // NewTracerProvider requires the SDK
)

func xrayEventToCarrier([]byte) propagation.TextMapCarrier {
	xrayTraceID := os.Getenv("_X_AMZN_TRACE_ID")
	return propagation.HeaderCarrier{"X-Amzn-Trace-Id": []string{xrayTraceID}}
}

// NewTracerProvider returns a TracerProvider configured with an exporter,
// ID generator, and lambda resource detector to send trace data to AWS X-Ray
// via a Collector instance listening on localhost.
func NewTracerProvider(ctx context.Context) (*sdktrace.TracerProvider, error) {
	exp, err := otlptracegrpc.New(ctx, otlptracegrpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	detector := lambdadetector.NewResourceDetector()
	resource, err := detector.Detect(ctx)
	if err != nil {
		return nil, err
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithIDGenerator(xray.NewIDGenerator()),
		sdktrace.WithResource(resource),
	), nil
}

// WithEventToCarrier returns an otellambda.Option to enable
// an otellambda.EventToCarrier function which reads the XRay trace
// information from the environment and returns this information in
// a propagation.HeaderCarrier.
func WithEventToCarrier() otellambda.Option {
	return otellambda.WithEventToCarrier(xrayEventToCarrier)
}

// WithPropagator returns an otellambda.Option to enable the xray.Propagator.
func WithPropagator() otellambda.Option {
	return otellambda.WithPropagator(xray.Propagator{})
}

// WithRecommendedOptions returns a list of all otellambda.Option(s)
// recommended for the otellambda package when using AWS XRay.
func WithRecommendedOptions(tp *sdktrace.TracerProvider) []otellambda.Option {
	return []otellambda.Option{WithEventToCarrier(), WithPropagator(), otellambda.WithTracerProvider(tp), otellambda.WithFlusher(tp)}
}
