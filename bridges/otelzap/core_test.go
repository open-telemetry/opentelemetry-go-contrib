// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelzap

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/log/logtest"
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
		got := rec.Result()[0].Records[0]
		assert.Equal(t, testMessage, got.Body().AsString())
		assert.Equal(t, log.SeverityInfo, got.Severity())
		assert.Equal(t, 1, got.AttributesLen())
		got.WalkAttributes(func(kv log.KeyValue) bool {
			assert.Equal(t, testKey, string(kv.Key))
			assert.Equal(t, testValue, value2Result(kv.Value))
			return true
		})
	})

	rec.Reset()

	// TODO: Add WriteContext test case.
	// TODO: Add WithContext test case.

	// test child logger with accumulated fields
	t.Run("With", func(t *testing.T) {
		testCases := [][]string{{"test1", "value1"}, {"test2", "value2"}}
		childlogger := logger.With(zap.String(testCases[0][0], testCases[0][1]))
		childlogger.Info(testMessage, zap.String(testCases[1][0], testCases[1][1]))

		got := rec.Result()[0].Records[0]
		assert.Equal(t, testMessage, got.Body().AsString())
		assert.Equal(t, log.SeverityInfo, got.Severity())
		assert.Equal(t, 2, got.AttributesLen())

		index := 0
		got.WalkAttributes(func(kv log.KeyValue) bool {
			assert.Equal(t, testCases[index][0], string(kv.Key))
			assert.Equal(t, testCases[index][1], value2Result(kv.Value))
			index++
			return true
		})
	})

	rec.Reset()

	t.Run("WithMultiple", func(t *testing.T) {
		testCases := [][]string{{"test1", "value1"}, {"test2", "value2"}, {"test3", "value3"}}
		childlogger := logger.With(zap.String(testCases[0][0], testCases[0][1]))
		childlogger2 := childlogger.With(zap.String(testCases[1][0], testCases[1][1]))
		childlogger2.Info(testMessage, zap.String(testCases[2][0], testCases[2][1]))

		got := rec.Result()[0].Records[0]
		assert.Equal(t, testMessage, got.Body().AsString())
		assert.Equal(t, log.SeverityInfo, got.Severity())
		assert.Equal(t, 3, got.AttributesLen())

		index := 0
		got.WalkAttributes(func(kv log.KeyValue) bool {
			assert.Equal(t, testCases[index][0], string(kv.Key))
			assert.Equal(t, testCases[index][1], value2Result(kv.Value))
			index++
			return true
		})
	})
}

func TestCoreEnabled(t *testing.T) {
	enabledFunc := func(c context.Context, r log.Record) bool {
		return r.Severity() >= log.SeverityInfo
	}

	r := logtest.NewRecorder(logtest.WithEnabledFunc(enabledFunc))
	logger := zap.New(NewCore(loggerName, WithLoggerProvider(r)))

	logger.Debug(testMessage)
	assert.Empty(t, r.Result()[0].Records)

	if ce := logger.Check(zap.DebugLevel, testMessage); ce != nil {
		ce.Write()
	}
	assert.Empty(t, r.Result()[0].Records)

	if ce := logger.Check(zap.InfoLevel, testMessage); ce != nil {
		ce.Write()
	}
	require.Len(t, r.Result()[0].Records, 1)
	got := r.Result()[0].Records[0]
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
		require.NotPanics(t, func() { h = NewCore(loggerName) })
		require.NotNil(t, h.logger)
		require.Len(t, r.Result(), 1)

		want := &logtest.ScopeRecords{Name: loggerName}
		got := r.Result()[0]
		assert.Equal(t, want, got)
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
			)
		})
		require.NotNil(t, h.logger)
		require.Len(t, r.Result(), 1)

		want := &logtest.ScopeRecords{Name: loggerName, Version: "1.0.0", SchemaURL: "url"}
		got := r.Result()[0]
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

// Benchmark with multiple Fields.
func BenchmarkMultipleFields(b *testing.B) {
	benchmarks := []struct {
		name  string
		field []zapcore.Field
	}{
		{
			name: "10 fields",
			field: []zapcore.Field{
				zap.Int16("a", 1),
				zap.String("k", "a"),
				zap.Bool("k", true),
				zap.Time("k", time.Unix(1000, 1000)),
				zap.Binary("k", []byte{1, 2}),
				zap.Binary("k", []byte{1, 2}),
				zap.Object("k", loggable{true}),
				zap.String("k", "a"),
				zap.String("k", "a"),
				zap.Ints("k", []int{1, 2}),
			},
		},
		{
			name: "20 fields",
			field: []zapcore.Field{
				zap.Int16("a", 1),
				zap.String("k", "a"),
				zap.Bool("k", true),
				zap.Time("k", time.Unix(1000, 1000)),
				zap.Binary("k", []byte{1, 2}),
				zap.Binary("k", []byte{1, 2}),
				zap.Object("k", loggable{true}),
				zap.String("k", "a"),
				zap.String("k", "a"),
				zap.Ints("k", []int{1, 2}),
				zap.Int16("a", 1),
				zap.String("k", "a"),
				zap.String("k", "a"),
				zap.Time("k", time.Unix(1000, 1000)),
				zap.String("k", "a"),
				zap.Ints("k", []int{1, 2}),
				zap.Object("k", loggable{true}),
			},
		},
	}

	for _, bm := range benchmarks {
		b.Run(fmt.Sprint("Core Write", bm.name), func(b *testing.B) {
			zc := NewCore(loggerName)
			b.ReportAllocs()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					err := zc.Write(testentry, bm.field)
					if err != nil {
						b.Errorf("Unexpected error: %v", err)
					}
				}
			})
		})
	}

	for _, bm := range benchmarks {
		b.Run(fmt.Sprint("With", bm.name), func(b *testing.B) {
			zc := NewCore(loggerName)
			zc1 := zc.With(bm.field)
			b.ReportAllocs()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					err := zc1.Write(testentry, []zapcore.Field{})
					if err != nil {
						b.Errorf("Unexpected error: %v", err)
					}
				}
			})
		})
	}
}
