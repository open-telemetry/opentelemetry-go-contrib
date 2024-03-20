// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelslog

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/embedded"
)

func TestNewLogger(t *testing.T) {
	assert.IsType(t, &Handler{}, NewLogger().Handler())
}

// embeddedLogger is a type alias so the embedded.Logger type doesn't conflict
// with the Logger method of the recorder when it is embedded.
type embeddedLogger = embedded.Logger // nolint:unused  // Used below.

// recorder records all [log.Record]s it is ased to emit.
type recorder struct {
	embedded.LoggerProvider
	embeddedLogger // nolint:unused  // Used to embed embedded.Logger.

	// MinSeverity is the minimum severity the recorder will return true for
	// when Enabled is called (unless enableKey is set).
	MinSeverity log.Severity
}

func (r *recorder) Logger(string, ...log.LoggerOption) log.Logger { return r }

func (r *recorder) Emit(context.Context, log.Record) {
}

type enablerKey uint

var enableKey enablerKey

func (r *recorder) Enabled(ctx context.Context, record log.Record) bool {
	return ctx.Value(enableKey) != nil || record.Severity() >= r.MinSeverity
}

func TestHandlerEnabled(t *testing.T) {
	r := new(recorder)
	r.MinSeverity = log.SeverityInfo

	h := NewHandler(WithLoggerProvider(r))
	h.logger = r.Logger("") // TODO: Remove when #5311 merged.

	ctx := context.Background()
	assert.False(t, h.Enabled(ctx, slog.LevelDebug), "level conversion: permissive")
	assert.True(t, h.Enabled(ctx, slog.LevelInfo), "level conversion: restrictive")

	ctx = context.WithValue(ctx, enableKey, true)
	assert.True(t, h.Enabled(ctx, slog.LevelDebug), "context not passed")
}
