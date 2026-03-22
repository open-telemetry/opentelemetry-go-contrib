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
	deviceThermostat = attribute.String("device_type", "thermostat")
	deviceLock       = attribute.String("device_type", "lock")
)

func otelUpDownCounterCallbackUsage(meter metric.Meter) {
	// The device manager maintains the count of connected devices.
	// Use an observable up-down counter to report that value when metrics are collected.
	_, err := meter.Int64ObservableUpDownCounter("devices.connected",
		metric.WithDescription("Number of smart home devices currently connected"),
		metric.WithInt64Callback(func(_ context.Context, o metric.Int64Observer) error {
			o.Observe(int64(connectedDeviceCount("thermostat")), metric.WithAttributes(deviceThermostat))
			o.Observe(int64(connectedDeviceCount("lock")), metric.WithAttributes(deviceLock))
			return nil
		}))
	if err != nil {
		panic(err)
	}
}
