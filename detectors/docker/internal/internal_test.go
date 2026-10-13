// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGoosToOSType(t *testing.T) {
	cases := []struct{ in, want string }{
		{"linux", "linux"},
		{"darwin", "darwin"},
		{"windows", "windows"},
		{"dragonfly", "dragonflybsd"},
		{"freebsd", "freebsd"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, GOOSToOSType(tc.in), "input: %s", tc.in)
	}
}
