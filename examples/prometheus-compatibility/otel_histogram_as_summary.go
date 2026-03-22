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
	summaryThermostatOpts = []metric.RecordOption{metric.WithAttributes(attribute.String("device_type", "thermostat"))}
	summaryLockOpts       = []metric.RecordOption{metric.WithAttributes(attribute.String("device_type", "lock"))}
)

func summaryReplacement(ctx context.Context, meter metric.Meter) {
	// No explicit bucket boundaries: captures count and sum only.
	// For quantile estimation, prefer a base2 exponential histogram instead.
	deviceCommandDuration, err := meter.Float64Histogram("device.command.duration",
		metric.WithDescription("Time to receive acknowledgment from a smart home device"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries()) // no boundaries
	if err != nil {
		panic(err)
	}

	deviceCommandDuration.Record(ctx, 0.35, summaryThermostatOpts...)
	deviceCommandDuration.Record(ctx, 0.85, summaryLockOpts...)
}
