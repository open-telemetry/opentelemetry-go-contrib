// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Copyright (c) 2016-2017 Uber Technologies, Inc.

package otelzap // import "go.opentelemetry.io/contrib/bridges/otelzap"

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/otel/log"
)

// pool for array encoder
// used by AddArray() method only.
var _arrayEncoderPool = sync.Pool{
	New: func() interface{} {
		return &arrayEncoder{elems: make([]log.Value, 0)}
	},
}

func getArrayEncoder() *arrayEncoder {
	return _arrayEncoderPool.Get().(*arrayEncoder)
}

func putArrayEncoder(e *arrayEncoder) {
	e.elems = e.elems[:0]
	_arrayEncoderPool.Put(e)
}

// pool for object encoder
// used by AddObject() method only.
var _objectEncoderPool = sync.Pool{
	New: func() interface{} {
		return newObjectEncoder(0)
	},
}

func getObjectEncoder() *objectEncoder {
	return _objectEncoderPool.Get().(*objectEncoder)
}

func putObjectEncoder(e *objectEncoder) {
	e.cur = e.cur[:0]
	_objectEncoderPool.Put(e)
}

var (
	_ zapcore.ObjectEncoder = (*objectEncoder)(nil)
	_ zapcore.ArrayEncoder  = (*arrayEncoder)(nil)
)

// Object Encoder implements zapcore.ObjectEncoder.
// It encodes given fields to OTel attribute.
type objectEncoder struct {
	// cur is a pointer to the namespace we're currently writing to.
	cur        []log.KeyValue
	ctxfield   context.Context
	reflectval log.Value
}

// NewObjectEncoder returns an instance of ObjectEncoder.
func newObjectEncoder(len int) *objectEncoder {
	m := make([]log.KeyValue, 0, len)
	return &objectEncoder{
		cur: m,
	}
}

// Converts Array to logSlice using ArrayEncoder.
func (m *objectEncoder) AddArray(key string, v zapcore.ArrayMarshaler) error {
	// check if possible to get array length here - to avoid zero memory allocation
	arr := getArrayEncoder()
	err := v.MarshalLogArray(arr)
	m.cur = append(m.cur, log.Slice(key, arr.elems...))
	putArrayEncoder(arr)
	return err
}

// Converts Object to logMap.
func (m *objectEncoder) AddObject(k string, v zapcore.ObjectMarshaler) error {
	fmt.Println("inside object")
	newobj := getObjectEncoder()
	err := v.MarshalLogObject(newobj)
	m.cur = append(m.cur, log.Map(k, newobj.cur...))
	putObjectEncoder(newobj)
	return err
}

// Converts Binary to logBytes.
func (m *objectEncoder) AddBinary(k string, v []byte) {
	m.cur = append(m.cur, log.Bytes(k, v))
}

// Converts ByteString to logString.
func (m *objectEncoder) AddByteString(k string, v []byte) {
	m.cur = append(m.cur, log.String(k, string(v)))
}

// Converts Bool to logBool.
func (m *objectEncoder) AddBool(k string, v bool) {
	m.cur = append(m.cur, log.Bool(k, v))
}

// Converts Duration to logInt.
func (m *objectEncoder) AddDuration(k string, v time.Duration) {
	m.AddInt64(k, v.Nanoseconds())
}

// Converts Complex128 to logString.
func (m *objectEncoder) AddComplex128(k string, v complex128) {
	stringValue := strconv.FormatComplex(v, 'f', -1, 64)
	m.cur = append(m.cur, log.String(k, stringValue))
}

// Converts Float64 to logFloat64.
func (m *objectEncoder) AddFloat64(k string, v float64) {
	m.cur = append(m.cur, log.Float64(k, v))
}

// Converts Float32 to logFloat64.
func (m *objectEncoder) AddFloat32(k string, v float32) {
	m.AddFloat64(k, float64(v))
}

// Converts Int64 to logInt64.
func (m *objectEncoder) AddInt64(k string, v int64) {
	m.cur = append(m.cur, log.Int64(k, v))
}

// Converts Int to logInt.
func (m *objectEncoder) AddInt(k string, v int) {
	m.cur = append(m.cur, log.Int(k, v))
}

// Converts String to logString.
func (m *objectEncoder) AddString(k string, v string) {
	m.cur = append(m.cur, log.String(k, v))
}

func (m *objectEncoder) AddUint64(k string, v uint64) {
	m.cur = append(m.cur,
		log.KeyValue{
			Key:   k,
			Value: assignUintValue(v),
		})
}

// It calls this method if interface cannot be mapped to supported zap types
// converts everything to a JSON string.
func (m *objectEncoder) AddReflected(k string, v interface{}) error {
	fmt.Println("inside reflect")
	// Check if v is of type context.Context
	if ctx, ok := v.(context.Context); ok {
		// assign ctx
		m.ctxfield = ctx
		return nil
	}

	enc := json.NewEncoder(m)
	if err := enc.Encode(v); err != nil {
		return err
	}
	m.AddString(k, m.reflectval.AsString())
	return nil
}

// Implements Write method to which json encoder writes to.
// Used by AddReflected method.
func (m *objectEncoder) Write(p []byte) (n int, err error) {
	m.reflectval = log.StringValue(string(p))
	return
}

// TODO:
// OpenNamespace opens an isolated namespace where all subsequent fields will
// be added.
func (m *objectEncoder) OpenNamespace(k string) {
	// ns := make([]log.KeyValue, 0)
	// // m.cur expects both key and value
	// // "Namespace" as value here should be confirmed
	// m.cur = append(m.cur, log.String(k, "Namesspace"))
	// m.cur = ns
}

func (m *objectEncoder) AddComplex64(k string, v complex64) { m.AddComplex128(k, complex128(v)) }
func (m *objectEncoder) AddInt32(k string, v int32)         { m.AddInt64(k, int64(v)) }
func (m *objectEncoder) AddInt16(k string, v int16)         { m.AddInt64(k, int64(v)) }
func (m *objectEncoder) AddInt8(k string, v int8)           { m.AddInt64(k, int64(v)) }
func (m *objectEncoder) AddTime(k string, v time.Time)      { m.AddInt64(k, v.UnixNano()) }
func (m *objectEncoder) AddUint(k string, v uint)           { m.AddUint64(k, uint64(v)) }
func (m *objectEncoder) AddUint32(k string, v uint32)       { m.AddInt64(k, int64(v)) }
func (m *objectEncoder) AddUint16(k string, v uint16)       { m.AddInt64(k, int64(v)) }
func (m *objectEncoder) AddUint8(k string, v uint8)         { m.AddInt64(k, int64(v)) }
func (m *objectEncoder) AddUintptr(k string, v uintptr)     { m.AddUint64(k, uint64(v)) }

// ArrayEncoder implements [zapcore.ArrayEncoder].
type arrayEncoder struct {
	elems []log.Value
}

func (a *arrayEncoder) AppendArray(v zapcore.ArrayMarshaler) error {
	enc := &arrayEncoder{}
	err := v.MarshalLogArray(enc)
	a.elems = append(a.elems, enc.elems...)
	return err
}

func (a *arrayEncoder) AppendObject(v zapcore.ObjectMarshaler) error {
	// passing 0 here - we do not know object's length
	// a minimum buffer capacity can be agreed upon?
	m := newObjectEncoder(0)
	err := v.MarshalLogObject(m)
	a.elems = append(a.elems, log.MapValue(m.cur...))
	return err
}

func (a *arrayEncoder) AppendReflected(v interface{}) error {
	enc := json.NewEncoder(a)
	return enc.Encode(v)
}

func (a *arrayEncoder) AppendComplex128(v complex128) {
	stringValue := strconv.FormatComplex(v, 'f', -1, 64)
	a.elems = append(a.elems, log.StringValue(stringValue))
}

func (a *arrayEncoder) AppendFloat32(v float32) {
	a.AppendFloat64(float64(v))
}

func (a *arrayEncoder) AppendByteString(v []byte) {
	a.elems = append(a.elems, log.StringValue(string(v)))
}

func (a *arrayEncoder) AppendBool(v bool)              { a.elems = append(a.elems, log.BoolValue(v)) }
func (a *arrayEncoder) AppendUint64(v uint64)          { a.elems = append(a.elems, assignUintValue(v)) }
func (a *arrayEncoder) AppendFloat64(v float64)        { a.elems = append(a.elems, log.Float64Value(v)) }
func (a *arrayEncoder) AppendInt(v int)                { a.elems = append(a.elems, log.IntValue(v)) }
func (a *arrayEncoder) AppendInt64(v int64)            { a.elems = append(a.elems, log.Int64Value(v)) }
func (a *arrayEncoder) AppendString(v string)          { a.elems = append(a.elems, log.StringValue(v)) }
func (a *arrayEncoder) AppendComplex64(v complex64)    { a.AppendComplex128(complex128(v)) }
func (a *arrayEncoder) AppendDuration(v time.Duration) { a.AppendInt64(v.Nanoseconds()) }
func (a *arrayEncoder) AppendInt32(v int32)            { a.AppendInt64(int64(v)) }
func (a *arrayEncoder) AppendInt16(v int16)            { a.AppendInt64(int64(v)) }
func (a *arrayEncoder) AppendInt8(v int8)              { a.AppendInt64(int64(v)) }
func (a *arrayEncoder) AppendTime(v time.Time)         { a.AppendInt64(int64(v.UnixNano())) }
func (a *arrayEncoder) AppendUint(v uint)              { a.AppendUint64(uint64(v)) }
func (a *arrayEncoder) AppendUint32(v uint32)          { a.AppendInt64(int64(v)) }
func (a *arrayEncoder) AppendUint16(v uint16)          { a.AppendInt64(int64(v)) }
func (a *arrayEncoder) AppendUint8(v uint8)            { a.AppendInt64(int64(v)) }
func (a *arrayEncoder) AppendUintptr(v uintptr)        { a.AppendUint64(uint64(v)) }

// Implements Write method to which json encoder writes to.
// Used by AppendReflected method.
func (a *arrayEncoder) Write(p []byte) (n int, err error) {
	a.elems = append(a.elems, log.StringValue(string(p)))
	return
}

// assigns Uint values to OTel's log value.
func assignUintValue(v uint64) log.Value {
	const maxInt64 = ^uint64(0) >> 1
	if v > maxInt64 {
		value := strconv.FormatUint(v, 10)
		return log.StringValue(value)
	}
	return log.Int64Value(int64(v))
}
