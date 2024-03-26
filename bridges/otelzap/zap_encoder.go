// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Copyright (c) 2016-2017 Uber Technologies, Inc.

package otelzap // import "go.opentelemetry.io/contrib/bridges/otelzap"

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/otel/log"
)

// this file implements object and array encoder - similar to memory encoder by zapcore.
var (
	_ zapcore.ObjectEncoder = (*ObjectEncoder)(nil)
	_ zapcore.ArrayEncoder  = (*ArrayEncoder)(nil)
)

// Object Encoder implements zapcore.ObjectEncoder.
// It encodes given fields to OTel attribute.
type ObjectEncoder struct {
	// cur is a pointer to the namespace we're currently writing to.
	cur []log.KeyValue

	reflectval log.Value
}

// NewObjectEncoder returns ObjectEncoder which maps zap fields to OTel attributes.
func NewObjectEncoder(len int) *ObjectEncoder {
	m := make([]log.KeyValue, 0, len)
	return &ObjectEncoder{
		cur: m,
	}
}

// Converts Array to logSlice using ArrayEncoder.
func (m *ObjectEncoder) AddArray(key string, v zapcore.ArrayMarshaler) error {
	// check if possible to get array length here - to avoid zero memory allocation
	arr := &ArrayEncoder{elems: make([]log.Value, 0)}
	err := v.MarshalLogArray(arr)
	fmt.Println(arr.elems[0].AsString())
	m.cur = append(m.cur, log.Slice(key, arr.elems...))
	return err
}

// Converts Object to logMap.
func (m *ObjectEncoder) AddObject(k string, v zapcore.ObjectMarshaler) error {
	fmt.Println("inside map")
	newMap := NewObjectEncoder(2) // min
	err := v.MarshalLogObject(newMap)
	m.cur = append(m.cur, log.Map(k, newMap.cur...))
	return err
}

// Converts Binary to logBytes.
func (m *ObjectEncoder) AddBinary(k string, v []byte) {
	m.cur = append(m.cur, log.Bytes(k, v))
}

// Converts ByteString to logString.
func (m *ObjectEncoder) AddByteString(k string, v []byte) {
	m.cur = append(m.cur, log.String(k, string(v)))
}

// Converts Bool to logBool.
func (m *ObjectEncoder) AddBool(k string, v bool) {
	m.cur = append(m.cur, log.Bool(k, v))
}

// Converts Duration to logInt.
func (m *ObjectEncoder) AddDuration(k string, v time.Duration) {
	m.AddInt64(k, v.Nanoseconds())
}

// Converts Complex128 to logString.
func (m *ObjectEncoder) AddComplex128(k string, v complex128) {
	stringValue := strconv.FormatComplex(v, 'f', -1, 64)
	m.cur = append(m.cur, log.String(k, stringValue))
}

// Converts Float64 to logFloat64.
func (m *ObjectEncoder) AddFloat64(k string, v float64) {
	m.cur = append(m.cur, log.Float64(k, v))
}

func (m *ObjectEncoder) AddInt64(k string, v int64) {
	m.cur = append(m.cur, log.Int64(k, v))
}

func (m *ObjectEncoder) AddInt(k string, v int) {
	m.cur = append(m.cur, log.Int(k, v))
}

func (m *ObjectEncoder) AddString(k string, v string) {
	m.cur = append(m.cur, log.String(k, v))
}

func (m *ObjectEncoder) AddUint64(k string, v uint64) {
	m.cur = append(m.cur,
		log.KeyValue{
			Key:   k,
			Value: assignUintValue(v),
		})
}

// It calls this method if interface cannot be mapped to supported zap types
// this converts everything to a JSON string.
func (m *ObjectEncoder) AddReflected(k string, v interface{}) error {
	enc := json.NewEncoder(m)
	if err := enc.Encode(v); err != nil {
		return err
	}
	m.AddString(k, m.reflectval.AsString())
	return nil
}

// Implements Write method to which json encoder writes to.
// Used by AddReflected method.
func (m *ObjectEncoder) Write(p []byte) (n int, err error) {
	m.reflectval = log.StringValue(string(p))
	return
}

// TODO:
// OpenNamespace opens an isolated namespace where all subsequent fields will
// be added. Applications can use namespaces to prevent key collisions when
// injecting loggers into sub-components or third-party libraries.
func (m *ObjectEncoder) OpenNamespace(k string) {
	// ns := make([]log.KeyValue, 0)
	// // m.cur expects both key and value
	// // "Namespace" as value here should be confirmed
	// m.cur = append(m.cur, log.String(k, "Namesspace"))
	// m.cur = ns
}
func (m *ObjectEncoder) AddComplex64(k string, v complex64) { m.AddComplex128(k, complex128(v)) }

func (m *ObjectEncoder) AddFloat32(k string, v float32) {
	// preserves float32 value
	value, _ := strconv.ParseFloat(strconv.FormatFloat(float64(v), 'f', -1, 32), 64)
	m.AddFloat64(k, value)
}
func (m *ObjectEncoder) AddInt32(k string, v int32)     { m.AddInt64(k, int64(v)) }
func (m *ObjectEncoder) AddInt16(k string, v int16)     { m.AddInt64(k, int64(v)) }
func (m *ObjectEncoder) AddInt8(k string, v int8)       { m.AddInt64(k, int64(v)) }
func (m *ObjectEncoder) AddTime(k string, v time.Time)  { m.AddInt64(k, v.UnixNano()) }
func (m *ObjectEncoder) AddUint(k string, v uint)       { m.AddUint64(k, uint64(v)) }
func (m *ObjectEncoder) AddUint32(k string, v uint32)   { m.AddInt64(k, int64(v)) }
func (m *ObjectEncoder) AddUint16(k string, v uint16)   { m.AddInt64(k, int64(v)) }
func (m *ObjectEncoder) AddUint8(k string, v uint8)     { m.AddInt64(k, int64(v)) }
func (m *ObjectEncoder) AddUintptr(k string, v uintptr) { m.AddUint64(k, uint64(v)) }

// assigns Uint values to OTel's log value.
func assignUintValue(v uint64) log.Value {
	const maxInt64 = ^uint64(0) >> 1
	if v > maxInt64 {
		value := strconv.FormatUint(v, 10)
		return log.StringValue(value)
	}
	return log.Int64Value(int64(v))
}

// ArrayEncoder implements zapcore.ArrayEncoder.
type ArrayEncoder struct {
	elems []log.Value
}

func (a *ArrayEncoder) AppendArray(v zapcore.ArrayMarshaler) error {
	enc := &ArrayEncoder{}
	err := v.MarshalLogArray(enc)
	a.elems = append(a.elems, enc.elems...)
	return err
}

func (a *ArrayEncoder) AppendObject(v zapcore.ObjectMarshaler) error {
	// passing 0 here - we do not know object's length
	// a minimum buffer capacity can be agreed upon?
	m := NewObjectEncoder(0)
	err := v.MarshalLogObject(m)
	a.elems = append(a.elems, log.MapValue(m.cur...))
	return err
}

func (a *ArrayEncoder) AppendReflected(v interface{}) error {
	enc := json.NewEncoder(a)
	if err := enc.Encode(v); err != nil {
		return err
	}
	return nil
}

func (a *ArrayEncoder) AppendBool(v bool)         { a.elems = append(a.elems, log.BoolValue(v)) }
func (a *ArrayEncoder) AppendByteString(v []byte) { a.elems = append(a.elems, log.BytesValue(v)) }

func (a *ArrayEncoder) AppendComplex128(v complex128) {
	stringValue := `"` + strconv.FormatComplex(v, 'f', -1, 64) + `"`
	a.elems = append(a.elems, log.StringValue(stringValue))
}

func (a *ArrayEncoder) AppendUint64(v uint64)   { a.elems = append(a.elems, assignUintValue(v)) }
func (a *ArrayEncoder) AppendFloat64(v float64) { a.elems = append(a.elems, log.Float64Value(v)) }
func (a *ArrayEncoder) AppendInt(v int)         { a.elems = append(a.elems, log.IntValue(v)) }
func (a *ArrayEncoder) AppendInt64(v int64)     { a.elems = append(a.elems, log.Int64Value(v)) }
func (a *ArrayEncoder) AppendString(v string)   { a.elems = append(a.elems, log.StringValue(v)) }

func (a *ArrayEncoder) AppendComplex64(v complex64)    { a.AppendComplex128(complex128(v)) }
func (a *ArrayEncoder) AppendDuration(v time.Duration) { a.AppendInt64(v.Nanoseconds()) }

func (a *ArrayEncoder) AppendFloat32(v float32) {
	// preserves float32 value
	value, _ := strconv.ParseFloat(strconv.FormatFloat(float64(v), 'f', -1, 32), 64)
	a.AppendFloat64(value)
}
func (a *ArrayEncoder) AppendInt32(v int32)     { a.AppendInt64(int64(v)) }
func (a *ArrayEncoder) AppendInt16(v int16)     { a.AppendInt64(int64(v)) }
func (a *ArrayEncoder) AppendInt8(v int8)       { a.AppendInt64(int64(v)) }
func (a *ArrayEncoder) AppendTime(v time.Time)  { a.AppendInt64(int64(v.UnixNano())) }
func (a *ArrayEncoder) AppendUint(v uint)       { a.AppendUint64(uint64(v)) }
func (a *ArrayEncoder) AppendUint32(v uint32)   { a.AppendInt64(int64(v)) }
func (a *ArrayEncoder) AppendUint16(v uint16)   { a.AppendInt64(int64(v)) }
func (a *ArrayEncoder) AppendUint8(v uint8)     { a.AppendInt64(int64(v)) }
func (a *ArrayEncoder) AppendUintptr(v uintptr) { a.AppendUint64(uint64(v)) }

// Implements Write method to which json encoder writes to.
// Used by AppendReflected method.
func (a *ArrayEncoder) Write(p []byte) (n int, err error) {
	a.elems = append(a.elems, log.StringValue(string(p)))
	return
}
