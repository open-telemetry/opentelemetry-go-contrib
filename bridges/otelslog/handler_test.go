// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelslog

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/embedded"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/sdk/instrumentation"
)

// embeddedLogger is a type alias so the embedded.Logger type doesn't conflict
// with the Logger method of the recorder when it is embedded.
type embeddedLogger = embedded.Logger // nolint:unused  // Used below.

// recorder records all [log.Record]s it is ased to emit.
type recorder struct {
	embedded.LoggerProvider
	embeddedLogger // nolint:unused  // Used to embed embedded.Logger.

	// Scope is the Logger scope recorder received when Logger was called.
	Scope instrumentation.Scope

	// MinSeverity is the minimum severity the recorder will return true for
	// when Enabled is called (unless enableKey is set).
	MinSeverity log.Severity
}

func (r *recorder) Logger(name string, opts ...log.LoggerOption) log.Logger {
	cfg := log.NewLoggerConfig(opts...)

	r2 := *r
	r2.Scope = instrumentation.Scope{
		Name:      name,
		Version:   cfg.InstrumentationVersion(),
		SchemaURL: cfg.SchemaURL(),
	}
	return &r2
}

func (r *recorder) Emit(context.Context, log.Record) {
	// TODO: implement.
}

type enablerKey uint

var enableKey enablerKey

func (r *recorder) Enabled(ctx context.Context, record log.Record) bool {
	return ctx.Value(enableKey) != nil || record.Severity() >= r.MinSeverity
}

func TestNewHandlerConfiguration(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		r := new(recorder)
		global.SetLoggerProvider(r)

		var h *Handler
		assert.NotPanics(t, func() { h = NewHandler() })
		assert.NotNil(t, h.logger)
		require.IsType(t, &recorder{}, h.logger)

		l := h.logger.(*recorder)
		want := instrumentation.Scope{Name: bridgeName, Version: version}
		assert.Equal(t, want, l.Scope)
	})

	t.Run("Options", func(t *testing.T) {
		r := new(recorder)
		scope := instrumentation.Scope{Name: "name", Version: "ver", SchemaURL: "url"}
		var h *Handler
		assert.NotPanics(t, func() {
			h = NewHandler(
				WithLoggerProvider(r),
				WithInstrumentationScope(scope),
			)
		})
		assert.NotNil(t, h.logger)
		require.IsType(t, &recorder{}, h.logger)

		l := h.logger.(*recorder)
		assert.Equal(t, scope, l.Scope)
	})
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
