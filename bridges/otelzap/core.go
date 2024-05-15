// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package otelzap provides a bridge between the [go.uber.org/zap] and
// OpenTelemetry logging.
package otelzap // import "go.opentelemetry.io/contrib/bridges/otelzap"

import (
	"context"

	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/sdk/instrumentation"
)

const (
	bridgeName = "go.opentelemetry.io/contrib/bridges/otelzap"
)

type config struct {
	provider log.LoggerProvider
	scope    instrumentation.Scope
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

// Option configures a [Core].
type Option interface {
	apply(config) config
}

type optFunc func(config) config

func (f optFunc) apply(c config) config { return f(c) }

// WithInstrumentationScope returns an [Option] that configures the scope of
// the [log.Logger] used by a [Core].
//
// By default if this Option is not provided, [Core] will use a default
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
// used by a [Core] to create its [log.Logger].
//
// By default if this Option is not provided, the Handler will use the global
// LoggerProvider.
func WithLoggerProvider(provider log.LoggerProvider) Option {
	return optFunc(func(c config) config {
		c.provider = provider
		return c
	})
}

// Core is a [zapcore.Core] that sends logging records to OpenTelemetry.
type Core struct {
	logger log.Logger
}

// Compile-time check *Core implements zapcore.Core.
var _ zapcore.Core = (*Core)(nil)

// NewCore creates a new [zapcore.Core] that can be used with [go.uber.org/zap.New].
func NewCore(opts ...Option) *Core {
	cfg := newConfig(opts)
	return &Core{
		logger: cfg.logger(),
	}
}

// TODO
// LevelEnabler decides whether a given logging level is enabled when logging a message.
func (o *Core) Enabled(level zapcore.Level) bool {
	return true
}

// TODO
// With adds structured context to the Core.
func (o *Core) With(fields []zapcore.Field) zapcore.Core {
	return o
}

// TODO
// Sync flushes buffered logs (if any).
func (o *Core) Sync() error {
	return nil
}

// Check determines whether the supplied Entry should be logged using core.Enabled method.
func (o *Core) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if o.Enabled(ent.Level) {
		return ce.AddCore(ent, o)
	}
	return ce
}

// Write method encodes zap fields to OTel logs and emits them.
func (o *Core) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	r := log.Record{}
	r.SetTimestamp(ent.Time)
	r.SetBody(log.StringValue(ent.Message))
	r.SetSeverity(getOTelLevel(ent.Level))
	// TODO: Handle attributes passed via fields (exceptions: context.Context and zap.Namespace).
	// TODO: Handle attributes passed via With (exceptions: context.Context and zap.Namespace).
	// TODO: Handle context.Context containing trace context.
	// TODO: Handle zap.Namespace.
	// TODO: Handle logger name.
	o.logger.Emit(context.Background(), r)
	return nil
}

// converts zap level to OTel log level.
func getOTelLevel(level zapcore.Level) log.Severity {
	switch level {
	case zapcore.DebugLevel:
		return log.SeverityDebug
	case zapcore.InfoLevel:
		return log.SeverityInfo
	case zapcore.WarnLevel:
		return log.SeverityWarn
	case zapcore.ErrorLevel:
		return log.SeverityError
	case zapcore.DPanicLevel:
		return log.SeverityFatal1
	case zapcore.PanicLevel:
		return log.SeverityFatal2
	case zapcore.FatalLevel:
		return log.SeverityFatal3
	default:
		return log.SeverityUndefined
	}
}
