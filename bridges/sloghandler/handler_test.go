// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Copyright 2023 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sloghandler_test

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/contrib/bridges/sloghandler"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/embedded"
	"go.opentelemetry.io/otel/log/noop"
)

var now = time.Now()

// testCase represents a complete setup/run/check of an slog handler to test.
// It is copied from "testing/slogtest" (1.22.1).
type testCase struct {
	// Subtest name.
	name string
	// If non-empty, explanation explains the violated constraint.
	explanation string
	// f executes a single log event using its argument logger.
	// So that mkdescs.sh can generate the right description,
	// the body of f must appear on a single line whose first
	// non-whitespace characters are "l.".
	f func(*slog.Logger)
	// If mod is not nil, it is called to modify the Record
	// generated by the Logger before it is passed to the Handler.
	mod func(*slog.Record)
	// checks is a list of checks to run on the result. Each item is a slice of
	// checks that will be evaluated for the corresponding record emitted.
	checks [][]check
}

var cases = []testCase{
	// Test cases copied from "testing/slogtest" (1.22.1).
	// #######################################################################

	{
		name:        "built-ins",
		explanation: withSource("this test expects slog.TimeKey, slog.LevelKey and slog.MessageKey"),
		f: func(l *slog.Logger) {
			l.Info("message")
		},
		checks: [][]check{{
			hasKey(slog.TimeKey),
			hasKey(slog.LevelKey),
			hasAttr(slog.MessageKey, "message"),
		}},
	},
	{
		name:        "attrs",
		explanation: withSource("a Handler should output attributes passed to the logging function"),
		f: func(l *slog.Logger) {
			l.Info("message", "k", "v")
		},
		checks: [][]check{{
			hasAttr("k", "v"),
		}},
	},
	{
		name:        "empty-attr",
		explanation: withSource("a Handler should ignore an empty Attr"),
		f: func(l *slog.Logger) {
			l.Info("msg", "a", "b", "", nil, "c", "d")
		},
		checks: [][]check{{
			hasAttr("a", "b"),
			missingKey(""),
			hasAttr("c", "d"),
		}},
	},
	{
		name:        "zero-time",
		explanation: withSource("a Handler should ignore a zero Record.Time"),
		f: func(l *slog.Logger) {
			l.Info("msg", "k", "v")
		},
		mod: func(r *slog.Record) { r.Time = time.Time{} },
		checks: [][]check{{
			missingKey(slog.TimeKey),
		}},
	},
	{
		name:        "WithAttrs",
		explanation: withSource("a Handler should include the attributes from the WithAttrs method"),
		f: func(l *slog.Logger) {
			l.With("a", "b").Info("msg", "k", "v")
		},
		checks: [][]check{{
			hasAttr("a", "b"),
			hasAttr("k", "v"),
		}},
	},
	{
		name:        "groups",
		explanation: withSource("a Handler should handle Group attributes"),
		f: func(l *slog.Logger) {
			l.Info("msg", "a", "b", slog.Group("G", slog.String("c", "d")), "e", "f")
		},
		checks: [][]check{{
			hasAttr("a", "b"),
			inGroup("G", hasAttr("c", "d")),
			hasAttr("e", "f"),
		}},
	},
	{
		name:        "empty-group",
		explanation: withSource("a Handler should ignore an empty group"),
		f: func(l *slog.Logger) {
			l.Info("msg", "a", "b", slog.Group("G"), "e", "f")
		},
		checks: [][]check{{
			hasAttr("a", "b"),
			missingKey("G"),
			hasAttr("e", "f"),
		}},
	},
	{
		name:        "inline-group",
		explanation: withSource("a Handler should inline the Attrs of a group with an empty key"),
		f: func(l *slog.Logger) {
			l.Info("msg", "a", "b", slog.Group("", slog.String("c", "d")), "e", "f")
		},
		checks: [][]check{{
			hasAttr("a", "b"),
			hasAttr("c", "d"),
			hasAttr("e", "f"),
		}},
	},
	{
		name:        "WithGroup",
		explanation: withSource("a Handler should handle the WithGroup method"),
		f: func(l *slog.Logger) {
			l.WithGroup("G").Info("msg", "a", "b")
		},
		checks: [][]check{{
			hasKey(slog.TimeKey),
			hasKey(slog.LevelKey),
			hasAttr(slog.MessageKey, "msg"),
			missingKey("a"),
			inGroup("G", hasAttr("a", "b")),
		}},
	},
	{
		name:        "multi-With",
		explanation: withSource("a Handler should handle multiple WithGroup and WithAttr calls"),
		f: func(l *slog.Logger) {
			l.With("a", "b").WithGroup("G").With("c", "d").WithGroup("H").Info("msg", "e", "f")
		},
		checks: [][]check{{
			hasKey(slog.TimeKey),
			hasKey(slog.LevelKey),
			hasAttr(slog.MessageKey, "msg"),
			hasAttr("a", "b"),
			inGroup("G", hasAttr("c", "d")),
			inGroup("G", inGroup("H", hasAttr("e", "f"))),
		}},
	},
	{
		name:        "empty-group-record",
		explanation: withSource("a Handler should not output groups if there are no attributes"),
		f: func(l *slog.Logger) {
			l.With("a", "b").WithGroup("G").With("c", "d").WithGroup("H").Info("msg")
		},
		checks: [][]check{{
			hasKey(slog.TimeKey),
			hasKey(slog.LevelKey),
			hasAttr(slog.MessageKey, "msg"),
			hasAttr("a", "b"),
			inGroup("G", hasAttr("c", "d")),
			inGroup("G", missingKey("H")),
		}},
	},
	{
		name:        "resolve",
		explanation: withSource("a Handler should call Resolve on attribute values"),
		f: func(l *slog.Logger) {
			l.Info("msg", "k", &replace{"replaced"})
		},
		checks: [][]check{{hasAttr("k", "replaced")}},
	},
	{
		name:        "resolve-groups",
		explanation: withSource("a Handler should call Resolve on attribute values in groups"),
		f: func(l *slog.Logger) {
			l.Info("msg",
				slog.Group("G",
					slog.String("a", "v1"),
					slog.Any("b", &replace{"v2"})))
		},
		checks: [][]check{{
			inGroup("G", hasAttr("a", "v1")),
			inGroup("G", hasAttr("b", "v2")),
		}},
	},
	{
		name:        "resolve-WithAttrs",
		explanation: withSource("a Handler should call Resolve on attribute values from WithAttrs"),
		f: func(l *slog.Logger) {
			l = l.With("k", &replace{"replaced"})
			l.Info("msg")
		},
		checks: [][]check{{hasAttr("k", "replaced")}},
	},
	{
		name:        "resolve-WithAttrs-groups",
		explanation: withSource("a Handler should call Resolve on attribute values in groups from WithAttrs"),
		f: func(l *slog.Logger) {
			l = l.With(slog.Group("G",
				slog.String("a", "v1"),
				slog.Any("b", &replace{"v2"})))
			l.Info("msg")
		},
		checks: [][]check{{
			inGroup("G", hasAttr("a", "v1")),
			inGroup("G", hasAttr("b", "v2")),
		}},
	},
	{
		name:        "empty-PC",
		explanation: withSource("a Handler should not output SourceKey if the PC is zero"),
		f: func(l *slog.Logger) {
			l.Info("message")
		},
		mod: func(r *slog.Record) { r.PC = 0 },
		checks: [][]check{{
			missingKey(slog.SourceKey),
		}},
	},

	// #######################################################################

	// OTel specific test cases.
	// #######################################################################

	{
		name:        "Values",
		explanation: withSource("all slog Values need to be supported"),
		f: func(l *slog.Logger) {
			l.Info(
				"msg",
				"any", struct{ data int64 }{data: 1},
				"bool", true,
				"duration", time.Minute,
				"float64", 3.14159,
				"int64", -2,
				"string", "str",
				"time", now,
				"uint64", uint64(3),
				// KindGroup and KindLogValuer are left for tests above.
			)
		},
		checks: [][]check{{
			hasKey(slog.TimeKey),
			hasKey(slog.LevelKey),
			hasAttr("any", "{data:1}"),
			hasAttr("bool", true),
			hasAttr("duration", int64(time.Minute)),
			hasAttr("float64", 3.14159),
			hasAttr("int64", int64(-2)),
			hasAttr("string", "str"),
			hasAttr("time", now.UnixNano()),
			hasAttr("uint64", int64(3)),
		}},
	},
	{
		name:        "multi-messages",
		explanation: withSource("this test expects multiple independent messages"),
		f: func(l *slog.Logger) {
			l.Info("one")
			l.Info("two")
		},
		checks: [][]check{{
			hasKey(slog.TimeKey),
			hasKey(slog.LevelKey),
			hasAttr(slog.MessageKey, "one"),
		}, {
			hasKey(slog.TimeKey),
			hasKey(slog.LevelKey),
			hasAttr(slog.MessageKey, "two"),
		}},
	},
	{
		name:        "multi-attrs",
		explanation: withSource("attributes from one message do not affect another"),
		f: func(l *slog.Logger) {
			l.Info("one", "k", "v")
			l.Info("two")
		},
		checks: [][]check{{
			hasAttr("k", "v"),
		}, {
			missingKey("k"),
		}},
	},
	{
		name:        "independent-WithAttrs",
		explanation: withSource("a Handler should only include attributes from its own WithAttr origin"),
		f: func(l *slog.Logger) {
			l1 := l.With("a", "b")
			l2 := l1.With("c", "d")

			l2.Info("msg", "k", "v")
			l1.Info("msg", "k", "v")
			l.Info("msg", "k", "v")
		},
		checks: [][]check{{
			hasAttr("a", "b"),
			hasAttr("c", "d"),
			hasAttr("k", "v"),
		}, {
			hasAttr("a", "b"),
			hasAttr("k", "v"),
		}, {
			hasAttr("k", "v"),
		}},
	},
	{
		name:        "independent-WithGroup",
		explanation: withSource("a Handler should only include attributes from its own WithGroup origin"),
		f: func(l *slog.Logger) {
			l1 := l.WithGroup("G").With("a", "b")
			l2 := l1.WithGroup("H").With("c", "d")

			l2.Info("msg", "k", "v")
			l1.Info("msg", "k", "v")
			l.Info("msg", "k", "v")
		},
		checks: [][]check{{
			hasKey(slog.TimeKey),
			hasKey(slog.LevelKey),
			hasAttr(slog.MessageKey, "msg"),
			missingKey("a"),
			missingKey("c"),
			inGroup("G", hasAttr("a", "b")),
			inGroup("G", inGroup("H", hasAttr("c", "d"))),
			inGroup("G", inGroup("H", hasAttr("k", "v"))),
		}, {
			hasKey(slog.TimeKey),
			hasKey(slog.LevelKey),
			hasAttr(slog.MessageKey, "msg"),
			missingKey("a"),
			missingKey("c"),
			missingKey("H"),
			inGroup("G", hasAttr("a", "b")),
			inGroup("G", hasAttr("k", "v")),
		}, {
			hasKey(slog.TimeKey),
			hasKey(slog.LevelKey),
			hasAttr("k", "v"),
			hasAttr(slog.MessageKey, "msg"),
			missingKey("a"),
			missingKey("c"),
			missingKey("G"),
			missingKey("H"),
		}},
	},

	// #######################################################################
}

func TestSLogHandler(t *testing.T) {
	// Based on slogtest.Run.
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := new(recorder)
			var h slog.Handler = sloghandler.New(r)
			if c.mod != nil {
				h = &wrapper{h, c.mod}
			}
			l := slog.New(h)
			c.f(l)
			got := r.Results()
			if len(got) != len(c.checks) {
				t.Fatalf("missing record checks: %d records, %d checks", len(got), len(c.checks))
			}
			for i, checks := range c.checks {
				for _, check := range checks {
					if p := check(got[i]); p != "" {
						t.Errorf("%s: %s", p, c.explanation)
					}
				}
			}
		})
	}
}

type check func(map[string]any) string

func hasKey(key string) check {
	return func(m map[string]any) string {
		if _, ok := m[key]; !ok {
			return fmt.Sprintf("missing key %q", key)
		}
		return ""
	}
}

func missingKey(key string) check {
	return func(m map[string]any) string {
		if _, ok := m[key]; ok {
			return fmt.Sprintf("unexpected key %q", key)
		}
		return ""
	}
}

func hasAttr(key string, wantVal any) check {
	return func(m map[string]any) string {
		if s := hasKey(key)(m); s != "" {
			return s
		}
		gotVal := m[key]
		if !reflect.DeepEqual(gotVal, wantVal) {
			return fmt.Sprintf("%q: got %#v, want %#v", key, gotVal, wantVal)
		}
		return ""
	}
}

func inGroup(name string, c check) check {
	return func(m map[string]any) string {
		v, ok := m[name]
		if !ok {
			return fmt.Sprintf("missing group %q", name)
		}
		g, ok := v.(map[string]any)
		if !ok {
			return fmt.Sprintf("value for group %q is not map[string]any", name)
		}
		return c(g)
	}
}

func withSource(s string) string {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		panic("runtime.Caller failed")
	}
	return fmt.Sprintf("%s (%s:%d)", s, file, line)
}

type wrapper struct {
	slog.Handler
	mod func(*slog.Record)
}

func (h *wrapper) Handle(ctx context.Context, r slog.Record) error {
	h.mod(&r)
	return h.Handler.Handle(ctx, r)
}

type replace struct {
	v any
}

func (r *replace) LogValue() slog.Value { return slog.AnyValue(r.v) }

func (r *replace) String() string {
	return fmt.Sprintf("<replace(%v)>", r.v)
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
