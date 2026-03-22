// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// #docregion createExponentialProvider
package main

import sdkmetric "go.opentelemetry.io/otel/sdk/metric"

func createExponentialProvider(reader sdkmetric.Reader) *sdkmetric.MeterProvider {
	// Configure base2 exponential histograms for all histogram instruments via a view.
	view := sdkmetric.NewView(
		sdkmetric.Instrument{Kind: sdkmetric.InstrumentKindHistogram},
		sdkmetric.Stream{Aggregation: sdkmetric.AggregationBase2ExponentialHistogram{}},
	)
	return sdkmetric.NewMeterProvider(sdkmetric.WithView(view), sdkmetric.WithReader(reader))
}

// #enddocregion createExponentialProvider

// #docregion createExponentialView
func createExponentialView() sdkmetric.View {
	// Use a view for per-instrument control — select a specific instrument by name
	// to use exponential histograms while keeping explicit buckets for others.
	return sdkmetric.NewView(
		sdkmetric.Instrument{Name: "device.command.duration"},
		sdkmetric.Stream{Aggregation: sdkmetric.AggregationBase2ExponentialHistogram{}},
	)
}

// #enddocregion createExponentialView
