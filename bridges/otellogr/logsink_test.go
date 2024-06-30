// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otellogr

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/logtest"
	"go.opentelemetry.io/otel/sdk/instrumentation"
)

type expectedRecord struct {
	Body       log.Value
	Severity   log.Severity
	Attributes []log.KeyValue
}

var now = time.Now()

func TestNewLogSinkConfiguration(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		var ls *LogSink
		assert.NotPanics(t, func() { ls = NewLogSink() })
		assert.NotNil(t, ls)
	})

	t.Run("with_options", func(t *testing.T) {
		rec := logtest.NewRecorder()
		wantScope := instrumentation.Scope{Name: "name", Version: "ver", SchemaURL: "url"}
		var ls *LogSink
		assert.NotPanics(t, func() {
			ls = NewLogSink(
				WithLoggerProvider(rec),
				WithInstrumentationScope(wantScope),
				WithLevelSeverity(func(i int) log.Severity {
					return log.SeverityFatal
				}),
			)
		})
		assert.NotNil(t, ls)
		assert.NotNil(t, ls.levelSeverity)
		assert.Equal(t, log.SeverityFatal, ls.levelSeverity(0))
	})
}

func TestLogSink(t *testing.T) {
	for _, tt := range []struct {
		name                string
		f                   func(*logr.Logger)
		expectedLoggerCount int
		expectedRecords     []expectedRecord
	}{
		{
			name: "info",
			f: func(l *logr.Logger) {
				l.Info("info message")
			},
			expectedLoggerCount: 1,
			expectedRecords: []expectedRecord{
				{
					Body:     log.StringValue("info message"),
					Severity: log.SeverityInfo,
				},
			},
		},
		{
			name: "info_multi_attrs",
			f: func(l *logr.Logger) {
				l.Info("msg",
					"struct", struct{ data int64 }{data: 1},
					"bool", true,
					"duration", time.Minute,
					"float64", 3.14159,
					"int64", -2,
					"string", "str",
					"time", now,
					"uint64", uint64(3),
				)
			},
			expectedLoggerCount: 1,
			expectedRecords: []expectedRecord{
				{
					Body:     log.StringValue("msg"),
					Severity: log.SeverityInfo,
					Attributes: []log.KeyValue{
						log.String("struct", "{data:1}"),
						log.Bool("bool", true),
						log.Int64("duration", 60_000_000_000),
						log.Float64("float64", 3.14159),
						log.Int64("int64", -2),
						log.String("string", "str"),
						log.Int64("time", now.UnixNano()),
						log.Int64("uint64", 3),
					},
				},
			},
		},
		{
			name: "error",
			f: func(l *logr.Logger) {
				l.Error(errors.New("test error"), "error message")
			},
			expectedLoggerCount: 1,
			expectedRecords: []expectedRecord{
				{
					Body:     log.StringValue("error message"),
					Severity: log.SeverityError,
					Attributes: []log.KeyValue{
						log.String(errorKey, "test error"),
					},
				},
			},
		},
		{
			name: "error_multi_attrs",
			f: func(l *logr.Logger) {
				l.Error(errors.New("error"), "msg",
					"struct", struct{ data int64 }{data: 1},
					"bool", true,
					"duration", time.Minute,
					"float64", 3.14159,
					"int64", -2,
					"string", "str",
					"time", now,
					"uint64", uint64(3),
				)
			},
			expectedLoggerCount: 1,
			expectedRecords: []expectedRecord{
				{
					Body:     log.StringValue("msg"),
					Severity: log.SeverityError,
					Attributes: []log.KeyValue{
						log.String(errorKey, "error"),
						log.String("struct", "{data:1}"),
						log.Bool("bool", true),
						log.Int64("duration", 60_000_000_000),
						log.Float64("float64", 3.14159),
						log.Int64("int64", -2),
						log.String("string", "str"),
						log.Int64("time", now.UnixNano()),
						log.Int64("uint64", 3),
					},
				},
			},
		},
		{
			name: "info_with_name",
			f: func(l *logr.Logger) {
				l.WithName("test").Info("info message with name")
			},
			expectedLoggerCount: 2,
			expectedRecords: []expectedRecord{
				{
					Body:     log.StringValue("info message with name"),
					Severity: log.SeverityInfo,
				},
			},
		},
		{
			name: "info_with_name_nested",
			f: func(l *logr.Logger) {
				l.WithName("test").WithName("test").Info("info message with name")
			},
			expectedLoggerCount: 3,
			expectedRecords: []expectedRecord{
				{
					Body:     log.StringValue("info message with name"),
					Severity: log.SeverityInfo,
				},
			},
		},
		{
			name: "info_with_attrs",
			f: func(l *logr.Logger) {
				l.WithValues("key", "value").Info("info message with attrs")
			},
			expectedLoggerCount: 1,
			expectedRecords: []expectedRecord{
				{
					Body:     log.StringValue("info message with attrs"),
					Severity: log.SeverityInfo,
					Attributes: []log.KeyValue{
						log.String("key", "value"),
					},
				},
			},
		},
		{
			name: "info_with_attrs_nested",
			f: func(l *logr.Logger) {
				l.WithValues("key1", "value1").Info("info message with attrs", "key2", "value2")
			},
			expectedLoggerCount: 1,
			expectedRecords: []expectedRecord{
				{
					Body:     log.StringValue("info message with attrs"),
					Severity: log.SeverityInfo,
					Attributes: []log.KeyValue{
						log.String("key1", "value1"),
						log.String("key2", "value2"),
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rec := logtest.NewRecorder()
			ls := NewLogSink(WithLoggerProvider(rec))
			l := logr.New(ls)
			tt.f(&l)

			require.Len(t, rec.Result(), tt.expectedLoggerCount)

			assert.Len(t, rec.Result()[tt.expectedLoggerCount-1].Records, len(tt.expectedRecords))
			for i, record := range rec.Result()[tt.expectedLoggerCount-1].Records {
				assert.Equal(t, tt.expectedRecords[i].Body, record.Body())
				assert.Equal(t, tt.expectedRecords[i].Severity, record.Severity())

				var attributes []log.KeyValue
				record.WalkAttributes(func(kv log.KeyValue) bool {
					attributes = append(attributes, kv)
					return true
				})
				assert.Equal(t, tt.expectedRecords[i].Attributes, attributes)
			}
		})
	}
}

func TestLogSinkWithName(t *testing.T) {
	rec := logtest.NewRecorder()
	ls := NewLogSink(WithLoggerProvider(rec))
	lsWithName := ls.WithName("test")
	require.NotEqual(t, ls, lsWithName)

	require.NotEqual(t, lsWithName, ls.WithName("test"))
}

func TestLogSinkEnabled(t *testing.T) {
	rec := logtest.NewRecorder(
		logtest.WithEnabledFunc(func(ctx context.Context, record log.Record) bool {
			return record.Severity() == log.SeverityInfo
		}),
	)
	ls := NewLogSink(
		WithLoggerProvider(rec),
		WithLevelSeverity(func(i int) log.Severity {
			switch i {
			case 0:
				return log.SeverityDebug
			default:
				return log.SeverityInfo
			}
		}),
	)

	assert.False(t, ls.Enabled(0))
	assert.True(t, ls.Enabled(1))
}

func TestConvertKVs(t *testing.T) {
	ctx := context.WithValue(context.Background(), "key", "value") // nolint: revive,staticcheck // test context

	for _, tt := range []struct {
		name string

		kvs         []any
		expectedKVs []log.KeyValue
		expectedCtx context.Context
	}{
		{
			name: "empty",
			kvs:  nil,
		},
		{
			name: "single_value",
			kvs:  []any{"key", "value"},
			expectedKVs: []log.KeyValue{
				log.String("key", "value"),
			},
		},
		{
			name: "multiple_values",
			kvs:  []any{"key1", "value1", "key2", "value2"},
			expectedKVs: []log.KeyValue{
				log.String("key1", "value1"),
				log.String("key2", "value2"),
			},
		},
		{
			name: "missing_value",
			kvs:  []any{"key1", "value1", "key2"},
			expectedKVs: []log.KeyValue{
				log.String("key1", "value1"),
				{Key: "key2", Value: log.Value{}},
			},
		},
		{
			name: "key_not_string",
			kvs:  []any{42, "value"},
			expectedKVs: []log.KeyValue{
				log.String("42", "value"),
			},
		},
		{
			name:        "context",
			kvs:         []any{"ctx", ctx, "key", "value"},
			expectedKVs: []log.KeyValue{log.String("key", "value")},
			expectedCtx: ctx,
		},
		{
			name:        "last_context",
			kvs:         []any{"key", context.Background(), "ctx", ctx},
			expectedKVs: []log.KeyValue{},
			expectedCtx: ctx,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, kvs := convertKVs(tt.kvs)
			assert.Equal(t, tt.expectedKVs, kvs)

			if tt.expectedCtx != nil {
				assert.Equal(t, tt.expectedCtx, ctx)
			}
		})
	}
}

func TestConvertValue(t *testing.T) {
	for _, tt := range []struct {
		name string

		value         any
		expectedValue log.Value
	}{
		{
			name:          "bool",
			value:         true,
			expectedValue: log.BoolValue(true),
		},
		{
			name:          "string",
			value:         "value",
			expectedValue: log.StringValue("value"),
		},
		{
			name:          "int",
			value:         10,
			expectedValue: log.Int64Value(10),
		},
		{
			name:          "int8",
			value:         int8(127),
			expectedValue: log.Int64Value(127),
		},
		{
			name:          "int16",
			value:         int16(32767),
			expectedValue: log.Int64Value(32767),
		},
		{
			name:          "int32",
			value:         int32(2147483647),
			expectedValue: log.Int64Value(2147483647),
		},
		{
			name:          "int64",
			value:         int64(9223372036854775807),
			expectedValue: log.Int64Value(9223372036854775807),
		},
		{
			name:          "uint",
			value:         uint(42),
			expectedValue: log.Int64Value(42),
		},
		{
			name:          "uint8",
			value:         uint8(255),
			expectedValue: log.Int64Value(255),
		},
		{
			name:          "uint16",
			value:         uint16(65535),
			expectedValue: log.Int64Value(65535),
		},
		{
			name:          "uint32",
			value:         uint32(4294967295),
			expectedValue: log.Int64Value(4294967295),
		},
		{
			name:          "uint64",
			value:         uint64(9223372036854775807),
			expectedValue: log.Int64Value(9223372036854775807),
		},
		{
			name:          "uint64-max",
			value:         uint64(18446744073709551615),
			expectedValue: log.StringValue("18446744073709551615"),
		},
		{
			name:          "uintptr",
			value:         uintptr(12345),
			expectedValue: log.Int64Value(12345),
		},
		{
			name:          "float64",
			value:         float64(3.14159),
			expectedValue: log.Float64Value(3.14159),
		},
		{
			name:          "time.Duration",
			value:         time.Second,
			expectedValue: log.Int64Value(1_000_000_000),
		},
		{
			name:          "complex64",
			value:         complex(float32(1), float32(2)),
			expectedValue: log.StringValue("(1+2i)"),
		},
		{
			name:          "complex128",
			value:         complex(float64(3), float64(4)),
			expectedValue: log.StringValue("(3+4i)"),
		},
		{
			name:          "time.Time",
			value:         now,
			expectedValue: log.Int64Value(now.UnixNano()),
		},
		{
			name:          "[]byte",
			value:         []byte("hello"),
			expectedValue: log.BytesValue([]byte("hello")),
		},
		{
			name:          "error",
			value:         errors.New("test error"),
			expectedValue: log.StringValue("test error"),
		},
		{
			name:          "error",
			value:         errors.New("test error"),
			expectedValue: log.StringValue("test error"),
		},
		{
			name:          "error-nested",
			value:         fmt.Errorf("test error: %w", errors.New("nested error")),
			expectedValue: log.StringValue("test error: nested error"),
		},
		{
			name:          "nil",
			value:         nil,
			expectedValue: log.Value{},
		},
		{
			name:          "nil_ptr",
			value:         (*int)(nil),
			expectedValue: log.Value{},
		},
		{
			name:          "int_ptr",
			value:         func() *int { i := 93; return &i }(),
			expectedValue: log.Int64Value(93),
		},
		{
			name:          "string_ptr",
			value:         func() *string { s := "hello"; return &s }(),
			expectedValue: log.StringValue("hello"),
		},
		{
			name:          "bool_ptr",
			value:         func() *bool { b := true; return &b }(),
			expectedValue: log.BoolValue(true),
		},
		{
			name:          "int_empty_array",
			value:         []int{},
			expectedValue: log.SliceValue([]log.Value{}...),
		},
		{
			name:  "int_array",
			value: []int{1, 2, 3},
			expectedValue: log.SliceValue([]log.Value{
				log.Int64Value(1),
				log.Int64Value(2),
				log.Int64Value(3),
			}...),
		},
		{
			name:  "key_value_map",
			value: map[string]int{"one": 1},
			expectedValue: log.MapValue(
				log.Int64("one", 1),
			),
		},
		{
			name:  "int_string_map",
			value: map[int]string{1: "one"},
			expectedValue: log.MapValue(
				log.String("1", "one"),
			),
		},
		{
			name:  "nested_map",
			value: map[string]map[string]int{"nested": {"one": 1}},
			expectedValue: log.MapValue(
				log.Map("nested",
					log.Int64("one", 1),
				),
			),
		},
		{
			name: "struct_key_map",
			value: map[struct{ Name string }]int{
				{Name: "John"}: 42,
			},
			expectedValue: log.MapValue(
				log.Int64("{Name:John}", 42),
			),
		},
		{
			name: "struct",
			value: struct {
				Name string
				Age  int
			}{
				Name: "John",
				Age:  42,
			},
			expectedValue: log.StringValue("{Name:John Age:42}"),
		},
		{
			name: "struct_ptr",
			value: &struct {
				Name string
				Age  int
			}{
				Name: "John",
				Age:  42,
			},
			expectedValue: log.StringValue("{Name:John Age:42}"),
		},
		{
			name:          "ctx",
			value:         context.Background(),
			expectedValue: log.StringValue("context.Background"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, convertValue(tt.value), tt.expectedValue)
		})
	}
}

func TestConvertValueFloat32(t *testing.T) {
	actual := convertValue(float32(3.14))
	expected := log.Float64Value(3.14)

	assert.InDelta(t, actual.AsFloat64(), expected.AsFloat64(), 0.0001)
}
