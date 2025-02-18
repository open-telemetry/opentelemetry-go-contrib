// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	config "go.opentelemetry.io/contrib/config/v0.3.0"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

var (
	// By default, use a no-op logger.
	logger = slog.Default()
)

// Setup configures the global providers for the application based on the provided config file.
func Setup(ctx context.Context, cfgFile string) (func(context.Context) error, error) {
	// Attempts to read the config file.
	b, err := os.ReadFile(cfgFile)
	if err != nil {
		// If the file does not exist, use the default logger.
		if errors.Is(err, os.ErrNotExist) {
			logger.Info("No config file found, using default logger")
			return func(ctx context.Context) error { return nil }, nil
		}

		return nil, err
	}

	// Optional: interopolate environment variables.
	b = []byte(os.ExpandEnv(string(b)))

	// Parse the contents of the configuration file.
	conf, err := config.ParseYAML(b)
	if err != nil {
		return nil, err
	}

	// Create the SDK with the parsed config.
	sdk, err := config.NewSDK(config.WithContext(ctx), config.WithOpenTelemetryConfiguration(*conf))
	if err != nil {
		return nil, err
	}

	// Set the global providers based on the parsed SDK config.
	otel.SetTracerProvider(sdk.TracerProvider())
	otel.SetMeterProvider(sdk.MeterProvider())
	global.SetLoggerProvider(sdk.LoggerProvider())

	// Optional: create an OTel bridge to Go's slog.
	logger = otelslog.NewLogger(Scope, otelslog.WithVersion(ScopeVersion), otelslog.WithLoggerProvider(sdk.LoggerProvider()))

	return sdk.Shutdown, nil
}

// Tracer returns the global tracer for this application, using the scope defined in this package.
func Tracer() trace.Tracer {
	return otel.Tracer(Scope, trace.WithInstrumentationVersion(ScopeVersion))
}

// Meter returns the global meter for this application, using the scope defined in this package.
func Meter() metric.Meter {
	return otel.Meter(Scope, metric.WithInstrumentationVersion(ScopeVersion))
}

// Logger returns the global logger for this application, which is either a noop logger (default), or a configured dual-writer.
func Logger() *slog.Logger {
	return logger
}
