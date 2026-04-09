// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelzap

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/embedded"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/log/logtest"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

type capturedRecord struct {
	err   error
	attrs []log.KeyValue
}

type captureLogger struct {
	embedded.Logger
	mu      sync.Mutex
	records []capturedRecord
}

func (*captureLogger) Enabled(context.Context, log.EnabledParameters) bool { return true }

func (l *captureLogger) Emit(_ context.Context, record log.Record) {
	attrs := make([]log.KeyValue, 0, record.AttributesLen())
	record.WalkAttributes(func(kv log.KeyValue) bool {
		attrs = append(attrs, kv)
		return true
	})

	l.mu.Lock()
	defer l.mu.Unlock()
	l.records = append(l.records, capturedRecord{
		err:   record.Err(),
		attrs: attrs,
	})
}

func (l *captureLogger) lastRecord(t *testing.T) capturedRecord {
	t.Helper()
	l.mu.Lock()
	defer l.mu.Unlock()
	require.NotEmpty(t, l.records)
	return l.records[len(l.records)-1]
}

type captureProvider struct {
	embedded.LoggerProvider
	logger *captureLogger
}

func newCaptureProvider() *captureProvider {
	return &captureProvider{logger: &captureLogger{}}
}

func (p *captureProvider) Logger(string, ...log.LoggerOption) log.Logger {
	return p.logger
}

func TestCore(t *testing.T) {
	rec := logtest.NewRecorder()
	zc := NewCore(loggerName, WithLoggerProvider(rec))
	logger := zap.New(zc)

	t.Run("Write", func(t *testing.T) {
		t.Cleanup(rec.Reset)

		logger.Info(testMessage, zap.String(testKey, testValue))

		want := logtest.Recording{
			logtest.Scope{Name: loggerName}: {
				{
					Body:         log.StringValue(testMessage),
					Severity:     log.SeverityInfo,
					SeverityText: zap.InfoLevel.String(),
					Attributes: []log.KeyValue{
						log.String(testKey, testValue),
					},
				},
			},
		}
		logtest.AssertEqual(t, want, rec.Result(),
			logtest.Transform(func(r logtest.Record) logtest.Record {
				cp := r.Clone()
				cp.Context = nil           // Ignore context for comparison.
				cp.Timestamp = time.Time{} // Ignore timestamp for comparison.
				return cp
			}),
		)
	})

	t.Run("WriteError", func(t *testing.T) {
		t.Cleanup(rec.Reset)

		logger.Error(testMessage, zap.Error(errors.New("test error")))

		want := logtest.Recording{
			logtest.Scope{Name: loggerName}: {
				{
					Body:         log.StringValue(testMessage),
					Severity:     log.SeverityError,
					SeverityText: zap.ErrorLevel.String(),
					Attributes:   []log.KeyValue{},
				},
			},
		}
		logtest.AssertEqual(t, want, rec.Result(),
			logtest.Transform(func(r logtest.Record) logtest.Record {
				cp := r.Clone()
				cp.Context = nil
				cp.Timestamp = time.Time{}
				return cp
			}),
		)
	})

	t.Run("WriteNamedError", func(t *testing.T) {
		t.Cleanup(rec.Reset)

		logger.Error(testMessage, zap.NamedError("db", errors.New("test error")))

		want := logtest.Recording{
			logtest.Scope{Name: loggerName}: {
				{
					Body:         log.StringValue(testMessage),
					Severity:     log.SeverityError,
					SeverityText: zap.ErrorLevel.String(),
					Attributes: []log.KeyValue{
						log.String("db", "test error"),
					},
				},
			},
		}
		logtest.AssertEqual(t, want, rec.Result(),
			logtest.Transform(func(r logtest.Record) logtest.Record {
				cp := r.Clone()
				cp.Context = nil
				cp.Timestamp = time.Time{}
				return cp
			}),
		)
	})

	t.Run("WriteContext", func(t *testing.T) {
		t.Cleanup(rec.Reset)

		ctx := t.Context()
		ctx = context.WithValue(ctx, testEntry, true)
		logger.Info(testMessage, zap.Any("ctx", ctx))

		want := logtest.Recording{
			logtest.Scope{Name: loggerName}: {
				{
					Context:      ctx,
					Body:         log.StringValue(testMessage),
					Severity:     log.SeverityInfo,
					SeverityText: zap.InfoLevel.String(),
				},
			},
		}
		logtest.AssertEqual(t, want, rec.Result(),
			logtest.Transform(func(r logtest.Record) logtest.Record {
				cp := r.Clone()
				cp.Timestamp = time.Time{} // Ignore timestamp for comparison.
				return cp
			}),
		)
	})

	t.Run("WithContext", func(t *testing.T) {
		t.Cleanup(rec.Reset)

		ctx := t.Context()
		ctx = context.WithValue(ctx, testEntry, false)
		childlogger := logger.With(zap.Reflect("ctx", ctx))
		childlogger.Info(testMessage)

		want := logtest.Recording{
			logtest.Scope{Name: loggerName}: {
				{
					Context:      ctx,
					Body:         log.StringValue(testMessage),
					Severity:     log.SeverityInfo,
					SeverityText: zap.InfoLevel.String(),
				},
			},
		}
		logtest.AssertEqual(t, want, rec.Result(),
			logtest.Transform(func(r logtest.Record) logtest.Record {
				cp := r.Clone()
				cp.Timestamp = time.Time{} // Ignore timestamp for comparison.
				return cp
			}),
		)
	})

	t.Run("With", func(t *testing.T) {
		t.Cleanup(rec.Reset)

		l := logger.With(zap.String("test1", "value1"))
		l = l.With(zap.String("test2", "value2"))
		l.Info(testMessage, zap.String("test3", "value3"))

		want := logtest.Recording{
			logtest.Scope{Name: loggerName}: {
				{
					Body:         log.StringValue(testMessage),
					Severity:     log.SeverityInfo,
					SeverityText: zap.InfoLevel.String(),
					Attributes: []log.KeyValue{
						log.String("test1", "value1"),
						log.String("test2", "value2"),
						log.String("test3", "value3"),
					},
				},
			},
		}
		logtest.AssertEqual(t, want, rec.Result(),
			logtest.Transform(func(r logtest.Record) logtest.Record {
				cp := r.Clone()
				cp.Context = nil           // Ignore context for comparison.
				cp.Timestamp = time.Time{} // Ignore timestamp for comparison.
				return cp
			}),
		)
	})

	t.Run("Named", func(t *testing.T) {
		t.Cleanup(rec.Reset)

		name := "my/pkg"
		childlogger := logger.Named(name)
		childlogger.Info(testMessage, zap.String(testKey, testValue))

		want := logtest.Recording{
			logtest.Scope{Name: loggerName}: {},
			logtest.Scope{Name: name}: {
				{
					Body:         log.StringValue(testMessage),
					Severity:     log.SeverityInfo,
					SeverityText: zap.InfoLevel.String(),
					Attributes: []log.KeyValue{
						log.String(testKey, testValue),
					},
				},
			},
		}
		logtest.AssertEqual(t, want, rec.Result(),
			logtest.Transform(func(r logtest.Record) logtest.Record {
				cp := r.Clone()
				cp.Context = nil           // Ignore context for comparison.
				cp.Timestamp = time.Time{} // Ignore timestamp for comparison.
				return cp
			}),
		)
	})
}

func TestCoreErrorFieldSetErrBehavior(t *testing.T) {
	t.Run("ErrorFieldUsesSetErr", func(t *testing.T) {
		p := newCaptureProvider()
		logger := zap.New(NewCore(loggerName, WithLoggerProvider(p)))

		logger.Error(testMessage, zap.Error(errors.New("test error")))

		r := p.logger.lastRecord(t)
		require.EqualError(t, r.err, "test error")
		require.NotContains(t, r.attrs, log.String("error", "test error"))
		require.NotContains(t, r.attrs, log.String(string(semconv.ExceptionMessageKey), "test error"))
		require.NotContains(t, r.attrs, log.String(string(semconv.ExceptionTypeKey), "*errors.errorString"))
	})

	t.Run("NamedErrorDoesNotUseSetErr", func(t *testing.T) {
		p := newCaptureProvider()
		logger := zap.New(NewCore(loggerName, WithLoggerProvider(p)))

		logger.Error(testMessage, zap.NamedError("db.error", errors.New("db failure")))

		r := p.logger.lastRecord(t)
		require.NoError(t, r.err)
		require.Contains(t, r.attrs, log.String("db.error", "db failure"))
	})

	t.Run("NamedErrorDoesNotOverrideErrorField", func(t *testing.T) {
		p := newCaptureProvider()
		logger := zap.New(NewCore(loggerName, WithLoggerProvider(p)))

		logger.Error(
			testMessage,
			zap.Error(errors.New("top level error")),
			zap.NamedError("db.error", errors.New("db failure")),
		)

		r := p.logger.lastRecord(t)
		require.EqualError(t, r.err, "top level error")
		require.Contains(t, r.attrs, log.String("db.error", "db failure"))
		require.NotContains(t, r.attrs, log.String("error", "top level error"))
	})
}

func TestCoreWriteContextConcurrentSafe(t *testing.T) {
	rec := logtest.NewRecorder()
	zc := NewCore(loggerName, WithLoggerProvider(rec))
	logger := zap.New(zc)

	ctx := t.Context()
	ctx = context.WithValue(ctx, testEntry, true)

	var wg sync.WaitGroup
	wg.Go(func() {
		logger.Debug(testMessage, zap.Any("ctx", ctx))
	})
	wg.Go(func() {
		logger.Debug(testMessage, zap.Any("ctx", ctx))
	})
	wg.Wait()

	want := logtest.Recording{
		logtest.Scope{Name: loggerName}: {
			{
				Context:      ctx,
				Body:         log.StringValue(testMessage),
				Severity:     log.SeverityDebug,
				SeverityText: zap.DebugLevel.String(),
			},
			{
				Context:      ctx,
				Body:         log.StringValue(testMessage),
				Severity:     log.SeverityDebug,
				SeverityText: zap.DebugLevel.String(),
			},
		},
	}
	logtest.AssertEqual(t, want, rec.Result(),
		logtest.Transform(func(r logtest.Record) logtest.Record {
			cp := r.Clone()
			cp.Timestamp = time.Time{} // Ignore timestamp for comparison.
			return cp
		}),
	)
}

func TestCoreEnabled(t *testing.T) {
	enabledFunc := func(_ context.Context, param log.EnabledParameters) bool {
		return param.Severity >= log.SeverityInfo
	}

	rec := logtest.NewRecorder(logtest.WithEnabledFunc(enabledFunc))
	logger := zap.New(NewCore(loggerName, WithLoggerProvider(rec)))

	wantEmpty := logtest.Recording{
		logtest.Scope{Name: loggerName}: nil,
	}

	logger.Debug(testMessage)
	logtest.AssertEqual(t, wantEmpty, rec.Result(),
		logtest.Desc("Debug message should not be recorded"),
	)

	if ce := logger.Check(zap.DebugLevel, testMessage); ce != nil {
		ce.Write()
	}
	logtest.AssertEqual(t, wantEmpty, rec.Result(),
		logtest.Desc("Debug message should not be recorded"),
	)

	if ce := logger.Check(zap.InfoLevel, testMessage); ce != nil {
		ce.Write()
	}
	want := logtest.Recording{
		logtest.Scope{Name: loggerName}: {
			{
				Body:         log.StringValue(testMessage),
				Severity:     log.SeverityInfo,
				SeverityText: zap.InfoLevel.String(),
				Attributes:   []log.KeyValue{},
			},
		},
	}
	logtest.AssertEqual(t, want, rec.Result(),
		logtest.Transform(func(r logtest.Record) logtest.Record {
			cp := r.Clone()
			cp.Context = nil           // Ignore context for comparison.
			cp.Timestamp = time.Time{} // Ignore timestamp for comparison.
			return cp
		}),
	)
}

func TestCoreWithCaller(t *testing.T) {
	rec := logtest.NewRecorder()
	zc := NewCore(loggerName, WithLoggerProvider(rec))
	logger := zap.New(zc, zap.AddCaller())

	logger.Info(testMessage)
	want := logtest.Recording{
		logtest.Scope{Name: "name"}: {
			{
				Body:         log.StringValue(testMessage),
				Severity:     log.SeverityInfo,
				SeverityText: zap.InfoLevel.String(),
				Attributes: []log.KeyValue{
					log.String(string(semconv.CodeFilePathKey), "core_test.go"), // The real filepth will vary based on the test environment. However, it should end with "core_test.go".
					log.Int64(string(semconv.CodeLineNumberKey), 1),             // Line number will vary.
					log.String(string(semconv.CodeFunctionNameKey), "go.opentelemetry.io/contrib/bridges/otelzap."+t.Name()),
				},
			},
		},
	}
	logtest.AssertEqual(t, want, rec.Result(),
		logtest.Transform(func(r logtest.Record) logtest.Record {
			cp := r.Clone()
			cp.Context = nil           // Ignore context for comparison.
			cp.Timestamp = time.Time{} // Ignore timestamp for comparison.

			for i, attr := range cp.Attributes {
				if attr.Key == string(semconv.CodeLineNumberKey) {
					// Adjust the line number to be non-zero, as it will vary based on the test environment.
					cp.Attributes[i].Value = log.Int64Value(1) // Set to 1 for consistency in tests.
				}
				if attr.Key == string(semconv.CodeFilePathKey) && strings.HasSuffix(attr.Value.AsString(), "core_test.go") {
					// Trim the prefix, as it will vary based on the test environment.
					cp.Attributes[i].Value = log.StringValue("core_test.go")
				}
			}
			return cp
		}),
	)
}

func TestCoreWithStacktrace(t *testing.T) {
	rec := logtest.NewRecorder()
	zc := NewCore(loggerName, WithLoggerProvider(rec))
	logger := zap.New(zc, zap.AddStacktrace(zapcore.ErrorLevel))

	logger.Error(testMessage)

	want := logtest.Recording{
		logtest.Scope{Name: "name"}: {
			{
				Body:         log.StringValue(testMessage),
				Severity:     log.SeverityError,
				SeverityText: zap.ErrorLevel.String(),
				Attributes: []log.KeyValue{
					log.String(string(semconv.CodeStacktraceKey), "stacktrace"), // Stacktrace will vary based on the test environment.
				},
			},
		},
	}
	logtest.AssertEqual(t, want, rec.Result(),
		logtest.Transform(func(r logtest.Record) logtest.Record {
			cp := r.Clone()
			cp.Context = nil           // Ignore context for comparison.
			cp.Timestamp = time.Time{} // Ignore timestamp for comparison.
			for i, attr := range cp.Attributes {
				if attr.Key == string(semconv.CodeStacktraceKey) {
					// Adjust the stacktrace to be non-empty, as it will vary based on the test environment.
					cp.Attributes[i].Value = log.StringValue("stacktrace") // Set to a placeholder for consistency in tests.
				}
			}
			return cp
		}),
	)
}

func TestCoreWithExceptionStacktrace(t *testing.T) {
	rec := logtest.NewRecorder()
	zc := NewCore(loggerName, WithLoggerProvider(rec))
	logger := zap.New(zc, zap.AddStacktrace(zapcore.ErrorLevel))

	logger.Error(testMessage, zap.Error(errors.New("test error")))

	want := logtest.Recording{
		logtest.Scope{Name: "name"}: {
			{
				Body:         log.StringValue(testMessage),
				Severity:     log.SeverityError,
				SeverityText: zap.ErrorLevel.String(),
				Attributes: []log.KeyValue{
					log.String(string(semconv.ExceptionStacktraceKey), "stacktrace"),
				},
			},
		},
	}
	logtest.AssertEqual(t, want, rec.Result(),
		logtest.Transform(func(r logtest.Record) logtest.Record {
			cp := r.Clone()
			cp.Context = nil
			cp.Timestamp = time.Time{}
			for i, attr := range cp.Attributes {
				if attr.Key == string(semconv.ExceptionStacktraceKey) {
					cp.Attributes[i].Value = log.StringValue("stacktrace")
				}
			}
			return cp
		}),
	)
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

func TestCoreWithErrorStacktraceDefault(t *testing.T) {
	rec := logtest.NewRecorder()
	zc := NewCore(loggerName, WithLoggerProvider(rec))
	logger := zap.New(zc, zap.AddStacktrace(zapcore.ErrorLevel))

	logger.Error(testMessage, zap.Error(errors.New("test error")))

	want := logtest.Recording{
		logtest.Scope{Name: "name"}: {
			{
				Body:         log.StringValue(testMessage),
				Severity:     log.SeverityError,
				SeverityText: zap.ErrorLevel.String(),
				Attributes: []log.KeyValue{
					log.String(string(semconv.ExceptionStacktraceKey), "stacktrace"),
				},
			},
		},
	}
	logtest.AssertEqual(t, want, rec.Result(),
		logtest.Transform(func(r logtest.Record) logtest.Record {
			cp := r.Clone()
			cp.Context = nil
			cp.Timestamp = time.Time{}
			for i, attr := range cp.Attributes {
				if attr.Key == string(semconv.ExceptionStacktraceKey) {
					cp.Attributes[i].Value = log.StringValue("stacktrace")
				}
			}
			return cp
		}),
	)
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
