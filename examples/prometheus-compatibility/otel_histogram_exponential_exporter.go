// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// #docregion createExponentialExporter
package main

import (
	"context"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func createExponentialExporter(ctx context.Context) (*otlpmetrichttp.Exporter, error) {
	// Configure the exporter to use exponential histograms for all histogram instruments.
	// This is the preferred approach — it applies globally without modifying instrumentation code.
	return otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithAggregationSelector(func(ik sdkmetric.InstrumentKind) sdkmetric.Aggregation {
			if ik == sdkmetric.InstrumentKindHistogram {
				return sdkmetric.AggregationBase2ExponentialHistogram{}
			}
			return sdkmetric.DefaultAggregationSelector(ik)
		}),
	)
}

// #enddocregion createExponentialExporter
