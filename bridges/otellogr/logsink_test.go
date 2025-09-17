// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otellogr

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/embedded"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/log/logtest"
)

type mockLoggerProvider struct {
	embedded.LoggerProvider
}

func (mockLoggerProvider) Logger(string, ...log.LoggerOption) log.Logger {
	return nil
}

func TestNewConfig(t *testing.T) {
	customLoggerProvider := mockLoggerProvider{}

	for _, tt := range []struct {
		name    string
		options []Option

		wantConfig config
	}{
		{
			name: "with no options",

			wantConfig: config{
				provider: global.GetLoggerProvider(),
			},
		},
		{
			name: "with a custom instrumentation scope",
			options: []Option{
				WithVersion("42.0"),
			},

			wantConfig: config{
				version:  "42.0",
				provider: global.GetLoggerProvider(),
			},
		},
		{
			name: "with a custom logger provider",
			options: []Option{
				WithLoggerProvider(customLoggerProvider),
			},

			wantConfig: config{
				provider: customLoggerProvider,
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			config := newConfig(tt.options)
			config.levelSeverity = nil // Ignore asserting level severity function, assert.Equal does not support function comparison
			assert.Equal(t, tt.wantConfig, config)
		})
	}
}

func TestNewLogSink(t *testing.T) {
	const name = "name"

	for _, tt := range []struct {
		name    string
		options []Option
		want    logtest.Recording
	}{
		{
			name: "with default options",
			want: logtest.Recording{
				logtest.Scope{Name: name}: nil,
			},
		},
		{
			name: "with custom options",
			options: []Option{
				WithVersion("1.0"),
				WithSchemaURL("https://example.com"),
				WithAttributes(attribute.String("testattr", "testval")),
			},
			want: logtest.Recording{
				logtest.Scope{
					Name:       name,
					Version:    "1.0",
					SchemaURL:  "https://example.com",
					Attributes: attribute.NewSet(attribute.String("testattr", "testval")),
				}: nil,
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rec := logtest.NewRecorder()

			NewLogSink(name, append(
				tt.options,
				WithLoggerProvider(rec),
			)...)

			logtest.AssertEqual(t, tt.want, rec.Result())
		})
	}
}

func TestLogSink(t *testing.T) {
	const name = "name"

	for _, tt := range []struct {
		name         string
		f            func(*logr.Logger)
		wantSeverity func(int) log.Severity
		want         logtest.Recording
	}{
		{
			name: "no_log",
			f:    func(*logr.Logger) {},
			want: logtest.Recording{
				logtest.Scope{Name: name}: nil,
			},
		},
		{
			name: "info",
			f: func(l *logr.Logger) {
				l.Info("msg")
			},
			want: logtest.Recording{
				logtest.Scope{Name: name}: {
					{Body: log.StringValue("msg"), Severity: log.SeverityInfo},
				},
			},
		},
		{
			name: "info_with_level_severity",
			f: func(l *logr.Logger) {
				l.V(0).Info("msg")
				l.V(1).Info("msg")
				l.V(2).Info("msg")
				l.V(3).Info("msg")
			},
			want: logtest.Recording{
				logtest.Scope{Name: name}: {
					{Body: log.StringValue("msg"), Severity: log.SeverityInfo},
					{Body: log.StringValue("msg"), Severity: log.SeverityDebug},
					{Body: log.StringValue("msg"), Severity: log.SeverityTrace},
					{Body: log.StringValue("msg"), Severity: log.SeverityTrace},
				},
			},
		},
		{
			name: "info_with_custom_level_severity",
			f: func(l *logr.Logger) {
				l.Info("msg")
				l.V(1).Info("msg")
				l.V(2).Info("msg")
			},
			wantSeverity: func(level int) log.Severity {
				switch level {
				case 1:
					return log.SeverityError
				case 2:
					return log.SeverityWarn
				default:
					return log.SeverityInfo
				}
			},
			want: logtest.Recording{
				logtest.Scope{Name: name}: {
					{Body: log.StringValue("msg"), Severity: log.SeverityInfo},
					{Body: log.StringValue("msg"), Severity: log.SeverityError},
					{Body: log.StringValue("msg"), Severity: log.SeverityWarn},
				},
			},
		},
		{
			name: "info_multi_attrs",
			f: func(l *logr.Logger) {
				l.Info("msg",
					"struct", struct{ data int64 }{data: 1},
					"bool", true,
					"duration", time.Minute,
					"float64", 3.14159,
					"int64", -2,
					"string", "str",
					"time", time.Unix(1000, 1000),
					"uint64", uint64(3),
					"log-attribute", log.MapValue(log.String("foo", "bar")),
					"standard-attribute", attribute.StringSliceValue([]string{"one", "two"}),
				)
			},
			want: logtest.Recording{
				logtest.Scope{Name: name}: {
					{
						Body:     log.StringValue("msg"),
						Severity: log.SeverityInfo,
						Attributes: []log.KeyValue{
							log.String("struct", "{data:1}"),
							log.Bool("bool", true),
							log.Int64("duration", 60_000_000_000),
							log.Float64("float64", 3.14159),
							log.Int64("int64", -2),
							log.String("string", "str"),
							log.Int64("time", time.Unix(1000, 1000).UnixNano()),
							log.Int64("uint64", 3),
							log.Map("log-attribute", log.String("foo", "bar")),
							log.Slice("standard-attribute", log.StringValue("one"), log.StringValue("two")),
						},
					},
				},
			},
		},
		{
			name: "info_with_name",
			f: func(l *logr.Logger) {
				l.WithName("test").Info("info message with name")
			},
			want: logtest.Recording{
				logtest.Scope{Name: name}: nil,
				logtest.Scope{Name: name + "/test"}: {
					{Body: log.StringValue("info message with name"), Severity: log.SeverityInfo},
				},
			},
		},
		{
			name: "info_with_name_nested",
			f: func(l *logr.Logger) {
				l.WithName("test").WithName("test").Info("info message with name")
			},
			want: logtest.Recording{
				logtest.Scope{Name: name}:           nil,
				logtest.Scope{Name: name + "/test"}: nil,
				logtest.Scope{Name: name + "/test/test"}: {
					{Body: log.StringValue("info message with name"), Severity: log.SeverityInfo},
				},
			},
		},
		{
			name: "info_with_attrs",
			f: func(l *logr.Logger) {
				l.WithValues("key", "value").Info("info message with attrs")
			},
			want: logtest.Recording{
				logtest.Scope{Name: name}: {
					{
						Body:     log.StringValue("info message with attrs"),
						Severity: log.SeverityInfo,
						Attributes: []log.KeyValue{
							log.String("key", "value"),
						},
					},
				},
			},
		},
		{
			name: "info_with_attrs_nested",
			f: func(l *logr.Logger) {
				l.WithValues("key1", "value1").Info("info message with attrs", "key2", "value2")
			},
			want: logtest.Recording{
				logtest.Scope{Name: name}: {
					{
						Body:     log.StringValue("info message with attrs"),
						Severity: log.SeverityInfo,
						Attributes: []log.KeyValue{
							log.String("key1", "value1"),
							log.String("key2", "value2"),
						},
					},
				},
			},
		},
		{
			name: "info_with_normal_attr_and_nil_pointer_attr",
			f: func(l *logr.Logger) {
				var p *int
				l.WithValues("key", "value", "nil_pointer", p).Info("info message with attrs")
			},
			want: logtest.Recording{
				logtest.Scope{Name: name}: {
					{
						Body:     log.StringValue("info message with attrs"),
						Severity: log.SeverityInfo,
						Attributes: []log.KeyValue{
							log.String("key", "value"),
							log.Empty("nil_pointer"),
						},
					},
				},
			},
		},
		{
			name: "error",
			f: func(l *logr.Logger) {
				l.Error(errors.New("test"), "error message")
			},
			want: logtest.Recording{
				logtest.Scope{Name: name}: []logtest.Record{
					{
						Body:     log.StringValue("error message"),
						Severity: log.SeverityError,
						Attributes: []log.KeyValue{
							log.String("exception.message", "test"),
						},
					},
				},
			},
		},
		{
			name: "error_multi_attrs",
			f: func(l *logr.Logger) {
				l.Error(errors.New("test error"), "msg",
					"struct", struct{ data int64 }{data: 1},
					"bool", true,
					"duration", time.Minute,
					"float64", 3.14159,
					"int64", -2,
					"string", "str",
					"time", time.Unix(1000, 1000),
					"uint64", uint64(3),
				)
			},
			want: logtest.Recording{
				logtest.Scope{Name: name}: []logtest.Record{
					{
						Body:     log.StringValue("msg"),
						Severity: log.SeverityError,
						Attributes: []log.KeyValue{
							{Key: "exception.message", Value: log.StringValue("test error")},
							log.String("struct", "{data:1}"),
							log.Bool("bool", true),
							log.Int64("duration", 60_000_000_000),
							log.Float64("float64", 3.14159),
							log.Int64("int64", -2),
							log.String("string", "str"),
							log.Int64("time", time.Unix(1000, 1000).UnixNano()),
							log.Int64("uint64", 3),
						},
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rec := logtest.NewRecorder()
			ls := NewLogSink(name,
				WithLoggerProvider(rec),
				WithLevelSeverity(tt.wantSeverity),
			)
			l := logr.New(ls)
			tt.f(&l)

			logtest.AssertEqual(t, tt.want, rec.Result(),
				logtest.Transform(func(r logtest.Record) logtest.Record {
					r.Context = nil // Ignore context for comparison.
					return r
				}),
			)
		})
	}
}

func TestLogSinkContext(t *testing.T) {
	name := "name"
	ctx := context.WithValue(t.Context(), "key", "value") //nolint:revive,staticcheck // test context

	tests := []struct {
		name string
		f    func(*logr.Logger)
		want logtest.Recording
	}{
		{
			name: "default",
			f: func(l *logr.Logger) {
				l.Info("msg")
			},
			want: logtest.Recording{
				logtest.Scope{Name: name}: {
					//nolint:usetesting // This place was originally intended to test the default context.
					{Context: context.Background()},
				},
			},
		},
		{
			name: "context in KeyAndValues",
			f: func(l *logr.Logger) {
				l.WithValues("ctx", ctx).Info("msg")
			},
			want: logtest.Recording{
				logtest.Scope{Name: name}: {
					{Context: ctx},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := logtest.NewRecorder()
			ls := NewLogSink(name, WithLoggerProvider(rec))
			l := logr.New(ls)
			tt.f(&l)

			logtest.AssertEqual(t, tt.want, rec.Result(),
				logtest.Transform(func(r logtest.Record) logtest.Record {
					// Only compare the context, ignore the rest.
					return logtest.Record{
						Context: r.Context,
					}
				}),
			)
		})
	}
}

func TestLogSinkEnabled(t *testing.T) {
	enabledFunc := func(_ context.Context, param log.EnabledParameters) bool {
		return param.Severity == log.SeverityInfo
	}

	rec := logtest.NewRecorder(logtest.WithEnabledFunc(enabledFunc))
	ls := NewLogSink(
		"name",
		WithLoggerProvider(rec),
		WithLevelSeverity(func(i int) log.Severity {
			switch i {
			case 0:
				return log.SeverityInfo
			default:
				return log.SeverityDebug
			}
		}),
	)

	assert.True(t, ls.Enabled(0))
	assert.False(t, ls.Enabled(1))
}

func TestConvertKVs(t *testing.T) {
	ctx := context.WithValue(t.Context(), "key", "value") //nolint:revive,staticcheck // test context

	for _, tt := range []struct {
		name    string
		kvs     []any
		wantKVs []log.KeyValue
		wantCtx context.Context
	}{
		{
			name: "empty",
			kvs:  []any{},
		},
		{
			name: "single_value",
			kvs:  []any{"key", "value"},
			wantKVs: []log.KeyValue{
				log.String("key", "value"),
			},
		},
		{
			name: "multiple_values",
			kvs:  []any{"key1", "value1", "key2", "value2"},
			wantKVs: []log.KeyValue{
				log.String("key1", "value1"),
				log.String("key2", "value2"),
			},
		},
		{
			name: "missing_value",
			kvs:  []any{"key1", "value1", "key2"},
			wantKVs: []log.KeyValue{
				log.String("key1", "value1"),
				{Key: "key2", Value: log.Value{}},
			},
		},
		{
			name: "key_not_string",
			kvs:  []any{42, "value"},
			wantKVs: []log.KeyValue{
				log.String("42", "value"),
			},
		},
		{
			name:    "context",
			kvs:     []any{"ctx", ctx, "key", "value"},
			wantKVs: []log.KeyValue{log.String("key", "value")},
			wantCtx: ctx,
		},
		{
			name:    "last_context",
			kvs:     []any{"key", t.Context(), "ctx", ctx},
			wantKVs: []log.KeyValue{},
			wantCtx: ctx,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, kvs := convertKVs(nil, tt.kvs...) //nolint:staticcheck // pass nil context
			assert.Equal(t, tt.wantKVs, kvs)
			assert.Equal(t, tt.wantCtx, ctx)
		})
	}
}
