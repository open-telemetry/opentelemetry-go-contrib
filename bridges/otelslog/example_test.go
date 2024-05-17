// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelslog_test

import (
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/log/noop"
)

func Example() {
	// Use a working LoggerProvider implementation instead e.g. using go.opentelemetry.io/otel/sdk/log.
	provider := noop.NewLoggerProvider()

	// Create an *slog.Logger and use it in your application.
	otelslog.NewLogger("my/pkg/name", otelslog.WithLoggerProvider(provider))
}
