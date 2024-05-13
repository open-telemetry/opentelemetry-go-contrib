// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelzap

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/log/logtest"
	"go.opentelemetry.io/otel/sdk/instrumentation"
)

var (
	testBodyString = "log message"
	testSeverity   = log.SeverityInfo
	testField      = zap.String("key", "testValue")
)

func TestNewCoreConfiguration(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		r := logtest.NewRecorder()
		prev := global.GetLoggerProvider()
		defer global.SetLoggerProvider(prev)
		global.SetLoggerProvider(r)

		var h *Core
		require.NotPanics(t, func() { h = NewCore() })
		require.NotNil(t, h.logger)
		require.IsType(t, &logtest.Recorder{}, h.logger)
		l := h.logger.(*logtest.Recorder)
		require.Len(t, l.Result(), 1)

		want := &logtest.ScopeRecords{Name: bridgeName, Version: version}
		got := l.Result()[0]
		assert.Equal(t, want, got)
	})

	t.Run("Options", func(t *testing.T) {
		r := logtest.NewRecorder()
		scope := instrumentation.Scope{Name: "name", Version: "ver", SchemaURL: "url"}
		var h *Core
		require.NotPanics(t, func() {
			h = NewCore(
				WithLoggerProvider(r),
				WithInstrumentationScope(scope),
			)
		})
		require.NotNil(t, h.logger)
		require.IsType(t, &logtest.Recorder{}, h.logger)
		l := h.logger.(*logtest.Recorder)
		require.Len(t, l.Result(), 1)

		want := &logtest.ScopeRecords{Name: scope.Name, Version: scope.Version, SchemaURL: scope.SchemaURL}
		got := l.Result()[0]
		assert.Equal(t, want, got)
	})
}

// Test conversion of Zap Level to OTel level.
func TestGetOTelLevel(t *testing.T) {
	tests := []struct {
		level       zapcore.Level
		expectedSev log.Severity
	}{
		{zapcore.DebugLevel, log.SeverityDebug},
		{zapcore.InfoLevel, log.SeverityInfo},
		{zapcore.WarnLevel, log.SeverityWarn},
		{zapcore.ErrorLevel, log.SeverityError},
		{zapcore.DPanicLevel, log.SeverityFatal1},
		{zapcore.PanicLevel, log.SeverityFatal2},
		{zapcore.FatalLevel, log.SeverityFatal3},
		{zapcore.InvalidLevel, log.SeverityUndefined},
	}

	for _, test := range tests {
		result := getOTelLevel(test.level)
		if result != test.expectedSev {
			t.Errorf("For level %v, expected %v but got %v", test.level, test.expectedSev, result)
		}
	}
}

// Tests [Core] write method.
func TestCore(t *testing.T) {
	rec := logtest.NewRecorder()
	zc := NewCore(WithLoggerProvider(rec))

	t.Run("test Write method of Core", func(t *testing.T) {
		logger := zap.New(zc)
		logger.Info(testBodyString, testField)

		// not sure index 1 populated with results and not 0
		got := rec.Result()[1].Records[0]
		assert.Equal(t, testBodyString, got.Body().AsString())
		assert.Equal(t, testSeverity, got.Severity())

		rec.Reset()
	})
}
