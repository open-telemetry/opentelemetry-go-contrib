// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// #docregion
package main

import "github.com/prometheus/client_golang/prometheus"

var deviceCommandDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "device_command_duration_seconds",
	Help:    "Time to receive acknowledgment from a smart home device",
	Buckets: []float64{0.1, 0.25, 0.5, 1.0, 2.5, 5.0},
}, []string{"device_type"})

func prometheusHistogramUsage(reg *prometheus.Registry) {
	reg.MustRegister(deviceCommandDuration)

	deviceCommandDuration.WithLabelValues("thermostat").Observe(0.35)
	deviceCommandDuration.WithLabelValues("lock").Observe(0.85)
}
