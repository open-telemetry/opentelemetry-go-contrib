// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package envcar

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpperWithUnderscores(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"ABC", "ABC"},
		{"abc", "ABC"},
		{"01239", "01239"},
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
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			assert.Equal(t, tc.want, upperWithUnderscores(tc.in))
		})
	}
}
