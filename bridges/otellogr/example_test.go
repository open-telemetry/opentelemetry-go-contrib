// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otellogr_test

import (
	"github.com/go-logr/logr"
	otellogr "go.opentelemetry.io/contrib/bridges/otellogr"
	"go.opentelemetry.io/otel/log/noop"
)

func Example() {
	// Use a working LoggerProvider implementation instead e.g. using go.opentelemetry.io/otel/sdk/log.
	provider := noop.NewLoggerProvider()

	// Create an *slog.Logger with *otelslog.Handler and use it in your application.
	logr.New(otellogr.NewLogSink(otellogr.WithLoggerProvider(provider)))
}
