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
	"runtime"

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

type asyncSafeFlusher struct {
	tp *sdktrace.TracerProvider
}

func (f asyncSafeFlusher) ForceFlush(ctx context.Context) error {
	// yield processor to attempt to ensure all spans have
	// been consumed and are ready to be flushed
	// - see https://github.com/open-telemetry/opentelemetry-go/issues/2080
	// to be removed upon resolution of above issue
	runtime.Gosched()

	return f.tp.ForceFlush(ctx)
}

// tracerProviderAndFlusher returns a list of otellambda.Option(s) to
// enable using a TracerProvider configured for AWS XRay via a collector
// and an otellambda.Flusher to flush this TracerProvider.
// tracerProviderAndFlusher is not exported because it should not be used
// without the provided EventToCarrier function and XRay Propagator
func tracerProviderAndFlusher() ([]otellambda.Option, error) {
	ctx := context.Background()

	// Do not need transport security in Lambda because collector
	// runs locally in Lambda execution environment
	exp, err := otlptracegrpc.New(ctx, otlptracegrpc.WithInsecure())
	if err != nil {
		return []otellambda.Option{}, err
	}

	detector := lambdadetector.NewResourceDetector()
	res, err := detector.Detect(ctx)
	if err != nil {
		return []otellambda.Option{}, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithIDGenerator(xray.NewIDGenerator()),
		sdktrace.WithResource(res),
	)

	return []otellambda.Option{otellambda.WithTracerProvider(tp), otellambda.WithFlusher(asyncSafeFlusher{tp: tp})}, nil
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
func AllRecommendedOptions() []otellambda.Option {
	options, err := tracerProviderAndFlusher()
	if err != nil {
		// should we fail to create the TracerProvider, do not alter otellambda's default configuration
		errorLogger.Println("failed to create recommended configuration: ", err)
		return []otellambda.Option{}
	}
	return append(options, []otellambda.Option{EventToCarrier(), Propagator()}...)
}
