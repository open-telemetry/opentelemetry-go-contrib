// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package otellogr provides a [LogSink], a [logr.LogSink] implementation that
// can be used to bridge between the [logr] API and [OpenTelemetry].
//
// # Record Conversion
//
// The logr records are converted to OpenTelemetry [log.Record] in the following
// way:
//
//   - Message is set as the Body using a [log.StringValue].
//   - Level is transformed and set as the Severity. The SeverityText is not
//     set.
//   - KeyAndValues are transformed and set as Attributes.
//   - Error is always logged as an additional attribute with the key
//     "exception.message" and with the severity [log.SeverityError].
//   - The [context.Context] value in KeyAndValues is propagated to OpenTelemetry
//     log record. All non-nested [context.Context] values are ignored and not
//     added as attributes. If there are multiple [context.Context] the last one
//     is used.
//
// The V-level is transformed by using the [WithLevelSeverity] option. If option is
// not provided then V-level is transformed in the following way:
//
//   - logr.Info and logr.V(0) are transformed to [log.SeverityInfo].
//   - logr.V(1) is transformed to [log.SeverityDebug].
//   - logr.V(2) and higher are transformed to [log.SeverityTrace].
//
// KeysAndValues values are transformed based on their type. The following types are
// supported:
//
//   - [bool] are transformed to [log.BoolValue].
//   - [string] are transformed to [log.StringValue].
//   - [int], [int8], [int16], [int32], [int64] are transformed to
//     [log.Int64Value].
//   - [uint], [uint8], [uint16], [uint32], [uint64], [uintptr] are transformed
//     to [log.Int64Value] or [log.StringValue] if the value is too large.
//   - [float32], [float64] are transformed to [log.Float64Value].
//   - [time.Duration] are transformed to [log.Int64Value] with the nanoseconds.
//   - [complex64], [complex128] are transformed to [log.MapValue] with the keys
//     "r" and "i" for the real and imaginary parts. The values are
//     [log.Float64Value].
//   - [time.Time] are transformed to [log.Int64Value] with the nanoseconds.
//   - [[]byte] are transformed to [log.BytesValue].
//   - [error] are transformed to [log.StringValue] with the error message.
//   - [nil] are transformed to an empty [log.Value].
//   - [struct] are transformed to [log.StringValue] with the struct fields.
//   - [slice], [array] are transformed to [log.SliceValue] with the elements.
//   - [map] are transformed to [log.MapValue] with the key-value pairs.
//   - [pointer], [interface] are transformed to the dereferenced value.
//
// [OpenTelemetry]: https://opentelemetry.io/docs/concepts/signals/logs/
package otellogr // import "go.opentelemetry.io/contrib/bridges/otellogr"

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
)

type config struct {
	provider  log.LoggerProvider
	version   string
	schemaURL string

	levelSeverity func(int) log.Severity
}

func newConfig(options []Option) config {
	var c config
	for _, opt := range options {
		c = opt.apply(c)
	}

	if c.provider == nil {
		c.provider = global.GetLoggerProvider()
	}

	if c.levelSeverity == nil {
		c.levelSeverity = func(level int) log.Severity {
			switch level {
			case 0:
				return log.SeverityInfo
			case 1:
				return log.SeverityDebug
			default:
				return log.SeverityTrace
			}
		}
	}

	return c
}

// Option configures a [LogSink].
type Option interface {
	apply(config) config
}

type optFunc func(config) config

func (f optFunc) apply(c config) config { return f(c) }

// WithVersion returns an [Option] that configures the version of the
// [log.Logger] used by a [LogSink]. The version should be the version of the
// package that is being logged.
func WithVersion(version string) Option {
	return optFunc(func(c config) config {
		c.version = version
		return c
	})
}

// WithSchemaURL returns an [Option] that configures the semantic convention
// schema URL of the [log.Logger] used by a [LogSink]. The schemaURL should be
// the schema URL for the semantic conventions used in log records.
func WithSchemaURL(schemaURL string) Option {
	return optFunc(func(c config) config {
		c.schemaURL = schemaURL
		return c
	})
}

// WithLoggerProvider returns an [Option] that configures [log.LoggerProvider]
// used by a [LogSink] to create its [log.Logger].
//
// By default if this Option is not provided, the LogSink will use the global
// LoggerProvider.
func WithLoggerProvider(provider log.LoggerProvider) Option {
	return optFunc(func(c config) config {
		c.provider = provider
		return c
	})
}

// WithLevelSeverity returns an [Option] that configures the function used to
// convert logr levels to OpenTelemetry log severities.
//
// By default if this Option is not provided, the LogSink will use a default
// conversion function that transforms in the following way:
//
//   - logr.Info and logr.V(0) are transformed to [log.SeverityInfo].
//   - logr.V(1) is transformed to [log.SeverityDebug].
//   - logr.V(2) and higher are transformed to [log.SeverityTrace].
func WithLevelSeverity(f func(int) log.Severity) Option {
	return optFunc(func(c config) config {
		c.levelSeverity = f
		return c
	})
}

// NewLogSink returns a new [LogSink] to be used as a [logr.LogSink].
//
// If [WithLoggerProvider] is not provided, the returned [LogSink] will use the
// global LoggerProvider.
func NewLogSink(name string, options ...Option) *LogSink {
	c := newConfig(options)

	var opts []log.LoggerOption
	if c.version != "" {
		opts = append(opts, log.WithInstrumentationVersion(c.version))
	}
	if c.schemaURL != "" {
		opts = append(opts, log.WithSchemaURL(c.schemaURL))
	}

	return &LogSink{
		name:          name,
		provider:      c.provider,
		logger:        c.provider.Logger(name, opts...),
		levelSeverity: c.levelSeverity,
		opts:          opts,
		ctx:           context.Background(),
	}
}

// LogSink is a [logr.LogSink] that sends all logging records it receives to
// OpenTelemetry. See package documentation for how conversions are made.
type LogSink struct {
	// Ensure forward compatibility by explicitly making this not comparable.
	noCmp [0]func() //nolint: unused  // This is indeed used.

	name          string
	provider      log.LoggerProvider
	logger        log.Logger
	levelSeverity func(int) log.Severity
	opts          []log.LoggerOption
	attr          []log.KeyValue
	ctx           context.Context
}

// Compile-time check *Handler implements logr.LogSink.
var _ logr.LogSink = (*LogSink)(nil)

// Enabled tests whether this LogSink is enabled at the specified V-level.
// For example, commandline flags might be used to set the logging
// verbosity and disable some info logs.
func (l *LogSink) Enabled(level int) bool {
	ctx := context.Background()
	param := log.EnabledParameters{Severity: l.levelSeverity(level)}
	return l.logger.Enabled(ctx, param)
}

// Error logs an error, with the given message and key/value pairs.
func (l *LogSink) Error(err error, msg string, keysAndValues ...any) {
	var record log.Record
	record.SetBody(log.StringValue(msg))
	record.SetSeverity(log.SeverityError)

	record.AddAttributes(
		log.String(string(semconv.ExceptionMessageKey), err.Error()),
	)

	record.AddAttributes(l.attr...)

	ctx, attr := convertKVs(l.ctx, keysAndValues...)
	record.AddAttributes(attr...)

	l.logger.Emit(ctx, record)
}

// Info logs a non-error message with the given key/value pairs.
func (l *LogSink) Info(level int, msg string, keysAndValues ...any) {
	var record log.Record
	record.SetBody(log.StringValue(msg))
	record.SetSeverity(l.levelSeverity(level))

	record.AddAttributes(l.attr...)

	ctx, attr := convertKVs(l.ctx, keysAndValues...)
	record.AddAttributes(attr...)

	l.logger.Emit(ctx, record)
}

// Init receives optional information about the logr library this
// implementation does not use it.
func (l *LogSink) Init(logr.RuntimeInfo) {
	// We don't need to do anything here.
	// CallDepth is used to calculate the caller's PC.
	// PC is dropped as part of the conversion to the OpenTelemetry log.Record.
}

// WithName returns a new LogSink with the specified name appended.
func (l LogSink) WithName(name string) logr.LogSink {
	l.name = l.name + "/" + name
	l.logger = l.provider.Logger(l.name, l.opts...)
	return &l
}

// WithValues returns a new LogSink with additional key/value pairs.
func (l LogSink) WithValues(keysAndValues ...any) logr.LogSink {
	ctx, attr := convertKVs(l.ctx, keysAndValues...)
	l.attr = append(l.attr, attr...)
	l.ctx = ctx
	return &l
}

// convertKVs converts a list of key-value pairs to a list of [log.KeyValue].
// The last [context.Context] value is returned as the context.
// If no context is found, the original context is returned.
func convertKVs(ctx context.Context, keysAndValues ...any) (context.Context, []log.KeyValue) {
	if len(keysAndValues) == 0 {
		return ctx, nil
	}
	if len(keysAndValues)%2 != 0 {
		// Ensure an odd number of items here does not corrupt the list.
		keysAndValues = append(keysAndValues, nil)
	}

	kvs := make([]log.KeyValue, 0, len(keysAndValues)/2)
	for i := 0; i < len(keysAndValues); i += 2 {
		k, ok := keysAndValues[i].(string)
		if !ok {
			// Ensure that the key is a string.
			k = fmt.Sprintf("%v", keysAndValues[i])
		}

		v := keysAndValues[i+1]
		if vCtx, ok := v.(context.Context); ok {
			// Special case when a field is of context.Context type.
			ctx = vCtx
			continue
		}

		kvs = append(kvs, log.KeyValue{
			Key:   k,
			Value: convertValue(v),
		})
	}

	return ctx, kvs
}
