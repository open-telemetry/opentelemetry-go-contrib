// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelzerolog_test

import (
	"os"

	"github.com/rs/zerolog"

	"go.opentelemetry.io/contrib/bridges/otelzerolog"
	"go.opentelemetry.io/otel/log/noop"
)

func Example() {
	// Use a working LoggerProvider implementation instead e.g. using go.opentelemetry.io/otel/sdk/log.
	provider := noop.NewLoggerProvider()

	// This will emit logs to both STDOUT and the OTel Go SDK.
	hook := otelzerolog.NewSeverityHook("my/pkg/name", otelzerolog.WithLoggerProvider(provider))

	logger := zerolog.New(os.Stdout).With().Logger()
	logger = logger.Hook(hook)
}
