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
	zoneUpstairsGaugeOpts   = []metric.RecordOption{metric.WithAttributes(attribute.String("zone", "upstairs"))}
	zoneDownstairsGaugeOpts = []metric.RecordOption{metric.WithAttributes(attribute.String("zone", "downstairs"))}
)

func otelGaugeUsage(ctx context.Context, meter metric.Meter) {
	thermostatSetpoint, err := meter.Float64Gauge("thermostat.setpoint",
		metric.WithDescription("Target temperature set on the thermostat"),
		metric.WithUnit("Cel"))
	if err != nil {
		panic(err)
	}

	thermostatSetpoint.Record(ctx, 22.5, zoneUpstairsGaugeOpts...)
	thermostatSetpoint.Record(ctx, 20.0, zoneDownstairsGaugeOpts...)
}
