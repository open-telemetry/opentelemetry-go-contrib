// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package prometheus_test

import (
	"go.opentelemetry.io/contrib/bridges/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
)

func ExampleNewMetricProducer() {
	// Create a Promethes bridge "Metric Producer" which adds metrics from the
	// prometheus.DefaultGatherer. Add the WithGatherer option to add metrics
	// from other registries.
	bridge := prometheus.NewMetricProducer()
	// This reader is used as a stand-in for a reader that will actually export
	// data. See https://pkg.go.dev/go.opentelemetry.io/otel/exporters for
	// exporters that can be used as or with readers. The metric.WithProducer
	// option adds metrics from the Prometheus bridge to the reader.
	reader := metric.NewManualReader(metric.WithProducer(bridge))
	// Create an OTel MeterProvider with our reader. Metrics from OpenTelemetry
	// instruments are combined with metrics from Prometheus instruments in
	// exported batches of metrics.
	_ = metric.NewMeterProvider(metric.WithReader(reader))
}
