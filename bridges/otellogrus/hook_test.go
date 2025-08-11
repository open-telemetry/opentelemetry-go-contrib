// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otellogrus

import (
	"slices"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
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
				levels:   logrus.AllLevels,
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
				levels:   logrus.AllLevels,
			},
		},
		{
			name: "with a custom logger provider",
			options: []Option{
				WithLoggerProvider(customLoggerProvider),
			},

			wantConfig: config{
				provider: customLoggerProvider,
				levels:   logrus.AllLevels,
			},
		},
		{
			name: "with custom log levels",
			options: []Option{
				WithLevels([]logrus.Level{logrus.FatalLevel}),
			},

			wantConfig: config{
				provider: global.GetLoggerProvider(),
				levels:   []logrus.Level{logrus.FatalLevel},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantConfig, newConfig(tt.options))
		})
	}
}

func TestNewHook(t *testing.T) {
	const name = "name"
	provider := global.GetLoggerProvider()

	for _, tt := range []struct {
		name    string
		options []Option

		wantLogger log.Logger
	}{
		{
			name: "with the default options",

			wantLogger: provider.Logger(name),
		},
		{
			name: "with custom options",
			options: []Option{
				WithVersion("42.1"),
				WithSchemaURL("https://example.com"),
				WithAttributes(attribute.String("testattr", "testval")),
			},

			wantLogger: provider.Logger(name,
				log.WithInstrumentationVersion("42.1"),
				log.WithSchemaURL("https://example.com"),
				log.WithInstrumentationAttributes(attribute.String("testattr", "testval")),
			),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			hook := NewHook(name, tt.options...)
			assert.NotNil(t, hook)

			assert.Equal(t, tt.wantLogger, hook.logger)
		})
	}
}

func TestHookLevels(t *testing.T) {
	for _, tt := range []struct {
		name    string
		options []Option

		wantLevels []logrus.Level
	}{
		{
			name:       "with the default levels",
			wantLevels: logrus.AllLevels,
		},
		{
			name: "with provided levels",
			options: []Option{
				WithLevels([]logrus.Level{logrus.PanicLevel}),
			},
			wantLevels: []logrus.Level{logrus.PanicLevel},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			levels := NewHook("", tt.options...).Levels()
			assert.Equal(t, tt.wantLevels, levels)
		})
	}
}

func TestHookFire(t *testing.T) {
	const name = "name"
	now := time.Now()
	var nilPointer *struct{}

	for _, tt := range []struct {
		name  string
		entry *logrus.Entry

		want    logtest.Recording
		wantErr error
	}{
		{
			name:  "emits an empty log entry",
			entry: &logrus.Entry{},

			want: logtest.Recording{
				logtest.Scope{Name: name}: {
					{
						Severity:     log.SeverityFatal4,
						SeverityText: "panic",
						Body:         log.StringValue(""),
					},
				},
			},
		},
		{
			name: "emits a log entry with a timestamp",
			entry: &logrus.Entry{
				Time: now,
			},
			want: logtest.Recording{
				logtest.Scope{Name: name}: {
					{
						Severity:     log.SeverityFatal4,
						SeverityText: "panic",
						Body:         log.StringValue(""),
						Timestamp:    now,
					},
				},
			},
		},
		{
			name: "emits a log entry with panic severity level",
			entry: &logrus.Entry{
				Level: logrus.PanicLevel,
			},
			want: logtest.Recording{
				logtest.Scope{Name: name}: {
					{
						Severity:     log.SeverityFatal4,
						SeverityText: "panic",
						Body:         log.StringValue(""),
					},
				},
			},
		},
		{
			name: "emits a log entry with fatal severity level",
			entry: &logrus.Entry{
				Level: logrus.FatalLevel,
			},
			want: logtest.Recording{
				logtest.Scope{Name: name}: {
					{
						Severity:     log.SeverityFatal,
						SeverityText: "fatal",
						Body:         log.StringValue(""),
					},
				},
			},
		},
		{
			name: "emits a log entry with error severity level",
			entry: &logrus.Entry{
				Level: logrus.ErrorLevel,
			},
			want: logtest.Recording{
				logtest.Scope{Name: name}: {
					{
						Severity:     log.SeverityError,
						SeverityText: "error",
						Body:         log.StringValue(""),
					},
				},
			},
		},
		{
			name: "emits a log entry with warn severity level",
			entry: &logrus.Entry{
				Level: logrus.WarnLevel,
			},
			want: logtest.Recording{
				logtest.Scope{Name: name}: {
					{
						Severity:     log.SeverityWarn,
						SeverityText: "warning",
						Body:         log.StringValue(""),
					},
				},
			},
		},
		{
			name: "emits a log entry with info severity level",
			entry: &logrus.Entry{
				Level: logrus.InfoLevel,
			},
			want: logtest.Recording{
				logtest.Scope{Name: name}: {
					{
						Severity:     log.SeverityInfo,
						SeverityText: "info",
						Body:         log.StringValue(""),
					},
				},
			},
		},
		{
			name: "emits a log entry with info severity level",
			entry: &logrus.Entry{
				Level: logrus.DebugLevel,
			},
			want: logtest.Recording{
				logtest.Scope{Name: name}: {
					{
						Severity:     log.SeverityDebug,
						SeverityText: "debug",
						Body:         log.StringValue(""),
					},
				},
			},
		},
		{
			name: "emits a log entry with info severity level",
			entry: &logrus.Entry{
				Level: logrus.TraceLevel,
			},
			want: logtest.Recording{
				logtest.Scope{Name: name}: {
					{
						Severity:     log.SeverityTrace,
						SeverityText: "trace",
						Body:         log.StringValue(""),
					},
				},
			},
		},
		{
			name: "emits a log entry with data",
			entry: &logrus.Entry{
				Data: logrus.Fields{
					"hello": "world",
				},
			},
			want: logtest.Recording{
				logtest.Scope{Name: name}: {
					{
						Severity:     log.SeverityFatal4,
						SeverityText: "panic",
						Attributes: []log.KeyValue{
							log.String("hello", "world"),
						},
						Body: log.StringValue(""),
					},
				},
			},
		},
		{
			name: "emits a log entry with data containing a nil pointer",
			entry: &logrus.Entry{
				Data: logrus.Fields{
					"nil_pointer": nilPointer,
				},
			},
			want: logtest.Recording{
				logtest.Scope{Name: name}: {
					{
						Severity:     log.SeverityFatal4,
						SeverityText: "panic",
						Attributes: []log.KeyValue{
							log.Empty("nil_pointer"),
						},
						Body: log.StringValue(""),
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rec := logtest.NewRecorder()

			err := NewHook(name, WithLoggerProvider(rec)).Fire(tt.entry)
			assert.Equal(t, tt.wantErr, err)

			logtest.AssertEqual(t, tt.want, rec.Result())
		})
	}
}

func TestConvertFields(t *testing.T) {
	for _, tt := range []struct {
		name string

		fields logrus.Fields
		want   []log.KeyValue
	}{
		{
			name:   "with a boolean",
			fields: logrus.Fields{"hello": true},
			want: []log.KeyValue{
				log.Bool("hello", true),
			},
		},
		{
			name:   "with a bytes array",
			fields: logrus.Fields{"hello": []byte("world")},
			want: []log.KeyValue{
				log.Bytes("hello", []byte("world")),
			},
		},
		{
			name:   "with a float64",
			fields: logrus.Fields{"hello": 6.5},
			want: []log.KeyValue{
				log.Float64("hello", 6.5),
			},
		},
		{
			name:   "with an int",
			fields: logrus.Fields{"hello": 42},
			want: []log.KeyValue{
				log.Int("hello", 42),
			},
		},
		{
			name:   "with an int64",
			fields: logrus.Fields{"hello": int64(42)},
			want: []log.KeyValue{
				log.Int64("hello", 42),
			},
		},
		{
			name:   "with a string",
			fields: logrus.Fields{"hello": "world"},
			want: []log.KeyValue{
				log.String("hello", "world"),
			},
		},
		{
			name:   "with nil",
			fields: logrus.Fields{"hello": nil},
			want: []log.KeyValue{
				{Key: "hello", Value: log.Value{}},
			},
		},
		{
			name:   "with a struct",
			fields: logrus.Fields{"hello": struct{ Name string }{Name: "foobar"}},
			want: []log.KeyValue{
				log.String("hello", "{Name:foobar}"),
			},
		},
		{
			name:   "with a slice",
			fields: logrus.Fields{"hello": []string{"foo", "bar"}},
			want: []log.KeyValue{
				log.Slice("hello",
					log.StringValue("foo"),
					log.StringValue("bar"),
				),
			},
		},
		{
			name:   "with an interface slice",
			fields: logrus.Fields{"hello": []any{"foo", 42}},
			want: []log.KeyValue{
				log.Slice("hello",
					log.StringValue("foo"),
					log.Int64Value(42),
				),
			},
		},
		{
			name:   "with a map",
			fields: logrus.Fields{"hello": map[string]int{"answer": 42}},
			want: []log.KeyValue{
				log.Map("hello", log.Int("answer", 42)),
			},
		},
		{
			name:   "with an interface map",
			fields: logrus.Fields{"hello": map[any]any{1: "question", "answer": 42}},
			want: []log.KeyValue{
				log.Map("hello", log.Int("answer", 42), log.String("1", "question")),
			},
		},
		{
			name:   "with a nested map",
			fields: logrus.Fields{"hello": map[string]map[string]int{"sublevel": {"answer": 42}}},
			want: []log.KeyValue{
				log.Map("hello", log.Map("sublevel", log.Int("answer", 42))),
			},
		},
		{
			name:   "with a struct map",
			fields: logrus.Fields{"hello": map[struct{ name string }]string{{name: "hello"}: "world"}},
			want: []log.KeyValue{
				log.Map("hello", log.String("{name:hello}", "world")),
			},
		},
		{
			name:   "with a pointer to struct",
			fields: logrus.Fields{"hello": &struct{ Name string }{Name: "foobar"}},
			want: []log.KeyValue{
				log.String("hello", "{Name:foobar}"),
			},
		},
		{
			name:   "with log attribute",
			fields: logrus.Fields{"hello": log.MapValue(log.String("foo", "bar"))},
			want: []log.KeyValue{
				log.Map("hello", log.String("foo", "bar")),
			},
		},
		{
			name:   "with standard attribute",
			fields: logrus.Fields{"hello": attribute.StringSliceValue([]string{"one", "two"})},
			want: []log.KeyValue{
				log.Slice("hello", log.StringValue("one"), log.StringValue("two")),
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := convertFields(tt.fields)
			if !slices.EqualFunc(tt.want, got, log.KeyValue.Equal) {
				t.Errorf("KeyValues are not equal:\nwant: %v\ngot:  %v", tt.want, got)
			}
		})
	}
}
