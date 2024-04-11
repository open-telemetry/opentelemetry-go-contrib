// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Copyright (c) 2016-2017 Uber Technologies, Inc.

package otelzap // import "go.opentelemetry.io/contrib/bridges/otelzap"

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/multierr"
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
	logger := zap.New(NewCore(WithLoggerProvider(rec)))
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

// // Benchmark on different Field types.
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
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			zc := NewCore()
			b.ReportAllocs()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					err := zc.Write(entry, []zapcore.Field{bm.field})
					if err != nil {
						b.Errorf("Unexpected error: %v", err)
					}
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
			name: "10 fields",
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
			},
		},
		{
			name: "20 fields",
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
			},
		},
	}

	b.Run("Core with 0 fields", func(b *testing.B) {
		zc := NewCore()
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				err := zc.Write(entry, []zapcore.Field{})
				if err != nil {
					b.Errorf("Unexpected error: %v", err)
				}
			}
		})
	})

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			zc := NewCore()
			b.ReportAllocs()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					err := zc.Write(entry, bm.field)
					if err != nil {
						b.Errorf("Unexpected error: %v", err)
					}
				}
			})
		})
	}

	for _, bm := range benchmarks {
		b.Run(fmt.Sprint("Core with", bm.name), func(b *testing.B) {
			zc := NewCore()
			zc = zc.With(bm.field)
			b.ReportAllocs()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					err := zc.Write(entry, []zapcore.Field{})
					if err != nil {
						b.Errorf("Unexpected error: %v", err)
					}
				}
			})
		})
	}
}

func TestConcurrentSafe(t *testing.T) {
	h := NewCore()

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

// Copied from field_test.go. https://github.com/uber-go/zap/blob/b39f8b6b6a44d8371a87610be50cce58eeeaabcb/zapcore/memory_encoder_test.go
func TestObjectEncoder(t *testing.T) {
	// Expected output of a turducken.
	wantTurducken := map[string]interface{}{
		"ducks": []interface{}{
			map[string]interface{}{"in": "chicken"},
			map[string]interface{}{"in": "chicken"},
		},
	}

	tests := []struct {
		desc     string
		f        func(zapcore.ObjectEncoder)
		expected interface{}
	}{
		{
			desc: "AddObject",
			f: func(e zapcore.ObjectEncoder) {
				assert.NoError(t, e.AddObject("k", loggable{true}), "Expected AddObject to succeed.")
			},
			expected: map[string]interface{}{"loggable": "yes"},
		},
		{
			desc: "AddObject (nested)",
			f: func(e zapcore.ObjectEncoder) {
				assert.NoError(t, e.AddObject("k", turducken{}), "Expected AddObject to succeed.")
			},
			expected: wantTurducken,
		},
		{
			desc: "AddArray",
			f: func(e zapcore.ObjectEncoder) {
				assert.NoError(t, e.AddArray("k", zapcore.ArrayMarshalerFunc(func(arr zapcore.ArrayEncoder) error {
					arr.AppendBool(true)
					arr.AppendBool(false)
					arr.AppendBool(true)
					return nil
				})), "Expected AddArray to succeed.")
			},
			expected: []interface{}{true, false, true},
		},
		{
			desc: "AddArray (nested)",
			f: func(e zapcore.ObjectEncoder) {
				assert.NoError(t, e.AddArray("k", turduckens(2)), "Expected AddArray to succeed.")
			},
			expected: []interface{}{wantTurducken, wantTurducken},
		},
		{
			desc:     "AddBinary",
			f:        func(e zapcore.ObjectEncoder) { e.AddBinary("k", []byte("foo")) },
			expected: []byte("foo"),
		},
		{
			desc:     "AddByteString",
			f:        func(e zapcore.ObjectEncoder) { e.AddByteString("k", []byte("foo")) },
			expected: "foo",
		},
		{
			desc:     "AddBool",
			f:        func(e zapcore.ObjectEncoder) { e.AddBool("k", true) },
			expected: true,
		},
		{
			desc:     "AddComplex128",
			f:        func(e zapcore.ObjectEncoder) { e.AddComplex128("k", 1+2i) },
			expected: "(1+2i)",
		},
		{
			desc:     "AddComplex64",
			f:        func(e zapcore.ObjectEncoder) { e.AddComplex64("k", 1+2i) },
			expected: "(1+2i)",
		},
		{
			desc:     "AddDuration",
			f:        func(e zapcore.ObjectEncoder) { e.AddDuration("k", time.Millisecond) },
			expected: int64(1000000),
		},
		{
			desc:     "AddFloat64",
			f:        func(e zapcore.ObjectEncoder) { e.AddFloat64("k", 3.14) },
			expected: 3.14,
		},
		{
			desc:     "AddFloat32",
			f:        func(e zapcore.ObjectEncoder) { e.AddFloat32("k", 3.14) },
			expected: float64(float32(3.14)),
		},
		{
			desc:     "AddInt",
			f:        func(e zapcore.ObjectEncoder) { e.AddInt("k", 42) },
			expected: int64(42),
		},
		{
			desc:     "AddInt64",
			f:        func(e zapcore.ObjectEncoder) { e.AddInt64("k", 42) },
			expected: int64(42),
		},
		{
			desc:     "AddInt32",
			f:        func(e zapcore.ObjectEncoder) { e.AddInt32("k", 42) },
			expected: int64(42),
		},
		{
			desc:     "AddInt16",
			f:        func(e zapcore.ObjectEncoder) { e.AddInt16("k", 42) },
			expected: int64(42),
		},
		{
			desc:     "AddInt8",
			f:        func(e zapcore.ObjectEncoder) { e.AddInt8("k", 42) },
			expected: int64(42),
		},
		{
			desc:     "AddString",
			f:        func(e zapcore.ObjectEncoder) { e.AddString("k", "v") },
			expected: "v",
		},
		{
			desc:     "AddTime",
			f:        func(e zapcore.ObjectEncoder) { e.AddTime("k", time.Unix(0, 100)) },
			expected: time.Unix(0, 100).UnixNano(),
		},
		{
			desc:     "AddUint32",
			f:        func(e zapcore.ObjectEncoder) { e.AddUint32("k", 42) },
			expected: int64(42),
		},
		{
			desc:     "AddUint16",
			f:        func(e zapcore.ObjectEncoder) { e.AddUint16("k", 42) },
			expected: int64(42),
		},
		{
			desc:     "AddUint8",
			f:        func(e zapcore.ObjectEncoder) { e.AddUint8("k", 42) },
			expected: int64(42),
		},
		{
			desc:     "AddUintptr",
			f:        func(e zapcore.ObjectEncoder) { e.AddUintptr("k", 42) },
			expected: int64(42),
		},
		{
			desc: "AddReflected",
			f: func(e zapcore.ObjectEncoder) {
				assert.NoError(t, e.AddReflected("k", map[string]interface{}{"foo": 5}), "Expected AddReflected to succeed.")
			},
			expected: "{\"foo\":5}\n",
		},
		{
			desc: "OpenNamespace",
			f: func(e zapcore.ObjectEncoder) {
				e.OpenNamespace("k")
				e.AddInt("foo", 1)
				e.OpenNamespace("middle")
				e.AddInt("foo", 2)
				e.OpenNamespace("inner")
				e.AddInt("foo", 3)
			},
			expected: map[string]interface{}{
				"foo": int64(1),
				"middle": map[string]interface{}{
					"foo": int64(2),
					"inner": map[string]interface{}{
						"foo": int64(3),
					},
				},
			},
		},
		{
			desc: "object (with nested namespace) then string",
			f: func(e zapcore.ObjectEncoder) {
				e.OpenNamespace("k")
				assert.NoError(t, e.AddObject("obj", maybeNamespace{true}))
				e.AddString("not-obj", "should-be-outside-obj")
			},
			expected: map[string]interface{}{
				"obj": map[string]interface{}{
					"obj-out": "obj-outside-namespace",
					"obj-namespace": map[string]interface{}{
						"obj-in": "obj-inside-namespace",
					},
				},
				"not-obj": "should-be-outside-obj",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			enc := newObjectEncoder(1)
			tt.f(enc)
			enc.getObjValue(enc.root)
			fmt.Println(enc.root.kv)
			assert.Equal(t, tt.expected, value2Result((enc.root.kv[0].Value)), "Unexpected encoder output.")
		})
	}
}

type turducken struct{}

func (t turducken) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	return enc.AddArray("ducks", zapcore.ArrayMarshalerFunc(func(arr zapcore.ArrayEncoder) error {
		for i := 0; i < 2; i++ {
			err := arr.AppendObject(zapcore.ObjectMarshalerFunc(func(inner zapcore.ObjectEncoder) error {
				inner.AddString("in", "chicken")
				return nil
			}))
			if err != nil {
				return err
			}
		}
		return nil
	}))
}

type turduckens int

func (t turduckens) MarshalLogArray(enc zapcore.ArrayEncoder) error {
	var err error
	tur := turducken{}
	for i := 0; i < int(t); i++ {
		err = multierr.Append(err, enc.AppendObject(tur))
	}
	return err
}

// maybeNamespace is an ObjectMarshaler that sometimes opens a namespace.
type maybeNamespace struct{ bool }

func (m maybeNamespace) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("obj-out", "obj-outside-namespace")
	if m.bool {
		enc.OpenNamespace("obj-namespace")
		enc.AddString("obj-in", "obj-inside-namespace")
	}
	return nil
}

type loggable struct{ bool }

func (l loggable) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if !l.bool {
		return errors.New("can't marshal")
	}
	enc.AddString("loggable", "yes")
	return nil
}

func (l loggable) MarshalLogArray(enc zapcore.ArrayEncoder) error {
	if !l.bool {
		return errors.New("can't marshal")
	}
	enc.AppendBool(true)
	return nil
}

func TestArrayEncoder(t *testing.T) {
	tests := []struct {
		desc     string
		f        func(zapcore.ArrayEncoder)
		expected interface{}
	}{
		// AppendObject and AppendArray are covered by the AddObject (nested) and
		// AddArray (nested) cases above.
		{"AppendBool", func(e zapcore.ArrayEncoder) { e.AppendBool(true) }, true},
		{"AppendByteString", func(e zapcore.ArrayEncoder) { e.AppendByteString([]byte("foo")) }, "foo"},
		{"AppendComplex128", func(e zapcore.ArrayEncoder) { e.AppendComplex128(1 + 2i) }, "(1+2i)"},
		{"AppendComplex64", func(e zapcore.ArrayEncoder) { e.AppendComplex64(1 + 2i) }, "(1+2i)"},
		{"AppendDuration", func(e zapcore.ArrayEncoder) { e.AppendDuration(time.Second) }, int64(1000000000)},
		{"AppendFloat64", func(e zapcore.ArrayEncoder) { e.AppendFloat64(3.14) }, 3.14},
		{"AppendFloat32", func(e zapcore.ArrayEncoder) { e.AppendFloat32(3.14) }, float64(float32(3.14))},
		{"AppendInt", func(e zapcore.ArrayEncoder) { e.AppendInt(42) }, int64(42)},
		{"AppendInt64", func(e zapcore.ArrayEncoder) { e.AppendInt64(42) }, int64(42)},
		{"AppendInt32", func(e zapcore.ArrayEncoder) { e.AppendInt32(42) }, int64(42)},
		{"AppendInt16", func(e zapcore.ArrayEncoder) { e.AppendInt16(42) }, int64(42)},
		{"AppendInt8", func(e zapcore.ArrayEncoder) { e.AppendInt8(42) }, int64(42)},
		{"AppendString", func(e zapcore.ArrayEncoder) { e.AppendString("foo") }, "foo"},
		{"AppendTime", func(e zapcore.ArrayEncoder) { e.AppendTime(time.Unix(0, 100)) }, time.Unix(0, 100).UnixNano()},
		{"AppendUint", func(e zapcore.ArrayEncoder) { e.AppendUint(42) }, int64(42)},
		{"AppendUint64", func(e zapcore.ArrayEncoder) { e.AppendUint64(42) }, int64(42)},
		{"AppendUint32", func(e zapcore.ArrayEncoder) { e.AppendUint32(42) }, int64(42)},
		{"AppendUint16", func(e zapcore.ArrayEncoder) { e.AppendUint16(42) }, int64(42)},
		{"AppendUint8", func(e zapcore.ArrayEncoder) { e.AppendUint8(42) }, int64(42)},
		{"AppendUintptr", func(e zapcore.ArrayEncoder) { e.AppendUintptr(42) }, int64(42)},
		{
			desc: "AppendReflected",
			f: func(e zapcore.ArrayEncoder) {
				assert.NoError(t, e.AppendReflected(map[string]interface{}{"foo": 5}))
			},
			expected: "{\"foo\":5}\n",
		},
		{
			desc: "AppendArray (arrays of arrays)",
			f: func(e zapcore.ArrayEncoder) {
				err := e.AppendArray(zapcore.ArrayMarshalerFunc(func(inner zapcore.ArrayEncoder) error {
					inner.AppendBool(true)
					inner.AppendBool(false)
					return nil
				}))
				assert.NoError(t, err)
			},
			expected: []interface{}{true, false},
		},
		{
			desc: "object (no nested namespace) then string",
			f: func(e zapcore.ArrayEncoder) {
				err := e.AppendArray(zapcore.ArrayMarshalerFunc(func(inner zapcore.ArrayEncoder) error {
					err := inner.AppendObject(maybeNamespace{false})
					inner.AppendString("should-be-outside-obj")
					return err
				}))
				assert.NoError(t, err)
			},
			expected: []interface{}{
				map[string]interface{}{
					"obj-out": "obj-outside-namespace",
				},
				"should-be-outside-obj",
			},
		},
		{
			desc: "object (with nested namespace) then string",
			f: func(e zapcore.ArrayEncoder) {
				err := e.AppendArray(zapcore.ArrayMarshalerFunc(func(inner zapcore.ArrayEncoder) error {
					err := inner.AppendObject(maybeNamespace{true})
					inner.AppendString("should-be-outside-obj")
					return err
				}))
				assert.NoError(t, err)
			},
			expected: []interface{}{
				map[string]interface{}{
					"obj-out": "obj-outside-namespace",
					"obj-namespace": map[string]interface{}{
						"obj-in": "obj-inside-namespace",
					},
				},
				"should-be-outside-obj",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			enc := newObjectEncoder(5)
			assert.NoError(t, enc.AddArray("k", zapcore.ArrayMarshalerFunc(func(arr zapcore.ArrayEncoder) error {
				tt.f(arr)
				tt.f(arr)
				return nil
			})), "Expected AddArray to succeed.")

			enc.getObjValue(enc.root)
			assert.Equal(t, []interface{}{tt.expected, tt.expected}, value2Result(enc.root.kv[0].Value), "Unexpected encoder output.")
		})
	}
}
