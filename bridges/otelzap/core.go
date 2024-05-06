// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package otelzap provides a bridge between the [go.uber.org/zap] and
// OpenTelemetry logging.
package otelzap // import "otelzap"

import (
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/otel/log"
)

const (
	bridgeName = "go.opentelemetry.io/contrib/bridges/otelzap"
)

// Core is a [zapcore.Core] that sends logging records to OpenTelemetry.
type Core struct {
	logger log.Logger
}

// Compile-time check *Core implements zapcore.Core.
var _ zapcore.Core = (*Core)(nil)

// NewOTelZapCore creates a new [zapcore.Core] that can be used with zap.New()
// this instance will translate zap logs to opentelemetry logs and export them.
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

// Sync flushes buffered logs (if any).
func (o *Core) Sync() error {
	return nil
}

// Check determines whether the supplied Entry should be logged using core.Enabled method.
func (o *Core) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	// TODO
	return ce
}

// TODO
// Write method encodes zap fields to OTel logs and emits them.
func (o *Core) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	return nil
}
