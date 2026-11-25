// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zpages_test

import (
	"context"
	"log"
	"net/http"
	"time"

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

	// Register the tracez HTTP handler on a dedicated administrative mux.
	adminMux := http.NewServeMux()
	adminMux.Handle("/debug/tracez", zpages.NewTracezHandler(sp))

	// Bind only to a loopback or private administrative listener, never to a
	// public interface without authentication and authorization.
	adminServer := &http.Server{
		Addr:              "127.0.0.1:8080",
		Handler:           adminMux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	_ = adminServer

	// Start adminServer (for example, in a goroutine) and call
	// adminServer.Shutdown before tp.Shutdown when the application exits.
}

func ExampleNewTracezHandler_auth() {
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

	// Replace authorize with your application's authentication and
	// authorization check (for example, verifying an mTLS client identity or
	// validating a signed token). This placeholder fails closed by default.
	authorize := func(*http.Request) bool {
		return false
	}

	requireAuth := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !authorize(r) {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	// Wrap the tracez handler with authentication and authorization middleware.
	mux := http.NewServeMux()
	mux.Handle("/debug/tracez", requireAuth(zpages.NewTracezHandler(sp)))

	srv := &http.Server{
		Addr:              "127.0.0.1:8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	_ = srv

	// Start srv (for example, in a goroutine) and call srv.Shutdown before
	// tp.Shutdown when the application exits.
}
