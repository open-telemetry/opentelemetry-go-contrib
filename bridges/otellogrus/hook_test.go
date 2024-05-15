// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package otellogrus

import (
	"slices"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/embedded"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/log/logtest"
	"go.opentelemetry.io/otel/sdk/instrumentation"
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
				scope: instrumentation.Scope{
					Name:    bridgeName,
					Version: version,
				},
				provider: global.GetLoggerProvider(),
				levels:   logrus.AllLevels,
			},
		},
		{
			name: "with a custom instrumentation scope",
			options: []Option{
				WithInstrumentationScope(instrumentation.Scope{
					Name:    "test",
					Version: "42.0",
				}),
			},

			wantConfig: config{
				scope: instrumentation.Scope{
					Name:    "test",
					Version: "42.0",
				},
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
				scope: instrumentation.Scope{
					Name:    bridgeName,
					Version: version,
				},
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
				scope: instrumentation.Scope{
					Name:    bridgeName,
					Version: version,
				},
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
	provider := global.GetLoggerProvider()

	for _, tt := range []struct {
		name    string
		options []Option

		wantLogger log.Logger
	}{
		{
			name: "with the default options",

			wantLogger: provider.Logger(bridgeName, log.WithInstrumentationVersion(version)),
		},
		{
			name: "with a schema URL",
			options: []Option{
				WithInstrumentationScope(instrumentation.Scope{
					Name:      "test",
					Version:   "42.1",
					SchemaURL: "https://example.com",
				}),
			},

			wantLogger: provider.Logger("test",
				log.WithInstrumentationVersion("42.1"),
				log.WithSchemaURL("https://example.com"),
			),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			hook := NewHook(tt.options...)
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
			levels := NewHook(tt.options...).Levels()
			assert.Equal(t, tt.wantLevels, levels)
		})
	}
}

func TestHookFire(t *testing.T) {
	now := time.Now()

	for _, tt := range []struct {
		name  string
		entry *logrus.Entry

		wantRecords map[string][]log.Record
		wantErr     error
	}{
		{
			name:  "emits an empty log entry",
			entry: &logrus.Entry{},

			wantRecords: map[string][]log.Record{
				bridgeName: {
					buildRecord(log.StringValue(""), time.Time{}, 0, nil),
				},
			},
		},
		{
			name: "emits a log entry with a timestamp",
			entry: &logrus.Entry{
				Time: now,
			},
			wantRecords: map[string][]log.Record{
				bridgeName: {
					buildRecord(log.StringValue(""), now, 0, nil),
				},
			},
		},
		{
			name: "emits a log entry with severity level",
			entry: &logrus.Entry{
				Level: logrus.FatalLevel,
			},
			wantRecords: map[string][]log.Record{
				bridgeName: {
					buildRecord(log.StringValue(""), time.Time{}, log.SeverityTrace1, nil),
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
			wantRecords: map[string][]log.Record{
				bridgeName: {
					buildRecord(log.StringValue(""), time.Time{}, 0, []log.KeyValue{
						log.String("hello", "world"),
					}),
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rec := logtest.NewRecorder()

			err := NewHook(WithLoggerProvider(rec)).Fire(tt.entry)
			assert.Equal(t, tt.wantErr, err)

			for k, v := range tt.wantRecords {
				found := false

				for _, s := range rec.Result() {
					if k == s.Name {
						assertRecords(t, v, s.Records)
						found = true
					}
				}

				assert.Truef(t, found, "want to find records with a scope named %q", k)
			}
		})
	}
}

func TestConvertFields(t *testing.T) {
	for _, tt := range []struct {
		name string

		fields       logrus.Fields
		wantKeyValue []log.KeyValue
	}{
		{
			name:   "with a boolean",
			fields: logrus.Fields{"hello": true},
			wantKeyValue: []log.KeyValue{
				log.Bool("hello", true),
			},
		},
		{
			name:   "with a bytes array",
			fields: logrus.Fields{"hello": []byte("world")},
			wantKeyValue: []log.KeyValue{
				log.Bytes("hello", []byte("world")),
			},
		},
		{
			name:   "with a float64",
			fields: logrus.Fields{"hello": 6.5},
			wantKeyValue: []log.KeyValue{
				log.Float64("hello", 6.5),
			},
		},
		{
			name:   "with an int",
			fields: logrus.Fields{"hello": 42},
			wantKeyValue: []log.KeyValue{
				log.Int("hello", 42),
			},
		},
		{
			name:   "with an int64",
			fields: logrus.Fields{"hello": int64(42)},
			wantKeyValue: []log.KeyValue{
				log.Int64("hello", 42),
			},
		},
		{
			name:   "with a string",
			fields: logrus.Fields{"hello": "world"},
			wantKeyValue: []log.KeyValue{
				log.String("hello", "world"),
			},
		},
		{
			name:   "with nil",
			fields: logrus.Fields{"hello": nil},
			wantKeyValue: []log.KeyValue{
				{Key: "hello", Value: log.Value{}},
			},
		},
		{
			name:   "with a struct",
			fields: logrus.Fields{"hello": struct{ Name string }{Name: "foobar"}},
			wantKeyValue: []log.KeyValue{
				log.String("hello", "{Name:foobar}"),
			},
		},
		{
			name:   "with a slice",
			fields: logrus.Fields{"hello": []string{"foo", "bar"}},
			wantKeyValue: []log.KeyValue{
				log.Slice("hello",
					log.StringValue("foo"),
					log.StringValue("bar"),
				),
			},
		},
		{
			name:   "with an interface slice",
			fields: logrus.Fields{"hello": []interface{}{"foo", 42}},
			wantKeyValue: []log.KeyValue{
				log.Slice("hello",
					log.StringValue("foo"),
					log.Int64Value(42),
				),
			},
		},
		{
			name:   "with a map",
			fields: logrus.Fields{"hello": map[string]int{"answer": 42}},
			wantKeyValue: []log.KeyValue{
				log.Map("hello", log.Int("answer", 42)),
			},
		},
		{
			name:   "with an interface map",
			fields: logrus.Fields{"hello": map[interface{}]interface{}{1: "question", "answer": 42}},
			wantKeyValue: []log.KeyValue{
				log.Map("hello", log.Int("answer", 42), log.String("1", "question")),
			},
		},
		{
			name:   "with a nested map",
			fields: logrus.Fields{"hello": map[string]map[string]int{"sublevel": {"answer": 42}}},
			wantKeyValue: []log.KeyValue{
				log.Map("hello", log.Map("sublevel", log.Int("answer", 42))),
			},
		},
		{
			name:   "with a struct map",
			fields: logrus.Fields{"hello": map[struct{ name string }]string{{name: "hello"}: "world"}},
			wantKeyValue: []log.KeyValue{
				log.Map("hello", log.String("{name:hello}", "world")),
			},
		},
		{
			name:   "with a pointer to struct",
			fields: logrus.Fields{"hello": &struct{ Name string }{Name: "foobar"}},
			wantKeyValue: []log.KeyValue{
				log.String("hello", "{Name:foobar}"),
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assertKeyValues(t, tt.wantKeyValue, convertFields(tt.fields))
		})
	}
}

func BenchmarkHook(b *testing.B) {
	record := &logrus.Entry{
		Data: map[string]interface{}{
			"string": "hello",
			"int":    42,
			"float":  1.5,
			"bool":   false,
		},
		Message: "body",
		Time:    time.Now(),
		Level:   logrus.InfoLevel,
	}

	b.Run("Fire", func(b *testing.B) {
		hooks := make([]*Hook, b.N)
		for i := range hooks {
			hooks[i] = NewHook()
		}

		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			_ = hooks[n].Fire(record)
		}
	})
}

func buildRecord(body log.Value, timestamp time.Time, severity log.Severity, attrs []log.KeyValue) log.Record {
	var record log.Record
	record.SetBody(body)
	record.SetTimestamp(timestamp)
	record.SetSeverity(severity)
	record.AddAttributes(attrs...)

	return record
}

func assertKeyValues(t *testing.T, want, got []log.KeyValue) {
	t.Helper()
	if !slices.EqualFunc(want, got, log.KeyValue.Equal) {
		t.Errorf("KeyValues are not equal:\nwant: %v\ngot:  %v", want, got)
	}
}

func assertBody(t *testing.T, want log.Value, got log.Value) {
	t.Helper()
	if !got.Equal(want) {
		t.Errorf("Body value is not equal:\nwant: %v\ngot:  %v", want, got)
	}
}

func assertAttributes(t *testing.T, want, got log.Record) {
	t.Helper()

	var wantAttr []log.KeyValue
	want.WalkAttributes(func(kv log.KeyValue) bool {
		wantAttr = append(wantAttr, kv)
		return true
	})
	var gotAttr []log.KeyValue
	got.WalkAttributes(func(kv log.KeyValue) bool {
		gotAttr = append(gotAttr, kv)
		return true
	})

	if !slices.EqualFunc(wantAttr, gotAttr, log.KeyValue.Equal) {
		t.Errorf("Attributes are not equal:\nwant: %v\ngot:  %v", want, got)
	}
}

func assertRecords(t *testing.T, want, got []log.Record) {
	t.Helper()

	assert.Equal(t, len(want), len(got))

	for i, j := range want {
		assert.Equal(t, j.Timestamp(), got[i].Timestamp())
		assert.Equal(t, j.ObservedTimestamp(), got[i].ObservedTimestamp())
		assert.Equal(t, j.Severity(), got[i].Severity())
		assert.Equal(t, j.SeverityText(), got[i].SeverityText())
		assertBody(t, j.Body(), got[i].Body())
		assertAttributes(t, j, got[i])
	}
}
