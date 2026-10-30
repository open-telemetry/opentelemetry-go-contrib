// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zpages_test

import (
	"context"
	"log"
	"net/http"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"go.opentelemetry.io/contrib/zpages"
)

func ExampleNewTracezHandler() {
	// Create a new SpanProcessor to collect zpages diagnostic data.
	sp := zpages.NewSpanProcessor()

	// Register the SpanProcessor with the OpenTelemetry SDK TracerProvider.
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sp),
	)
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()
	otel.SetTracerProvider(tp)

	// Register the tracez HTTP handler on an HTTP server mux.
	mux := http.NewServeMux()
	mux.Handle("/debug/tracez", zpages.NewTracezHandler(sp))

	// The mux can now be passed to http.ListenAndServe, for example:
	// log.Fatal(http.ListenAndServe(":8080", mux))
}
