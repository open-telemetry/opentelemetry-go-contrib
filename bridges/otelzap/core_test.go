// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelzap

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/log/logtest"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

var (
	testMessage = "log message"
	loggerName  = "name"
	testKey     = "key"
	testValue   = "value"
	testEntry   = zapcore.Entry{
		Level:   zap.InfoLevel,
		Message: testMessage,
	}
)

func TestCore(t *testing.T) {
	rec := logtest.NewRecorder()
	zc := NewCore(loggerName, WithLoggerProvider(rec))
	logger := zap.New(zc)

	t.Run("Write", func(t *testing.T) {
		logger.Info(testMessage, zap.String(testKey, testValue))

		result := rec.Result()
		require.Len(t, result, 1)
		require.Len(t, result[logtest.Scope{Name: "name"}], 1)
		got := result[logtest.Scope{Name: "name"}][0]

		assert.Equal(t, testMessage, got.Body.AsString())
		assert.Equal(t, log.SeverityInfo, got.Severity)
		assert.Equal(t, zap.InfoLevel.String(), got.SeverityText)
		assert.Equal(t, []log.KeyValue{
			log.String(testKey, testValue),
		}, got.Attributes)
	})

	rec.Reset()

	t.Run("Write Context", func(t *testing.T) {
		ctx := context.Background()
		ctx = context.WithValue(ctx, testEntry, true)
		logger.Info(testMessage, zap.Any("ctx", ctx))

		got := rec.Result()
		records := got[logtest.Scope{Name: "name"}]
		require.Len(t, records, 1)
		assert.Equal(t, records[0].Context, ctx)
	})

	rec.Reset()

	t.Run("With Context", func(t *testing.T) {
		ctx := context.Background()
		ctx = context.WithValue(ctx, testEntry, false)
		childlogger := logger.With(zap.Reflect("ctx", ctx))
		childlogger.Info(testMessage)

		got := rec.Result()
		records := got[logtest.Scope{Name: "name"}]
		require.Len(t, records, 1)
		assert.Equal(t, records[0].Context, ctx)
	})

	rec.Reset()

	// test child logger with accumulated fields
	t.Run("With", func(t *testing.T) {
		testCases := [][]string{{"test1", "value1"}, {"test2", "value2"}}
		childlogger := logger.With(zap.String(testCases[0][0], testCases[0][1]))
		childlogger.Info(testMessage, zap.String(testCases[1][0], testCases[1][1]))

		result := rec.Result()
		require.Len(t, result, 1)
		require.Len(t, result[logtest.Scope{Name: "name"}], 1)
		got := result[logtest.Scope{Name: "name"}][0]

		assert.Equal(t, testMessage, got.Body.AsString())
		assert.Equal(t, log.SeverityInfo, got.Severity)
		assert.Equal(t, zap.InfoLevel.String(), got.SeverityText)
		assert.Equal(t, []log.KeyValue{
			log.String("test1", "value1"),
			log.String("test2", "value2"),
		}, got.Attributes)
	})

	rec.Reset()

	t.Run("Named", func(t *testing.T) {
		name := "my/pkg"
		childlogger := logger.Named(name)
		childlogger.Info(testMessage, zap.String(testKey, testValue))

		result := rec.Result()
		require.Len(t, result, 2)
		require.Len(t, result[logtest.Scope{Name: "my/pkg"}], 1)
		got := result[logtest.Scope{Name: "my/pkg"}][0]

		assert.Equal(t, testMessage, got.Body.AsString())
		assert.Equal(t, log.SeverityInfo, got.Severity)
		assert.Equal(t, zap.InfoLevel.String(), got.SeverityText)
		assert.Equal(t, []log.KeyValue{
			log.String(testKey, testValue),
		}, got.Attributes)
	})

	rec.Reset()

	t.Run("WithMultiple", func(t *testing.T) {
		testCases := [][]string{{"test1", "value1"}, {"test2", "value2"}, {"test3", "value3"}}
		childlogger := logger.With(zap.String(testCases[0][0], testCases[0][1]))
		childlogger2 := childlogger.With(zap.String(testCases[1][0], testCases[1][1]))
		childlogger2.Info(testMessage, zap.String(testCases[2][0], testCases[2][1]))

		result := rec.Result()
		require.Len(t, result, 2)
		require.Len(t, result[logtest.Scope{Name: "name"}], 1)
		got := result[logtest.Scope{Name: "name"}][0]

		assert.Equal(t, testMessage, got.Body.AsString())
		assert.Equal(t, log.SeverityInfo, got.Severity)
		assert.Equal(t, zap.InfoLevel.String(), got.SeverityText)
		assert.Equal(t, []log.KeyValue{
			log.String("test1", "value1"),
			log.String("test2", "value2"),
			log.String("test3", "value3"),
		}, got.Attributes)
	})
}

func TestCoreConcurrentSafe(t *testing.T) {
	rec := logtest.NewRecorder()
	zc := NewCore(loggerName, WithLoggerProvider(rec))
	logger := zap.New(zc)

	t.Run("Write", func(t *testing.T) {
		var wg sync.WaitGroup
		const n = 2
		wg.Add(n)
		ctx := context.Background()
		for i := 0; i < n; i++ {
			go func() {
				defer wg.Done()
				logger.Info(testMessage, zap.String(testKey, testValue), zap.Any("ctx", ctx))
			}()
		}
		wg.Wait()

		result := rec.Result()
		require.Len(t, result, 1)
		require.Len(t, result[logtest.Scope{Name: "name"}], 2)
		got := result[logtest.Scope{Name: "name"}][0]

		assert.Equal(t, testMessage, got.Body.AsString())
		assert.Equal(t, log.SeverityInfo, got.Severity)
		assert.Equal(t, zap.InfoLevel.String(), got.SeverityText)
		assert.Equal(t, []log.KeyValue{
			log.String(testKey, testValue),
		}, got.Attributes)
	})
}

func TestCoreEnabled(t *testing.T) {
	enabledFunc := func(c context.Context, param log.EnabledParameters) bool {
		return param.Severity >= log.SeverityInfo
	}

	r := logtest.NewRecorder(logtest.WithEnabledFunc(enabledFunc))
	logger := zap.New(NewCore(loggerName, WithLoggerProvider(r)))

	logger.Debug(testMessage)
	assert.Empty(t, r.Result()[logtest.Scope{Name: "name"}])

	if ce := logger.Check(zap.DebugLevel, testMessage); ce != nil {
		ce.Write()
	}
	assert.Empty(t, r.Result()[logtest.Scope{Name: "name"}])

	if ce := logger.Check(zap.InfoLevel, testMessage); ce != nil {
		ce.Write()
	}

	result := r.Result()
	require.Len(t, result, 1)
	require.Len(t, result[logtest.Scope{Name: "name"}], 1)
	got := result[logtest.Scope{Name: "name"}][0]

	assert.Equal(t, testMessage, got.Body.AsString())
	assert.Equal(t, log.SeverityInfo, got.Severity)
	assert.Equal(t, zap.InfoLevel.String(), got.SeverityText)
}

func TestCoreWithCaller(t *testing.T) {
	rec := logtest.NewRecorder()
	zc := NewCore(loggerName, WithLoggerProvider(rec))
	logger := zap.New(zc, zap.AddCaller())

	logger.Info(testMessage)
	result := rec.Result()
	require.Len(t, result, 1)
	require.Len(t, result[logtest.Scope{Name: "name"}], 1)
	got := result[logtest.Scope{Name: "name"}][0]

	assert.Equal(t, testMessage, got.Body.AsString())
	assert.Equal(t, log.SeverityInfo, got.Severity)
	assert.Equal(t, zap.InfoLevel.String(), got.SeverityText)

	assert.Len(t, got.Attributes, 3)
	assert.Equal(t, string(semconv.CodeFilepathKey), got.Attributes[0].Key)
	assert.Contains(t, got.Attributes[0].Value.AsString(), "core_test.go")

	assert.Equal(t, string(semconv.CodeLineNumberKey), got.Attributes[1].Key)
	assert.Positive(t, got.Attributes[1].Value.AsInt64())

	assert.Equal(t, string(semconv.CodeFunctionKey), got.Attributes[2].Key)
	assert.Positive(t, "go.opentelemetry.io/contrib/bridges/otelzap."+t.Name(), got.Attributes[2].Value.AsString())
}

func TestCoreWithStacktrace(t *testing.T) {
	rec := logtest.NewRecorder()
	zc := NewCore(loggerName, WithLoggerProvider(rec))
	logger := zap.New(zc, zap.AddStacktrace(zapcore.ErrorLevel))

	logger.Error(testMessage)
	result := rec.Result()
	require.Len(t, result, 1)
	require.Len(t, result[logtest.Scope{Name: "name"}], 1)
	got := result[logtest.Scope{Name: "name"}][0]

	assert.Equal(t, testMessage, got.Body.AsString())
	assert.Equal(t, log.SeverityError, got.Severity)
	assert.Equal(t, zap.ErrorLevel.String(), got.SeverityText)

	assert.Len(t, got.Attributes, 1)
	assert.Equal(t, string(semconv.CodeStacktraceKey), got.Attributes[0].Key)
	assert.NotEmpty(t, got.Attributes[0].Value.AsString())
}

func TestNewCoreConfiguration(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		r := logtest.NewRecorder()
		prev := global.GetLoggerProvider()
		defer global.SetLoggerProvider(prev)
		global.SetLoggerProvider(r)

		var h *Core
		require.NotPanics(t, func() { h = NewCore(loggerName) })
		require.NotNil(t, h.logger)
		require.Len(t, r.Result(), 1)

		want := logtest.Recording{
			logtest.Scope{Name: "name"}: nil,
		}
		logtest.AssertEqual(t, want, r.Result())
	})

	t.Run("Options", func(t *testing.T) {
		r := logtest.NewRecorder()
		var h *Core
		require.NotPanics(t, func() {
			h = NewCore(
				loggerName,
				WithLoggerProvider(r),
				WithVersion("1.0.0"),
				WithSchemaURL("url"),
				WithAttributes(attribute.String("testattr", "testval")),
			)
		})
		require.NotNil(t, h.logger)
		require.Len(t, r.Result(), 1)

		want := logtest.Recording{
			logtest.Scope{
				Name:       "name",
				Version:    "1.0.0",
				SchemaURL:  "url",
				Attributes: attribute.NewSet(attribute.String("testattr", "testval")),
			}: nil,
		}
		logtest.AssertEqual(t, want, r.Result())
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
