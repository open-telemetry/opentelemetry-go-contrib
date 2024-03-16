package zap

import (
	"fmt"
	"time"

	"go.opentelemetry.io/otel/log"
	"go.uber.org/zap/zapcore"
)

// not optimizes yet
// this file implements object and array encoder
type OtelObjectEncoder struct {
	zapcore.Encoder
	// cur is a pointer to the namespace we're currently writing to.
	cur []log.KeyValue
}

// NewMapObjectEncoder creates a new map-backed ObjectEncoder.
func NewOtelObjectEncoder() *OtelObjectEncoder {
	//m := make(map[string]log.Value)
	return &OtelObjectEncoder{}
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
	newMap := NewOtelObjectEncoder()
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
func (m OtelObjectEncoder) AddDuration(k string, v time.Duration) {
	m.cur = append(m.cur, log.KeyValue{Key: k, Value: log.BoolValue(true)})
}

// AddComplex128 implements ObjectEncoder.
func (m *OtelObjectEncoder) AddComplex128(k string, v complex128) {
	m.cur = append(m.cur, log.KeyValue{Key: k, Value: log.BoolValue(true)})
}

// AddComplex64 implements ObjectEncoder.
func (m *OtelObjectEncoder) AddComplex64(k string, v complex64) {
	m.cur = append(m.cur, log.KeyValue{Key: k, Value: log.BoolValue(true)})
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
func (m *OtelObjectEncoder) AddInt(k string, v int) { m.cur = append(m.cur, log.Int(k, v)) }

// AddInt64 implements ObjectEncoder.
func (m *OtelObjectEncoder) AddInt64(k string, v int64) { m.cur = append(m.cur, log.Int64(k, v)) }

// AddInt32 implements ObjectEncoder.
func (m *OtelObjectEncoder) AddInt32(k string, v int32) {
	m.cur = append(m.cur, log.Int64(k, int64(v)))
}

// AddInt16 implements ObjectEncoder.
func (m *OtelObjectEncoder) AddInt16(k string, v int16) {
	m.cur = append(m.cur, log.Int64(k, int64(v)))
}

// AddInt8 implements ObjectEncoder.
func (m *OtelObjectEncoder) AddInt8(k string, v int8) { m.cur = append(m.cur, log.Int64(k, int64(v))) }

// AddString implements ObjectEncoder.
func (m *OtelObjectEncoder) AddString(k string, v string) {
	fmt.Println(k)
	m.cur = append(m.cur, log.String(k, v))
}

// AddTime implements ObjectEncoder.
func (m OtelObjectEncoder) AddTime(k string, v time.Time) {
	m.cur = append(m.cur, log.KeyValue{Key: k, Value: log.BoolValue(true)})
}

// AddUint implements ObjectEncoder.
func (m *OtelObjectEncoder) AddUint(k string, v uint) {
	m.cur = append(m.cur, log.KeyValue{Key: k, Value: log.BoolValue(true)})
}

// AddUint64 implements ObjectEncoder.
func (m *OtelObjectEncoder) AddUint64(k string, v uint64) {
	m.cur = append(m.cur, log.KeyValue{Key: k, Value: log.BoolValue(true)})
}

// AddUint32 implements ObjectEncoder.
func (m *OtelObjectEncoder) AddUint32(k string, v uint32) {
	m.cur = append(m.cur, log.KeyValue{Key: k, Value: log.BoolValue(true)})
}

// AddUint16 implements ObjectEncoder.
func (m *OtelObjectEncoder) AddUint16(k string, v uint16) {
	m.cur = append(m.cur, log.KeyValue{Key: k, Value: log.BoolValue(true)})
}

// AddUint8 implements ObjectEncoder.
func (m *OtelObjectEncoder) AddUint8(k string, v uint8) {
	m.cur = append(m.cur, log.KeyValue{Key: k, Value: log.BoolValue(true)})
}

// AddUintptr implements ObjectEncoder.
func (m *OtelObjectEncoder) AddUintptr(k string, v uintptr) {
	m.cur = append(m.cur, log.KeyValue{Key: k, Value: log.BoolValue(true)})
}

// AddReflected implements ObjectEncoder.
func (m *OtelObjectEncoder) AddReflected(k string, v interface{}) error {
	m.cur = append(m.cur, log.KeyValue{Key: k, Value: log.BoolValue(true)})
	return nil
}

// OpenNamespace implements ObjectEncoder.
func (m *OtelObjectEncoder) OpenNamespace(k string) {

	m.cur = append(m.cur, log.KeyValue{Key: k, Value: log.BoolValue(true)})
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
	m := NewOtelObjectEncoder()
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

// TODO
func (s *sliceArrayEncoder) AppendComplex128(v complex128) {
	s.elems = append(s.elems, log.BoolValue(true))
}

// TODO
func (s *sliceArrayEncoder) AppendComplex64(v complex64) {
	s.elems = append(s.elems, log.BoolValue(true))
}

// TODO
func (s *sliceArrayEncoder) AppendDuration(v time.Duration) {
	s.elems = append(s.elems, log.BoolValue(true))
}
func (s *sliceArrayEncoder) AppendFloat64(v float64) { s.elems = append(s.elems, log.Float64Value(v)) }
func (s *sliceArrayEncoder) AppendFloat32(v float32) {
	s.elems = append(s.elems, log.Float64Value(float64(v)))
}
func (s *sliceArrayEncoder) AppendInt(v int)       { s.elems = append(s.elems, log.IntValue(v)) }
func (s *sliceArrayEncoder) AppendInt64(v int64)   { s.elems = append(s.elems, log.Int64Value(v)) }
func (s *sliceArrayEncoder) AppendInt32(v int32)   { s.elems = append(s.elems, log.Int64Value(int64(v))) }
func (s *sliceArrayEncoder) AppendInt16(v int16)   { s.elems = append(s.elems, log.Int64Value(int64(v))) }
func (s *sliceArrayEncoder) AppendInt8(v int8)     { s.elems = append(s.elems, log.Int64Value(int64(v))) }
func (s *sliceArrayEncoder) AppendString(v string) { s.elems = append(s.elems, log.StringValue(v)) }

// TO DO
func (s *sliceArrayEncoder) AppendTime(v time.Time) {
	s.elems = append(s.elems, log.Int64Value(int64(v.Nanosecond())))
}

// TO DO
func (s *sliceArrayEncoder) AppendUint(v uint) { s.elems = append(s.elems, log.Int64Value(int64(v))) }

// TO DO
func (s *sliceArrayEncoder) AppendUint64(v uint64) {
	s.elems = append(s.elems, log.Int64Value(int64(v)))
}

// TO DO
func (s *sliceArrayEncoder) AppendUint32(v uint32) {
	s.elems = append(s.elems, log.Int64Value(int64(v)))
}

// TO DO
func (s *sliceArrayEncoder) AppendUint16(v uint16) {
	s.elems = append(s.elems, log.Int64Value(int64(v)))
}

// TO DO
func (s *sliceArrayEncoder) AppendUint8(v uint8) { s.elems = append(s.elems, log.Int64Value(int64(v))) }

// TO DO
func (s *sliceArrayEncoder) AppendUintptr(v uintptr) {
	s.elems = append(s.elems, log.Int64Value(int64(v)))
}
