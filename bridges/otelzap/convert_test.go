// Code created by gotmpl. DO NOT MODIFY.
// source: internal/shared/logutil/convert_test.go.tmpl

// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelzap

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/log"
)

func TestConvertValue(t *testing.T) {
	for _, tt := range []struct {
		name      string
		value     any
		wantValue log.Value
	}{
		{
			name:      "bool",
			value:     true,
			wantValue: log.BoolValue(true),
		},
		{
			name:      "string",
			value:     "value",
			wantValue: log.StringValue("value"),
		},
		{
			name:      "int",
			value:     10,
			wantValue: log.Int64Value(10),
		},
		{
			name:      "int8",
			value:     int8(127),
			wantValue: log.Int64Value(127),
		},
		{
			name:      "int16",
			value:     int16(32767),
			wantValue: log.Int64Value(32767),
		},
		{
			name:      "int32",
			value:     int32(2147483647),
			wantValue: log.Int64Value(2147483647),
		},
		{
			name:      "int64",
			value:     int64(9223372036854775807),
			wantValue: log.Int64Value(9223372036854775807),
		},
		{
			name:      "uint",
			value:     uint(42),
			wantValue: log.Int64Value(42),
		},
		{
			name:      "uint8",
			value:     uint8(255),
			wantValue: log.Int64Value(255),
		},
		{
			name:      "uint16",
			value:     uint16(65535),
			wantValue: log.Int64Value(65535),
		},
		{
			name:      "uint32",
			value:     uint32(4294967295),
			wantValue: log.Int64Value(4294967295),
		},
		{
			name:      "uint64",
			value:     uint64(9223372036854775807),
			wantValue: log.Int64Value(9223372036854775807),
		},
		{
			name:      "uint64-max",
			value:     uint64(18446744073709551615),
			wantValue: log.StringValue("18446744073709551615"),
		},
		{
			name:      "uintptr",
			value:     uintptr(12345),
			wantValue: log.Int64Value(12345),
		},
		{
			name:      "float64",
			value:     float64(3.14159),
			wantValue: log.Float64Value(3.14159),
		},
		{
			name:      "time.Duration",
			value:     time.Second,
			wantValue: log.Int64Value(1_000_000_000),
		},
		{
			name:      "complex64",
			value:     complex64(complex(float32(1), float32(2))),
			wantValue: log.MapValue(log.Float64("r", 1), log.Float64("i", 2)),
		},
		{
			name:      "complex128",
			value:     complex(float64(3), float64(4)),
			wantValue: log.MapValue(log.Float64("r", 3), log.Float64("i", 4)),
		},
		{
			name:      "time.Time",
			value:     time.Unix(1000, 1000),
			wantValue: log.Int64Value(time.Unix(1000, 1000).UnixNano()),
		},
		{
			name:      "[]byte",
			value:     []byte("hello"),
			wantValue: log.BytesValue([]byte("hello")),
		},
		{
			name:      "error",
			value:     errors.New("test error"),
			wantValue: log.StringValue("test error"),
		},
		{
			name:      "error",
			value:     errors.New("test error"),
			wantValue: log.StringValue("test error"),
		},
		{
			name:      "error-nested",
			value:     fmt.Errorf("test error: %w", errors.New("nested error")),
			wantValue: log.StringValue("test error: nested error"),
		},
		{
			name:      "nil",
			value:     nil,
			wantValue: log.Value{},
		},
		{
			name:      "nil_ptr",
			value:     (*int)(nil),
			wantValue: log.Value{},
		},
		{
			name:      "int_ptr",
			value:     func() *int { i := 93; return &i }(),
			wantValue: log.Int64Value(93),
		},
		{
			name:      "string_ptr",
			value:     func() *string { s := "hello"; return &s }(),
			wantValue: log.StringValue("hello"),
		},
		{
			name:      "bool_ptr",
			value:     func() *bool { b := true; return &b }(),
			wantValue: log.BoolValue(true),
		},
		{
			name:      "int_empty_array",
			value:     []int{},
			wantValue: log.SliceValue([]log.Value{}...),
		},
		{
			name:  "int_array",
			value: []int{1, 2, 3},
			wantValue: log.SliceValue([]log.Value{
				log.Int64Value(1),
				log.Int64Value(2),
				log.Int64Value(3),
			}...),
		},
		{
			name:  "key_value_map",
			value: map[string]int{"one": 1},
			wantValue: log.MapValue(
				log.Int64("one", 1),
			),
		},
		{
			name:  "int_string_map",
			value: map[int]string{1: "one"},
			wantValue: log.MapValue(
				log.String("1", "one"),
			),
		},
		{
			name:  "nested_map",
			value: map[string]map[string]int{"nested": {"one": 1}},
			wantValue: log.MapValue(
				log.Map("nested",
					log.Int64("one", 1),
				),
			),
		},
		{
			name: "struct_key_map",
			value: map[struct{ Name string }]int{
				{Name: "John"}: 42,
			},
			wantValue: log.MapValue(
				log.Int64("{Name:John}", 42),
			),
		},
		{
			name: "struct",
			value: struct {
				Name string
				Age  int
			}{
				Name: "John",
				Age:  42,
			},
			wantValue: log.StringValue("{Name:John Age:42}"),
		},
		{
			name: "struct_ptr",
			value: &struct {
				Name string
				Age  int
			}{
				Name: "John",
				Age:  42,
			},
			wantValue: log.StringValue("{Name:John Age:42}"),
		},
		{
			name: "nil_struct_ptr",
			value: (*struct {
				Name string
				Age  int
			})(nil),
			wantValue: log.Value{},
		},
		{
			name:      "ctx",
			value:     context.Background(),
			wantValue: log.StringValue("context.Background"),
		},
		{
			name:      "unhandled type",
			value:     chan int(nil),
			wantValue: log.StringValue("unhandled: (chan int) <nil>"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantValue, convertValue(tt.value))
		})
	}
}

func TestConvertValueFloat32(t *testing.T) {
	value := convertValue(float32(3.14))
	want := log.Float64Value(3.14)

	assert.InDelta(t, value.AsFloat64(), want.AsFloat64(), 0.0001)
}
