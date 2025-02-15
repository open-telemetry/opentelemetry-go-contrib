// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"context"
	"errors"
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	config "go.opentelemetry.io/contrib/config/v0.3.0"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

var (
	// Ensure that the logger is only created once.
	loggerOnce sync.Once

	// By default, use a no-op logger.
	logger = zap.NewNop()
)

// Setup configures the global providers for the application based on the provided config file.
func Setup(ctx context.Context, cfgFile string) (func(context.Context) error, error) {
	// attempts to read the config file
	b, err := os.ReadFile(cfgFile)
	if err != nil {
		// if the file does not exist, use the default logger
		if errors.Is(err, os.ErrNotExist) {
			logger = zap.Must(zap.NewProduction())
			logger.Info("No config file found, using default logger")
			return func(ctx context.Context) error { return nil }, nil
		}

		return nil, err
	}

	// optional: interopolate environment variables
	b = []byte(os.ExpandEnv(string(b)))

	// parse the config
	conf, err := config.ParseYAML(b)
	if err != nil {
		return nil, err
	}

	// create the SDK with the parsed config
	sdk, err := config.NewSDK(config.WithContext(ctx), config.WithOpenTelemetryConfiguration(*conf))
	if err != nil {
		return nil, err
	}

	// set the global providers based on the parsed SDK config
	otel.SetTracerProvider(sdk.TracerProvider())
	otel.SetMeterProvider(sdk.MeterProvider())
	global.SetLoggerProvider(sdk.LoggerProvider())

	// optional: create a zap logger that logs to stdout and the OTel logger
	loggerOnce.Do(func() {
		core := zapcore.NewTee(
			zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(os.Stdout), zapcore.InfoLevel),
			otelzap.NewCore(Scope, otelzap.WithVersion(ScopeVersion), otelzap.WithLoggerProvider(global.GetLoggerProvider())),
		)
		logger = zap.New(core)
	})

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
func Logger() *zap.Logger {
	return logger
}
