// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package envcar

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var normalizeCases = []struct {
	in, want string
}{
	{"", ""},
	{"ABC", "ABC"},
	{"abc", "ABC"},
	{"01239", "_01239"},
	{"0abc", "_0ABC"},
	{"9", "_9"},
	{"a_b_c", "A_B_C"},
	{"hello-world", "HELLO_WORLD"},
	{"foo.bar", "FOO_BAR"},
	{"Content-Type", "CONTENT_TYPE"},
	{"traceparent", "TRACEPARENT"},
	{"key with spaces", "KEY_WITH_SPACES"},
	{"MiXeD_123!", "MIXED_123_"},
	{"🧳", "_"},
	{"Mój Bagaż", "M_J_BAGA_"},
}

func TestNormalize(t *testing.T) {
	for _, tc := range normalizeCases {
		t.Run(tc.in, func(t *testing.T) {
			assert.Equal(t, tc.want, normalize(tc.in))
		})
	}
}

func BenchmarkNormalize(b *testing.B) {
	for _, tc := range normalizeCases {
		b.Run(tc.in, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				normalize(tc.in)
			}
		})
	}
}
