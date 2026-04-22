// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// #docregion
package main

import "github.com/prometheus/client_golang/prometheus"

type energyCollector struct{ desc *prometheus.Desc }

func newEnergyCollector() *energyCollector {
	return &energyCollector{desc: prometheus.NewDesc(
		"energy_consumed_joules_total",
		"Total energy consumed in joules",
		[]string{"zone"}, nil,
	)}
}

func (c *energyCollector) Describe(ch chan<- *prometheus.Desc) { ch <- c.desc }
func (c *energyCollector) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(c.desc, prometheus.CounterValue, totalEnergyJoules("upstairs"), "upstairs")
	ch <- prometheus.MustNewConstMetric(c.desc, prometheus.CounterValue, totalEnergyJoules("downstairs"), "downstairs")
}

func prometheusCounterCallbackUsage(reg *prometheus.Registry) {
	// Each zone has its own smart energy meter tracking cumulative joule totals.
	// Implement prometheus.Collector to report those values at scrape time.
	reg.MustRegister(newEnergyCollector())
}
