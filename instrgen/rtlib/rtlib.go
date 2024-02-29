// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Basic runtime library

package rtlib // import "go.opentelemetry.io/contrib/instrgen/rtlib"

import (
	"context"
	"io"
	"log"
	"os"

	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	trace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
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
	tracingState.File, err = os.Create("traces.txt")
	if err != nil {
		tracingState.Logger.Fatal(err)
	}
	var exp trace.SpanExporter
	exp, err = NewExporter(tracingState.File)
	if err != nil {
		tracingState.Logger.Fatal(err)
	}
	tracingState.Tp = trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithResource(NewResource()),
	)
	return tracingState
}

// NewExporter returns a console exporter.
func NewExporter(w io.Writer) (trace.SpanExporter, error) {
	return stdouttrace.New(
		stdouttrace.WithWriter(w),
		// Use human readable output.
		stdouttrace.WithPrettyPrint(),
		// Do not print timestamps for the demo.
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
