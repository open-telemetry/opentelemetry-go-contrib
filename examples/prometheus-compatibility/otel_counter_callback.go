// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// #docregion
package main

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	zoneUpstairs   = attribute.String("zone", "upstairs")
	zoneDownstairs = attribute.String("zone", "downstairs")
)

func otelCounterCallbackUsage(meter metric.Meter) {
	// Each zone has its own smart energy meter tracking cumulative joule totals.
	// Use an observable counter to report those values when metrics are collected.
	_, err := meter.Float64ObservableCounter("energy.consumed",
		metric.WithDescription("Total energy consumed"),
		metric.WithUnit("J"),
		metric.WithFloat64Callback(func(_ context.Context, o metric.Float64Observer) error {
			o.Observe(totalEnergyJoules("upstairs"), metric.WithAttributes(zoneUpstairs))
			o.Observe(totalEnergyJoules("downstairs"), metric.WithAttributes(zoneDownstairs))
			return nil
		}))
	if err != nil {
		panic(err)
	}
}
