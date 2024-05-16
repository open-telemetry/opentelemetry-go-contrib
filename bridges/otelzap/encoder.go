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
	kv []log.KeyValue
}

// nolint:unused
func newObjectEncoder(len int) *objectEncoder {
	keyval := make([]log.KeyValue, 0, len)

	return &objectEncoder{
		kv: keyval,
	}
}

// AddArray converts array to log.Slice using ArrayEncoder.
func (m *objectEncoder) AddArray(key string, v zapcore.ArrayMarshaler) error {
	// TODO
	return nil
}

// AddObject converts object to log.Map using ObjectEncoder.
func (m *objectEncoder) AddObject(k string, v zapcore.ObjectMarshaler) error {
	// TODO
	return nil
}

// AddBinary converts binary to log.Bytes.
func (m *objectEncoder) AddBinary(k string, v []byte) {
	m.kv = append(m.kv, log.Bytes(k, v))
}

// AddByteString converts byte to log.String.
func (m *objectEncoder) AddByteString(k string, v []byte) {
	m.kv = append(m.kv, log.String(k, string(v)))
}

// AddBool converts bool to log.Bool.
func (m *objectEncoder) AddBool(k string, v bool) {
	m.kv = append(m.kv, log.Bool(k, v))
}

// AddDuration converts duration to log.Int.
func (m *objectEncoder) AddDuration(k string, v time.Duration) {
	// TODO
}

// TODO.
func (m *objectEncoder) AddComplex128(k string, v complex128) {
}

// AddFloat64 converts float64 to log.Float64.
func (m *objectEncoder) AddFloat64(k string, v float64) {
	m.kv = append(m.kv, log.Float64(k, v))
}

// AddInt64 converts int64 to log.Int64.
func (m *objectEncoder) AddInt64(k string, v int64) {
	m.kv = append(m.kv, log.Int64(k, v))
}

// AddInt converts int to log.Int.
func (m *objectEncoder) AddInt(k string, v int) {
	m.kv = append(m.kv, log.Int(k, v))
}

// AddString converts string to log.String.
func (m *objectEncoder) AddString(k string, v string) {
	m.kv = append(m.kv, log.String(k, v))
}

// TODO.
func (m *objectEncoder) AddUint64(k string, v uint64) {
}

// TODO.
func (m *objectEncoder) AddReflected(k string, v interface{}) error {
	return nil
}

// OpenNamespace opens an isolated namespace where all subsequent fields will
// be added.
func (m *objectEncoder) OpenNamespace(k string) {
	// TODO
}

// TODO.
func (m *objectEncoder) AddFloat32(k string, v float32)     { m.AddFloat64(k, float64(v)) }
func (m *objectEncoder) AddComplex64(k string, v complex64) {}
func (m *objectEncoder) AddInt32(k string, v int32)         { m.AddInt64(k, int64(v)) }
func (m *objectEncoder) AddInt16(k string, v int16)         { m.AddInt64(k, int64(v)) }
func (m *objectEncoder) AddInt8(k string, v int8)           { m.AddInt64(k, int64(v)) }
func (m *objectEncoder) AddTime(k string, v time.Time)      {}
func (m *objectEncoder) AddUint(k string, v uint)           {}
func (m *objectEncoder) AddUint32(k string, v uint32)       {}
func (m *objectEncoder) AddUint16(k string, v uint16)       {}
func (m *objectEncoder) AddUint8(k string, v uint8)         {}
func (m *objectEncoder) AddUintptr(k string, v uintptr)     {}
