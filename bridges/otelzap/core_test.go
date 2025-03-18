// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelzap

import (
	"context"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/log/logtest"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
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
							log.String("fizz", "buzz"),
							log.String("foo", "bar"),
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
			logtest.AssertEqual(t, got, tc.want, logtest.IgnoreTimestamp())
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
	logtest.AssertEqual(t, got, want, logtest.IgnoreTimestamp())
}

func TestCoreWithCaller(t *testing.T) {
	rec := logtest.NewRecorder()
	zc := NewCore(loggerName, WithLoggerProvider(rec))
	logger := zap.New(zc, zap.AddCaller())

	logger.Info(testMessage)

	attrs := recordedAttributes(t, rec.Result())
	var key string

	key = string(semconv.CodeFilepathKey)
	if v, ok := attrs[key]; !ok {
		t.Errorf("%q attribute is missing, got = %v", key, attrs)
	} else if !strings.Contains(v.AsString(), "core_test.go") {
		t.Errorf("%q attribute has bad value, got = %v", key, v)
	}

	key = string(semconv.CodeLineNumberKey)
	if v, ok := attrs[key]; !ok {
		t.Errorf("%q attribute is missing, got = %v", key, attrs)
	} else if v.AsInt64() <= 0 {
		t.Errorf("%q attribute is not a number, got = %v", key, v)
	}

	key = string(semconv.CodeFunctionKey)
	if v, ok := attrs[key]; !ok {
		t.Errorf("%q attribute is missing, got = %v", key, attrs)
	} else if want := t.Name(); v.AsString() != want {
		t.Errorf("%q attribute has bad value, got = %v, want = %q", key, v, want)
	}

	key = string(semconv.CodeNamespaceKey)
	if v, ok := attrs[key]; !ok {
		t.Errorf("%q attribute is missing, got = %v", key, attrs)
	} else if want := "go.opentelemetry.io/contrib/bridges/otelzap"; v.AsString() != want {
		t.Errorf("%q attribute has bad value, got = %v, want = %q", key, v, want)
	}
}

func TestCoreWithStacktrace(t *testing.T) {
	rec := logtest.NewRecorder()
	zc := NewCore(loggerName, WithLoggerProvider(rec))
	logger := zap.New(zc, zap.AddStacktrace(zapcore.ErrorLevel))

	logger.Error(testMessage)

	attrs := recordedAttributes(t, rec.Result())

	key := string(semconv.CodeStacktraceKey)
	if v, ok := attrs[key]; !ok {
		t.Errorf("%q attribute is missing, got = %v", key, attrs)
	} else if v.AsString() == "" {
		t.Fatalf("%q attribute does not contain a stacktrace, got = %v", key, v)
	}
}

func TestNewCoreConfiguration(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		rec := logtest.NewRecorder()
		prev := global.GetLoggerProvider()
		defer global.SetLoggerProvider(prev)
		global.SetLoggerProvider(rec)

		NewCore(loggerName)

		want := logtest.Recording{
			logtest.Scope{Name: loggerName}: nil,
		}
		got := rec.Result()
		logtest.AssertEqual(t, got, want)
	})

	t.Run("Options", func(t *testing.T) {
		rec := logtest.NewRecorder()
		NewCore(
			loggerName,
			WithLoggerProvider(rec),
			WithVersion("1.0.0"),
			WithSchemaURL("url"),
		)

		want := logtest.Recording{
			logtest.Scope{
				Name:      loggerName,
				Version:   "1.0.0",
				SchemaURL: "url",
			}: nil,
		}
		got := rec.Result()
		logtest.AssertEqual(t, got, want)
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
			if gotFuncName != tc.wantFuncName || gotNamespace != tc.wantNamespace {
				t.Errorf("splitFuncName(%q) = (%q, %q), want (%q, %q)",
					tc.fullFuncName, gotFuncName, gotNamespace, tc.wantFuncName, tc.wantNamespace)
			}
		})
	}
}

// recordedAttributes returns record's attributes when only a single record was emitted.
func recordedAttributes(t *testing.T, rec logtest.Recording) map[string]log.Value {
	t.Helper()

	if len(rec) != 1 {
		t.Fatalf("should have only 1 scope as a single logger is used, got = %v", rec)
	}

	var records []logtest.Record
	for _, v := range rec {
		records = v
	}
	if len(records) != 1 {
		t.Fatalf("should have 1 record, got = %v", records)
	}
	a := records[0].Attributes
	attrs := make(map[string]log.Value, len(a))
	for _, v := range a {
		attrs[v.Key] = v.Value
	}
	return attrs
}
