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

package example

import (
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/exporters/stdout"
	"go.opentelemetry.io/otel/propagators"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv"
)

var (
	Tracer        trace.Tracer
	TraceProvider trace.TracerProvider
	Propagators   otel.TextMapPropagator
)

// InitTracer with stdout exporter.
func InitTracer() {
	exporter, err := stdout.NewExporter(stdout.WithPrettyPrint())
	if err != nil {
		log.Fatal(err)
	}

	TraceProvider = sdktrace.NewTracerProvider(
		sdktrace.WithConfig(sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),
		sdktrace.WithResource(resource.New(
			semconv.TelemetrySDKNameKey.String("opentelemetry"),
			semconv.TelemetrySDKLanguageGo,
			semconv.TelemetrySDKVersionKey.String("0.13.0"),
			semconv.ServiceNameKey.String("myKafka"),
		)),
		sdktrace.WithSyncer(exporter),
	)

	// Set global propagator to baggage (the default is no-op).
	// TraceContext is a propagator that supports the W3C Trace Context format
	Propagators = otel.NewCompositeTextMapPropagator(propagators.TraceContext{}, propagators.Baggage{})
	global.SetTextMapPropagator(Propagators)

	// Tracer
	Tracer = global.Tracer("example/service")
}
