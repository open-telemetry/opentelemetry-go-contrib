// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelzap_test

import (
	"context"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/sdk/log"
	"os"
	"testing"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/otel/log/noop"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestAltAppender(t *testing.T) {
	// Configure otel log provider, which uses simple processor and stdout exporter for simplicity
	logExporter, err := stdoutlog.New()
	if err != nil {
		t.Error(err)
		return
	}
	provider := log.NewLoggerProvider(
		log.WithProcessor(log.NewSimpleProcessor(logExporter)),
	)
	if err != nil {
		t.Error(err)
		return
	}

	// Configure a zap core using whatever configuration you typically use
	core := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(os.Stdout), zapcore.WarnLevel)
	// Wrap the core in an otel wrapper, which bridges all logs the core decides to write (i.e. all logs for which core.Write is invoked) to the otel log provider
	wrappedCore := otelzap.NewWrappedCore(core, provider)
	// Create a logger from the wrapped core, and proceed as usual
	logger := zap.New(wrappedCore).Named("my.logger")

	logger.Warn("message level warn")
	logger.Info("message level info")

	// Output:
	// {"level":"warn","ts":1742414003.3163369,"msg":"message level warn"}
	// {"Timestamp":"2025-03-19T14:53:23.316337-05:00","ObservedTimestamp":"2025-03-19T14:53:23.316363-05:00","Severity":13,"SeverityText":"warn","Body":{"Type":"String","Value":"message level warn"},"Attributes":[],"TraceID":"00000000000000000000000000000000","SpanID":"0000000000000000","TraceFlags":"00","Resource":[{"Key":"service.name","Value":{"Type":"STRING","Value":"unknown_service:___TestAltAppender_in_go_opentelemetry_io_contrib_bridges_otelzap.test"}},{"Key":"telemetry.sdk.language","Value":{"Type":"STRING","Value":"go"}},{"Key":"telemetry.sdk.name","Value":{"Type":"STRING","Value":"opentelemetry"}},{"Key":"telemetry.sdk.version","Value":{"Type":"STRING","Value":"1.35.0"}}],"Scope":{"Name":"unknown","Version":"","SchemaURL":"","Attributes":{}},"DroppedAttributes":0}

	// Notes:
	// - Only the warn level log is written, since the core specifies zapcore.WarnLevel
	// - The log is written to stdout twice:
	//    - Once by the core's configured stdout writer w/ JSON encoded
	//    - Second by the otel log provider's stdout log exporter.
	// - Typically, the otel log provider would be configured with a batch processor and otlp exporter, which would result in the log being written to stdout once, and OTLP once.
}

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
