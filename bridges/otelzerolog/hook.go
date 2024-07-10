// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package otelzerolog provides a SeverityHook, a zerolog.Hook implementation that
// can be used to bridge between the [zerolog] API and [OpenTelemetry].
// [zerolog]: https://github.com/rs/zerolog
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

// SeverityHook is a [zerolog.Hook] that sends all logging records it receives to
// OpenTelemetry. See package documentation for how conversions are made.
type SeverityHook struct {
	logger log.Logger
}

// NewSeverityHook returns a new [SeverityHook] to be used as a [Zerolog.Hook].
// If [WithLoggerProvider] is not provided, the returned SeverityHook will use the
// global LoggerProvider.
func NewSeverityHook(name string, options ...Option) *SeverityHook {
	cfg := newConfig(options)
	return &SeverityHook{
		logger: cfg.logger(name),
	}
}

// TODO
// Run handles the passed record, and sends it to OpenTelemetry.
func (h SeverityHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
}
