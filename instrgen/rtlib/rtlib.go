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

// Basic runtime library

package rtlib // import "go.opentelemetry.io/contrib/instrgen/rtlib"

import (
	"context"
	"io"
	"log"
	"os"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/sdk/resource"
	trace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

const (
	serviceName           = "OTEL_SERVICE_NAME"
	defaultServiceName    = "instrgen"
	tracesExporter        = "OTEL_TRACES_EXPORTER"
	zipkinExporter        = "zipkin"
	otlpExporter          = "otlp"
	zipkinEndpoint        = "OTEL_EXPORTER_ZIPKIN_ENDPOINT"
	otlpExporterEndpoint  = "OTEL_EXPORTER_OTLP_ENDPOINT"
	defaultZipkinEndpoint = "http://localhost:9411/api/v2/spans"
	defaultGrpcEndpoint   = "localhost:4317"
	defaultHTTPEndpoint   = "http://localhost:4318"
	exporterProtocol      = "OTEL_EXPORTER_OTLP_PROTOCOL"
	exporterHTTPProtocol  = "http/protobuf"
	traceFile             = "traces.txt"
)

// TracingState type.
type TracingState struct {
	Logger *log.Logger
	File   *os.File
	Tp     *trace.TracerProvider
}

// NewTracingState.
func NewTracingState() TracingState {
	var tracingState TracingState
	tracingState.Logger = log.New(os.Stdout, "", 0)

	// Write telemetry data to a file.
	var err error
	serviceName := os.Getenv(serviceName)
	// fallback to instrgen
	if serviceName == "" {
		serviceName = defaultServiceName
	}
	exporterVar := os.Getenv(tracesExporter)
	switch exporterVar {
	case zipkinExporter:
		exporterEndpoint := os.Getenv(zipkinEndpoint)
		// fallback to localhost
		if exporterEndpoint == "" {
			exporterEndpoint = defaultZipkinEndpoint
		}
		exporter, _ := zipkin.New(
			exporterEndpoint,
			zipkin.WithLogger(tracingState.Logger),
		)

		batcher := trace.NewBatchSpanProcessor(exporter)

		tracingState.Tp = trace.NewTracerProvider(
			trace.WithSpanProcessor(batcher),
			trace.WithResource(resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceName(serviceName),
			)),
		)
	case otlpExporter:
		ctx := context.Background()
		res, err := resource.New(ctx,
			resource.WithAttributes(
				semconv.ServiceNameKey.String(serviceName),
				semconv.TelemetrySDKLanguageGo,
			),
		)
		if err != nil {
			tracingState.Logger.Fatal(err)
		}

		var client otlptrace.Client
		protocol := os.Getenv(exporterProtocol)
		exporterEndpoint := os.Getenv(otlpExporterEndpoint)

		if protocol == exporterHTTPProtocol {
			if exporterEndpoint == "" {
				exporterEndpoint = defaultHTTPEndpoint
			}

			client = otlptracehttp.NewClient(
				otlptracehttp.WithInsecure(),
				otlptracehttp.WithEndpoint(exporterEndpoint),
			)
		} else {
			if exporterEndpoint == "" {
				exporterEndpoint = defaultGrpcEndpoint
			}

			client = otlptracegrpc.NewClient(
				otlptracegrpc.WithInsecure(),
				otlptracegrpc.WithEndpoint(exporterEndpoint),
			)
		}
		traceExporter, err := otlptrace.New(
			context.Background(),
			client,
		)

		if err != nil {
			tracingState.Logger.Fatal(err)
		}

		bsp := trace.NewBatchSpanProcessor(traceExporter)
		tracingState.Tp = trace.NewTracerProvider(
			trace.WithSampler(trace.AlwaysSample()),
			trace.WithResource(res),
			trace.WithSpanProcessor(bsp),
		)
	default:
		// fallback to file exporting
		tracingState.File, err = os.Create(traceFile)

		if err != nil {
			tracingState.Logger.Fatal(err)
		}
		var exp trace.SpanExporter
		exp, err = NewConsoleExporter(tracingState.File)
		if err != nil {
			tracingState.Logger.Fatal(err)
		}
		tracingState.Tp = trace.NewTracerProvider(
			trace.WithBatcher(exp),
			trace.WithResource(NewResource()),
		)
	}
	return tracingState
}

// NewConsoleExporter returns a console exporter.
func NewConsoleExporter(w io.Writer) (trace.SpanExporter, error) {
	return stdouttrace.New(
		stdouttrace.WithWriter(w),
		// Use human readable output.
		stdouttrace.WithPrettyPrint(),
		stdouttrace.WithoutTimestamps(),
	)
}

// NewResource returns a resource describing this application.
func NewResource() *resource.Resource {
	r, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
		),
	)
	return r
}

// Shutdown.
func Shutdown(ts TracingState) {
	if err := ts.Tp.Shutdown(context.Background()); err != nil {
		ts.Logger.Fatal(err)
	}
}

// AutoEntryPoint.
func AutotelEntryPoint() {

}
