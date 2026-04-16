// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// #docregion
package main

import "github.com/prometheus/client_golang/prometheus"

type temperatureCollector struct{ desc *prometheus.Desc }

func newTemperatureCollector() *temperatureCollector {
	return &temperatureCollector{desc: prometheus.NewDesc(
		"room_temperature_celsius",
		"Current temperature in the room",
		[]string{"room"}, nil,
	)}
}

func (c *temperatureCollector) Describe(ch chan<- *prometheus.Desc) { ch <- c.desc }
func (c *temperatureCollector) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, livingRoomTemperatureCelsius(), "living_room")
	ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, bedroomTemperatureCelsius(), "bedroom")
}

func prometheusGaugeCallbackUsage(reg *prometheus.Registry) {
	// Temperature sensors maintain their own readings in firmware.
	// Implement prometheus.Collector to report those values at scrape time.
	reg.MustRegister(newTemperatureCollector())
}
