// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build ignore

// #docregion
package main

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func main() {
	ctx := context.Background()
	// Configure the SDK: export metrics over OTLP/HTTP on a fixed interval.
	// The endpoint defaults to localhost:4318 and can be configured via
	// the OTEL_EXPORTER_OTLP_ENDPOINT environment variable.
	exporter, err := otlpmetrichttp.New(ctx)
	if err != nil {
		panic(err)
	}
	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter)),
	)
	defer provider.Shutdown(ctx) //nolint:errcheck

	meter := provider.Meter("smart.home")
	doorOpens, err := meter.Int64Counter("door.opens",
		metric.WithDescription("Total number of times a door has been opened"))
	if err != nil {
		panic(err)
	}

	doorOpens.Add(ctx, 1, metric.WithAttributes(attribute.String("door", "front")))

	select {} // sleep forever
}
