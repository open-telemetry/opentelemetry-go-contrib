// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kv

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
)

func TestFromNameValue(t *testing.T) {
	other := struct{}{}
	for _, tt := range []struct {
		name string
		val  any
		want attribute.KeyValue
	}{
		{name: "attr-bool", val: true, want: attribute.Bool("attr-bool", true)},
		{name: "attr-uint64", val: uint64(164), want: attribute.String("attr-uint64", fmt.Sprintf("%d", 164))},
		{name: "attr-int64", val: int64(-164), want: attribute.Int64("attr-int64", int64(-164))},
		{name: "attr-float64", val: float64(64.0), want: attribute.Float64("attr-float64", float64(64.0))},
		{name: "attr-int8", val: int8(-18), want: attribute.Int64("attr-int8", int64(-18))},
		{name: "attr-uint8", val: uint8(18), want: attribute.Int64("attr-uint8", int64(18))},
		{name: "attr-int16", val: int16(-116), want: attribute.Int64("attr-int16", int64(-116))},
		{name: "attr-uint16", val: uint16(116), want: attribute.Int64("attr-uint16", int64(116))},
		{name: "attr-int32", val: int32(-132), want: attribute.Int64("attr-int32", int64(-132))},
		{name: "attr-uint32", val: uint32(132), want: attribute.Int64("attr-uint32", int64(132))},
		{name: "attr-float32", val: float32(32.0), want: attribute.Float64("attr-float32", float64(32.0))},
		{name: "attr-int", val: int(-1), want: attribute.Int64("attr-int", int64(-1))},
		{name: "attr-uint", val: uint(1), want: attribute.String("attr-uint", fmt.Sprintf("%d", 1))},
		{name: "attr-string", val: "string-val", want: attribute.String("attr-string", "string-val")},
		{name: "attr-default", val: other, want: attribute.String("attr-default", fmt.Sprintf("%v", other))},
	} {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, FromNameValue(tt.name, tt.val))
		})
	}
}
