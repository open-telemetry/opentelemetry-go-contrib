// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build ignore

// #docregion
package main

import "github.com/prometheus/client_golang/prometheus"

func nativeHistogramUsage(reg *prometheus.Registry) {
	deviceCommandDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:                        "device_command_duration_seconds",
		Help:                        "Time to receive acknowledgment from a smart home device",
		NativeHistogramBucketFactor: 1.1,
	}, []string{"device_type"})
	reg.MustRegister(deviceCommandDuration)

	deviceCommandDuration.WithLabelValues("thermostat").Observe(0.35)
	deviceCommandDuration.WithLabelValues("lock").Observe(0.85)
}
