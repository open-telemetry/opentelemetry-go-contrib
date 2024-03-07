// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package sloghandler_test

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"testing/slogtest"
	"time"

	"go.opentelemetry.io/contrib/bridges/sloghandler"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/embedded"
	"go.opentelemetry.io/otel/log/noop"
)

type loggerProvider struct {
	embedded.LoggerProvider

	loggerN int
	logger  *logger
}

func (p *loggerProvider) Logger(string, ...log.LoggerOption) log.Logger {
	p.logger = newLogger(p.loggerN)
	return p.logger
}

func (p *loggerProvider) String() string {
	return fmt.Sprintf("&loggerProvider{%s}", p.logger.String())
}

type logger struct {
	embedded.Logger

	Records []log.Record
}

func newLogger(n int) *logger {
	if n == 0 {
		return &logger{}
	}
	return &logger{Records: make([]log.Record, 0, n)}
}

func (l *logger) Emit(_ context.Context, r log.Record) {
	l.Records = append(l.Records, r)
}

func (l *logger) String() string {
	var buf strings.Builder
	_, _ = buf.WriteString("&logger{")
	for _, r := range l.Records {
		_, _ = buf.WriteString("Record{body: ")
		_, _ = buf.WriteString(value2Str(r.Body()))
		_, _ = buf.WriteString(", attr: ")
		r.WalkAttributes(func(kv log.KeyValue) bool {
			_, _ = buf.WriteString(kv.Key)
			_, _ = buf.WriteRune(':')
			_, _ = buf.WriteString(value2Str(kv.Value))
			_, _ = buf.WriteRune(',')
			return true
		})
		_, _ = buf.WriteString("},")
	}
	_, _ = buf.WriteString("}")
	return buf.String()
}

func value2Str(v log.Value) string {
	var buf strings.Builder
	switch v.Kind() {
	case log.KindBool:
		if v.AsBool() {
			_, _ = buf.WriteString("true")
		} else {
			_, _ = buf.WriteString("false")
		}
	case log.KindFloat64:
		_, _ = buf.WriteString(fmt.Sprintf("%g", v.AsFloat64()))
	case log.KindInt64:
		_, _ = buf.WriteString(fmt.Sprintf("%d", v.AsInt64()))
	case log.KindString:
		_, _ = buf.WriteString(v.AsString())
	case log.KindBytes:
		_, _ = buf.Write(v.AsBytes())
	case log.KindSlice:
		_, _ = buf.WriteRune('[')
		for _, s := range v.AsSlice() {
			_, _ = buf.WriteString(value2Str(s))
			_, _ = buf.WriteRune(',')
		}
		_, _ = buf.WriteRune(']')
	case log.KindMap:
		_, _ = buf.WriteRune('{')
		for _, m := range v.AsMap() {
			_, _ = buf.WriteString(m.Key)
			_, _ = buf.WriteRune(':')
			_, _ = buf.WriteString(value2Str(m.Value))
			_, _ = buf.WriteRune(',')
		}
		_, _ = buf.WriteRune('}')
	}
	return buf.String()
}

func value2Result(v log.Value) any {
	switch v.Kind() {
	case log.KindBool, log.KindFloat64, log.KindInt64, log.KindString, log.KindBytes, log.KindSlice:
		return value2Str(v)
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
	lp := new(loggerProvider)

	results := func() []map[string]any {
		out := make([]map[string]any, len(lp.logger.Records))
		for i := range out {
			r := lp.logger.Records[i]

			m := make(map[string]any)
			if tStamp := r.Timestamp(); !tStamp.IsZero() {
				m[slog.TimeKey] = tStamp
			}
			if lvl := r.Severity(); lvl != 0 {
				m[slog.LevelKey] = lvl - 9
			}
			if body := r.Body(); body.Kind() != log.KindEmpty {
				m[slog.MessageKey] = value2Str(body)
			}
			r.WalkAttributes(func(kv log.KeyValue) bool {
				m[kv.Key] = value2Result(kv.Value)
				return true
			})

			out[i] = m
		}
		return out
	}
	// TODO: Use slogtest.Run when we drop support for Go 1.21.
	err := slogtest.TestHandler(sloghandler.New(lp), results)
	if err != nil {
		t.Fatal(err)
	}
}

func BenchmarkHandler(b *testing.B) {
	var (
		h   slog.Handler
		err error
	)

	attrs := []slog.Attr{slog.Any("Key", "Value")}
	record := slog.NewRecord(time.Now(), slog.LevelInfo, "body", 0)
	ctx := context.Background()

	b.Run("Handle", func(b *testing.B) {
		handlers := make([]*sloghandler.Handler, b.N)
		for i := range handlers {
			lp := noop.NewLoggerProvider()
			handlers[i] = sloghandler.New(lp)
		}

		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			err = handlers[n].Handle(ctx, record)
		}
	})

	b.Run("WithAttrs", func(b *testing.B) {
		handlers := make([]*sloghandler.Handler, b.N)
		for i := range handlers {
			lp := noop.NewLoggerProvider()
			handlers[i] = sloghandler.New(lp)
		}

		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			h = handlers[n].WithAttrs(attrs)
		}
	})

	b.Run("WithGroup", func(b *testing.B) {
		handlers := make([]*sloghandler.Handler, b.N)
		for i := range handlers {
			lp := noop.NewLoggerProvider()
			handlers[i] = sloghandler.New(lp)
		}

		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			h = handlers[n].WithGroup("group")
		}
	})

	b.Run("WithGroup.WithAttrs", func(b *testing.B) {
		handlers := make([]*sloghandler.Handler, b.N)
		for i := range handlers {
			lp := noop.NewLoggerProvider()
			handlers[i] = sloghandler.New(lp)
		}

		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			h = handlers[n].WithGroup("group").WithAttrs(attrs)
		}
	})

	b.Run("(WithGroup.WithAttrs).Handle", func(b *testing.B) {
		handlers := make([]slog.Handler, b.N)
		for i := range handlers {
			lp := noop.NewLoggerProvider()
			handlers[i] = sloghandler.New(lp).WithGroup("group").WithAttrs(attrs)
		}

		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			err = handlers[n].Handle(ctx, record)
		}
	})

	_, _ = h, err
}
