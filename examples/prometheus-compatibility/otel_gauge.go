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
	zoneUpstairsOpts   = []metric.RecordOption{metric.WithAttributes(attribute.String("zone", "upstairs"))}
	zoneDownstairsOpts = []metric.RecordOption{metric.WithAttributes(attribute.String("zone", "downstairs"))}
)

func gaugeUsage(ctx context.Context, meter metric.Meter) {
	thermostatSetpoint, err := meter.Float64Gauge("thermostat.setpoint",
		metric.WithDescription("Target temperature set on the thermostat"),
		metric.WithUnit("Cel"))
	if err != nil {
		panic(err)
	}

	thermostatSetpoint.Record(ctx, 22.5, zoneUpstairsOpts...)
	thermostatSetpoint.Record(ctx, 20.0, zoneDownstairsOpts...)
}
