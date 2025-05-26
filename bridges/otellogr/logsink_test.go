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
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/embedded"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/log/logtest"
)

type mockLoggerProvider struct {
	embedded.LoggerProvider
}

func (mockLoggerProvider) Logger(name string, options ...log.LoggerOption) log.Logger {
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
		name          string
		options       []Option
		wantRecording logtest.Recording
	}{
		{
			name: "with default options",
			wantRecording: logtest.Recording{
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
			wantRecording: logtest.Recording{
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
			recorder := logtest.NewRecorder()

			var l *LogSink
			require.NotPanics(t, func() {
				l = NewLogSink(name, append(
					tt.options,
					WithLoggerProvider(recorder),
				)...)
			})
			require.NotNil(t, l)
			require.Len(t, recorder.Result(), 1)

			logtest.AssertEqual(t, tt.wantRecording, recorder.Result())
		})
	}
}

func TestLogSink(t *testing.T) {
	const name = "name"

	for _, tt := range []struct {
		name          string
		f             func(*logr.Logger)
		levelSeverity func(int) log.Severity
		wantRecording logtest.Recording
	}{
		{
			name: "no_log",
			f:    func(l *logr.Logger) {},
			wantRecording: logtest.Recording{
				logtest.Scope{Name: name}: []logtest.Record{},
			},
		},
		{
			name: "info",
			f: func(l *logr.Logger) {
				l.Info("msg")
			},
			wantRecording: logtest.Recording{
				logtest.Scope{Name: name}: []logtest.Record{
					buildRecord(log.StringValue("msg"), log.SeverityInfo, nil),
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
			wantRecording: logtest.Recording{
				logtest.Scope{Name: name}: []logtest.Record{
					buildRecord(log.StringValue("msg"), log.SeverityInfo, nil),
					buildRecord(log.StringValue("msg"), log.SeverityDebug, nil),
					buildRecord(log.StringValue("msg"), log.SeverityTrace, nil),
					buildRecord(log.StringValue("msg"), log.SeverityTrace, nil),
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
			levelSeverity: func(level int) log.Severity {
				switch level {
				case 1:
					return log.SeverityError
				case 2:
					return log.SeverityWarn
				default:
					return log.SeverityInfo
				}
			},
			wantRecording: logtest.Recording{
				logtest.Scope{Name: name}: []logtest.Record{
					buildRecord(log.StringValue("msg"), log.SeverityInfo, nil),
					buildRecord(log.StringValue("msg"), log.SeverityError, nil),
					buildRecord(log.StringValue("msg"), log.SeverityWarn, nil),
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
				)
			},
			wantRecording: logtest.Recording{
				logtest.Scope{Name: name}: []logtest.Record{
					buildRecord(log.StringValue("msg"), log.SeverityInfo, []log.KeyValue{
						log.String("struct", "{data:1}"),
						log.Bool("bool", true),
						log.Int64("duration", 60_000_000_000),
						log.Float64("float64", 3.14159),
						log.Int64("int64", -2),
						log.String("string", "str"),
						log.Int64("time", time.Unix(1000, 1000).UnixNano()),
						log.Int64("uint64", 3),
					}),
				},
			},
		},
		{
			name: "info_with_name",
			f: func(l *logr.Logger) {
				l.WithName("test").Info("info message with name")
			},
			wantRecording: logtest.Recording{
				logtest.Scope{Name: name}: nil,
				logtest.Scope{Name: name + "/test"}: []logtest.Record{
					buildRecord(log.StringValue("info message with name"), log.SeverityInfo, nil),
				},
			},
		},
		{
			name: "info_with_name_nested",
			f: func(l *logr.Logger) {
				l.WithName("test").WithName("test").Info("info message with name")
			},
			wantRecording: logtest.Recording{
				logtest.Scope{Name: name}:           nil,
				logtest.Scope{Name: name + "/test"}: nil,
				logtest.Scope{Name: name + "/test/test"}: []logtest.Record{
					buildRecord(log.StringValue("info message with name"), log.SeverityInfo, nil),
				},
			},
		},
		{
			name: "info_with_attrs",
			f: func(l *logr.Logger) {
				l.WithValues("key", "value").Info("info message with attrs")
			},
			wantRecording: logtest.Recording{
				logtest.Scope{Name: name}: []logtest.Record{
					buildRecord(log.StringValue("info message with attrs"), log.SeverityInfo, []log.KeyValue{
						log.String("key", "value"),
					}),
				},
			},
		},
		{
			name: "info_with_attrs_nested",
			f: func(l *logr.Logger) {
				l.WithValues("key1", "value1").Info("info message with attrs", "key2", "value2")
			},
			wantRecording: logtest.Recording{
				logtest.Scope{Name: name}: []logtest.Record{
					buildRecord(log.StringValue("info message with attrs"), log.SeverityInfo, []log.KeyValue{
						log.String("key1", "value1"),
						log.String("key2", "value2"),
					}),
				},
			},
		},
		{
			name: "info_with_normal_attr_and_nil_pointer_attr",
			f: func(l *logr.Logger) {
				var p *int
				l.WithValues("key", "value", "nil_pointer", p).Info("info message with attrs")
			},
			wantRecording: logtest.Recording{
				logtest.Scope{Name: name}: []logtest.Record{
					buildRecord(log.StringValue("info message with attrs"), log.SeverityInfo, []log.KeyValue{
						log.String("key", "value"),
						log.Empty("nil_pointer"),
					}),
				},
			},
		},
		{
			name: "error",
			f: func(l *logr.Logger) {
				l.Error(errors.New("test"), "error message")
			},
			wantRecording: logtest.Recording{
				logtest.Scope{Name: name}: []logtest.Record{
					buildRecord(log.StringValue("error message"), log.SeverityError, []log.KeyValue{
						{Key: "exception.message", Value: log.StringValue("test")},
					}),
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
			wantRecording: logtest.Recording{
				logtest.Scope{Name: name}: []logtest.Record{
					buildRecord(log.StringValue("msg"), log.SeverityError, []log.KeyValue{
						{Key: "exception.message", Value: log.StringValue("test error")},
						log.String("struct", "{data:1}"),
						log.Bool("bool", true),
						log.Int64("duration", 60_000_000_000),
						log.Float64("float64", 3.14159),
						log.Int64("int64", -2),
						log.String("string", "str"),
						log.Int64("time", time.Unix(1000, 1000).UnixNano()),
						log.Int64("uint64", 3),
					}),
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rec := logtest.NewRecorder()
			ls := NewLogSink(name,
				WithLoggerProvider(rec),
				WithLevelSeverity(tt.levelSeverity),
			)
			l := logr.New(ls)
			tt.f(&l)

			logtest.AssertEqual(t, tt.wantRecording, rec.Result())
		})
	}
}

// fix https://github.com/open-telemetry/opentelemetry-go-contrib/issues/6509
func TestLogSinkCtxInInfo(t *testing.T) {
	rec := logtest.NewRecorder()
	ls := NewLogSink("name", WithLoggerProvider(rec))
	l := logr.New(ls)
	ctx := context.WithValue(context.Background(), "key", "value") // nolint:revive,staticcheck

	tests := []struct {
		name        string
		keyValues   []any
		wantCtxFunc func(ctx context.Context)
	}{
		{"default", nil, func(ctx context.Context) {
			assert.Equal(t, context.Background(), ctx)
		}},
		{"default with context", []any{"ctx", ctx}, func(ctx context.Context) {
			assert.Equal(t, "value", ctx.Value("key"))
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer rec.Reset()

			l.Info("msg", tt.keyValues...)
			require.Len(t, rec.Result(), 1)

			got := rec.Result()
			records := got[logtest.Scope{Name: "name"}]
			require.Len(t, records, 1)
			tt.wantCtxFunc(records[0].Context)
		})
	}
}

func TestLogSinkEnabled(t *testing.T) {
	enabledFunc := func(ctx context.Context, param log.EnabledParameters) bool {
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

func buildRecord(body log.Value, severity log.Severity, attrs []log.KeyValue) logtest.Record {
	return logtest.Record{
		Context:    context.Background(),
		Body:       body,
		Severity:   severity,
		Attributes: attrs,
	}
}

func TestConvertKVs(t *testing.T) {
	ctx := context.WithValue(context.Background(), "key", "value") // nolint: revive,staticcheck // test context

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
			kvs:     []any{"key", context.Background(), "ctx", ctx},
			wantKVs: []log.KeyValue{},
			wantCtx: ctx,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, kvs := convertKVs(nil, tt.kvs...) // nolint: staticcheck // pass nil context
			assert.Equal(t, tt.wantKVs, kvs)
			assert.Equal(t, tt.wantCtx, ctx)
		})
	}
}
