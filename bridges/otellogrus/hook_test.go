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
	} {
		t.Run(tt.name, func(t *testing.T) {
			lp := sdklog.NewLoggerProvider()

			err := NewHook(WithLoggerProvider(lp)).Fire(tt.entry)
			assert.Equal(t, tt.expectedErr, err)
		})
	}
}
