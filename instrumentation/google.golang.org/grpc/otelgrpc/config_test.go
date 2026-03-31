// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgrpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSemconvMode(t *testing.T) {
	tests := []struct {
		name string
		val  string
		want semconvMode
	}{
		{
			name: "empty",
			val:  "",
			want: semconvModeNew,
		},
		{
			name: "old",
			val:  "rpc/old",
			want: semconvModeOld,
		},
		{
			name: "dup",
			val:  "rpc/dup",
			want: semconvModeDup,
		},
		{
			name: "unknown",
			val:  "unknown",
			want: semconvModeNew,
		},
		{
			name: "multiple with valid",
			val:  "foo, rpc/dup",
			want: semconvModeDup,
		},
		{
			name: "multiple with old first",
			val:  "rpc/old, rpc/dup",
			want: semconvModeOld,
		},
		{
			name: "spaces",
			val:  "  rpc/old  ",
			want: semconvModeOld,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("OTEL_SEMCONV_STABILITY_OPT_IN", tt.val)
			got := parseSemconvMode()
			assert.Equal(t, tt.want, got)
		})
	}
}
