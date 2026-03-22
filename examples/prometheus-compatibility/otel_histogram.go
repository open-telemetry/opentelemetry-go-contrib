// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// #docregion
package main

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Preallocate attribute options when values are static to avoid per-call allocation.
var (
	deviceThermostatOpts = []metric.RecordOption{metric.WithAttributes(attribute.String("device_type", "thermostat"))}
	deviceLockOpts       = []metric.RecordOption{metric.WithAttributes(attribute.String("device_type", "lock"))}
)

func otelHistogramUsage(ctx context.Context, meter metric.Meter) {
	// WithExplicitBucketBoundaries sets default boundaries as a hint to the SDK.
	// Views configured at the SDK level take precedence over this hint.
	deviceCommandDuration, err := meter.Float64Histogram("device.command.duration",
		metric.WithDescription("Time to receive acknowledgment from a smart home device"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.1, 0.25, 0.5, 1.0, 2.5, 5.0))
	if err != nil {
		panic(err)
	}

	deviceCommandDuration.Record(ctx, 0.35, deviceThermostatOpts...)
	deviceCommandDuration.Record(ctx, 0.85, deviceLockOpts...)
}
