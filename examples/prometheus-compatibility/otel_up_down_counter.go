// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build ignore

// #docregion
package main

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Preallocate attribute options when values are static to avoid per-call allocation.
var (
	deviceThermostatOpts = []metric.AddOption{metric.WithAttributes(attribute.String("device_type", "thermostat"))}
	deviceLockOpts       = []metric.AddOption{metric.WithAttributes(attribute.String("device_type", "lock"))}
)

func upDownCounterUsage(ctx context.Context, meter metric.Meter) {
	devicesConnected, err := meter.Int64UpDownCounter("devices.connected",
		metric.WithDescription("Number of smart home devices currently connected"))
	if err != nil {
		panic(err)
	}

	// Add() accepts positive and negative values.
	devicesConnected.Add(ctx, 1, deviceThermostatOpts...)
	devicesConnected.Add(ctx, 1, deviceThermostatOpts...)
	devicesConnected.Add(ctx, 1, deviceLockOpts...)
	devicesConnected.Add(ctx, -1, deviceLockOpts...)
}
