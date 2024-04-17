// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Copyright (c) 2016-2017 Uber Technologies, Inc.

package otelzap // import "go.opentelemetry.io/contrib/bridges/otelzap"

import (
	"encoding/json"
	"strconv"
	"sync"
	"time"

	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/otel/log"
)

// pool for array encoder.
var arrayEncoderPool = sync.Pool{
	New: func() interface{} {
		// console_encoder by zap uses capacity of 2
		return &arrayEncoder{elems: make([]log.Value, 0, 2)}
	},
}

func getArrayEncoder() (arr *arrayEncoder, free func()) {
	arr = arrayEncoderPool.Get().(*arrayEncoder)
	return arr, func() {
		// TODO: Limit the capacity of the slice
		arr.elems = arr.elems[:0]
		arrayEncoderPool.Put(arr)
	}
}

// pool for object encoder.
var objectEncoderPool = sync.Pool{
	New: func() interface{} {
		// console_encoder by zap uses capacity of 2
		return newObjectEncoder(2)
	},
}

func getObjectEncoder() (obj *objectEncoder, free func()) {
	obj = objectEncoderPool.Get().(*objectEncoder)
	return obj, func() {
		// TODO: Limit the capacity of the slice
		obj.root.kv = obj.root.kv[:0]
		obj.root.ns = ""
		obj.root.next = nil
		obj.cur = obj.root
		objectEncoderPool.Put(obj)
	}
}

var (
	_ zapcore.ObjectEncoder = (*objectEncoder)(nil)
	_ zapcore.ArrayEncoder  = (*arrayEncoder)(nil)
)

type newNameSpace struct {
	ns   string
	kv   []log.KeyValue
	next *newNameSpace
}

// Object Encoder implements zapcore.ObjectEncoder.
// It encodes given fields to OTel attribute.
type objectEncoder struct {
	// root is a pointer to the default namespace
	root *newNameSpace
	// cur is a pointer to the namespace we're currently writing to.
	cur        *newNameSpace
	reflectval log.Value
}

// NewObjectEncoder returns new ObjectEncoder.
func newObjectEncoder(len int) *objectEncoder {
	keyval := make([]log.KeyValue, 0, len)
	m := &newNameSpace{
		kv: keyval,
	}
	return &objectEncoder{
		root: m,
		cur:  m,
	}
}

// It iterates to the end of the linked list and appends namespace data.
func (m *objectEncoder) getObjValue(o *newNameSpace) *newNameSpace {
	if o.next == nil {
		return nil
	}
	ok := m.getObjValue(o.next)
	if ok == nil {
		o.kv = append(o.kv, log.Map(o.next.ns, o.next.kv...))
		return nil
	}
	return o
}

// Converts Array to logSlice using ArrayEncoder.
func (m *objectEncoder) AddArray(key string, v zapcore.ArrayMarshaler) error {
	arr, free := getArrayEncoder()
	defer free()
	err := v.MarshalLogArray(arr)
	m.cur.kv = append(m.cur.kv, log.Slice(key, arr.elems...))
	return err
}

// Converts Object to logMap.
func (m *objectEncoder) AddObject(k string, v zapcore.ObjectMarshaler) error {
	newobj, free := getObjectEncoder()
	defer free()
	err := v.MarshalLogObject(newobj)
	m.getObjValue(newobj.root)
	m.cur.kv = append(m.cur.kv, log.Map(k, newobj.root.kv...))
	return err
}

// Converts Binary to logBytes.
func (m *objectEncoder) AddBinary(k string, v []byte) {
	m.cur.kv = append(m.cur.kv, log.Bytes(k, v))
}

// Converts ByteString to logString.
func (m *objectEncoder) AddByteString(k string, v []byte) {
	m.cur.kv = append(m.cur.kv, log.String(k, string(v)))
}

// Converts Bool to logBool.
func (m *objectEncoder) AddBool(k string, v bool) {
	m.cur.kv = append(m.cur.kv, log.Bool(k, v))
}

// Converts Duration to logInt.
func (m *objectEncoder) AddDuration(k string, v time.Duration) {
	m.AddInt64(k, v.Nanoseconds())
}

// Converts Complex128 to logString.
func (m *objectEncoder) AddComplex128(k string, v complex128) {
	stringValue := strconv.FormatComplex(v, 'f', -1, 64)
	m.cur.kv = append(m.cur.kv, log.String(k, stringValue))
}

// Converts Float64 to logFloat64.
func (m *objectEncoder) AddFloat64(k string, v float64) {
	m.cur.kv = append(m.cur.kv, log.Float64(k, v))
}

// Converts Float32 to logFloat64.
func (m *objectEncoder) AddFloat32(k string, v float32) {
	m.AddFloat64(k, float64(v))
}

// Converts Int64 to logInt64.
func (m *objectEncoder) AddInt64(k string, v int64) {
	m.cur.kv = append(m.cur.kv, log.Int64(k, v))
}

// Converts Int to logInt.
func (m *objectEncoder) AddInt(k string, v int) {
	m.cur.kv = append(m.cur.kv, log.Int(k, v))
}

// Converts String to logString.
func (m *objectEncoder) AddString(k string, v string) {
	m.cur.kv = append(m.cur.kv, log.String(k, v))
}

// Converts Uint64 to logInt64/logString.
func (m *objectEncoder) AddUint64(k string, v uint64) {
	m.cur.kv = append(m.cur.kv,
		log.KeyValue{
			Key:   k,
			Value: assignUintValue(v),
		})
}

// Converts all non-primitive types to JSON string.
func (m *objectEncoder) AddReflected(k string, v interface{}) error {
	enc := json.NewEncoder(m)
	if err := enc.Encode(v); err != nil {
		return err
	}
	m.AddString(k, m.reflectval.AsString())
	return nil
}

// Implements io.Writer to which json encoder writes to.
// Used by AddReflected method.
func (m *objectEncoder) Write(p []byte) (n int, err error) {
	m.reflectval = log.StringValue(string(p))
	return
}

// OpenNamespace opens an isolated namespace where all subsequent fields will
// be added.
func (m *objectEncoder) OpenNamespace(k string) {
	keyValue := make([]log.KeyValue, 0, 5)
	s := &newNameSpace{
		ns: k,
		kv: keyValue,
	}
	m.cur.next = s
	m.cur = s
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
	arr, free := getArrayEncoder()
	defer free()
	err := v.MarshalLogArray(arr)
	a.elems = append(a.elems, log.SliceValue(arr.elems...))
	return err
}

func (a *arrayEncoder) AppendObject(v zapcore.ObjectMarshaler) error {
	m := newObjectEncoder(2)
	err := v.MarshalLogObject(m)
	m.getObjValue(m.root)
	a.elems = append(a.elems, log.MapValue(m.root.kv...))
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

// Implements io.Writer to which json encoder writes to.
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
