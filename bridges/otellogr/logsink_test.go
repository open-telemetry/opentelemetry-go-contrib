// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package otellogr

import (
	"context"
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
		{
			name: "with a custom levels",
			options: []Option{
				WithLevels([]log.Severity{log.SeverityFatal, log.SeverityError, log.SeverityWarn, log.SeverityInfo}),
			},

			wantConfig: config{
				levels:   []log.Severity{log.SeverityFatal, log.SeverityError, log.SeverityWarn, log.SeverityInfo},
				provider: global.GetLoggerProvider(),
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantConfig, newConfig(tt.options))
		})
	}
}

func TestNewLogSink(t *testing.T) {
	const name = "test_logsink"
	provider := global.GetLoggerProvider()

	for _, tt := range []struct {
		name       string
		options    []Option
		wantLogger log.Logger
	}{
		{
			name:       "with default options",
			wantLogger: provider.Logger(name),
		},
		{
			name: "with version and schema URL",
			options: []Option{
				WithVersion("1.0"),
				WithSchemaURL("https://example.com"),
			},
			wantLogger: provider.Logger(name,
				log.WithInstrumentationVersion("1.0"),
				log.WithSchemaURL("https://example.com"),
			),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			hook := NewLogSink(name, tt.options...)
			assert.NotNil(t, hook)
			assert.Equal(t, tt.wantLogger, hook.logger)
		})
	}
}

type wantRecord struct {
	Body       log.Value
	Severity   log.Severity
	Attributes []log.KeyValue
}

func TestLogSink(t *testing.T) {
	now := time.Now()

	for _, tt := range []struct {
		name            string
		f               func(*logr.Logger)
		wantLoggerCount int
		wantRecords     []wantRecord
	}{
		{
			name: "info",
			f: func(l *logr.Logger) {
				l.Info("info message")
			},
			wantRecords: []wantRecord{
				{
					Body:     log.StringValue("info message"),
					Severity: log.SeverityInfo,
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
					"time", now,
					"uint64", uint64(3),
				)
			},
			wantRecords: []wantRecord{
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
						log.Int64("time", now.UnixNano()),
						log.Int64("uint64", 3),
					},
				},
			},
		},
		{
			name: "info_with_name",
			f: func(l *logr.Logger) {
				l.WithName("test").Info("info message with name")
			},
			wantRecords: []wantRecord{
				{
					Body:     log.StringValue("info message with name"),
					Severity: log.SeverityInfo,
				},
			},
		},
		{
			name: "info_with_name_nested",
			f: func(l *logr.Logger) {
				l.WithName("test").WithName("test").Info("info message with name")
			},
			wantRecords: []wantRecord{
				{
					Body:     log.StringValue("info message with name"),
					Severity: log.SeverityInfo,
				},
			},
		},
		{
			name: "info_with_attrs",
			f: func(l *logr.Logger) {
				l.WithValues("key", "value").Info("info message with attrs")
			},
			wantRecords: []wantRecord{
				{
					Body:     log.StringValue("info message with attrs"),
					Severity: log.SeverityInfo,
					Attributes: []log.KeyValue{
						log.String("key", "value"),
					},
				},
			},
		},
		{
			name: "info_with_attrs_nested",
			f: func(l *logr.Logger) {
				l.WithValues("key1", "value1").Info("info message with attrs", "key2", "value2")
			},
			wantRecords: []wantRecord{
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
	} {
		t.Run(tt.name, func(t *testing.T) {
			rec := logtest.NewRecorder()
			ls := NewLogSink("name", WithLoggerProvider(rec))
			l := logr.New(ls)
			tt.f(&l)

			last := len(rec.Result()) - 1

			assert.Len(t, rec.Result()[last].Records, len(tt.wantRecords))
			for i, record := range rec.Result()[last].Records {
				assert.Equal(t, tt.wantRecords[i].Body, record.Body())
				assert.Equal(t, tt.wantRecords[i].Severity, record.Severity())

				var attributes []log.KeyValue
				record.WalkAttributes(func(kv log.KeyValue) bool {
					attributes = append(attributes, kv)
					return true
				})
				assert.Equal(t, tt.wantRecords[i].Attributes, attributes)
			}
		})
	}
}

func TestLogSinkWithName(t *testing.T) {
	rec := logtest.NewRecorder()
	ls := NewLogSink("name", WithLoggerProvider(rec))
	lsWithName := ls.WithName("test")
	require.NotEqual(t, ls, lsWithName)
	require.Equal(t, lsWithName, ls.WithName("test"))
}

func TestLogSinkEnabled(t *testing.T) {
	rec := logtest.NewRecorder(
		logtest.WithEnabledFunc(func(ctx context.Context, record log.Record) bool {
			return record.Severity() == log.SeverityInfo
		}),
	)
	ls := NewLogSink(
		"name",
		WithLoggerProvider(rec),
		WithLevels([]log.Severity{log.SeverityDebug, log.SeverityInfo}),
	)

	assert.False(t, ls.Enabled(0))
	assert.True(t, ls.Enabled(1))
}
