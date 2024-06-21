// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelzerolog

import (
	"bytes"
	// "context"
	"testing"
	// "time"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/log/embedded"

)

type mockLoggerProvider struct{
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
			name: "with a schema URL",
			options: []Option{
				WithSchemaURL("https://example.com"),
			},
			wantConfig: config{
				schemaURL: "https://example.com",
				provider:  global.GetLoggerProvider(),
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantConfig, newConfig(tt.options))
		})
	}
}

func TestNewSeverityHook(t *testing.T) {
	const name = "test_hook"
	provider := global.GetLoggerProvider()

	for _, tt := range []struct {
		name    string
		options []Option
		wantLogger log.Logger
	}{
		{
			name: "with default options",
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
			hook := NewSeverityHook(name, tt.options...)
			assert.NotNil(t, hook)
			assert.Equal(t, tt.wantLogger, hook.logger)
		})
	}
}

func TestSeverityHookLevels(t *testing.T) {
	hook := NewSeverityHook("test")
	expectedLevels := []zerolog.Level{
		zerolog.PanicLevel,
		zerolog.FatalLevel,
		zerolog.ErrorLevel,
		zerolog.WarnLevel,
		zerolog.InfoLevel,
		zerolog.DebugLevel,
	}

	assert.Equal(t, expectedLevels, hook.Levels())
}

func TestSeverityHookRun(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf).With().Timestamp().Logger()
	hook := NewSeverityHook("test")

	e := logger.Info()
	level := zerolog.InfoLevel
	msg := "test message"

	err := hook.Run(e, level, msg)
	assert.NoError(t, err)
}

//TODO
func TestConvertEvent(t *testing.T) {

}

func BenchmarkSeverityHookRun(b *testing.B) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf).With().Timestamp().Logger()
	hook := NewSeverityHook("test")

	e := logger.Info()
	level := zerolog.InfoLevel
	msg := "test message"

	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = hook.Run(e, level, msg)
	}
}
