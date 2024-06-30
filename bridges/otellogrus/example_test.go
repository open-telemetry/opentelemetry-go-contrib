// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otellogrus_test

import (
	"github.com/sirupsen/logrus"

	"go.opentelemetry.io/contrib/bridges/otellogrus"
	"go.opentelemetry.io/otel/log/noop"
)

func Example() {
	// Use a working LoggerProvider implementation instead e.g. using go.opentelemetry.io/otel/sdk/log.
	provider := noop.NewLoggerProvider()

	// Create an *otellogrus.Hook and use it in your application.
	hook := otellogrus.NewHook("my/pkg/name", otellogrus.WithLoggerProvider(provider))

	// Set the newly created hook as a global logrus hook
	logrus.AddHook(hook)
}
