// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build ignore

// #docregion
package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Create a counter and register it with a custom registry.
	reg := prometheus.NewRegistry()
	doorOpens := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "door_opens_total",
		Help: "Total number of times a door has been opened",
	}, []string{"door"})
	reg.MustRegister(doorOpens)

	// Prometheus scrapes http://localhost:9464/metrics.
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	go http.ListenAndServe(":9464", nil) //nolint:errcheck

	doorOpens.WithLabelValues("front").Inc()

	select {} // sleep forever
}
