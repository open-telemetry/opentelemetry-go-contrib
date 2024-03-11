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

func TestSLogHandler(t *testing.T) {
	r := new(recorder)

	// TODO: Use slogtest.Run when we drop support for Go 1.21.
	err := slogtest.TestHandler(sloghandler.New(r), r.Results)
	if err != nil {
		t.Fatal(err)
	}
}

// embeddedLogger is a type alias so the embedded.Logger type doesn't conflict
// with the Logger method of the recorder when it is embedded.
type embeddedLogger = embedded.Logger

// recorder records all [log.Record]s it is ased to emit.
type recorder struct {
	embedded.LoggerProvider
	embeddedLogger

	Records []log.Record
}

func (r *recorder) Logger(string, ...log.LoggerOption) log.Logger { return r }

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
		if data := v.AsSlice(); len(data) > 0 {
			_, _ = buf.WriteString(value2Str(data[0]))
			for _, s := range data[1:] {
				_, _ = buf.WriteRune(',')
				_, _ = buf.WriteString(value2Str(s))
			}
		}
		_, _ = buf.WriteRune(']')
	case log.KindMap:
		_, _ = buf.WriteRune('{')
		if data := v.AsMap(); len(data) > 0 {
			_, _ = buf.WriteString(data[0].Key)
			_, _ = buf.WriteRune(':')
			_, _ = buf.WriteString(value2Str(data[0].Value))
			for _, m := range data[1:] {
				_, _ = buf.WriteRune(',')
				_, _ = buf.WriteString(m.Key)
				_, _ = buf.WriteRune(':')
				_, _ = buf.WriteString(value2Str(m.Value))
			}
		}
		_, _ = buf.WriteRune('}')
	}
	return buf.String()
}

func value2Result(v log.Value) any {
	switch v.Kind() {
	case log.KindBool,
		log.KindFloat64,
		log.KindInt64,
		log.KindString,
		log.KindBytes,
		log.KindSlice:
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
