package zap

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/embedded"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

func NewTestOtelLogger(log log.Logger) zapcore.Core {
	return &OtelZapCore{
		logger: log,
	}
}

// Basic Logger Test and Child Logger test
func TestZapCore(t *testing.T) {
	spy := &spyLogger{}
	logger := zap.New(NewTestOtelLogger(spy))
	logger.Info(testBodyString, zap.Strings("key", []string{"1", "2"}))

	a := []interface{}{"1", "2"}
	assert.Equal(t, testBodyString, spy.Record.Body().AsString())
	assert.Equal(t, testSeverity, spy.Record.Severity())
	assert.Equal(t, 1, spy.Record.AttributesLen())
	spy.Record.WalkAttributes(func(kv log.KeyValue) bool {
		assert.Equal(t, "key", string(kv.Key))
		assert.Equal(t, a, value2Result(kv.Value))
		fmt.Println(value2Result(kv.Value))
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

// Test Inline Marshaler and Object Encoder
type addr struct {
	IP   string
	Port int
}

type request struct {
	URL    string
	Listen addr
	Remote addr
}

func (a addr) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("ip", a.IP)
	enc.AddInt("port", a.Port)
	return nil
}

func (r *request) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("url", r.URL)
	zap.Inline(r.Listen).AddTo(enc)
	return enc.AddObject("remote", r.Remote)
}

func TestObjectEncoder(t *testing.T) {
	spy := &spyLogger{}
	logger := zap.New(NewTestOtelLogger(spy))
	req := &request{
		URL:    "/test",
		Listen: addr{"127.0.0.1", 8080},
		Remote: addr{"127.0.0.1", 31200},
	}

	// expected int values are all of type int64
	expValue := map[string]any{
		"url":  "/test",
		"ip":   "127.0.0.1",
		"port": int64(8080),
		"remote": map[string]any{
			"ip":   "127.0.0.1",
			"port": int64(31200),
		},
	}

	logger.Info("new request, in nested object", zap.Object("req", req))
	spy.Record.WalkAttributes(func(kv log.KeyValue) bool {
		assert.Equal(t, "req", string(kv.Key))
		assert.Equal(t, expValue, value2Result(kv.Value))
		return true
	})
}

// Copied from field_test.go
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
// Uint are converted to int64
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
		{t: zapcore.ByteStringType, iface: []byte("foo"), want: []byte("foo")},
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
		// {t: zapcore.TimeType, i: 1000, iface: time.UTC, want: time.Unix(0, 1000).In(time.UTC)},
		// {t: zapcore.TimeType, i: 1000, want: time.Unix(0, 1000)},
		// All Uint types are converted to Int64
		{t: zapcore.Uint64Type, i: 42, want: int64(42)},
		{t: zapcore.Uint32Type, i: 42, want: int64(42)},
		{t: zapcore.Uint16Type, i: 42, want: int64(42)},
		{t: zapcore.Uint8Type, i: 42, want: int64(42)},
		{t: zapcore.UintptrType, i: 42, want: int64(42)},
		// {t: zapcore.ReflectType, iface: users(2), want: users(2)},
		// {t: zapcore.NamespaceType, want: map[string]interface{}{}},
		{t: zapcore.StringerType, iface: users(2), want: "2 users"},
		{t: zapcore.StringerType, iface: &obj{}, want: "obj"},
		{t: zapcore.StringerType, iface: (*obj)(nil), want: "nil obj"},
		//{t: zapcore.SkipType, want: interface{}(nil)},
		{t: zapcore.StringerType, iface: (*url.URL)(nil), want: "<nil>"},
		{t: zapcore.StringerType, iface: (*users)(nil), want: "<nil>"},
		{t: zapcore.ErrorType, iface: (*errObj)(nil), want: "<nil>"},
	}

	for _, tt := range tests {
		enc := NewOtelObjectEncoder(1)
		f := zapcore.Field{Key: "k", Type: tt.t, Integer: tt.i, Interface: tt.iface, String: tt.s}
		f.AddTo(enc)
		assert.Equal(t, tt.want, value2Result(enc.cur[0].Value), "Unexpected output from field %+v.", f)
	}
}

// converts value to result
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
