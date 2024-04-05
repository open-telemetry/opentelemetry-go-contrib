// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Copyright (c) 2016-2017 Uber Technologies, Inc.

package otelzap // import "go.opentelemetry.io/contrib/bridges/otelzap"

import (
	"context"
	"math"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/otel/log"
)

var (
	testBodyString = "log message"
	testSeverity   = log.SeverityInfo
	entry          = zapcore.Entry{
		Level:   zap.InfoLevel,
		Message: testBodyString,
	}
	field = zap.String("key", "testValue")
)

// Basic Logger Test and Child Logger test.
func TestZapCore(t *testing.T) {
	rec := &recorder{}
	logger := zap.New(NewOTelZapCore(WithLoggerProvider(rec)))
	logger.Info(testBodyString, zap.String("key", "testValue"))

	assert.Equal(t, testBodyString, rec.Record.Body().AsString())
	assert.Equal(t, testSeverity, rec.Record.Severity())
	assert.Equal(t, 1, rec.Record.AttributesLen())
	rec.Record.WalkAttributes(func(kv log.KeyValue) bool {
		assert.Equal(t, "key", string(kv.Key))
		assert.Equal(t, "testValue", value2Result(kv.Value))
		return true
	})

	// test child logger with accumulated fields
	childlogger := logger.With(zap.String("workplace", "otel"))
	childlogger.Info(testBodyString)
	rec.Record.WalkAttributes(func(kv log.KeyValue) bool {
		assert.Equal(t, "workplace", string(kv.Key))
		assert.Equal(t, "otel", kv.Value.AsString())
		return true
	})
}

// Test conversion of Zap Level to OTel level.
func TestGetOTelLevel(t *testing.T) {
	tests := []struct {
		level       zapcore.Level
		expectedSev log.Severity
	}{
		{zapcore.DebugLevel, log.SeverityDebug},   // Expected value for DebugLevel
		{zapcore.InfoLevel, log.SeverityInfo},     // Expected value for InfoLevel
		{zapcore.WarnLevel, log.SeverityWarn},     // Expected value for WarnLevel
		{zapcore.ErrorLevel, log.SeverityError},   // Expected value for ErrorLevel
		{zapcore.DPanicLevel, log.SeverityFatal1}, // Expected value for DPanicLevel
		{zapcore.PanicLevel, log.SeverityFatal2},  // Expected value for PanicLevel
		{zapcore.FatalLevel, log.SeverityFatal3},  // Expected value for FatalLevel
		{zapcore.InvalidLevel, log.SeverityUndefined},
	}

	for _, test := range tests {
		result := getOtelLevel(test.level)
		if result != test.expectedSev {
			t.Errorf("For level %v, expected %v but got %v", test.level, test.expectedSev, result)
		}
	}
}

// Copied from field_test.go. https://github.com/uber-go/zap/blob/b39f8b6b6a44d8371a87610be50cce58eeeaabcb/zapcore/field_test.go#L131

// NOTE:
// int, int8, int16, int32 types are converted to int64
// Complex128 are converted to string of complex values
// Uint are converted to int64.
// Reflect Types are converted to JSON string.
func TestFields(t *testing.T) {
	tests := []struct {
		t     zapcore.FieldType
		i     int64
		s     string
		iface interface{}
		want  interface{}
	}{
		{t: zapcore.ArrayMarshalerType, iface: users(2), want: []any{"user", "user"}},
		{t: zapcore.ObjectMarshalerType, iface: users(2), want: map[string]interface{}{"users": int64(2)}},
		{t: zapcore.BoolType, i: 0, want: false},
		{t: zapcore.ByteStringType, iface: []byte("foo"), want: "foo"},
		{t: zapcore.Complex128Type, iface: 1 + 2i, want: "(1+2i)"},
		{t: zapcore.Complex64Type, iface: complex64(1 + 2i), want: "(1+2i)"},
		{t: zapcore.DurationType, i: 1000, want: int64(1000)},
		{t: zapcore.Float64Type, i: int64(math.Float64bits(3.14)), want: 3.14},
		{t: zapcore.Float32Type, i: int64(math.Float32bits(3.14)), want: 3.14},
		{t: zapcore.Int64Type, i: 42, want: int64(42)},
		{t: zapcore.Int32Type, i: 42, want: int64(42)},
		{t: zapcore.Int16Type, i: 42, want: int64(42)},
		{t: zapcore.Int8Type, i: 42, want: int64(42)},
		{t: zapcore.StringType, s: "foo", want: "foo"},
		{t: zapcore.TimeType, i: 1000, iface: time.UTC, want: int64(1000)},
		{t: zapcore.TimeType, i: 1000, want: int64(1000)},
		{t: zapcore.Uint64Type, i: 42, want: int64(42)},
		{t: zapcore.Uint32Type, i: 42, want: int64(42)},
		{t: zapcore.Uint16Type, i: 42, want: int64(42)},
		{t: zapcore.Uint8Type, i: 42, want: int64(42)},
		{t: zapcore.UintptrType, i: 42, want: int64(42)},
		// Encode writes the JSON encoding of v to the stream, followed by a newline character.
		{t: zapcore.ReflectType, iface: users(2), want: "2\n"},
		// Opens a new Namespace with the value in Key
		{t: zapcore.NamespaceType, want: "k"},
		{t: zapcore.StringerType, iface: users(2), want: "2 users"},
		{t: zapcore.StringerType, iface: &obj{}, want: "obj"},
		{t: zapcore.StringerType, iface: (*obj)(nil), want: "nil obj"},
		{t: zapcore.SkipType, want: nil},
		{t: zapcore.StringerType, iface: (*url.URL)(nil), want: "<nil>"},
		{t: zapcore.StringerType, iface: (*users)(nil), want: "<nil>"},
		{t: zapcore.ErrorType, iface: (*errObj)(nil), want: "<nil>"},
	}

	for _, tt := range tests {
		enc := newObjectEncoder(1)
		f := zapcore.Field{Key: "k", Type: tt.t, Integer: tt.i, Interface: tt.iface, String: tt.s}
		f.AddTo(enc)
		if f.Type == zapcore.SkipType {
			assert.Nil(t, tt.want, enc.cur)
			continue
		}
		if f.Type == zapcore.Float32Type {
			assert.InDelta(t, tt.want, value2Result(enc.cur[0].Value), 0.001)
			continue
		}

		assert.Equal(t, tt.want, value2Result(enc.cur[0].Value), "Unexpected output from field %+v.", f)
	}
}

// copied from field_test.go https://github.com/uber-go/zap/blob/b39f8b6b6a44d8371a87610be50cce58eeeaabcb/zapcore/field_test.go#L184
func TestInlineMarshaler(t *testing.T) {
	enc := newObjectEncoder(3)

	topLevelStr := zapcore.Field{Key: "k", Type: zapcore.StringType, String: "s"}
	topLevelStr.AddTo(enc)

	inlineObj := zapcore.Field{Key: "ignored", Type: zapcore.InlineMarshalerType, Interface: users(10)}
	inlineObj.AddTo(enc)

	nestedObj := zapcore.Field{Key: "nested", Type: zapcore.ObjectMarshalerType, Interface: users(11)}
	nestedObj.AddTo(enc)

	gotVal := make(map[string]interface{})
	for _, encc := range enc.cur {
		gotVal[encc.Key] = value2Result(encc.Value)
	}

	assert.Equal(t, map[string]interface{}{
		"k":     "s",
		"users": int64(10),
		"nested": map[string]interface{}{
			"users": int64(11),
		},
	}, gotVal)
}

// Benchmark on different Field types.
func BenchmarkZapWrite(b *testing.B) {
	benchmarks := []struct {
		name  string
		field zapcore.Field
	}{
		{
			name:  "Int",
			field: zap.Int16("a", 1),
		},
		{
			name:  "String",
			field: zap.String("k", "a"),
		},
		{
			name:  "Time",
			field: zap.Time("k", time.Unix(1000, 1000)),
		},
		{
			name:  "Binary",
			field: zap.Binary("k", []byte{1, 2}),
		},
		{
			name:  "ByteString",
			field: zap.ByteString("k", []byte("abc")),
		},
		{
			name:  "Array",
			field: zap.Ints("k", []int{1, 2}),
		},
		{
			name:  "Object",
			field: zap.Object("k", users(10)),
		},
		{
			name:  "Map",
			field: zap.Any("k", map[string]string{"a": "b"}),
		},

		{
			name:  "Dict",
			field: zap.Dict("k", zap.String("a", "b")),
		},
		{
			name:  "Context",
			field: Context("k", context.Background()),
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			zc := NewOTelZapCore()
			b.ReportAllocs()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					zc.Write(entry, []zapcore.Field{bm.field})
				}
			})
		})
	}
}

// Benchmark with multiple Fields.
func BenchmarkMultipleFields(b *testing.B) {
	benchmarks := []struct {
		name  string
		field []zapcore.Field
	}{
		{
			name: "With 10 fields",
			field: []zapcore.Field{
				zap.Int16("a", 1),
				zap.String("k", "a"),
				zap.String("k", "a"),
				zap.Time("k", time.Unix(1000, 1000)),
				zap.Binary("k", []byte{1, 2}),
				zap.Binary("k", []byte{1, 2}),
				zap.Object("k", users(10)),
				zap.String("k", "a"),
				zap.String("k", "a"),
				zap.String("k", "a"),
				zap.ByteString("k", []byte("abc")),
			},
		},
		{
			name: "With 20 fields",
			field: []zapcore.Field{
				zap.Int16("a", 1),
				zap.String("k", "a"),
				zap.String("k", "a"),
				zap.Time("k", time.Unix(1000, 1000)),
				zap.Binary("k", []byte{1, 2}),
				zap.Binary("k", []byte{1, 2}),
				zap.Object("k", users(10)),
				zap.String("k", "a"),
				zap.String("k", "a"),
				zap.String("k", "a"),
				zap.ByteString("k", []byte("abc")),
				zap.Int16("a", 1),
				zap.String("k", "a"),
				zap.String("k", "a"),
				zap.Time("k", time.Unix(1000, 1000)),
				zap.Binary("k", []byte{1, 2}),
				zap.Binary("k", []byte{1, 2}),
				zap.Object("k", users(10)),
				zap.String("k", "a"),
				zap.String("k", "a"),
				zap.String("k", "a"),
				zap.ByteString("k", []byte("abc")),
			},
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			zc := NewOTelZapCore()
			b.ReportAllocs()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					zc.Write(entry, bm.field)
				}
			})
		})
	}
}

func TestConcurrentSafe(t *testing.T) {
	h := NewOTelZapCore()

	const goroutineN = 10

	var wg sync.WaitGroup
	wg.Add(goroutineN)

	for i := 0; i < goroutineN; i++ {
		go func() {
			defer wg.Done()

			_ = h.Enabled(zapcore.DebugLevel)

			_ = h.Write(entry, []zapcore.Field{field})

			_ = h.With([]zapcore.Field{field})
		}()
	}
	wg.Wait()
}
