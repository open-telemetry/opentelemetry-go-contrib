// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otellogr

import (
	"testing"

	"go.opentelemetry.io/otel/attribute"
)

func BenchmarkConvertValue(b *testing.B) {
	integer := 42
	mapping := map[string]int{"one": 1}

	for _, tt := range []struct {
		name  string
		value any
	}{
		{name: "Bool", value: true},
		{name: "String", value: "value"},
		{name: "Int64", value: int64(42)},
		{name: "Nil", value: nil},
		{name: "AttributeValue", value: attribute.StringValue("value")},
		{name: "PointerScalar", value: &integer},
		{name: "EmptySlice", value: []int{}},
		{name: "SliceInt3", value: []int{1, 2, 3}},
		{name: "SliceAny3", value: []any{1, "two", true}},
		{name: "MapOneEntry", value: mapping},
		{name: "MapThreeEntries", value: map[string]int{"one": 1, "two": 2, "three": 3}},
		{name: "StructScalar", value: struct{ Value int }{Value: 42}},
		{name: "StructMap", value: struct{ Value map[string]int }{Value: mapping}},
	} {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				convertValue(tt.value)
			}
		})
	}
}

func BenchmarkConvertKVs(b *testing.B) {
	ctx := b.Context()
	keyValues := []any{
		"string", "hello",
		"int64", int64(42),
		"bool", true,
		"attribute", attribute.StringValue("value"),
	}

	b.ReportAllocs()
	for b.Loop() {
		convertKVs(ctx, keyValues...)
	}
}
