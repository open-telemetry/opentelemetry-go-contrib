// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// #docregion
package main

import "github.com/prometheus/client_golang/prometheus"

type deviceCountCollector struct{ desc *prometheus.Desc }

func newDeviceCountCollector() *deviceCountCollector {
	return &deviceCountCollector{desc: prometheus.NewDesc(
		"devices_connected",
		"Number of smart home devices currently connected",
		[]string{"device_type"}, nil,
	)}
}

func (c *deviceCountCollector) Describe(ch chan<- *prometheus.Desc) { ch <- c.desc }
func (c *deviceCountCollector) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, float64(connectedDeviceCount("thermostat")), "thermostat")
	ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, float64(connectedDeviceCount("lock")), "lock")
}

func prometheusUpDownCounterCallbackUsage(reg *prometheus.Registry) {
	// The device manager maintains the count of connected devices.
	// Implement prometheus.Collector to report those values at scrape time.
	reg.MustRegister(newDeviceCountCollector())
}
