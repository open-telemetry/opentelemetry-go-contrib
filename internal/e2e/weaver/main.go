// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Weaver E2E exercises instrumentation libraries and exports the
// resulting telemetry via OTLP gRPC to a running weaver live-check
// instance for semantic-convention compliance assessment.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func main() {
	ctx := context.Background()

	shutdown, err := initOTLP(ctx, "localhost:4317")
	if err != nil {
		log.Fatalf("init OTLP: %v", err)
	}
	defer func() {
		if err := shutdown(ctx); err != nil {
			log.Printf("shutdown: %v", err)
		}
	}()

	if err := exerciseOtelHTTP(ctx); err != nil {
		log.Fatalf("otelhttp: %v", err)
	}

	// Allow the batched exporter to flush before the process exits.
	time.Sleep(2 * time.Second)
	log.Println("done")
}

// initOTLP configures the global TracerProvider and MeterProvider with
// OTLP gRPC exporters pointing at the given endpoint.
func initOTLP(ctx context.Context, endpoint string) (func(context.Context) error, error) {
	traceExp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("trace exporter: %w", err)
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(traceExp))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	metricExp, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("metric exporter: %w", err)
	}
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExp)),
	)
	otel.SetMeterProvider(mp)

	return func(c context.Context) error {
		tpErr := tp.Shutdown(c)
		mpErr := mp.Shutdown(c)
		if tpErr != nil {
			return fmt.Errorf("trace provider: %w", tpErr)
		}
		if mpErr != nil {
			return fmt.Errorf("metric provider: %w", mpErr)
		}
		return nil
	}, nil
}

// exerciseOtelHTTP spins up a local test server wrapped with otelhttp
// and issues requests through an instrumented client transport.
func exerciseOtelHTTP(ctx context.Context) error {
	handler := otelhttp.NewHandler(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		}),
		"test-server",
	)

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client := &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	for _, method := range []string{http.MethodGet, http.MethodPost} {
		req, err := http.NewRequestWithContext(ctx, method, srv.URL+"/test", http.NoBody)
		if err != nil {
			return fmt.Errorf("new request: %w", err)
		}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("do request: %w", err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}

	return nil
}
