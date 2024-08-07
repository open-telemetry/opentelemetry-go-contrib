// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelzap_test

import (
	"context"
	"os"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/otel/log/noop"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Example() {
	// Use a working LoggerProvider implementation instead e.g. use go.opentelemetry.io/otel/sdk/log.
	provider := noop.NewLoggerProvider()

	// Initialize a zap logger with the otelzap bridge core.
	// This method actually doesn't log anything on your STDOUT, as everything
	// is shipped to a configured otel endpoint.
	logger := zap.New(otelzap.NewCore("my/pkg/name", otelzap.WithLoggerProvider(provider)))

	// You can now use your logger in your code.
	logger.Info("something really cool")

	// You can set context for trace correlation using zap.Any or zap.Reflect
	ctx := context.Background()
	logger.Info("setting context", zap.Any("context", ctx))
}

func Example_multiple() {
	// Use a working LoggerProvider implementation instead e.g. use go.opentelemetry.io/otel/sdk/log.
	provider := noop.NewLoggerProvider()

	// If you want to log also on stdout, you can initialize a new zap.Core
	// that has multiple outputs using the method zap.NewTee(). With the following code,
	// logs will be written to stdout and also exported to the OTEL endpoint through the bridge.
	core := zapcore.NewTee(
		zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(os.Stdout), zapcore.InfoLevel),
		otelzap.NewCore("my/pkg/name", otelzap.WithLoggerProvider(provider)),
	)
	logger := zap.New(core)

	// You can now use your logger in your code.
	logger.Info("something really cool")
}
