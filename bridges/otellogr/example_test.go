// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otellogr_test

import (
	"github.com/go-logr/logr"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/noop"

	otellogr "go.opentelemetry.io/contrib/bridges/otellogr"
)

func Example() {
	// Use a working LoggerProvider implementation instead e.g. using go.opentelemetry.io/otel/sdk/log.
	provider := noop.NewLoggerProvider()

	// Create an *slog.Logger with *otelslog.Handler and use it in your application.
	logr.New(otellogr.NewLogSink(
		otellogr.WithLoggerProvider(provider),
		// Optionally, set the log level severity mapping.
		otellogr.WithLevelSeverity(func(i int) log.Severity {
			switch i {
			case 0:
				return log.SeverityInfo
			case 1:
				return log.SeverityWarn
			default:
				return log.SeverityFatal
			}
		})),
	)
}
