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

package xrayconfig

import (
	"context"
	"log"
	"os"

	lambdadetector "go.opentelemetry.io/contrib/detectors/aws/lambda"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
	"go.opentelemetry.io/contrib/propagators/aws/xray"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var errorLogger = log.New(log.Writer(), "OTel Lambda XRay Configuration Error: ", 0)

func xrayEventToCarrier([]byte) propagation.TextMapCarrier {
	xrayTraceID := os.Getenv("_X_AMZN_TRACE_ID")
	return propagation.HeaderCarrier{"X-Amzn-Trace-Id": []string{xrayTraceID}}
}

// PrepareTracerProvider returns a TracerProvider configured with exporter,
// id generator and lambda resource detector to send trace data to AWS X-Ray via Collector
func PrepareTracerProvider(ctx context.Context) (*sdktrace.TracerProvider, error) {
	log.Println("creating trace exporter")
	exp, err := otlptracegrpc.New(ctx, otlptracegrpc.WithInsecure())
	if err != nil {
		errorLogger.Println("failed to create exporter: ", err)
		return nil, err
	}

	detector := lambdadetector.NewResourceDetector()
	resource, err := detector.Detect(ctx)
	if err != nil {
		errorLogger.Println("failed to detect lambda resources: ", err)
		return nil, err
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithIDGenerator(xray.NewIDGenerator()),
		sdktrace.WithResource(resource),
	), nil
}

// tracerProviderAndFlusher returns a list of otellambda.Option(s) to
// enable using a TracerProvider configured for AWS XRay via a collector
// and an otellambda.Flusher to flush this TracerProvider.
// tracerProviderAndFlusher is not exported because it should not be used
// without the provided EventToCarrier function and XRay Propagator
func tracerProviderAndFlusher(ctx context.Context) ([]otellambda.Option, *sdktrace.TracerProvider, error) {
	tp, err := PrepareTracerProvider(ctx)
	if err != nil {
		errorLogger.Println("failed to prepare tracer provider: ", err)
		return nil, nil, err
	}

	return []otellambda.Option{otellambda.WithTracerProvider(tp), otellambda.WithAsyncSafeFlusher(tp)}, tp, nil
}

// EventToCarrier returns an otellambda.Option to enable
// an otellambda.EventToCarrier function which reads the XRay trace
// information from the environment and returns this information in
// a propagation.HeaderCarrier
func EventToCarrier() otellambda.Option {
	return otellambda.WithEventToCarrier(xrayEventToCarrier)
}

// Propagator returns an otellambda.Option to enable the xray.Propagator
func Propagator() otellambda.Option {

	return otellambda.WithPropagator(xray.Propagator{})
}

// AllRecommendedOptions returns a list of all otellambda.Option(s)
// recommended for the otellambda package when using AWS XRay
func AllRecommendedOptions(ctx context.Context) ([]otellambda.Option, *sdktrace.TracerProvider) {
	options, tp, err := tracerProviderAndFlusher(ctx)
	if err != nil {
		// should we fail to create the TracerProvider, do not alter otellambda's default configuration
		errorLogger.Println("failed to create recommended configuration: ", err)
		return []otellambda.Option{}, nil
	}
	return append(options, []otellambda.Option{EventToCarrier(), Propagator()}...), tp
}
