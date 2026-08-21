// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otellogr

import (
	"errors"
	"testing"

	"github.com/go-logr/logr"

	"go.opentelemetry.io/otel/attribute"
)

var benchmarkValue attribute.Value

func benchmarkPointerChain(depth int) any {
	value := any(42)
	for range depth {
		value = func(v any) *any { return &v }(value)
	}
	return value
}

func BenchmarkConvertValue(b *testing.B) {
	shared := map[string]int{"one": 1}
	integer := 42

	for _, tt := range []struct {
		name  string
		value any
	}{
		{name: "Bool", value: true},
		{name: "Int", value: 42},
		{name: "Int8", value: int8(42)},
		{name: "Int64", value: int64(42)},
		{name: "Uint64", value: uint64(42)},
		{name: "Float32", value: float32(42)},
		{name: "String", value: "value"},
		{name: "AttributeValue", value: attribute.StringValue("value")},
		{name: "Nil", value: nil},
		{name: "PointerInt", value: &integer},
		{name: "PointerDepth1", value: benchmarkPointerChain(1)},
		{name: "PointerDepth8", value: benchmarkPointerChain(8)},
		{name: "PointerDepth64", value: benchmarkPointerChain(64)},
		{name: "EmptySlice", value: []int{}},
		{name: "Slice", value: []int{1, 2, 3}},
		{name: "SliceAny", value: []any{1, 2, 3}},
		{name: "OneEntryMap", value: map[string]int{"one": 1}},
		{name: "OneEntryMapAny", value: map[string]any{"one": 1}},
		{name: "Map", value: map[string]int{"one": 1, "two": 2, "three": 3}},
		{name: "NestedMap", value: map[string]any{"outer": map[string]int{"one": 1}}},
		{name: "SharedDAG", value: []any{shared, shared}},
		{name: "Struct", value: struct{ Value int }{Value: 42}},
		{name: "StructInterface", value: struct{ Value any }{Value: 42}},
		{name: "StructMap", value: struct{ Value map[string]int }{Value: shared}},
	} {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				benchmarkValue = convertValue(tt.value)
			}
		})
	}
}

func BenchmarkConvertValueCycle(b *testing.B) {
	cyclicMap := map[string]any{}
	cyclicMap["self"] = cyclicMap
	branchingCycle := map[string]any{}
	branchingCycle["left"] = branchingCycle
	branchingCycle["right"] = branchingCycle
	type wrapper struct{ Value any }
	structCycle := map[string]any{}
	structCycle["wrapper"] = wrapper{Value: structCycle}

	for _, tt := range []struct {
		name  string
		value any
	}{
		{name: "Cycle", value: cyclicMap},
		{name: "BranchingCycle", value: branchingCycle},
		{name: "StructCycle", value: wrapper{Value: structCycle}},
	} {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				benchmarkValue = convertValue(tt.value)
			}
		})
	}
}

func BenchmarkLogSink(b *testing.B) {
	message := "body"
	keyValues := []any{
		"string", "hello",
		"int", 42,
		"float", 3.14,
		"bool", false,
	}
	err := errors.New("error")

	b.Run("Info", func(b *testing.B) {
		logSinks := make([]logr.LogSink, b.N)
		for i := range logSinks {
			logSinks[i] = NewLogSink("")
		}

		b.ReportAllocs()
		b.ResetTimer()
		for n := range b.N {
			logSinks[n].Info(0, message, keyValues...)
		}
	})

	b.Run("Error", func(b *testing.B) {
		logSinks := make([]logr.LogSink, b.N)
		for i := range logSinks {
			logSinks[i] = NewLogSink("")
		}

		b.ReportAllocs()
		b.ResetTimer()
		for n := range b.N {
			logSinks[n].Error(err, message, keyValues...)
		}
	})

	b.Run("WithValues", func(b *testing.B) {
		logSinks := make([]logr.LogSink, b.N)
		for i := range logSinks {
			logSinks[i] = NewLogSink("")
		}

		b.ReportAllocs()
		b.ResetTimer()
		for n := range b.N {
			logSinks[n].WithValues(keyValues...)
		}
	})

	b.Run("WithName", func(b *testing.B) {
		logSinks := make([]logr.LogSink, b.N)
		for i := range logSinks {
			logSinks[i] = NewLogSink("")
		}

		b.ReportAllocs()
		b.ResetTimer()
		for n := range b.N {
			logSinks[n].WithName("name")
		}
	})

	b.Run("WithName.WithValues", func(b *testing.B) {
		logSinks := make([]logr.LogSink, b.N)
		for i := range logSinks {
			logSinks[i] = NewLogSink("")
		}

		b.ReportAllocs()
		b.ResetTimer()
		for n := range b.N {
			logSinks[n].WithName("name").WithValues(keyValues...)
		}
	})

	b.Run("(WithName.WithValues).Info", func(b *testing.B) {
		logSinks := make([]logr.LogSink, b.N)
		for i := range logSinks {
			logSinks[i] = NewLogSink("").WithName("name").WithValues(keyValues...)
		}

		b.ReportAllocs()
		b.ResetTimer()
		for n := range b.N {
			logSinks[n].Info(0, message)
		}
	})
}

func BenchmarkLogSinkErrorField(b *testing.B) {
	err := errors.New("error")
	message := "body"
	keyValues := []any{
		"string", "hello",
		"int", 42,
		"float", 3.14,
		"bool", false,
		"bytes", []byte("bytes"),
		"uint", uint(5),
		"duration", 1,
		"slice",
		[]int{1, 2, 3},
		"map",
		map[string]int{"value": 1},
	}

	b.Run("NoErrorField", func(b *testing.B) {
		logSinks := make([]logr.LogSink, b.N)
		for i := range logSinks {
			logSinks[i] = NewLogSink("")
		}

		b.ReportAllocs()
		b.ResetTimer()
		for n := range b.N {
			logSinks[n].Info(0, message)
		}
	})

	b.Run("WithErrorField", func(b *testing.B) {
		logSinks := make([]logr.LogSink, b.N)
		for i := range logSinks {
			logSinks[i] = NewLogSink("")
		}

		b.ReportAllocs()
		b.ResetTimer()
		for n := range b.N {
			logSinks[n].Error(err, message)
		}
	})

	b.Run("TenFieldsNoError", func(b *testing.B) {
		logSinks := make([]logr.LogSink, b.N)
		for i := range logSinks {
			logSinks[i] = NewLogSink("")
		}

		b.ReportAllocs()
		b.ResetTimer()
		for n := range b.N {
			logSinks[n].Info(0, message, keyValues...)
		}
	})

	b.Run("TenFieldsWithError", func(b *testing.B) {
		logSinks := make([]logr.LogSink, b.N)
		for i := range logSinks {
			logSinks[i] = NewLogSink("")
		}

		b.ReportAllocs()
		b.ResetTimer()
		for n := range b.N {
			logSinks[n].Error(err, message, keyValues...)
		}
	})
}
