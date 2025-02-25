// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelzap

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/logtest"
)

var (
	ctx         = context.WithValue(context.Background(), testEntry, false)
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
	testCases := []struct {
		name string
		fn   func(l *zap.Logger)
		want logtest.Recording
	}{
		{
			name: "Info",
			fn: func(l *zap.Logger) {
				l.Info(testMessage, zap.String(testKey, testValue))
			},
			want: logtest.Recording{
				logtest.Scope{Name: loggerName}: []logtest.Record{
					{
						Context:      context.Background(),
						Severity:     log.SeverityInfo,
						SeverityText: zap.InfoLevel.String(),
						Body:         log.StringValue(testMessage),
						Attributes: []log.KeyValue{
							log.String(testKey, testValue),
						},
					},
				},
			},
		},
		{
			name: "InfoContextField",
			fn: func(l *zap.Logger) {
				l.Info(testMessage, zap.Any("ctx", ctx))
			},
			want: logtest.Recording{
				logtest.Scope{Name: loggerName}: []logtest.Record{
					{
						Context:      ctx,
						Severity:     log.SeverityInfo,
						SeverityText: zap.InfoLevel.String(),
						Body:         log.StringValue(testMessage),
					},
				},
			},
		},
		{
			name: "WithContextInfo",
			fn: func(l *zap.Logger) {
				l = l.With(zap.Reflect("ctx", ctx))
				l.Info(testMessage)
			},
			want: logtest.Recording{
				logtest.Scope{Name: loggerName}: []logtest.Record{
					{
						Context:      ctx,
						Severity:     log.SeverityInfo,
						SeverityText: zap.InfoLevel.String(),
						Body:         log.StringValue(testMessage),
					},
				},
			},
		},
		{
			name: "WithFields",
			fn: func(l *zap.Logger) {
				l = l.With(zap.String("foo", "bar"))
				l.Info(testMessage, zap.String("fizz", "buzz"))
			},
			want: logtest.Recording{
				logtest.Scope{Name: loggerName}: []logtest.Record{
					{
						Context:      context.Background(),
						Severity:     log.SeverityInfo,
						SeverityText: zap.InfoLevel.String(),
						Body:         log.StringValue(testMessage),
						Attributes: []log.KeyValue{
							log.String("foo", "bar"),
							log.String("fizz", "buzz"),
						},
					},
				},
			},
		},
		{
			name: "WithMultiple",
			fn: func(l *zap.Logger) {
				l = l.With(zap.String("foo", "bar"))
				l = l.With(zap.String("a", "b"))
				l.Info(testMessage, zap.String("fizz", "buzz"))
			},
			want: logtest.Recording{
				logtest.Scope{Name: loggerName}: []logtest.Record{
					{
						Context:      context.Background(),
						Severity:     log.SeverityInfo,
						SeverityText: zap.InfoLevel.String(),
						Body:         log.StringValue(testMessage),
						Attributes: []log.KeyValue{
							log.String("a", "b"),
							log.String("foo", "bar"),
							log.String("fizz", "buzz"),
						},
					},
				},
			},
		},
		{
			name: "Named",
			fn: func(l *zap.Logger) {
				l = l.Named("custom")
				l.Info(testMessage)
			},
			want: logtest.Recording{
				logtest.Scope{Name: "name"}: nil, // Acquired but not used.
				logtest.Scope{Name: "custom"}: []logtest.Record{
					{
						Context:      context.Background(),
						Severity:     log.SeverityInfo,
						SeverityText: zap.InfoLevel.String(),
						Body:         log.StringValue(testMessage),
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rec := logtest.NewRecorder()
			zc := NewCore(loggerName, WithLoggerProvider(rec))
			l := zap.New(zc)
			tc.fn(l)
			got := rec.Result()
			if err := validate(got, tc.want); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestCoreEnabled(t *testing.T) {
	enabledFunc := func(c context.Context, param log.EnabledParameters) bool {
		return param.Severity >= log.SeverityInfo
	}
	rec := logtest.NewRecorder(logtest.WithEnabledFunc(enabledFunc))
	logger := zap.New(NewCore(loggerName, WithLoggerProvider(rec)))

	logger.Debug(testMessage)

	if ce := logger.Check(zap.DebugLevel, testMessage); ce != nil {
		ce.Write()
	}

	if ce := logger.Check(zap.InfoLevel, testMessage); ce != nil {
		ce.Write()
	}

	want := logtest.Recording{
		logtest.Scope{Name: loggerName}: []logtest.Record{
			{
				Context:      context.Background(),
				Severity:     log.SeverityInfo,
				SeverityText: zap.InfoLevel.String(),
				Body:         log.StringValue(testMessage),
			},
		},
	}
	got := rec.Result()
	if err := validate(got, want); err != nil {
		t.Error(err)
	}
}

// func TestCoreWithCaller(t *testing.T) {
// 	rec := logtest.NewRecorder()
// 	zc := NewCore(loggerName, WithLoggerProvider(rec))
// 	logger := zap.New(zc, zap.AddCaller())

// 	logger.Info(testMessage)
// 	got := rec.Result()[0].Records[0]
// 	assert.Equal(t, testMessage, got.Body().AsString())
// 	assert.Equal(t, log.SeverityInfo, got.Severity())
// 	assert.Equal(t, zap.InfoLevel.String(), got.SeverityText())
// 	assert.Equal(t, 4, got.AttributesLen())
// 	got.WalkAttributes(func(kv log.KeyValue) bool {
// 		switch kv.Key {
// 		case string(semconv.CodeFilepathKey):
// 			assert.Contains(t, kv.Value.AsString(), "core_test.go")
// 		case string(semconv.CodeLineNumberKey):
// 			assert.Positive(t, kv.Value.AsInt64())
// 		case string(semconv.CodeFunctionKey):
// 			assert.Equal(t, t.Name(), kv.Value.AsString())
// 		case string(semconv.CodeNamespaceKey):
// 			assert.Equal(t, "go.opentelemetry.io/contrib/bridges/otelzap", kv.Value.AsString())
// 		default:
// 			assert.Fail(t, "unexpected attribute key", kv.Key)
// 		}
// 		return true
// 	})
// }

// func TestCoreWithStacktrace(t *testing.T) {
// 	rec := logtest.NewRecorder()
// 	zc := NewCore(loggerName, WithLoggerProvider(rec))
// 	logger := zap.New(zc, zap.AddStacktrace(zapcore.ErrorLevel))

// 	logger.Error(testMessage)
// 	got := rec.Result()[0].Records[0]
// 	assert.Equal(t, testMessage, got.Body().AsString())
// 	assert.Equal(t, log.SeverityError, got.Severity())
// 	assert.Equal(t, zap.ErrorLevel.String(), got.SeverityText())
// 	assert.Equal(t, 1, got.AttributesLen())
// 	got.WalkAttributes(func(kv log.KeyValue) bool {
// 		assert.Equal(t, string(semconv.CodeStacktraceKey), kv.Key)
// 		assert.NotEmpty(t, kv.Value.AsString())
// 		return true
// 	})
// }

// func TestNewCoreConfiguration(t *testing.T) {
// 	t.Run("Default", func(t *testing.T) {
// 		r := logtest.NewRecorder()
// 		prev := global.GetLoggerProvider()
// 		defer global.SetLoggerProvider(prev)
// 		global.SetLoggerProvider(r)

// 		var h *Core
// 		require.NotPanics(t, func() { h = NewCore(loggerName) })
// 		require.NotNil(t, h.logger)
// 		require.Len(t, r.Result(), 1)

// 		want := &logtest.ScopeRecords{Name: loggerName}
// 		got := r.Result()[0]
// 		assert.Equal(t, want, got)
// 	})

// 	t.Run("Options", func(t *testing.T) {
// 		r := logtest.NewRecorder()
// 		var h *Core
// 		require.NotPanics(t, func() {
// 			h = NewCore(
// 				loggerName,
// 				WithLoggerProvider(r),
// 				WithVersion("1.0.0"),
// 				WithSchemaURL("url"),
// 			)
// 		})
// 		require.NotNil(t, h.logger)
// 		require.Len(t, r.Result(), 1)

// 		want := &logtest.ScopeRecords{Name: loggerName, Version: "1.0.0", SchemaURL: "url"}
// 		got := r.Result()[0]
// 		assert.Equal(t, want, got)
// 	})
// }

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

func validate(got, want logtest.Recording) error {
	// Compare Context.
	cmpCtx := cmpopts.EquateComparable(context.Background(), ctx)
	// Ignore Timestamps.
	cmpStmps := cmpopts.IgnoreTypes(time.Time{})
	// Unordered compare of the key values.
	cmpKVs := cmpopts.SortSlices(func(a, b log.KeyValue) bool { return a.Key < b.Key })
	// Empty and nil collections are equal.
	cmpEpty := cmpopts.EquateEmpty()

	if diff := cmp.Diff(want, got, cmpCtx, cmpStmps, cmpKVs, cmpEpty); diff != "" {
		return fmt.Errorf("records mismatch (-want +got):\n%s", diff)
	}
	return nil
}
