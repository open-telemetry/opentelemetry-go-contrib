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

func (m *objectEncoder) AddArray(key string, v zapcore.ArrayMarshaler) error {
	// TODO
	return nil
}

func (m *objectEncoder) AddObject(k string, v zapcore.ObjectMarshaler) error {
	// TODO
	return nil
}

func (m *objectEncoder) AddBinary(k string, v []byte) {
	// TODO
}

func (m *objectEncoder) AddByteString(k string, v []byte) {
	// TODO
}

func (m *objectEncoder) AddBool(k string, v bool) {
	// TODO
}

func (m *objectEncoder) AddDuration(k string, v time.Duration) {
	// TODO
}

// TODO.
func (m *objectEncoder) AddComplex128(k string, v complex128) {
}

func (m *objectEncoder) AddFloat64(k string, v float64) {
	// TODO
}

func (m *objectEncoder) AddFloat32(k string, v float32) {
	// TODO
}

func (m *objectEncoder) AddInt64(k string, v int64) {
	// TODO
}

func (m *objectEncoder) AddInt(k string, v int) {
	// TODO
}

func (m *objectEncoder) AddString(k string, v string) {
	// TODO
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
