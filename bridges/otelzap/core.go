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
)

const (
	bridgeName = "go.opentelemetry.io/contrib/bridge/zapcore"
)

// Core is a [zapcore.Core] that sends logging records to OpenTelemetry.
type Core struct {
	logger log.Logger
	attr   []log.KeyValue
	ctx    context.Context
}

// // Compile-time check *Core implements zapcore.Core.
var _ zapcore.Core = (*Core)(nil)

// NewOTelZapCore creates a new [zapcore.Core] that can be used with zap.New()
// this instance will translate zap logs to opentelemetry logs and export them.
func NewCore(opts ...Option) zapcore.Core {
	cfg := newConfig(opts)
	return &Core{
		logger: cfg.logger(),
		ctx:    context.Background(),
	}
}

// LevelEnabler decides whether a given logging level is enabled when logging a message.
func (o *Core) Enabled(level zapcore.Level) bool {
	r := log.Record{}
	r.SetSeverity(getOTelLevel(level))
	return o.logger.Enabled(context.Background(), r)
}

// With adds structured context to the Core.
func (o *Core) With(fields []zapcore.Field) zapcore.Core {
	clone := o.clone()
	if len(fields) > 0 {
		attrbuf := getAttr(&clone.ctx, &fields)
		clone.attr = append(clone.attr, attrbuf...)
	}
	return clone
}

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

	if len(fields) > 0 {
		attrbuf := getAttr(&o.ctx, &fields)
		attrbuf = append(attrbuf, o.attr...)
		r.AddAttributes(attrbuf...)
	}

	o.logger.Emit(o.ctx, r)
	return nil
}

func (o *Core) clone() *Core {
	return &Core{
		logger: o.logger,
		attr:   slices.Clone(o.attr),
		ctx:    o.ctx,
	}
}

// converts zap fields to OTel log attributes.
func getAttr(ctx *context.Context, fields *[]zapcore.Field) []log.KeyValue {
	enc, free := getObjectEncoder()
	defer free()
	for _, field := range *fields {
		if ctxFld, ok := field.Interface.(context.Context); ok {
			*ctx = ctxFld
			continue
		}
		field.AddTo(enc)
	}
	enc.getObjValue(enc.root)
	return enc.root.kv
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
