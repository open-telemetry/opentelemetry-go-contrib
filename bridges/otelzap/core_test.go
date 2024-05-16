// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelzap

import (
	"context"
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

var testMessage = "log message"

func TestCore(t *testing.T) {
	rec := logtest.NewRecorder()
	zc := NewCore(WithLoggerProvider(rec))
	logger := zap.New(zc)

	logger.Info(testMessage)

	// TODO (#5580): Not sure why index 1 is populated with results and not 0.
	got := rec.Result()[1].Records[0]
	assert.Equal(t, testMessage, got.Body().AsString())
	assert.Equal(t, log.SeverityInfo, got.Severity())
}

func TestCoreEnabled(t *testing.T) {
	enabledFunc := func(c context.Context, r log.Record) bool {
		return r.Severity() >= log.SeverityInfo
	}

	r := logtest.NewRecorder(logtest.WithEnabledFunc(enabledFunc))
	logger := zap.New(NewCore(WithLoggerProvider(r)))

	if ce := logger.Check(zap.DebugLevel, testMessage); ce != nil {
		ce.Write()
	}

	assert.Empty(t, r.Result()[1].Records)
	logger.Debug(testMessage)
	assert.Empty(t, r.Result()[1].Records)

	if ce := logger.Check(zap.InfoLevel, testMessage); ce != nil {
		ce.Write()
	}
	require.Len(t, r.Result()[1].Records, 1)
	got := r.Result()[1].Records[0]
	assert.Equal(t, testMessage, got.Body().AsString())
	assert.Equal(t, log.SeverityInfo, got.Severity())
}

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

func TestConvertLevel(t *testing.T) {
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
		result := convertLevel(test.level)
		if result != test.expectedSev {
			t.Errorf("For level %v, expected %v but got %v", test.level, test.expectedSev, result)
		}
	}
}
