// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// #docregion
package main

import "github.com/prometheus/client_golang/prometheus"

var summaryDeviceCommandDuration = prometheus.NewSummaryVec(prometheus.SummaryOpts{
	Name:       "device_command_duration_seconds",
	Help:       "Time to receive acknowledgment from a smart home device",
	Objectives: map[float64]float64{0.5: 0.05, 0.95: 0.01, 0.99: 0.001},
}, []string{"device_type"})

func summaryUsage(reg *prometheus.Registry) {
	reg.MustRegister(summaryDeviceCommandDuration)

	summaryDeviceCommandDuration.WithLabelValues("thermostat").Observe(0.35)
	summaryDeviceCommandDuration.WithLabelValues("lock").Observe(0.85)
}
