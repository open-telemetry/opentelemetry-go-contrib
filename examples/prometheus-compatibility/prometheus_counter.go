// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build ignore

// #docregion
package main

import "github.com/prometheus/client_golang/prometheus"

func counterUsage(reg *prometheus.Registry) {
	hvacOnTime := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "hvac_on_seconds_total",
		Help: "Total time the HVAC system has been running, in seconds",
	}, []string{"zone"})
	reg.MustRegister(hvacOnTime)

	// Pre-bind to label value sets: subsequent calls avoid the series lookup.
	upstairs := hvacOnTime.WithLabelValues("upstairs")
	downstairs := hvacOnTime.WithLabelValues("downstairs")

	upstairs.Add(127.5)
	downstairs.Add(3600.0)

	// Pre-initialize a series so it appears in /metrics with value 0.
	hvacOnTime.WithLabelValues("basement")
}
