// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// #docregion
package main

import "github.com/prometheus/client_golang/prometheus"

func prometheusGaugeUsage(reg *prometheus.Registry) {
	thermostatSetpoint := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "thermostat_setpoint_celsius",
		Help: "Target temperature set on the thermostat",
	}, []string{"zone"})
	reg.MustRegister(thermostatSetpoint)

	thermostatSetpoint.WithLabelValues("upstairs").Set(22.5)
	thermostatSetpoint.WithLabelValues("downstairs").Set(20.0)
}
