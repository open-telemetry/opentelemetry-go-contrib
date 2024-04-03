// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package otellogrus // import "go.opentelemetry.io/contrib/bridges/otellogrus"

import (
	"github.com/sirupsen/logrus"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/sdk/instrumentation"
)

type config struct {
	provider log.LoggerProvider
	scope    instrumentation.Scope

	levels []logrus.Level
}

func newConfig(options []Option) config {
	var c config
	for _, opt := range options {
		c = opt.apply(c)
	}

	var emptyScope instrumentation.Scope
	if c.scope == emptyScope {
		c.scope = instrumentation.Scope{
			Name:    bridgeName,
			Version: version,
		}
	}

	if c.provider == nil {
		c.provider = global.GetLoggerProvider()
	}

	if c.levels == nil {
		c.levels = logrus.AllLevels
	}

	return c
}

func (c config) logger() log.Logger {
	var opts []log.LoggerOption
	if c.scope.Version != "" {
		opts = append(opts, log.WithInstrumentationVersion(c.scope.Version))
	}
	if c.scope.SchemaURL != "" {
		opts = append(opts, log.WithSchemaURL(c.scope.SchemaURL))
	}
	return c.provider.Logger(c.scope.Name, opts...)
}

// Option configures a [Hook].
type Option interface {
	apply(config) config
}

type optFunc func(config) config

func (f optFunc) apply(c config) config { return f(c) }

// WithInstrumentationScope returns an [Option] that configures the scope of
// the logs that will be emitted by the configured [Hook].
//
// By default if this Option is not provided, the Hook will use a default
// instrumentation scope describing this bridge package. It is recommended to
// provide this so log data can be associated with its source package or
// module.
func WithInstrumentationScope(scope instrumentation.Scope) Option {
	return optFunc(func(c config) config {
		c.scope = scope
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

// WithLevels returns an [Option] that configures the log levels that will fire
// the configured [Hook].
//
// By default if this Option is not provided, the Hook will fire for all levels.
// LoggerProvider.
func WithLevels(l []logrus.Level) Option {
	return optFunc(func(c config) config {
		c.levels = l
		return c
	})
}
