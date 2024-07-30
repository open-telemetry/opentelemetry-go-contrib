// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package otelzerolog provides a [Hook], a [zerolog.Hook] implementation that
// can be used to bridge between the [zerolog] API and [OpenTelemetry].
//
// # Record Conversion
//
// The [zerolog.Event] records are converted to OpenTelemetry [log.Record] in
// the following way:
//
//   - Message is set as the Body using a [log.StringValue].
//   - Level is transformed and set as the Severity. The SeverityText is also
//     set.
//
// The Level is transformed to the OpenTelemetry Severity types in the following way.
//
//   - [zerolog.DebugLevel] is transformed to [log.SeverityDebug]
//   - [zerolog.InfoLevel] is transformed to [log.SeverityInfo]
//   - [zerolog.WarnLevel] is transformed to [log.SeverityWarn]
//   - [zerolog.ErrorLevel] is transformed to [log.SeverityError]
//   - [zerolog.PanicLevel] is transformed to [log.SeverityFatal1]
//   - [zerolog.FatalLevel] is transformed to [log.SeverityFatal2]
//
// NOTE: Fields are not transformed because of https://github.com/rs/zerolog/issues/493.
//
// [OpenTelemetry]: https://opentelemetry.io/docs/concepts/signals/logs/
package otelzerolog // import "go.opentelemetry.io/contrib/bridges/otelzerolog"

import (
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

// Option configures a Hook.
type Option interface {
	apply(config) config
}
type optFunc func(config) config

func (f optFunc) apply(c config) config { return f(c) }

// WithVersion returns an [Option] that configures the version of the
// [log.Logger] used by a [Hook]. The version should be the version of the
// package that is being logged.
func WithVersion(version string) Option {
	return optFunc(func(c config) config {
		c.version = version
		return c
	})
}

// WithSchemaURL returns an [Option] that configures the semantic convention
// schema URL of the [log.Logger] used by a [Hook]. The schemaURL should be
// the schema URL for the semantic conventions used in log records.
func WithSchemaURL(schemaURL string) Option {
	return optFunc(func(c config) config {
		c.schemaURL = schemaURL
		return c
	})
}

// WithLoggerProvider returns an [Option] that configures [log.LoggerProvider]
// used by a [Hook].
//
// By default if this Option is not provided, the Hook will use the global
// LoggerProvider.
func WithLoggerProvider(provider log.LoggerProvider) Option {
	return optFunc(func(c config) config {
		c.provider = provider
		return c
	})
}

// Hook is a [zerolog.Hook] that sends all logging records it receives to
// OpenTelemetry. See package documentation for how conversions are made.
type Hook struct {
	logger log.Logger
}

// NewHook returns a new [Hook] to be used as a [Zerolog.Hook].
// If [WithLoggerProvider] is not provided, the returned Hook will use the
// global LoggerProvider.
func NewHook(name string, options ...Option) *Hook {
	cfg := newConfig(options)
	return &Hook{
		logger: cfg.logger(name),
	}
}

// Run handles the passed record, and sends it to OpenTelemetry.
func (h Hook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	r := log.Record{}
	r.SetSeverity(convertLevel(level))
	r.SetBody(log.StringValue(msg))
	r.SetSeverityText(level.String())

	// TODO: add support for attributes
	// This is limited by zerolog's inability to retrieve fields.
	// https://github.com/rs/zerolog/issues/493

	h.logger.Emit(e.GetCtx(), r)
}

func convertLevel(level zerolog.Level) log.Severity {
	switch level {
	case zerolog.DebugLevel:
		return log.SeverityDebug
	case zerolog.InfoLevel:
		return log.SeverityInfo
	case zerolog.WarnLevel:
		return log.SeverityWarn
	case zerolog.ErrorLevel:
		return log.SeverityError
	case zerolog.PanicLevel:
		return log.SeverityFatal1
	case zerolog.FatalLevel:
		return log.SeverityFatal2
	default:
		return log.SeverityUndefined
	}
}
