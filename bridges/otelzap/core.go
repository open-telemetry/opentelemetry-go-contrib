// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package otelzap provides a bridge between the [go.uber.org/zap] and
// OpenTelemetry logging.
package otelzap // import "go.opentelemetry.io/contrib/bridges/otelzap"

import (
	"context"
	"slices"

	"go.uber.org/zap/zapcore"

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

// Option configures a [Core].
type Option interface {
	apply(config) config
}

type optFunc func(config) config

func (f optFunc) apply(c config) config { return f(c) }

// WithVersion returns an [Option] that configures the version of the
// [log.Logger] used by a [Core]. The version should be the version of the
// package that is being logged.
func WithVersion(version string) Option {
	return optFunc(func(c config) config {
		c.version = version
		return c
	})
}

// WithSchemaURL returns an [Option] that configures the semantic convention
// schema URL of the [log.Logger] used by a [Core]. The schemaURL should be
// the schema URL for the semantic conventions used in log records.
func WithSchemaURL(schemaURL string) Option {
	return optFunc(func(c config) config {
		c.schemaURL = schemaURL
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
	attr   []log.KeyValue
	ctx    context.Context
}

// Compile-time check *Core implements zapcore.Core.
var _ zapcore.Core = (*Core)(nil)

// NewCore creates a new [zapcore.Core] that can be used with [go.uber.org/zap.New].
func NewCore(name string, opts ...Option) *Core {
	cfg := newConfig(opts)
	return &Core{
		logger: cfg.logger(name),
		ctx:    context.Background(),
	}
}

// Enabled decides whether a given logging level is enabled when logging a message.
func (o *Core) Enabled(level zapcore.Level) bool {
	r := log.Record{}
	r.SetSeverity(convertLevel(level))
	return o.logger.Enabled(context.Background(), r)
}

// With adds structured context to the Core.
func (o *Core) With(fields []zapcore.Field) zapcore.Core {
	cloned := o.clone()
	if len(fields) > 0 {
		var attrbuf []log.KeyValue
		cloned.ctx, attrbuf = convertField(cloned.ctx, fields)
		cloned.attr = append(cloned.attr, attrbuf...)
	}
	return cloned
}

func (o *Core) clone() *Core {
	return &Core{
		logger: o.logger,
		attr:   slices.Clone(o.attr),
		ctx:    o.ctx,
	}
}

// TODO
// Sync flushes buffered logs (if any).
func (o *Core) Sync() error {
	return nil
}

// Check determines whether the supplied Entry should be logged using core.Enabled method.
// If the entry should be logged, the Core adds itself to the CheckedEntry and returns the result.
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
	r.SetSeverity(convertLevel(ent.Level))

	// TODO: Handle attributes passed via With (exceptions: context.Context and zap.Namespace).
	// TODO: Handle zap.Namespace.
	// TODO: Handle ent.LoggerName.

	r.AddAttributes(o.attr...)
	if len(fields) > 0 {
		var attrbuf []log.KeyValue
		o.ctx, attrbuf = convertField(o.ctx, fields)
		r.AddAttributes(attrbuf...)
	}

	o.logger.Emit(o.ctx, r)
	return nil
}

func convertField(ctx context.Context, fields []zapcore.Field) (context.Context, []log.KeyValue) {
	// TODO: Use objectEncoder from a pool instead of newObjectEncoder.
	enc := newObjectEncoder(len(fields))
	for _, field := range fields {
		if ctxFld, ok := field.Interface.(context.Context); ok {
			ctx = ctxFld
			continue
		}
		field.AddTo(enc)
	}

	return ctx, enc.kv
}

func convertLevel(level zapcore.Level) log.Severity {
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
