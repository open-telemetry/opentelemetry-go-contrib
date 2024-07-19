// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package otelzerolog

import (
	"os"
	"testing"

	"github.com/rs/zerolog"
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
			name: "with a custom version",
			options: []Option{
				WithVersion("1.0"),
			},
			wantConfig: config{
				version:  "1.0",
				provider: global.GetLoggerProvider(),
			},
		},
		{
			name: "with a custom schema URL",
			options: []Option{
				WithSchemaURL("https://example.com"),
			},
			wantConfig: config{
				schemaURL: "https://example.com",
				provider:  global.GetLoggerProvider(),
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
			assert.Equal(t, tt.wantConfig, newConfig(tt.options))
		})
	}
}

func TestNewHook(t *testing.T) {
	const name = "test_hook"
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
			hook := NewHook(name, tt.options...)
			assert.NotNil(t, hook)
			assert.Equal(t, tt.wantLogger, hook.logger)
		})
	}
}

var (
	testMessage = "log message"
	loggerName  = "name"
	testKey     = "key"
	testValue   = "value"
	testEntry   = zerolog.InfoLevel
)

func TestHookRun(t *testing.T) {
	rec := logtest.NewRecorder()
	hook := NewHook(loggerName, WithLoggerProvider(rec))

	logger := zerolog.New(os.Stderr).Hook(hook)

	t.Run("Run", func(t *testing.T) {
		// Create an event and run the hook
		event := logger.Info().Str(testKey, testValue)
		hook.Run(event, testEntry, testMessage)

		// Check the results
		require.Len(t, rec.Result(), 1)
		require.Len(t, rec.Result()[0].Records, 1)
		got := rec.Result()[0].Records[0]
		assert.Equal(t, testMessage, got.Body().AsString())
		assert.Equal(t, log.SeverityInfo, got.Severity())
		assert.Equal(t, zerolog.InfoLevel.String(), got.SeverityText())
	})
}

func TestConvertLevel(t *testing.T) {
	tests := []struct {
		name         string
		zerologLevel zerolog.Level
		expected     log.Severity
	}{
		{
			name:         "DebugLevel",
			zerologLevel: zerolog.DebugLevel,
			expected:     log.SeverityDebug,
		},
		{
			name:         "InfoLevel",
			zerologLevel: zerolog.InfoLevel,
			expected:     log.SeverityInfo,
		},
		{
			name:         "WarnLevel",
			zerologLevel: zerolog.WarnLevel,
			expected:     log.SeverityWarn,
		},
		{
			name:         "ErrorLevel",
			zerologLevel: zerolog.ErrorLevel,
			expected:     log.SeverityError,
		},
		{
			name:         "PanicLevel",
			zerologLevel: zerolog.PanicLevel,
			expected:     log.SeverityFatal1,
		},
		{
			name:         "FatalLevel",
			zerologLevel: zerolog.FatalLevel,
			expected:     log.SeverityFatal2,
		},
		{
			name:         "UnknownLevel",
			zerologLevel: zerolog.NoLevel, // An unknown level
			expected:     log.SeverityUndefined,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := convertLevel(tt.zerologLevel)
			assert.Equal(t, tt.expected, actual, "severity mismatch")
		})
	}
}
