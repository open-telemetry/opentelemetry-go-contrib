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
	roomLivingRoom = attribute.String("room", "living_room")
	roomBedroom    = attribute.String("room", "bedroom")
)

func otelGaugeCallbackUsage(meter metric.Meter) {
	// Temperature sensors maintain their own readings in firmware.
	// Use an observable gauge to report those values when metrics are collected.
	_, err := meter.Float64ObservableGauge("room.temperature",
		metric.WithDescription("Current temperature in the room"),
		metric.WithUnit("Cel"),
		metric.WithFloat64Callback(func(_ context.Context, o metric.Float64Observer) error {
			o.Observe(livingRoomTemperatureCelsius(), metric.WithAttributes(roomLivingRoom))
			o.Observe(bedroomTemperatureCelsius(), metric.WithAttributes(roomBedroom))
			return nil
		}))
	if err != nil {
		panic(err)
	}
}
