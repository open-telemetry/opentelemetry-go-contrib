// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelslog

import (
	"context"
	"log/slog"
	"testing"
	"testing/slogtest"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/embedded"
)

// embeddedLogger is a type alias so the embedded.Logger type doesn't conflict
// with the Logger method of the recorder when it is embedded.
type embeddedLogger = embedded.Logger // nolint:unused  // Used below.

// recorder records all [log.Record]s it is ased to emit.
type recorder struct {
	embedded.LoggerProvider
	embeddedLogger // nolint:unused  // Used to embed embedded.Logger.

	Records []log.Record
	// MinSeverity is the minimum severity the recorder will return true for
	// when Enabled is called (unless enableKey is set).
	MinSeverity log.Severity
}

func (r *recorder) Logger(string, ...log.LoggerOption) log.Logger { return r }

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
		h.logger = r.Logger("") // TODO: Remove when #5311 merged.

		// TODO: use slogtest.Run when Go 1.21 is no longer supported.
		err := slogtest.TestHandler(h, r.Results)
		if err != nil {
			t.Fatal(err)
		}
	})

	// TODO: Add multi-logged testing. See #5195.
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
