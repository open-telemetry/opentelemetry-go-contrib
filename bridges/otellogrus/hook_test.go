// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package otellogrus

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

func TestNewHook(t *testing.T) {
	assert.NotNil(t, NewHook())
}

func TestHookLevels(t *testing.T) {
	for _, tt := range []struct {
		name    string
		options []Option

		expectedLevels []logrus.Level
	}{
		{
			name:           "with the default levels",
			expectedLevels: logrus.AllLevels,
		},
		{
			name: "with provided levels",
			options: []Option{
				WithLevels([]logrus.Level{logrus.PanicLevel}),
			},
			expectedLevels: []logrus.Level{logrus.PanicLevel},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			levels := NewHook(tt.options...).Levels()
			assert.Equal(t, tt.expectedLevels, levels)
		})
	}
}

func TestHookFire(t *testing.T) {
	for _, tt := range []struct {
		name  string
		entry *logrus.Entry

		expectedRecord log.Record
		expectedErr    error
	}{
		{
			name:  "emits an empty log entry",
			entry: &logrus.Entry{},
		},
		{
			name: "emits a log entry with severity level",
			entry: &logrus.Entry{
				Level: logrus.FatalLevel,
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
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			lp := sdklog.NewLoggerProvider()

			err := NewHook(WithLoggerProvider(lp)).Fire(tt.entry)
			assert.Equal(t, tt.expectedErr, err)
		})
	}
}

func TestConvertFields(t *testing.T) {
	for _, tt := range []struct {
		name string

		fields           logrus.Fields
		expectedKeyValue []log.KeyValue
	}{
		{
			name: "with a boolean",

			fields: logrus.Fields{"hello": true},
			expectedKeyValue: []log.KeyValue{
				log.Bool("hello", true),
			},
		},
		{
			name: "with a bytes array",

			fields: logrus.Fields{"hello": []byte("world")},
			expectedKeyValue: []log.KeyValue{
				log.Bytes("hello", []byte("world")),
			},
		},
		{
			name: "with a float64",

			fields: logrus.Fields{"hello": 6.5},
			expectedKeyValue: []log.KeyValue{
				log.Float64("hello", 6.5),
			},
		},
		{
			name: "with an int",

			fields: logrus.Fields{"hello": 42},
			expectedKeyValue: []log.KeyValue{
				log.Int("hello", 42),
			},
		},
		{
			name: "with an int64",

			fields: logrus.Fields{"hello": int64(42)},
			expectedKeyValue: []log.KeyValue{
				log.Int64("hello", 42),
			},
		},
		{
			name: "with a string",

			fields: logrus.Fields{"hello": "world"},
			expectedKeyValue: []log.KeyValue{
				log.String("hello", "world"),
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, convertFields(tt.fields), tt.expectedKeyValue)
		})
	}
}
