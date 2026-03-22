// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// #docregion
package main

import "github.com/prometheus/client_golang/prometheus"

func summaryUsage(reg *prometheus.Registry) {
	deviceCommandDuration := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name:       "device_command_duration_seconds",
		Help:       "Time to receive acknowledgment from a smart home device",
		Objectives: map[float64]float64{0.5: 0.05, 0.95: 0.01, 0.99: 0.001},
	}, []string{"device_type"})
	reg.MustRegister(deviceCommandDuration)

	deviceCommandDuration.WithLabelValues("thermostat").Observe(0.35)
	deviceCommandDuration.WithLabelValues("lock").Observe(0.85)
}
