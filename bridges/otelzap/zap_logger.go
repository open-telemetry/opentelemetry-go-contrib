// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package otelzap provides a bridge between the ["go.uber.org/zap"] and
// OpenTelemetry logging.
package otelzap // import "go.opentelemetry.io/contrib/bridges/otelzap"

import (
	"context"

	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/otel/log"
)

const (
	bridgeName = "go.opentelemetry.io/contrib/bridge/zapcore"
)

// Core is a [zapcore.Core] that sends all logging records it receives to
// OpenTelemetry.
type Core struct {
	logger log.Logger
	attr   []log.KeyValue
}

// Compile-time check *Core implements zapcore.Core.
var _ zapcore.Core = (*Core)(nil)

// this function creates a new zapcore.Core that can be used with zap.New()
// this instance will translate zap logs to opentelemetry logs and export them.
func NewOtelZapCore(opts ...Option) zapcore.Core {
	cfg := newConfig(opts)
	return &Core{
		logger: cfg.logger(),
	}
}

// LevelEnabler decides whether a given logging level is enabled when logging a
// message.
func (o *Core) Enabled(level zapcore.Level) bool {
	r := log.Record{}
	r.SetSeverity(getOtelLevel(level))

	// check how to propagate context
	ctx := context.Background()
	return o.logger.Enabled(ctx, r)
}

// With adds structured context to the Core.
// by returning a child logger with provided fields.
func (o *Core) With(fields []zapcore.Field) zapcore.Core {
	clone := o.clone()
	clone.attr = append(clone.attr, getAttr(fields)...)
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

// Writes to the destination.
func (o *Core) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	r := log.Record{}
	r.SetTimestamp(ent.Time)
	r.SetBody(log.StringValue(ent.Message))
	r.SetSeverity(getOtelLevel(ent.Level))

	// get attr from fields
	attr := getAttr(fields)
	// append attributes received from from parent logger
	addattr := append(attr, o.attr...)

	if len(addattr) > 0 {
		r.AddAttributes(addattr...)
	}

	// need to check how to propagate context here
	ctx := context.Background()
	o.logger.Emit(ctx, r)
	return nil
}

func (o *Core) clone() *Core {
	return &Core{
		logger: o.logger,
		attr:   o.attr,
	}
}

// converts zap fields to Otel's log attributes.
func getAttr(fields []zapcore.Field) []log.KeyValue {
	enc := NewObjectEncoder(len(fields))
	for i := range fields {
		fields[i].AddTo(enc)
	}
	return enc.cur
}

// converts zap level to Otel's log level.
func getOtelLevel(level zapcore.Level) log.Severity {
	// should confirm this
	// the logic here is that
	// zapcore.Debug = -1 & logger.Debug = 3
	// zapcore.Info = 0   & logger.Info = 7 and so on
	sevOffset := 4*(level+2) + 1
	return log.Severity(level + sevOffset)
}
