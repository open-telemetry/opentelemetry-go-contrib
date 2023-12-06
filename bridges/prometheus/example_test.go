// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
