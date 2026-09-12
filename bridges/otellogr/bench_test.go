// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otellogr

import (
	"errors"
	"reflect"
	"strconv"
	"testing"

	"github.com/go-logr/logr"

	"go.opentelemetry.io/otel/attribute"
)

type recursiveBenchmarkValue struct {
	Values []recursiveBenchmarkValue
}

type formattingBenchmarkKey struct {
	Value any
}

type safeFormattingBenchmarkValue struct {
	Values []int
}

func benchmarkPointerChain(depth int) any {
	value := any(42)
	for range depth {
		value = func(v any) *any { return &v }(value)
	}
	return value
}

func benchmarkFormattingTypeValue(depth int) any {
	typ := reflect.TypeFor[int]()
	for range depth {
		typ = reflect.ArrayOf(1, typ)
	}
	typ = reflect.StructOf([]reflect.StructField{{Name: "Value", Type: typ}})
	return reflect.New(typ).Elem().Interface()
}

func benchmarkFormattingTypeMap(depth, size int) any {
	deepType := reflect.TypeFor[int]()
	for range depth {
		deepType = reflect.ArrayOf(1, deepType)
	}
	keyType := reflect.StructOf([]reflect.StructField{
		{Name: "Value", Type: deepType},
		{Name: "ID", Type: reflect.TypeFor[int]()},
	})
	value := reflect.MakeMapWithSize(reflect.MapOf(keyType, reflect.TypeFor[int]()), size)
	for i := range size {
		key := reflect.New(keyType).Elem()
		key.Field(1).SetInt(int64(i))
		value.SetMapIndex(key, reflect.ValueOf(i))
	}
	return value.Interface()
}

func benchmarkWideFormattingStruct(size int) any {
	fields := make([]reflect.StructField, size)
	for i := range fields {
		fields[i] = reflect.StructField{
			Name: "F" + strconv.Itoa(i),
			Type: reflect.TypeFor[int](),
		}
	}
	return reflect.New(reflect.StructOf(fields)).Elem().Interface()
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
		{name: "StructRecursiveSlice", value: recursiveBenchmarkValue{
			Values: []recursiveBenchmarkValue{{}},
		}},
		{name: "IntMapKey", value: map[int]int{42: 1}},
		{name: "NonStringMapKey", value: map[formattingBenchmarkKey]int{{Value: 42}: 1}},
	} {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				convertValue(tt.value)
			}
		})
	}
}

func BenchmarkConvertValueFormattingTypeWork(b *testing.B) {
	for _, tt := range []struct {
		name  string
		value any
	}{
		{name: "DeepType", value: benchmarkFormattingTypeValue(998)},
		{name: "DeepTypeMap100", value: benchmarkFormattingTypeMap(998, 100)},
	} {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				convertValue(tt.value)
			}
		})
	}
}

func BenchmarkConvertValueWorkLimit(b *testing.B) {
	branchingCycle := func(depth int) any {
		nodes := make([][]any, depth)
		for i := range nodes {
			nodes[i] = make([]any, 2)
		}
		for i := range nodes {
			next := nodes[(i+1)%len(nodes)]
			nodes[i][0], nodes[i][1] = next, next
		}
		return nodes[0]
	}
	formattingSharedDAG := func(depth int) any {
		cyclic := map[string]any{}
		cyclic["self"] = cyclic
		var shared any = []any{42}
		for range depth {
			shared = []any{shared, shared}
		}
		return struct{ Value [2]any }{Value: [2]any{shared, cyclic}}
	}
	wideMutualCycle := func(width int) any {
		first := make([]any, width)
		second := make([]any, width)
		for i := range width {
			first[i] = second
			second[i] = first
		}
		return first
	}
	formattingSharedSafeValue := func(width int) any {
		shared := &safeFormattingBenchmarkValue{Values: make([]int, width)}
		values := make([]any, width)
		for i := range values {
			values[i] = shared
		}
		return values
	}
	sharedMapArray := map[string][16_384]int{"value": {}}
	sharedMapArrays := make([]any, 256)
	for i := range sharedMapArrays {
		sharedMapArrays[i] = sharedMapArray
	}
	wideFormattingStruct := benchmarkWideFormattingStruct(10_000)
	emptyWideArray := reflect.Zero(reflect.ArrayOf(
		0, reflect.TypeOf(wideFormattingStruct),
	)).Interface()

	for _, tt := range []struct {
		name  string
		value any
	}{
		{name: "BranchingCycleDepth18", value: branchingCycle(18)},
		{name: "BranchingCycleDepth64", value: branchingCycle(64)},
		{name: "FormattingSharedDAGDepth24", value: formattingSharedDAG(24)},
		{name: "FormattingSharedDAGDepth32", value: formattingSharedDAG(32)},
		{name: "WideMutualCycle256", value: wideMutualCycle(256)},
		{name: "FormattingSharedSafeValue256", value: formattingSharedSafeValue(256)},
		{name: "RejectedPointerArray16384", value: new([16_384]int)},
		{name: "RejectedNestedPointerArray16384", value: new([1][16_384]int)},
		{name: "SharedMapArray256x16384", value: sharedMapArrays},
		{name: "ZeroSizedSlice100000", value: make([]struct{}, 100_000)},
		{name: "WideFormattingStruct10000", value: wideFormattingStruct},
		{name: "EmptyWideElementType10000", value: emptyWideArray},
	} {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				convertValue(tt.value)
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
			for b.Loop() {
				convertValue(tt.value)
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
