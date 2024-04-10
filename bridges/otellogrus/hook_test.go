// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package otellogrus

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/logtest"
)

func TestNewHook(t *testing.T) {
	assert.NotNil(t, NewHook())
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
				bridgeName: []log.Record{
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
				bridgeName: []log.Record{
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
				bridgeName: []log.Record{
					buildRecord(log.StringValue(""), time.Time{}, log.SeverityTrace1, nil),
				},
			},
		},
		{
			name: "emits a log entry with data",
			entry: &logrus.Entry{
				Data: logrus.Fields{
					"hello":  "world",
					"answer": 42,
				},
			},
			wantRecords: map[string][]log.Record{
				bridgeName: []log.Record{
					buildRecord(log.StringValue(""), time.Time{}, 0, []log.KeyValue{
						log.String("hello", "world"),
						log.Int("answer", 42),
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
						assert.Equal(t, v, s.Records)
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
			name: "with a boolean",

			fields: logrus.Fields{"hello": true},
			wantKeyValue: []log.KeyValue{
				log.Bool("hello", true),
			},
		},
		{
			name: "with a bytes array",

			fields: logrus.Fields{"hello": []byte("world")},
			wantKeyValue: []log.KeyValue{
				log.Bytes("hello", []byte("world")),
			},
		},
		{
			name: "with a float64",

			fields: logrus.Fields{"hello": 6.5},
			wantKeyValue: []log.KeyValue{
				log.Float64("hello", 6.5),
			},
		},
		{
			name: "with an int",

			fields: logrus.Fields{"hello": 42},
			wantKeyValue: []log.KeyValue{
				log.Int("hello", 42),
			},
		},
		{
			name: "with an int64",

			fields: logrus.Fields{"hello": int64(42)},
			wantKeyValue: []log.KeyValue{
				log.Int64("hello", 42),
			},
		},
		{
			name: "with a string",

			fields: logrus.Fields{"hello": "world"},
			wantKeyValue: []log.KeyValue{
				log.String("hello", "world"),
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, convertFields(tt.fields), tt.wantKeyValue)
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
