// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelzap // import "go.opentelemetry.io/contrib/bridges/otelzap"

import (
	"time"

	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/otel/log"
)

var _ zapcore.ObjectEncoder = (*objectEncoder)(nil)

// Object Encoder implements zapcore.ObjectEncoder.
// It encodes given fields to OTel attribute.
type objectEncoder struct {
	kv []log.KeyValue
}

// NewObjectEncoder returns new ObjectEncoder.
func newObjectEncoder(len int) *objectEncoder {
	keyval := make([]log.KeyValue, 0, len)

	return &objectEncoder{
		kv: keyval,
	}
}

// TODO
// Converts Array to log.Slice using ArrayEncoder.
func (m *objectEncoder) AddArray(key string, v zapcore.ArrayMarshaler) error {
	return nil
}

// TODO
// Converts Object to log.Map.
func (m *objectEncoder) AddObject(k string, v zapcore.ObjectMarshaler) error {
	return nil
}

// TODO
// Converts Binary to log.Bytes.
func (m *objectEncoder) AddBinary(k string, v []byte) {
}

// TODO
// Converts ByteString to log.String.
func (m *objectEncoder) AddByteString(k string, v []byte) {
}

// TODO
// Converts Bool to log.Bool.
func (m *objectEncoder) AddBool(k string, v bool) {
}

// TODO
// Converts Duration to log.Int.
func (m *objectEncoder) AddDuration(k string, v time.Duration) {
}

// TODO
// Converts Complex128 to log.String.
func (m *objectEncoder) AddComplex128(k string, v complex128) {
}

// TODO
// Converts Float64 to log.Float64.
func (m *objectEncoder) AddFloat64(k string, v float64) {
}

// TODO
// Converts Float32 to log.Float64.
func (m *objectEncoder) AddFloat32(k string, v float32) {
}

// TODO
// Converts Int64 to logInt64.
func (m *objectEncoder) AddInt64(k string, v int64) {
}

// TODO
// Converts Int to logInt.
func (m *objectEncoder) AddInt(k string, v int) {
}

// TODO
// Converts String to logString.
func (m *objectEncoder) AddString(k string, v string) {
}

// TODO
// Converts Uint64 to logInt64/logString.
func (m *objectEncoder) AddUint64(k string, v uint64) {
}

// TODO
// Converts all non-primitive types to JSON string.
func (m *objectEncoder) AddReflected(k string, v interface{}) error {
	return nil
}

// TODO
// OpenNamespace opens an isolated namespace where all subsequent fields will
// be added.
func (m *objectEncoder) OpenNamespace(k string) {
}

// TODO.
func (m *objectEncoder) AddComplex64(k string, v complex64) {}
func (m *objectEncoder) AddInt32(k string, v int32)         {}
func (m *objectEncoder) AddInt16(k string, v int16)         {}
func (m *objectEncoder) AddInt8(k string, v int8)           {}
func (m *objectEncoder) AddTime(k string, v time.Time)      {}
func (m *objectEncoder) AddUint(k string, v uint)           {}
func (m *objectEncoder) AddUint32(k string, v uint32)       {}
func (m *objectEncoder) AddUint16(k string, v uint16)       {}
func (m *objectEncoder) AddUint8(k string, v uint8)         {}
func (m *objectEncoder) AddUintptr(k string, v uintptr)     {}
