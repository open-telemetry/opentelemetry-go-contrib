// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package otellogr

import (
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
		name             string
		options          []Option
		wantScopeRecords *logtest.ScopeRecords
	}{
		{
			name:             "with default options",
			wantScopeRecords: &logtest.ScopeRecords{Name: name},
		},
		{
			name: "with version and schema URL",
			options: []Option{
				WithVersion("1.0"),
				WithSchemaURL("https://example.com"),
			},
			wantScopeRecords: &logtest.ScopeRecords{
				Name:      name,
				Version:   "1.0",
				SchemaURL: "https://example.com",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			provider := logtest.NewRecorder()

			var l *LogSink
			require.NotPanics(t, func() {
				l = NewLogSink(name, append(
					tt.options,
					WithLoggerProvider(provider),
				)...)
			})
			require.NotNil(t, l)
			require.Len(t, provider.Result(), 1)

			got := provider.Result()[0]
			assert.Equal(t, tt.wantScopeRecords, got)
		})
	}
}

func TestLogSink(t *testing.T) {
	const name = "name"

	for _, tt := range []struct {
		name          string
		f             func(*logr.Logger)
		levelSeverity func(int) log.Severity
		wantRecords   map[string][]log.Record
	}{
		{
			name: "no_log",
			f:    func(l *logr.Logger) {},
			wantRecords: map[string][]log.Record{
				name: {},
			},
		},
		{
			name: "info",
			f: func(l *logr.Logger) {
				l.Info("msg")
			},
			wantRecords: map[string][]log.Record{
				name: {
					buildRecord(log.StringValue("msg"), time.Time{}, log.SeverityInfo, nil),
				},
			},
		},
		{
			name: "info_with_level_severity",
			f: func(l *logr.Logger) {
				l.V(1).Info("msg")
				l.V(4).Info("msg")
			},
			wantRecords: map[string][]log.Record{
				name: {
					buildRecord(log.StringValue("msg"), time.Time{}, log.SeverityInfo2, nil),
					buildRecord(log.StringValue("msg"), time.Time{}, log.SeverityWarn, nil),
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
			wantRecords: map[string][]log.Record{
				name: {
					buildRecord(log.StringValue("msg"), time.Time{}, log.SeverityInfo, nil),
					buildRecord(log.StringValue("msg"), time.Time{}, log.SeverityError, nil),
					buildRecord(log.StringValue("msg"), time.Time{}, log.SeverityWarn, nil),
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
			wantRecords: map[string][]log.Record{
				name: {
					buildRecord(log.StringValue("msg"), time.Time{}, log.SeverityInfo, []log.KeyValue{
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
			wantRecords: map[string][]log.Record{
				name + "/test": {
					buildRecord(log.StringValue("info message with name"), time.Time{}, log.SeverityInfo, nil),
				},
			},
		},
		{
			name: "info_with_name_nested",
			f: func(l *logr.Logger) {
				l.WithName("test").WithName("test").Info("info message with name")
			},
			wantRecords: map[string][]log.Record{
				name + "/test/test": {
					buildRecord(log.StringValue("info message with name"), time.Time{}, log.SeverityInfo, nil),
				},
			},
		},
		{
			name: "info_with_attrs",
			f: func(l *logr.Logger) {
				l.WithValues("key", "value").Info("info message with attrs")
			},
			wantRecords: map[string][]log.Record{
				name: {
					buildRecord(log.StringValue("info message with attrs"), time.Time{}, log.SeverityInfo, []log.KeyValue{
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
			wantRecords: map[string][]log.Record{
				name: {
					buildRecord(log.StringValue("info message with attrs"), time.Time{}, log.SeverityInfo, []log.KeyValue{
						log.String("key1", "value1"),
						log.String("key2", "value2"),
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

			for k, v := range tt.wantRecords {
				found := false

				want := make([]logtest.EmittedRecord, len(v))
				for i := range want {
					want[i] = logtest.EmittedRecord{Record: v[i]}
				}

				for _, s := range rec.Result() {
					if k == s.Name {
						assertRecords(t, want, s.Records)
						found = true
					}
				}

				assert.Truef(t, found, "want to find records with a scope named %q", k)
			}
		})
	}
}

func buildRecord(body log.Value, timestamp time.Time, severity log.Severity, attrs []log.KeyValue) log.Record {
	var record log.Record
	record.SetBody(body)
	record.SetTimestamp(timestamp)
	record.SetSeverity(severity)
	record.AddAttributes(attrs...)

	return record
}

func assertRecords(t *testing.T, want, got []logtest.EmittedRecord) {
	t.Helper()

	assert.Equal(t, len(want), len(got))

	for i, j := range want {
		logtest.AssertRecordEqual(t, j.Record, got[i].Record)
	}
}
