// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Copyright (c) 2016-2017 Uber Technologies, Inc.
package otelzap

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/embedded"
)

var (
	testBodyString = "log message"
	testSeverity   = log.SeverityInfo
)

type spyLogger struct {
	embedded.Logger
	Context context.Context
	Record  log.Record
}

func (l *spyLogger) Emit(ctx context.Context, r log.Record) {
	l.Context = ctx
	l.Record = r
}

func (l *spyLogger) Enabled(ctx context.Context, r log.Record) bool {
	return true
}

func NewTestOtelLogger(l log.Logger) zapcore.Core {
	return &Core{
		logger: l,
	}
}

type addr struct {
	IP   string
	Port int
}

// Basic Logger Test and Child Logger test.
func TestZapCore(t *testing.T) {
	spy := &spyLogger{}
	logger := zap.New(NewTestOtelLogger(spy))
	// logger.Info(testBodyString, zap.Any("key", []string{"1", "2"}))
	logger.Info(testBodyString, zap.Any("key", &addr{IP: "ip", Port: 1}))
	a := []interface{}{"1", "2"}
	// logger.Info("foo", zap.Any("bar", [][]string{{"a", "b"}, {"c", "d"}}))
	assert.Equal(t, testBodyString, spy.Record.Body().AsString())
	assert.Equal(t, testSeverity, spy.Record.Severity())
	assert.Equal(t, 1, spy.Record.AttributesLen())
	spy.Record.WalkAttributes(func(kv log.KeyValue) bool {
		assert.Equal(t, "b", string(kv.Key))
		assert.Equal(t, a, value2Result(kv.Value))

		return true
	})

	// test child logger
	childlogger := logger.With(zap.String("workplace", "otel"))
	childlogger.Info(testBodyString)
	spy.Record.WalkAttributes(func(kv log.KeyValue) bool {
		assert.Equal(t, "workplace", string(kv.Key))
		assert.Equal(t, "otel", kv.Value.AsString())
		return true
	})
}

// Copied from field_test.go. https://github.com/uber-go/zap/blob/b39f8b6b6a44d8371a87610be50cce58eeeaabcb/zapcore/field_test.go#L131
type users int

func (u users) String() string {
	return fmt.Sprintf("%d users", int(u))
}

func (u users) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if int(u) < 0 {
		return errors.New("too few users")
	}
	enc.AddInt("users", int(u))
	return nil
}

func (u users) MarshalLogArray(enc zapcore.ArrayEncoder) error {
	if int(u) < 0 {
		return errors.New("too few users")
	}
	for i := 0; i < int(u); i++ {
		enc.AppendString("user")
	}
	return nil
}

type obj struct {
	kind int
}

func (o *obj) String() string {
	if o == nil {
		return "nil obj"
	}

	if o.kind == 1 {
		panic("panic with string")
	} else if o.kind == 2 {
		panic(errors.New("panic with error"))
	} else if o.kind == 3 {
		// panic with an arbitrary object that causes a panic itself
		// when being converted to a string
		panic((*url.URL)(nil))
	}

	return "obj"
}

type errObj struct {
	kind   int
	errMsg string
}

func (eobj *errObj) Error() string {
	if eobj.kind == 1 {
		panic("panic in Error() method")
	}
	return eobj.errMsg
}

// NOTE:
// int, int8, int16, int32 types are converted to int64 by Otel's log
// Complex types are converted to string of complex values
// Uint are converted to int64.
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
		// All Uint types are converted to Int64
		{t: zapcore.Uint64Type, i: 42, want: int64(42)},
		{t: zapcore.Uint32Type, i: 42, want: int64(42)},
		{t: zapcore.Uint16Type, i: 42, want: int64(42)},
		{t: zapcore.Uint8Type, i: 42, want: int64(42)},
		{t: zapcore.UintptrType, i: 42, want: int64(42)},
		// Encode writes the JSON encoding of v to the stream,
		// followed by a newline character.
		{t: zapcore.ReflectType, iface: users(2), want: "2\n"},
		//{t: zapcore.NamespaceType, want: map[string]interface{}{}}, TODO
		{t: zapcore.StringerType, iface: users(2), want: "2 users"},
		{t: zapcore.StringerType, iface: &obj{}, want: "obj"},
		{t: zapcore.StringerType, iface: (*obj)(nil), want: "nil obj"},
		{t: zapcore.SkipType, want: nil},
		{t: zapcore.StringerType, iface: (*url.URL)(nil), want: "<nil>"},
		{t: zapcore.StringerType, iface: (*users)(nil), want: "<nil>"},
		{t: zapcore.ErrorType, iface: (*errObj)(nil), want: "<nil>"},
	}

	for _, tt := range tests {
		enc := NewObjectEncoder(1)
		f := zapcore.Field{Key: "k", Type: tt.t, Integer: tt.i, Interface: tt.iface, String: tt.s}
		f.AddTo(enc)
		fmt.Println(enc.Fields)
		if f.Type == zapcore.SkipType {
			assert.Nil(t, tt.want, enc.cur)
			continue
		}
		assert.Equal(t, tt.want, value2Result(enc.cur[0].Value), "Unexpected output from field %+v.", f)
	}
}

func TestInlineMarshaler(t *testing.T) {
	enc := NewObjectEncoder(3)

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

// converts value to result.
func value2Result(v log.Value) any {
	switch v.Kind() {
	case log.KindBool:
		return v.AsBool()
	case log.KindFloat64:
		return v.AsFloat64()
	case log.KindInt64:
		return v.AsInt64()
	case log.KindString:
		return v.AsString()
	case log.KindBytes:
		return v.AsBytes()
	case log.KindSlice:
		var s []any
		for _, val := range v.AsSlice() {
			s = append(s, value2Result(val))
		}
		return s
	case log.KindMap:
		m := make(map[string]any)
		for _, val := range v.AsMap() {
			m[val.Key] = value2Result(val.Value)
		}
		return m
	}
	return nil
}

// benchmark logging
// complex attributes take longer time.
func BenchmarkZapLogging(b *testing.B) {
	// var (
	// 	z   zapcore.Core
	// 	err error
	// )
	// testBody := "log message"
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

	// entry := zapcore.Entry{
	// 	Level:   zap.InfoLevel,
	// 	Message: testBodyString,
	// }
	// ctx := context.Background()
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			// zapcore := make([]zapcore.Core, b.N)
			// for i := 0; i < b.N; i++ {
			// 	zapcore[i] = NewOtelZapCore(nil)
			// }
			zapcore := zap.New(NewOtelZapCore(nil))
			b.ReportAllocs()
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				zapcore.Info(testBodyString, bm.field)
			}
		})
	}
}

func BenchmarkMultipleAttr(b *testing.B) {
	// testBody := "log message"
	benchmarks := []struct {
		name  string
		field []zapcore.Field
	}{
		{
			name: "With 3 fields",
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
		// {name: "String",
		// 	field: zap.String("k", "a"),
		// },
		// {name: "Time",
		// 	field: zap.Time("k", time.Unix(1000, 1000)),
		// },
		// {name: "Binary",
		// 	field: zap.Binary("k", []byte{1, 2}),
		// },
		// {name: "ByteString",
		// 	field: zap.ByteString("k", []byte("abc")),
		// },
		// {name: "Array",
		// 	field: zap.Ints("k", []int{1, 2}),
		// },
		// {name: "Object",
		// 	field: zap.Object("k", users(10)),
		// },
		// {name: "Map",
		// 	field: zap.Any("k", map[string]string{"a": "b"}),
		// },

		// {name: "Dict",
		// 	field: zap.Dict("k", zap.String("a", "b")),
		// },
	}

	// ctx := context.Background()
	for _, bm := range benchmarks {
		// entry := zapcore.Entry{
		// 	Level:   zap.InfoLevel,
		// 	Message: testBodyString,
		// }
		b.Run(bm.name, func(b *testing.B) {
			// zapLogger := make([]*zap.Logger, b.N)
			// for i := 0; i < b.N; i++ {
			// 	zapLogger[i] = zap.New(NewOtelZapCore(nil))
			// }
			zapcore := zap.New(NewOtelZapCore(nil))
			b.ReportAllocs()
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				zapcore.Info(testBodyString, bm.field...)
			}
		})
	}
}
