// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelzap // import "go.opentelemetry.io/contrib/bridges/otelzap"

import (
	"time"

	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/otel/log"
)

var _ zapcore.ObjectEncoder = (*objectEncoder)(nil)

// objectEncoder implements zapcore.ObjectEncoder.
// It encodes given fields to OTel key-values.
type objectEncoder struct {
	kv []log.KeyValue // nolint:unused
}

// nolint:unused
func newObjectEncoder(len int) *objectEncoder {
	keyval := make([]log.KeyValue, 0, len)

	return &objectEncoder{
		kv: keyval,
	}
}

// TODO
// AddArray converts array to log.Slice using ArrayEncoder.
func (m *objectEncoder) AddArray(key string, v zapcore.ArrayMarshaler) error {
	return nil
}

// TODO
// AddObject converts object to log.Map using ObjectEncoder.
func (m *objectEncoder) AddObject(k string, v zapcore.ObjectMarshaler) error {
	return nil
}

// TODO
// AddBinary converts binary to log.Bytes.
func (m *objectEncoder) AddBinary(k string, v []byte) {
}

// TODO
// AddByteString converts byte to log.String.
func (m *objectEncoder) AddByteString(k string, v []byte) {
}

// TODO
// AddBool converts bool to log.Bool.
func (m *objectEncoder) AddBool(k string, v bool) {
}

// TODO
// AddDuration converts duration to log.Int.
func (m *objectEncoder) AddDuration(k string, v time.Duration) {
}

// TODO
func (m *objectEncoder) AddComplex128(k string, v complex128) {
}

// TODO
// AddFloat64 converts float64 to log.Float64.
func (m *objectEncoder) AddFloat64(k string, v float64) {
}

// TODO
// AddFloat32 converts float32 to log.Float64.
func (m *objectEncoder) AddFloat32(k string, v float32) {
}

// TODO
// AddInt64 converts int64 to log.Int64.
func (m *objectEncoder) AddInt64(k string, v int64) {
}

// TODO
// AddInt converts int to log.Int.
func (m *objectEncoder) AddInt(k string, v int) {
}

// TODO
// AddString converts string to log.String.
func (m *objectEncoder) AddString(k string, v string) {
}

// TODO
func (m *objectEncoder) AddUint64(k string, v uint64) {
}

// TODO
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
