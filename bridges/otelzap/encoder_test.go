// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Copyright (c) 2016-2017 Uber Technologies, Inc.

package otelzap

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/otel/log"
)

// Copied from https://github.com/uber-go/zap/blob/b39f8b6b6a44d8371a87610be50cce58eeeaabcb/zapcore/memory_encoder_test.go.
func TestObjectEncoder(t *testing.T) {
	tests := []struct {
		desc     string
		f        func(zapcore.ObjectEncoder)
		expected interface{}
	}{
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
			f:        func(e zapcore.ObjectEncoder) { e.AddUint64("k", ^uint64(0)) },
			expected: float64(^uint64(0)),
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
