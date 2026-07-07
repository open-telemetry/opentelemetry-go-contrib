// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgrpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	grpc_codes "google.golang.org/grpc/codes"
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

func TestWithNonErrorCodes(t *testing.T) {
	t.Run("configures map and wires it correctly", func(t *testing.T) {
		nonErrorCodes := map[grpc_codes.Code]struct{}{
			grpc_codes.NotFound: {},
		}

		c := newConfig([]Option{WithNonErrorCodes(nonErrorCodes)})

		if c.NonErrorCodes == nil {
			t.Fatal("expected NonErrorCodes to be configured, got nil")
		}
		assert.Len(t, c.NonErrorCodes, 1)
		assert.Contains(t, c.NonErrorCodes, grpc_codes.NotFound)
	})

	t.Run("nil map is ignored", func(t *testing.T) {
		c := newConfig([]Option{WithNonErrorCodes(nil)})
		assert.Nil(t, c.NonErrorCodes)
	})

	t.Run("empty map is ignored", func(t *testing.T) {
		c := newConfig([]Option{WithNonErrorCodes(map[grpc_codes.Code]struct{}{})})
		assert.Nil(t, c.NonErrorCodes)
	})
}
