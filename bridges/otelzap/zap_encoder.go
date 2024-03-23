// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelzap // import "go.opentelemetry.io/contrib/bridges/otelzap"

import (
	"encoding/json"
	"strconv"
	"time"

	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/otel/log"
)

// this file implements object and array encoder - similar to memory encoder by zapcore.
var (
	_ zapcore.ObjectEncoder = (*OtelObjectEncoder)(nil)
	_ zapcore.ArrayEncoder  = (*sliceArrayEncoder)(nil)
)

type OtelObjectEncoder struct {
	// Fields contains the entire encoded log context.
	Fields []log.KeyValue
	// cur is a pointer to the namespace we're currently writing to.
	cur []log.KeyValue

	reflectval log.Value
	zapcore.Encoder
}

// NewOtelObjectEncoder creates otel encoder
func NewOtelObjectEncoder(len int) *OtelObjectEncoder {
	m := make([]log.KeyValue, len)
	return &OtelObjectEncoder{
		Fields: m,
		cur:    m,
	}
}

// AddArray implements ObjectEncoder.
func (m *OtelObjectEncoder) AddArray(key string, v zapcore.ArrayMarshaler) error {
	arr := &sliceArrayEncoder{elems: make([]log.Value, 0)}
	err := v.MarshalLogArray(arr)
	m.cur = append(m.cur, log.Slice(key, arr.elems...))
	return err
}

// AddObject implements ObjectEncoder.
func (m *OtelObjectEncoder) AddObject(k string, v zapcore.ObjectMarshaler) error {
	// fmt.Println(v, "inside object")
	newMap := NewOtelObjectEncoder(0) // min
	err := v.MarshalLogObject(newMap)
	m.cur = append(m.cur, log.Map(k, newMap.cur...))
	return err
}

// AddBinary implements ObjectEncoder.
func (m *OtelObjectEncoder) AddBinary(k string, v []byte) {
	m.cur = append(m.cur, log.Bytes(k, v))
}

// AddByteString implements ObjectEncoder.
func (m *OtelObjectEncoder) AddByteString(k string, v []byte) {
	m.cur = append(m.cur, log.String(k, string(v)))
}

// AddBool implements ObjectEncoder.
func (m *OtelObjectEncoder) AddBool(k string, v bool) {
	m.cur = append(m.cur, log.Bool(k, v))
}

// AddDuration implements ObjectEncoder.
func (m *OtelObjectEncoder) AddDuration(k string, v time.Duration) { m.AddInt64(k, v.Nanoseconds()) }

// AddComplex128 implements ObjectEncoder.
func (m *OtelObjectEncoder) AddComplex128(k string, v complex128) {
	stringValue := strconv.FormatComplex(v, 'f', -1, 64)
	m.cur = append(m.cur, log.String(k, stringValue))
}

func (m *OtelObjectEncoder) AddFloat64(k string, v float64) {
	m.cur = append(m.cur, log.Float64(k, v))
}

func (m *OtelObjectEncoder) AddInt64(k string, v int64) {
	m.cur = append(m.cur, log.Int64(k, v))
}

func (m *OtelObjectEncoder) AddInt(k string, v int) {
	m.cur = append(m.cur, log.Int(k, v))
}

func (m *OtelObjectEncoder) AddString(k string, v string) {
	m.cur = append(m.cur, log.String(k, v))
}

func (m *OtelObjectEncoder) AddUint64(k string, v uint64) {
	m.cur = append(m.cur,
		log.KeyValue{
			Key:   k,
			Value: assignUintValue(v),
		})
}

// AddReflected implements ObjectEncoder.
// It calls this func if interface cannot be mapped to supported zap types
// For ex: an array of arrays or Objects passed using zap.Any()
// this converts everything to a JSON string.
func (m *OtelObjectEncoder) AddReflected(k string, v interface{}) error {
	enc := json.NewEncoder(m)
	if err := enc.Encode(v); err != nil {
		return err
	}
	// fmt.Println(m.reflectval.AsString(), "inside reflect")
	m.AddString(k, m.reflectval.AsString())
	return nil
}

// Implements Write method to which json encoder writes to.
func (r *OtelObjectEncoder) Write(p []byte) (n int, err error) {
	r.reflectval = log.StringValue(string(p))
	return
}

// OpenNamespace implements ObjectEncoder.
func (m *OtelObjectEncoder) OpenNamespace(k string) {
	ns := make([]log.KeyValue, 0)
	m.cur = append(m.cur, log.String(k, "Namesspace"))
	m.cur = ns
}
func (m *OtelObjectEncoder) AddComplex64(k string, v complex64) { m.AddComplex128(k, complex128(v)) }

func (m *OtelObjectEncoder) AddFloat32(k string, v float32) {
	// preserves float32 value
	value, _ := strconv.ParseFloat(strconv.FormatFloat(float64(v), 'f', -1, 32), 64)
	m.AddFloat64(k, value)
}
func (m *OtelObjectEncoder) AddInt32(k string, v int32)     { m.AddInt64(k, int64(v)) }
func (m *OtelObjectEncoder) AddInt16(k string, v int16)     { m.AddInt64(k, int64(v)) }
func (m *OtelObjectEncoder) AddInt8(k string, v int8)       { m.AddInt64(k, int64(v)) }
func (m *OtelObjectEncoder) AddTime(k string, v time.Time)  { m.AddInt64(k, v.UnixNano()) }
func (m *OtelObjectEncoder) AddUint(k string, v uint)       { m.AddUint64(k, uint64(v)) }
func (m *OtelObjectEncoder) AddUint32(k string, v uint32)   { m.AddInt64(k, int64(v)) }
func (m *OtelObjectEncoder) AddUint16(k string, v uint16)   { m.AddInt64(k, int64(v)) }
func (m *OtelObjectEncoder) AddUint8(k string, v uint8)     { m.AddInt64(k, int64(v)) }
func (m *OtelObjectEncoder) AddUintptr(k string, v uintptr) { m.AddUint64(k, uint64(v)) }

func assignUintValue(v uint64) log.Value {
	const maxInt64 = ^uint64(0) >> 1
	if v > maxInt64 {
		value := strconv.FormatUint(v, 10)
		return log.StringValue(value)

	}
	return log.Int64Value(int64(v))

}

// sliceArrayEncoder implements zapcore.ArrayEncoder.
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
	// passing 0 here - we do not of object's length
	m := NewOtelObjectEncoder(0)
	err := v.MarshalLogObject(m)
	s.elems = append(s.elems, log.MapValue(m.cur...))
	return err
}

func (s *sliceArrayEncoder) AppendReflected(v interface{}) error {
	// s.elems = append(s.elems, v)
	return nil
}

func (s *sliceArrayEncoder) AppendBool(v bool)         { s.elems = append(s.elems, log.BoolValue(v)) }
func (s *sliceArrayEncoder) AppendByteString(v []byte) { s.elems = append(s.elems, log.BytesValue(v)) }

func (s *sliceArrayEncoder) AppendComplex128(v complex128) {
	stringValue := `"` + strconv.FormatComplex(v, 'f', -1, 64) + `"`
	s.elems = append(s.elems, log.StringValue(stringValue))
}

func (s *sliceArrayEncoder) AppendUint64(v uint64)   { s.elems = append(s.elems, assignUintValue(v)) }
func (s *sliceArrayEncoder) AppendFloat64(v float64) { s.elems = append(s.elems, log.Float64Value(v)) }
func (s *sliceArrayEncoder) AppendInt(v int)         { s.elems = append(s.elems, log.IntValue(v)) }
func (s *sliceArrayEncoder) AppendInt64(v int64)     { s.elems = append(s.elems, log.Int64Value(v)) }
func (s *sliceArrayEncoder) AppendString(v string)   { s.elems = append(s.elems, log.StringValue(v)) }

func (s *sliceArrayEncoder) AppendComplex64(v complex64)    { s.AppendComplex128(complex128(v)) }
func (s *sliceArrayEncoder) AppendDuration(v time.Duration) { s.AppendInt64(v.Nanoseconds()) }

func (s *sliceArrayEncoder) AppendFloat32(v float32) {
	// preserves float32 value
	value, _ := strconv.ParseFloat(strconv.FormatFloat(float64(v), 'f', -1, 32), 64)
	s.AppendFloat64(value)
}
func (s *sliceArrayEncoder) AppendInt32(v int32)     { s.AppendInt64(int64(v)) }
func (s *sliceArrayEncoder) AppendInt16(v int16)     { s.AppendInt64(int64(v)) }
func (s *sliceArrayEncoder) AppendInt8(v int8)       { s.AppendInt64(int64(v)) }
func (s *sliceArrayEncoder) AppendTime(v time.Time)  { s.AppendInt64(int64(v.UnixNano())) }
func (s *sliceArrayEncoder) AppendUint(v uint)       { s.AppendUint64(uint64(v)) }
func (s *sliceArrayEncoder) AppendUint32(v uint32)   { s.AppendInt64(int64(v)) }
func (s *sliceArrayEncoder) AppendUint16(v uint16)   { s.AppendInt64(int64(v)) }
func (s *sliceArrayEncoder) AppendUint8(v uint8)     { s.AppendInt64(int64(v)) }
func (s *sliceArrayEncoder) AppendUintptr(v uintptr) { s.AppendUint64(uint64(v)) }
