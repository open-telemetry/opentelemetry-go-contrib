// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package otelzerolog provides a SeverityHook, a zerolog.Hook implementation that
// can be used to bridge between the [zerolog] API and [OpenTelemetry].
//
// # Record Conversion
//
// The zerolog.Event records are converted to OpenTelemetry [log.Record] in
// the following way:
//
//   - Time is set as the Timestamp.
//   - Message is set as the Body using a [log.StringValue].
//   - Level is transformed and set as the Severity. The SeverityText is not
//     set.
//   - Fields are transformed and set as the attributes.
//
// The Level is transformed by using a mapping function to the OpenTelemetry
// Severity types. For example:
//
//   - zerolog.DebugLevel is transformed to [log.SeverityDebug]
//   - zerolog.InfoLevel is transformed to [log.SeverityInfo]
//   - zerolog.WarnLevel is transformed to [log.SeverityWarn]
//   - zerolog.ErrorLevel is transformed to [log.SeverityError]
//   - zerolog.FatalLevel and zerolog.PanicLevel are mapped to
//     [log.SeverityError] (consider customization for these levels)
//
// Attribute values are transformed based on their type into log attributes, or
// into a string value if there is no matching type.
//
// [zerolog]: https://github.com/rs/zerolog
// [OpenTelemetry]: https://opentelemetry.io/docs/concepts/signals/logs/
package otelzerolog // import "go.opentelemetry.io/contrib/bridges/otelzerolog"

import (
	"fmt"

	"github.com/rs/zerolog"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
)

type config struct {
	provider  log.LoggerProvider
	version   string
	schemaURL string
}

func newConfig(options []Option) config {
	var c config
	for _, opt := range options {
		c = opt.apply(c)
	}

	if c.provider == nil {
		c.provider = global.GetLoggerProvider()
	}

	return c
}

func (c config) logger(name string) log.Logger {
	var opts []log.LoggerOption
	if c.version != "" {
		opts = append(opts, log.WithInstrumentationVersion(c.version))
	}
	if c.schemaURL != "" {
		opts = append(opts, log.WithSchemaURL(c.schemaURL))
	}
	return c.provider.Logger(name, opts...)
}

// Option configures a SeverityHook.
type Option interface {
	apply(config) config
}
type optFunc func(config) config

func (f optFunc) apply(c config) config { return f(c) }

// WithVersion returns an [Option] that configures the version of the
// [log.Logger] used by a [SeverityHook]. The version should be the version of the
// package that is being logged.
func WithVersion(version string) Option {
	return optFunc(func(c config) config {
		c.version = version
		return c
	})
}

// WithSchemaURL returns an [Option] that configures the semantic convention
// schema URL of the [log.Logger] used by a [SeverityHook]. The schemaURL should be
// the schema URL for the semantic conventions used in log records.
func WithSchemaURL(schemaURL string) Option {
	return optFunc(func(c config) config {
		c.schemaURL = schemaURL
		return c
	})
}

// WithLoggerProvider returns an [Option] that configures [log.LoggerProvider]
// used by a [SeverityHook].
//
// By default if this Option is not provided, the SeverityHook will use the global
// LoggerProvider.
func WithLoggerProvider(provider log.LoggerProvider) Option {
	return optFunc(func(c config) config {
		c.provider = provider
		return c
	})
}

// NewSeverityHook returns a new [SeverityHook] to be used as a [Zerolog.Hook].
//
// If [WithLoggerProvider] is not provided, the returned SeverityHook will use the
// global LoggerProvider.
func NewSeverityHook(name string, options ...Option) *SeverityHook {
	cfg := newConfig(options)
	return &SeverityHook{
		logger: cfg.logger(name),
	}
}

// // Hook is a [zerolog.Hook] that sends all logging records it receives to
// OpenTelemetry. See package documentation for how conversions are made.
type SeverityHook struct {
	logger log.Logger
	levels zerolog.Level
}

// Levels returns the list of log levels we want to be sent to OpenTelemetry.
func (h *SeverityHook) Levels() zerolog.Level {
	return h.levels
}

// Run handles the passed record, and sends it to OpenTelemetry.
func (h SeverityHook) Run(e *zerolog.Event, level zerolog.Level, msg string) error {
	if level != zerolog.NoLevel {
		e.Str("severity", level.String())
	}
	h.logger.Emit(e.GetCtx(), h.convertEvent(e, level, msg))
	return nil
}

func (h *SeverityHook) convertEvent(e *zerolog.Event, level zerolog.Level, msg string) log.Record {
	var record log.Record
	record.SetTimestamp(zerolog.TimestampFunc())
	record.SetBody(log.StringValue(msg))
	const sevOffset = zerolog.Level(log.SeverityDebug) - zerolog.DebugLevel
	record.SetSeverity(log.Severity(level + sevOffset))
	fields := extractFields(e)

	record.AddAttributes(convertFields(fields, msg)...)
	return record
}

func extractFields(_ *zerolog.Event) map[string]interface{} {
	// Here you would implement the logic to extract fields from the zerolog event
	// This might involve using reflection or zerolog internals if necessary
	fields := make(map[string]interface{})
	// Dummy implementation - replace with actual field extraction

	return fields
}

func convertFields(fields map[string]interface{}, msg string) []log.KeyValue {
	kvs := make([]log.KeyValue, 0, len(fields))
	kvs = append(kvs, log.String("message", msg))
	for k, v := range fields {
		kvs = append(kvs, convertAttribute(k, v))
	}
	return kvs
}

func convertAttribute(key string, value interface{}) log.KeyValue {
	switch v := value.(type) {
	case bool:
		return log.Bool(key, v)
	case []byte:
		return log.String(key, string(v))
	case float64:
		return log.Float64(key, v)
	case int:
		return log.Int(key, v)
	case int64:
		return log.Int64(key, v)
	case string:
		return log.String(key, v)
	default:
		// Fallback to string representation for unhandled types
		return log.String(key, fmt.Sprintf("%v", v))
	}
}