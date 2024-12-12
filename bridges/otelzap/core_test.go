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
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
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
		assert.Equal(t, zap.InfoLevel.String(), got.SeverityText())
		assert.Equal(t, 1, got.AttributesLen())
		got.WalkAttributes(func(kv log.KeyValue) bool {
			assert.Equal(t, testKey, kv.Key)
			assert.Equal(t, testValue, value2Result(kv.Value))
			return true
		})
	})

	rec.Reset()

	t.Run("Write Context", func(t *testing.T) {
		ctx := context.Background()
		ctx = context.WithValue(ctx, testEntry, true)
		logger.Info(testMessage, zap.Any("ctx", ctx))
		got := rec.Result()[0].Records[0]
		assert.Equal(t, got.Context(), ctx)
	})

	rec.Reset()

	t.Run("With Context", func(t *testing.T) {
		ctx := context.Background()
		ctx = context.WithValue(ctx, testEntry, false)
		childlogger := logger.With(zap.Reflect("ctx", ctx))
		childlogger.Info(testMessage)
		got := rec.Result()[0].Records[0]
		assert.Equal(t, got.Context(), ctx)
	})

	rec.Reset()

	// test child logger with accumulated fields
	t.Run("With", func(t *testing.T) {
		testCases := [][]string{{"test1", "value1"}, {"test2", "value2"}}
		childlogger := logger.With(zap.String(testCases[0][0], testCases[0][1]))
		childlogger.Info(testMessage, zap.String(testCases[1][0], testCases[1][1]))

		got := rec.Result()[0].Records[0]
		assert.Equal(t, testMessage, got.Body().AsString())
		assert.Equal(t, log.SeverityInfo, got.Severity())
		assert.Equal(t, zap.InfoLevel.String(), got.SeverityText())
		assert.Equal(t, 2, got.AttributesLen())

		index := 0
		got.WalkAttributes(func(kv log.KeyValue) bool {
			assert.Equal(t, testCases[index][0], kv.Key)
			assert.Equal(t, testCases[index][1], value2Result(kv.Value))
			index++
			return true
		})
	})

	rec.Reset()

	t.Run("Named", func(t *testing.T) {
		name := "my/pkg"
		childlogger := logger.Named(name)
		childlogger.Info(testMessage, zap.String(testKey, testValue))

		found := false
		for _, got := range rec.Result() {
			found = got.Name == name
			if found {
				break
			}
		}
		assert.True(t, found)
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
		assert.Equal(t, zap.InfoLevel.String(), got.SeverityText())
		assert.Equal(t, 3, got.AttributesLen())

		index := 0
		got.WalkAttributes(func(kv log.KeyValue) bool {
			assert.Equal(t, testCases[index][0], kv.Key)
			assert.Equal(t, testCases[index][1], value2Result(kv.Value))
			index++
			return true
		})
	})
}

func TestCoreEnabled(t *testing.T) {
	enabledFunc := func(c context.Context, param log.EnabledParameters) bool {
		return param.Severity >= log.SeverityInfo
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
	assert.Equal(t, zap.InfoLevel.String(), got.SeverityText())
}

func TestCoreWithCaller(t *testing.T) {
	rec := logtest.NewRecorder()
	zc := NewCore(loggerName, WithLoggerProvider(rec))
	logger := zap.New(zc, zap.AddCaller())

	logger.Info(testMessage)
	got := rec.Result()[0].Records[0]
	assert.Equal(t, testMessage, got.Body().AsString())
	assert.Equal(t, log.SeverityInfo, got.Severity())
	assert.Equal(t, zap.InfoLevel.String(), got.SeverityText())
	assert.Equal(t, 4, got.AttributesLen())
	got.WalkAttributes(func(kv log.KeyValue) bool {
		switch kv.Key {
		case string(semconv.CodeFilepathKey):
			assert.Contains(t, kv.Value.AsString(), "core_test.go")
		case string(semconv.CodeLineNumberKey):
			assert.Positive(t, kv.Value.AsInt64())
		case string(semconv.CodeFunctionKey):
			assert.Equal(t, t.Name(), kv.Value.AsString())
		case string(semconv.CodeNamespaceKey):
			assert.Equal(t, "go.opentelemetry.io/contrib/bridges/otelzap", kv.Value.AsString())
		default:
			assert.Fail(t, "unexpected attribute key", kv.Key)
		}
		return true
	})
}

func TestCoreWithStacktrace(t *testing.T) {
	rec := logtest.NewRecorder()
	zc := NewCore(loggerName, WithLoggerProvider(rec))
	logger := zap.New(zc, zap.AddStacktrace(zapcore.ErrorLevel))

	logger.Error(testMessage)
	got := rec.Result()[0].Records[0]
	assert.Equal(t, testMessage, got.Body().AsString())
	assert.Equal(t, log.SeverityError, got.Severity())
	assert.Equal(t, zap.ErrorLevel.String(), got.SeverityText())
	assert.Equal(t, 1, got.AttributesLen())
	got.WalkAttributes(func(kv log.KeyValue) bool {
		assert.Equal(t, string(semconv.CodeStacktraceKey), kv.Key)
		assert.NotEmpty(t, kv.Value.AsString())
		return true
	})
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

func TestSplitFuncName(t *testing.T) {
	testCases := []struct {
		fullFuncName  string
		wantFuncName  string
		wantNamespace string
	}{
		{
			fullFuncName:  "github.com/my/repo/pkg.foo",
			wantFuncName:  "foo",
			wantNamespace: "github.com/my/repo/pkg",
		},
		{
			// anonymous function
			fullFuncName:  "github.com/my/repo/pkg.foo.func5",
			wantFuncName:  "func5",
			wantNamespace: "github.com/my/repo/pkg.foo",
		},
		{
			fullFuncName:  "net/http.Get",
			wantFuncName:  "Get",
			wantNamespace: "net/http",
		},
		{
			fullFuncName:  "invalid",
			wantFuncName:  "",
			wantNamespace: "",
		},
		{
			fullFuncName:  ".",
			wantFuncName:  "",
			wantNamespace: "",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.fullFuncName, func(t *testing.T) {
			gotFuncName, gotNamespace := splitFuncName(tc.fullFuncName)
			assert.Equal(t, tc.wantFuncName, gotFuncName)
			assert.Equal(t, tc.wantNamespace, gotNamespace)
		})
	}
}

func BenchmarkCoreWrite(b *testing.B) {
	benchmarks := []struct {
		name   string
		fields []zapcore.Field
	}{
		{
			name: "10 fields",
			fields: []zapcore.Field{
				zap.Int16("a", 1),
				zap.String("k", "a"),
				zap.Bool("k", true),
				zap.Time("k", time.Unix(1000, 1000)),
				zap.Binary("k", []byte{1, 2}),
				zap.ByteString("k", []byte{1, 2}),
				zap.Object("k", loggable{true}),
				zap.Array("k", loggable{true}),
				zap.String("k", "a"),
				zap.Ints("k", []int{1, 2}),
			},
		},
		{
			name: "20 fields",
			fields: []zapcore.Field{
				zap.Int16("a", 1),
				zap.String("k", "a"),
				zap.Bool("k", true),
				zap.Time("k", time.Unix(1000, 1000)),
				zap.Binary("k", []byte{1, 2}),
				zap.ByteString("k", []byte{1, 2}),
				zap.Object("k", loggable{true}),
				zap.String("k", "a"),
				zap.Array("k", loggable{true}),
				zap.Ints("k", []int{1, 2}),
				zap.Int16("a", 1),
				zap.String("k", "a"),
				zap.Bool("k", true),
				zap.Time("k", time.Unix(1000, 1000)),
				zap.Binary("k", []byte{1, 2}),
				zap.ByteString("k", []byte{1, 2}),
				zap.Object("k", loggable{true}),
				zap.Array("k", loggable{true}),
				zap.String("k", "a"),
				zap.Ints("k", []int{1, 2}),
			},
		},
		{ // Benchmark with nested namespace
			name: "Namespace",
			fields: []zapcore.Field{
				zap.Namespace("a"),
				zap.Int16("a", 1),
				zap.String("k", "a"),
				zap.Bool("k", true),
				zap.Time("k", time.Unix(1000, 1000)),
				zap.Binary("k", []byte{1, 2}),
				zap.Namespace("b"),
				zap.Binary("k", []byte{1, 2}),
				zap.Object("k", loggable{true}),
				zap.String("k", "a"),
				zap.Array("k", loggable{true}),
				zap.Ints("k", []int{1, 2}),
			},
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			zc := NewCore(loggerName)
			b.ReportAllocs()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					err := zc.Write(testEntry, bm.fields)
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
			zc1 := zc.With(bm.fields)
			b.ReportAllocs()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					err := zc1.Write(testEntry, []zapcore.Field{})
					if err != nil {
						b.Errorf("Unexpected error: %v", err)
					}
				}
			})
		})
	}
}
