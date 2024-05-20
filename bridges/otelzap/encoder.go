// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelzap // import "go.opentelemetry.io/contrib/bridges/otelzap"

import (
	"time"

	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/otel/log"
)

var (
	_ zapcore.ObjectEncoder = (*objectEncoder)(nil)
	_ zapcore.ArrayEncoder  = (*arrayEncoder)(nil)
)

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

func (m *objectEncoder) AddArray(key string, v zapcore.ArrayMarshaler) error {
	// TODO
	return nil
}

func (m *objectEncoder) AddObject(k string, v zapcore.ObjectMarshaler) error {
	// TODO
	return nil
}

func (m *objectEncoder) AddBinary(k string, v []byte) {
	m.kv = append(m.kv, log.Bytes(k, v))
}

func (m *objectEncoder) AddByteString(k string, v []byte) {
	m.kv = append(m.kv, log.String(k, string(v)))
}

func (m *objectEncoder) AddBool(k string, v bool) {
	m.kv = append(m.kv, log.Bool(k, v))
}

func (m *objectEncoder) AddDuration(k string, v time.Duration) {
	// TODO
}

func (m *objectEncoder) AddComplex128(k string, v complex128) {
	// TODO.
}

func (m *objectEncoder) AddFloat64(k string, v float64) {
	m.kv = append(m.kv, log.Float64(k, v))
}

func (m *objectEncoder) AddInt64(k string, v int64) {
	m.kv = append(m.kv, log.Int64(k, v))
}

func (m *objectEncoder) AddInt(k string, v int) {
	m.kv = append(m.kv, log.Int(k, v))
}

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

func (m *objectEncoder) AddFloat32(k string, v float32) { m.AddFloat64(k, float64(v)) }
func (m *objectEncoder) AddInt32(k string, v int32)     { m.AddInt64(k, int64(v)) }
func (m *objectEncoder) AddInt16(k string, v int16)     { m.AddInt64(k, int64(v)) }
func (m *objectEncoder) AddInt8(k string, v int8)       { m.AddInt64(k, int64(v)) }

// TODO.
func (m *objectEncoder) AddComplex64(k string, v complex64) {}
func (m *objectEncoder) AddTime(k string, v time.Time)      {}
func (m *objectEncoder) AddUint(k string, v uint)           {}
func (m *objectEncoder) AddUint32(k string, v uint32)       {}
func (m *objectEncoder) AddUint16(k string, v uint16)       {}
func (m *objectEncoder) AddUint8(k string, v uint8)         {}
func (m *objectEncoder) AddUintptr(k string, v uintptr)     {}

// arrayEncoder implements [zapcore.ArrayEncoder].
type arrayEncoder struct {
	elems []log.Value // nolint:unused
}

// TODO
func (a *arrayEncoder) AppendArray(v zapcore.ArrayMarshaler) error {
	return nil
}

// TODO
func (a *arrayEncoder) AppendObject(v zapcore.ObjectMarshaler) error {
	return nil
}

// TODO
func (a *arrayEncoder) AppendReflected(v interface{}) error {
	return nil
}

// TODO
func (a *arrayEncoder) AppendComplex128(v complex128)  {}
func (a *arrayEncoder) AppendFloat32(v float32)        {}
func (a *arrayEncoder) AppendByteString(v []byte)      {}
func (a *arrayEncoder) AppendBool(v bool)              {}
func (a *arrayEncoder) AppendUint64(v uint64)          {}
func (a *arrayEncoder) AppendFloat64(v float64)        {}
func (a *arrayEncoder) AppendInt(v int)                {}
func (a *arrayEncoder) AppendInt64(v int64)            {}
func (a *arrayEncoder) AppendString(v string)          {}
func (a *arrayEncoder) AppendComplex64(v complex64)    {}
func (a *arrayEncoder) AppendDuration(v time.Duration) {}
func (a *arrayEncoder) AppendInt32(v int32)            {}
func (a *arrayEncoder) AppendInt16(v int16)            {}
func (a *arrayEncoder) AppendInt8(v int8)              {}
func (a *arrayEncoder) AppendTime(v time.Time)         {}
func (a *arrayEncoder) AppendUint(v uint)              {}
func (a *arrayEncoder) AppendUint32(v uint32)          {}
func (a *arrayEncoder) AppendUint16(v uint16)          {}
func (a *arrayEncoder) AppendUint8(v uint8)            {}
func (a *arrayEncoder) AppendUintptr(v uintptr)        {}
