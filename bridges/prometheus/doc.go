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

// Package prometheus provides a bridge from Prometheus to OpenTelemetry.
//
// The Prometheus Bridge allows using the [Prometheus Golang client library]
// with the OpenTelemetry SDK. This enables prometheus instrumentation libraries
// to be used with OpenTelemetry exporters, including OTLP.
//
// Prometheus histograms are translated to OpenTelemetry exponential histograms
// when native histograms are enabled in the Prometheus client. To enable
// Prometheus native histograms, set the (currently experimental) NativeHistogram...
// options of the prometheus [HistogramOpts] when creating prometheus histograms.
//
// [Prometheus Golang client library]: https://github.com/prometheus/client_golang
// [HistogramOpts]: https://pkg.go.dev/github.com/prometheus/client_golang/prometheus#HistogramOpts
package prometheus // import "go.opentelemetry.io/contrib/bridges/prometheus"
