// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelslog

import (
	"context"
	"log/slog"
	"testing"
	"testing/slogtest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/embedded"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/sdk/instrumentation"
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

	// Records are the records emitted.
	Records []log.Record

	// Scope is the Logger scope recorder received when Logger was called.
	Scope instrumentation.Scope

	// MinSeverity is the minimum severity the recorder will return true for
	// when Enabled is called (unless enableKey is set).
	MinSeverity log.Severity
}

func (r *recorder) Logger(name string, opts ...log.LoggerOption) log.Logger {
	cfg := log.NewLoggerConfig(opts...)

	r.Scope = instrumentation.Scope{
		Name:      name,
		Version:   cfg.InstrumentationVersion(),
		SchemaURL: cfg.SchemaURL(),
	}
	return r
}

type enablerKey uint

var enableKey enablerKey

func (r *recorder) Enabled(ctx context.Context, record log.Record) bool {
	return ctx.Value(enableKey) != nil || record.Severity() >= r.MinSeverity
}

func (r *recorder) Emit(_ context.Context, record log.Record) {
	r.Records = append(r.Records, record)
}

func (r *recorder) Results() []map[string]any {
	out := make([]map[string]any, len(r.Records))
	for i := range out {
		r := r.Records[i]

		m := make(map[string]any)
		if tStamp := r.Timestamp(); !tStamp.IsZero() {
			m[slog.TimeKey] = tStamp
		}
		if lvl := r.Severity(); lvl != 0 {
			m[slog.LevelKey] = lvl - 9
		}
		if body := r.Body(); body.Kind() != log.KindEmpty {
			m[slog.MessageKey] = value2Result(body)
		}
		r.WalkAttributes(func(kv log.KeyValue) bool {
			m[kv.Key] = value2Result(kv.Value)
			return true
		})

		out[i] = m
	}
	return out
}

func value2Result(v log.Value) any {
	switch v.Kind() {
	case log.KindBool:
		return v.AsBool()
	case log.KindFloat64:
		return v.AsFloat64()
	case log.KindInt64:
		return v.AsInt64()
	case log.KindString:
		return v.AsString()
	case log.KindBytes:
		return v.AsBytes()
	case log.KindSlice:
		return v.AsSlice()
	case log.KindMap:
		m := make(map[string]any)
		for _, val := range v.AsMap() {
			m[val.Key] = value2Result(val.Value)
		}
		return m
	}
	return nil
}

func TestSLogHandler(t *testing.T) {
	t.Run("slogtest.TestHandler", func(t *testing.T) {
		r := new(recorder)
		h := NewHandler(WithLoggerProvider(r))

		// TODO: use slogtest.Run when Go 1.21 is no longer supported.
		err := slogtest.TestHandler(h, r.Results)
		if err != nil {
			t.Fatal(err)
		}
	})

	// TODO: Add multi-logged testing. See #5195.
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

	ctx := context.Background()
	assert.False(t, h.Enabled(ctx, slog.LevelDebug), "level conversion: permissive")
	assert.True(t, h.Enabled(ctx, slog.LevelInfo), "level conversion: restrictive")

	ctx = context.WithValue(ctx, enableKey, true)
	assert.True(t, h.Enabled(ctx, slog.LevelDebug), "context not passed")
}
