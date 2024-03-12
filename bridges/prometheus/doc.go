// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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
