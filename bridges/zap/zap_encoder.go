package zap

import (
	"fmt"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/log"
	"go.uber.org/zap/zapcore"
)

// not optimizes yet
// this file implements object and array encoder
var (
	_ zapcore.ObjectEncoder = (*OtelObjectEncoder)(nil)
	_ zapcore.ArrayEncoder  = (*sliceArrayEncoder)(nil)
)

type OtelObjectEncoder struct {

	// cur is a pointer to the namespace we're currently writing to.
	cur []log.KeyValue
}

// NewMapObjectEncoder creates a new map-backed ObjectEncoder.
func NewOtelObjectEncoder(len int) *OtelObjectEncoder {
	return &OtelObjectEncoder{
		cur: make([]log.KeyValue, 0, len),
	}
}

// AddArray implements ObjectEncoder.
func (m *OtelObjectEncoder) AddArray(key string, v zapcore.ArrayMarshaler) error {
	arr := &sliceArrayEncoder{elems: make([]log.Value, 0)}
	err := v.MarshalLogArray(arr)
	fmt.Println(arr.elems, "from array")
	m.cur = append(m.cur, log.KeyValue{Key: key, Value: log.SliceValue(arr.elems...)})
	return err
}

// AddObject implements ObjectEncoder.
func (m *OtelObjectEncoder) AddObject(k string, v zapcore.ObjectMarshaler) error {
	newMap := NewOtelObjectEncoder(0)
	fmt.Println(k, "inside map object")
	err := v.MarshalLogObject(newMap)
	m.cur = append(m.cur, log.KeyValue{Key: k, Value: log.MapValue(newMap.cur...)})
	return err
}

// AddBinary implements ObjectEncoder.
func (m *OtelObjectEncoder) AddBinary(k string, v []byte) {
	m.cur = append(m.cur, log.KeyValue{Key: k, Value: log.BytesValue(v)})
}

// AddByteString implements ObjectEncoder.
func (m *OtelObjectEncoder) AddByteString(k string, v []byte) {
	m.cur = append(m.cur, log.KeyValue{Key: k, Value: log.BytesValue(v)})
}

// AddBool implements ObjectEncoder.
func (m *OtelObjectEncoder) AddBool(k string, v bool) {
	m.cur = append(m.cur, log.KeyValue{Key: k, Value: log.BoolValue(v)})
}

// AddDuration implements ObjectEncoder.
func (m *OtelObjectEncoder) AddDuration(k string, v time.Duration) {
	m.cur = append(m.cur, log.Int64(k, v.Nanoseconds()))
}

// AddComplex128 implements ObjectEncoder.
func (m *OtelObjectEncoder) AddComplex128(k string, v complex128) {
	stringValue := `"` + strconv.FormatComplex(v, 'f', -1, 64) + `"`
	m.cur = append(m.cur, log.String(k, stringValue))
}

// AddComplex64 implements ObjectEncoder.
func (m *OtelObjectEncoder) AddComplex64(k string, v complex64) {
	m.AddComplex128(k, complex128(v))
}

// AddFloat64 implements ObjectEncoder.
func (m *OtelObjectEncoder) AddFloat64(k string, v float64) {
	m.cur = append(m.cur, log.Float64(k, v))
}

// AddFloat32 implements ObjectEncoder.
func (m *OtelObjectEncoder) AddFloat32(k string, v float32) {
	m.cur = append(m.cur, log.Float64(k, float64(v)))
}

// AddInt implements ObjectEncoder.
func (m *OtelObjectEncoder) AddInt(k string, v int) {
	m.cur = append(m.cur, log.Int(k, v))
}

// AddInt64 implements ObjectEncoder.
func (m *OtelObjectEncoder) AddInt64(k string, v int64) {
	m.cur = append(m.cur, log.Int64(k, v))
}

// AddInt32 implements ObjectEncoder.
func (m *OtelObjectEncoder) AddInt32(k string, v int32) {
	m.cur = append(m.cur, log.Int64(k, int64(v)))
}

// AddInt16 implements ObjectEncoder.
func (m *OtelObjectEncoder) AddInt16(k string, v int16) {
	m.cur = append(m.cur, log.Int64(k, int64(v)))
}

// AddInt8 implements ObjectEncoder.
func (m *OtelObjectEncoder) AddInt8(k string, v int8) {
	m.cur = append(m.cur, log.Int64(k, int64(v)))
}

// AddString implements ObjectEncoder.
func (m *OtelObjectEncoder) AddString(k string, v string) {
	m.cur = append(m.cur, log.String(k, v))
}

// AddTime implements ObjectEncoder.
func (m *OtelObjectEncoder) AddTime(k string, v time.Time) {
	m.cur = append(m.cur, log.Int64(k, int64(v.Nanosecond())))
}

// AddUint64 implements ObjectEncoder.
func (m *OtelObjectEncoder) AddUint64(k string, v uint64) {
	m.cur = append(m.cur,
		log.KeyValue{
			Key:   k,
			Value: assignUintValue(v),
		})
}

// AddUint implements ObjectEncoder.
func (m *OtelObjectEncoder) AddUint(k string, v uint) { m.AddUint64(k, uint64(v)) }

// AddUint32 implements ObjectEncoder.
func (m *OtelObjectEncoder) AddUint32(k string, v uint32) { m.AddInt64(k, int64(v)) }

// AddUint16 implements ObjectEncoder.
func (m *OtelObjectEncoder) AddUint16(k string, v uint16) { m.AddInt64(k, int64(v)) }

// AddUint8 implements ObjectEncoder.
func (m *OtelObjectEncoder) AddUint8(k string, v uint8) { m.AddInt64(k, int64(v)) }

// AddUintptr implements ObjectEncoder.
func (m *OtelObjectEncoder) AddUintptr(k string, v uintptr) { m.AddUint64(k, uint64(v)) }

// AddReflected implements ObjectEncoder.
// It falls here if interface cannot be mapped to supported zap types
// For ex: an array of arrays
func (m *OtelObjectEncoder) AddReflected(k string, v interface{}) error {
	// don't know
	fmt.Println(k, v)
	return nil
}

// OpenNamespace implements ObjectEncoder.
func (m *OtelObjectEncoder) OpenNamespace(k string) {
	//
}

func assignUintValue(v uint64) log.Value {
	const maxInt64 = ^uint64(0) >> 1
	if v > maxInt64 {
		value := strconv.FormatUint(v, 10)
		return log.StringValue(value)

	} else {
		return log.Int64Value(int64(v))
	}
}

// can we choose to support only few type for array?
type sliceArrayEncoder struct {
	elems []log.Value
}

func (s *sliceArrayEncoder) AppendArray(v zapcore.ArrayMarshaler) error {
	enc := &sliceArrayEncoder{}
	err := v.MarshalLogArray(enc)
	s.elems = append(s.elems, enc.elems...)
	return err
}

func (s *sliceArrayEncoder) AppendObject(v zapcore.ObjectMarshaler) error {
	m := NewOtelObjectEncoder(0)
	err := v.MarshalLogObject(m)
	s.elems = append(s.elems, log.MapValue(m.cur...))
	return err
}

func (s *sliceArrayEncoder) AppendReflected(v interface{}) error {
	//s.elems = append(s.elems, v)
	return nil
}

func (s *sliceArrayEncoder) AppendBool(v bool)         { s.elems = append(s.elems, log.BoolValue(v)) }
func (s *sliceArrayEncoder) AppendByteString(v []byte) { s.elems = append(s.elems, log.BytesValue(v)) }

func (s *sliceArrayEncoder) AppendComplex128(v complex128) {
	stringValue := `"` + strconv.FormatComplex(v, 'f', -1, 64) + `"`
	s.elems = append(s.elems, log.StringValue(stringValue))
}

func (s *sliceArrayEncoder) AppendComplex64(v complex64) { s.AppendComplex128(complex128(v)) }

// TODO
func (s *sliceArrayEncoder) AppendDuration(v time.Duration) {
	s.AppendInt64(v.Nanoseconds())
}
func (s *sliceArrayEncoder) AppendFloat64(v float64) { s.elems = append(s.elems, log.Float64Value(v)) }
func (s *sliceArrayEncoder) AppendFloat32(v float32) { s.AppendFloat64(float64(v)) }
func (s *sliceArrayEncoder) AppendInt(v int)         { s.elems = append(s.elems, log.IntValue(v)) }
func (s *sliceArrayEncoder) AppendInt64(v int64)     { s.elems = append(s.elems, log.Int64Value(v)) }
func (s *sliceArrayEncoder) AppendInt32(v int32)     { s.elems = append(s.elems, log.Int64Value(int64(v))) }
func (s *sliceArrayEncoder) AppendInt16(v int16)     { s.elems = append(s.elems, log.Int64Value(int64(v))) }
func (s *sliceArrayEncoder) AppendInt8(v int8)       { s.elems = append(s.elems, log.Int64Value(int64(v))) }
func (s *sliceArrayEncoder) AppendString(v string)   { s.elems = append(s.elems, log.StringValue(v)) }
func (s *sliceArrayEncoder) AppendTime(v time.Time) {
	s.elems = append(s.elems, log.Int64Value(int64(v.Nanosecond())))
}

func (s *sliceArrayEncoder) AppendUint64(v uint64)   { s.elems = append(s.elems, assignUintValue(v)) }
func (s *sliceArrayEncoder) AppendUint(v uint)       { s.AppendUint64(uint64(v)) }
func (s *sliceArrayEncoder) AppendUint32(v uint32)   { s.AppendInt64(int64(v)) }
func (s *sliceArrayEncoder) AppendUint16(v uint16)   { s.AppendInt64(int64(v)) }
func (s *sliceArrayEncoder) AppendUint8(v uint8)     { s.AppendInt64(int64(v)) }
func (s *sliceArrayEncoder) AppendUintptr(v uintptr) { s.AppendUint64(uint64(v)) }
