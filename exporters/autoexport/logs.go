// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package autoexport // import "go.opentelemetry.io/contrib/exporters/autoexport"

import (
	"context"
	"errors"

	"go.opentelemetry.io/contrib/exporters/autoexport/utils/env"
	"go.opentelemetry.io/contrib/exporters/autoexport/utils/functional"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/sdk/log"
)

const (
	otelLogsExporterEnvKey         = "OTEL_LOGS_EXPORTER"
	otelLogsExporterProtocolEnvKey = "OTEL_EXPORTER_OTLP_LOGS_PROTOCOL"
)

var (
	logsSignal = newSignal[log.Exporter](otelLogsExporterEnvKey)

	errLogsUnsupportedGRPCProtocol = errors.New("log exporter do not support 'grpc' protocol yet - consider using 'http/protobuf' instead")
)

// LogExporterOption applies an autoexport configuration option.
type LogExporterOption = functional.Option[config[log.Exporter]]

// WithFallbackLogExporter sets the fallback exporter to use when no exporter
// is configured through the OTEL_LOGS_EXPORTER environment variable.
func WithFallbackLogExporter(factoryFn factory[log.Exporter]) LogExporterOption {
	return withFallbackFactory(factoryFn)
}

// NewLogExporters returns one or more configured [go.opentelemetry.io/otel/sdk/log.Exporter]
// defined using the environment variables described below.
//
// OTEL_LOGS_EXPORTER defines the logs exporter; this value accepts a comma-separated list of values to enable multiple exporters; supported values:
//   - "none" - "no operation" exporter
//   - "otlp" (default) - OTLP exporter; see [go.opentelemetry.io/otel/exporters/otlp/otlplog]
//   - "console" - Standard output exporter; see [go.opentelemetry.io/otel/exporters/stdout/stdoutlog]
//
// OTEL_EXPORTER_OTLP_PROTOCOL defines OTLP exporter's transport protocol;
// supported values:
//   - "http/protobuf" (default) -  protobuf-encoded data over HTTP connection;
//     see: [go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp]
//
// An error is returned if an environment value is set to an unhandled value.
// Use [WithFallbackLogExporter] option to change the returned exporter
// when OTEL_LOGS_EXPORTER is unset or empty.
//
// Use [RegisterLogExporter] to handle more values of OTEL_LOGS_EXPORTER.
//
// Use [IsNoneLogExporter] to check if the returned exporter is a "no operation" exporter.
func NewLogExporters(ctx context.Context, options ...LogExporterOption) ([]log.Exporter, error) {
	return logsSignal.create(ctx, options...)
}

// NewLogExporter returns a configured [go.opentelemetry.io/otel/sdk/log.Exporter]
// defined using the environment variables described below.
//
// DEPRECATED: consider using [NewLogExporters] instead.
//
// OTEL_LOGS_EXPORTER defines the logs exporter; supported values:
//   - "none" - "no operation" exporter
//   - "otlp" (default) - OTLP exporter; see [go.opentelemetry.io/otel/exporters/otlp/otlplog]
//   - "console" - Standard output exporter; see [go.opentelemetry.io/otel/exporters/stdout/stdoutlog]
//
// OTEL_EXPORTER_OTLP_PROTOCOL defines OTLP exporter's transport protocol;
// supported values:
//   - "http/protobuf" (default) -  protobuf-encoded data over HTTP connection;
//     see: [go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp]
//
// OTEL_EXPORTER_OTLP_LOGS_PROTOCOL defines OTLP exporter's transport protocol for the logs signal;
// supported values are the same as OTEL_EXPORTER_OTLP_PROTOCOL.
//
// An error is returned if an environment value is set to an unhandled value.
// Use [WithFallbackLogExporter] option to change the returned exporter
// when OTEL_LOGS_EXPORTER is unset or empty.
//
// Use [RegisterLogExporter] to handle more values of OTEL_LOGS_EXPORTER.
//
// Use [IsNoneLogExporter] to check if the returned exporter is a "no operation" exporter.
func NewLogExporter(ctx context.Context, options ...LogExporterOption) (log.Exporter, error) {
	exporters, err := NewLogExporters(ctx, options...)
	if err != nil {
		return nil, err
	}
	return exporters[0], nil
}

// RegisterLogExporter sets the log.Exporter factory to be used when the
// OTEL_LOGS_EXPORTER environment variable contains the exporter name.
// This will panic if name has already been registered.
func RegisterLogExporter(name string, factoryFn factory[log.Exporter]) {
	must(logsSignal.registry.store(name, factoryFn))
}

func init() {
	RegisterLogExporter("otlp", func(ctx context.Context) (log.Exporter, error) {
		// The transport protocol used by the exporter is determined using the
		// following environment variables, ordered by priority:
		//   - OTEL_EXPORTER_OTLP_LOGS_PROTOCOL
		//   - OTEL_EXPORTER_OTLP_PROTOCOL
		//   - fallback to 'http/protobuf' if variables above are not set or empty.
		proto := env.WithDefaultString(
			otelLogsExporterProtocolEnvKey,
			env.WithDefaultString(otelExporterOTLPProtoEnvKey, "http/protobuf"),
		)

		switch proto {
		case "grpc":
			// grpc is not supported yet, should uncomment when it is supported.
			// return otlplogrpc.New(ctx)
			return nil, errLogsUnsupportedGRPCProtocol
		case "http/protobuf":
			return otlploghttp.New(ctx)
		default:
			return nil, errInvalidOTLPProtocol
		}
	})
	RegisterLogExporter("console", func(_ context.Context) (log.Exporter, error) {
		return stdoutlog.New()
	})
	RegisterLogExporter("none", func(_ context.Context) (log.Exporter, error) {
		return noopLogExporter{}, nil
	})
}
