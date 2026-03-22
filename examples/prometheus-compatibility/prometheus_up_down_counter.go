// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// #docregion
package main

import "github.com/prometheus/client_golang/prometheus"

func prometheusUpDownCounterUsage(reg *prometheus.Registry) {
	// Prometheus uses Gauge for values that can increase or decrease.
	devicesConnected := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "devices_connected",
		Help: "Number of smart home devices currently connected",
	}, []string{"device_type"})
	reg.MustRegister(devicesConnected)

	// Increment when a device connects, decrement when it disconnects.
	devicesConnected.WithLabelValues("thermostat").Inc()
	devicesConnected.WithLabelValues("thermostat").Inc()
	devicesConnected.WithLabelValues("lock").Inc()
	devicesConnected.WithLabelValues("lock").Dec()
}
