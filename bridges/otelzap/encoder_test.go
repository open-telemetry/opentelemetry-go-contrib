// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Copyright (c) 2016-2017 Uber Technologies, Inc.

package otelzap

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/multierr"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/otel/log"
)

// Copied from https://github.com/uber-go/zap/blob/b39f8b6b6a44d8371a87610be50cce58eeeaabcb/zapcore/memory_encoder_test.go.
func TestObjectEncoder(t *testing.T) {
	// Expected output of a turducken.
	wantTurducken := map[string]interface{}{
		"ducks": []interface{}{
			map[string]interface{}{"in": "chicken"},
			map[string]interface{}{"in": "chicken"},
		},
	}

	tests := []struct {
		desc     string
		f        func(zapcore.ObjectEncoder)
		expected interface{}
	}{
		{
			desc: "AddObject",
			f: func(e zapcore.ObjectEncoder) {
				assert.NoError(t, e.AddObject("k", loggable{true}), "Expected AddObject to succeed.")
			},
			expected: map[string]interface{}{"loggable": "yes"},
		},
		{
			desc: "AddObject (nested)",
			f: func(e zapcore.ObjectEncoder) {
				assert.NoError(t, e.AddObject("k", turducken{}), "Expected AddObject to succeed.")
			},
			expected: wantTurducken,
		},
		{
			desc: "AddArray",
			f: func(e zapcore.ObjectEncoder) {
				assert.NoError(t, e.AddArray("k", zapcore.ArrayMarshalerFunc(func(arr zapcore.ArrayEncoder) error {
					arr.AppendBool(true)
					arr.AppendBool(false)
					arr.AppendBool(true)
					return nil
				})), "Expected AddArray to succeed.")
			},
			expected: []interface{}{true, false, true},
		},
		{
			desc: "AddArray (nested)",
			f: func(e zapcore.ObjectEncoder) {
				assert.NoError(t, e.AddArray("k", turduckens(2)), "Expected AddArray to succeed.")
			},
			expected: []interface{}{wantTurducken, wantTurducken},
		},
		{
			desc:     "AddBinary",
			f:        func(e zapcore.ObjectEncoder) { e.AddBinary("k", []byte("foo")) },
			expected: []byte("foo"),
		},
		{
			desc:     "AddByteString",
			f:        func(e zapcore.ObjectEncoder) { e.AddByteString("k", []byte("foo")) },
			expected: "foo",
		},
		{
			desc:     "AddBool",
			f:        func(e zapcore.ObjectEncoder) { e.AddBool("k", true) },
			expected: true,
		},
		{
			desc:     "AddFloat64",
			f:        func(e zapcore.ObjectEncoder) { e.AddFloat64("k", 3.14) },
			expected: 3.14,
		},
		{
			desc:     "AddFloat32",
			f:        func(e zapcore.ObjectEncoder) { e.AddFloat32("k", 3.14) },
			expected: float64(float32(3.14)),
		},
		{
			desc:     "AddInt",
			f:        func(e zapcore.ObjectEncoder) { e.AddInt("k", 42) },
			expected: int64(42),
		},
		{
			desc:     "AddInt64",
			f:        func(e zapcore.ObjectEncoder) { e.AddInt64("k", 42) },
			expected: int64(42),
		},
		{
			desc:     "AddInt32",
			f:        func(e zapcore.ObjectEncoder) { e.AddInt32("k", 42) },
			expected: int64(42),
		},
		{
			desc:     "AddInt16",
			f:        func(e zapcore.ObjectEncoder) { e.AddInt16("k", 42) },
			expected: int64(42),
		},
		{
			desc:     "AddInt8",
			f:        func(e zapcore.ObjectEncoder) { e.AddInt8("k", 42) },
			expected: int64(42),
		},
		{
			desc:     "AddString",
			f:        func(e zapcore.ObjectEncoder) { e.AddString("k", "v") },
			expected: "v",
		},
		{
			desc:     "AddUint64",
			f:        func(e zapcore.ObjectEncoder) { e.AddUint64("k", 42) },
			expected: int64(42),
		},
		{
			desc:     "AddUint64-Overflow",
			f:        func(e zapcore.ObjectEncoder) { e.AddUint64("k", ^uint64(0)) },
			expected: float64(^uint64(0)),
		},
		{
			desc:     "AddUint",
			f:        func(e zapcore.ObjectEncoder) { e.AddUint("k", 42) },
			expected: int64(42),
		},
		{
			desc:     "AddUint32",
			f:        func(e zapcore.ObjectEncoder) { e.AddUint32("k", 42) },
			expected: int64(42),
		},
		{
			desc:     "AddUint16",
			f:        func(e zapcore.ObjectEncoder) { e.AddUint16("k", 42) },
			expected: int64(42),
		},
		{
			desc:     "AddUint8",
			f:        func(e zapcore.ObjectEncoder) { e.AddUint8("k", 42) },
			expected: int64(42),
		},
		{
			desc:     "AddUintptr",
			f:        func(e zapcore.ObjectEncoder) { e.AddUintptr("k", 42) },
			expected: int64(42),
		},
		{
			desc:     "AddDuration",
			f:        func(e zapcore.ObjectEncoder) { e.AddDuration("k", time.Millisecond) },
			expected: int64(1000000),
		},
		{
			desc:     "AddTime",
			f:        func(e zapcore.ObjectEncoder) { e.AddTime("k", time.Unix(0, 100)) },
			expected: time.Unix(0, 100).UnixNano(),
		},
		{
			desc:     "AddComplex128",
			f:        func(e zapcore.ObjectEncoder) { e.AddComplex128("k", 1+2i) },
			expected: map[string]interface{}{"i": float64(2), "r": float64(1)},
		},
		{
			desc:     "AddComplex64",
			f:        func(e zapcore.ObjectEncoder) { e.AddComplex64("k", 1+2i) },
			expected: map[string]interface{}{"i": float64(2), "r": float64(1)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			enc := newObjectEncoder(1)
			tt.f(enc)
			require.Len(t, enc.kv, 1)
			assert.Equal(t, tt.expected, value2Result((enc.kv[0].Value)), "Unexpected encoder output.")
		})
	}
}

// Copied from https://github.com/uber-go/zap/blob/b39f8b6b6a44d8371a87610be50cce58eeeaabcb/zapcore/memory_encoder_test.go.
func TestArrayEncoder(t *testing.T) {
	tests := []struct {
		desc     string
		f        func(zapcore.ArrayEncoder)
		expected interface{}
	}{
		// AppendObject and AppendArray are covered by the AddObject (nested) and
		// AddArray (nested) cases above.
		{"AppendBool", func(e zapcore.ArrayEncoder) { e.AppendBool(true) }, true},
		{"AppendByteString", func(e zapcore.ArrayEncoder) { e.AppendByteString([]byte("foo")) }, "foo"},
		{"AppendFloat64", func(e zapcore.ArrayEncoder) { e.AppendFloat64(3.14) }, 3.14},
		{"AppendFloat32", func(e zapcore.ArrayEncoder) { e.AppendFloat32(3.14) }, float64(float32(3.14))},
		{"AppendInt64", func(e zapcore.ArrayEncoder) { e.AppendInt64(42) }, int64(42)},
		{"AppendInt", func(e zapcore.ArrayEncoder) { e.AppendInt(42) }, int64(42)},
		{"AppendString", func(e zapcore.ArrayEncoder) { e.AppendString("foo") }, "foo"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			enc := newObjectEncoder(1)
			assert.NoError(t, enc.AddArray("k", zapcore.ArrayMarshalerFunc(func(arr zapcore.ArrayEncoder) error {
				tt.f(arr)
				tt.f(arr)
				return nil
			})), "Expected AddArray to succeed.")

			assert.Equal(t, []interface{}{tt.expected, tt.expected}, value2Result(enc.kv[0].Value), "Unexpected encoder output.")
		})
	}
}

type turducken struct{}

func (t turducken) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	return enc.AddArray("ducks", zapcore.ArrayMarshalerFunc(func(arr zapcore.ArrayEncoder) error {
		for i := 0; i < 2; i++ {
			err := arr.AppendObject(zapcore.ObjectMarshalerFunc(func(inner zapcore.ObjectEncoder) error {
				inner.AddString("in", "chicken")
				return nil
			}))
			if err != nil {
				return err
			}
		}
		return nil
	}))
}

type turduckens int

func (t turduckens) MarshalLogArray(enc zapcore.ArrayEncoder) error {
	var err error
	tur := turducken{}
	for i := 0; i < int(t); i++ {
		err = multierr.Append(err, enc.AppendObject(tur))
	}
	return err
}

type loggable struct{ bool }

func (l loggable) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if !l.bool {
		return errors.New("can't marshal")
	}
	enc.AddString("loggable", "yes")
	return nil
}

func (l loggable) MarshalLogArray(enc zapcore.ArrayEncoder) error {
	if !l.bool {
		return errors.New("can't marshal")
	}
	enc.AppendBool(true)
	return nil
}

func value2Result(v log.Value) any {
	switch v.Kind() {
	case log.KindBool:
		return v.AsBool()
	case log.KindFloat64:
		return v.AsFloat64()
	case log.KindInt64:
		return v.AsInt64()
	case log.KindString:
		return v.AsString()
	case log.KindBytes:
		return v.AsBytes()
	case log.KindSlice:
		var s []any
		for _, val := range v.AsSlice() {
			s = append(s, value2Result(val))
		}
		return s
	case log.KindMap:
		m := make(map[string]any)
		for _, val := range v.AsMap() {
			m[val.Key] = value2Result(val.Value)
		}
		return m
	}
	return nil
}
