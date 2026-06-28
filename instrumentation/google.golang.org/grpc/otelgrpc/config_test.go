// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgrpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/codes"
	grpc_status "google.golang.org/grpc/status"
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

func TestWithSpanStatusFn(t *testing.T) {
	t.Run("configures fn and wires it correctly", func(t *testing.T) {
		expectedCode := codes.Error
		expectedMsg := "custom error message"

		customFn := func(_ context.Context, _ string, _ *grpc_status.Status) (codes.Code, string) {
			return expectedCode, expectedMsg
		}

		c := newConfig([]Option{WithSpanStatusFn(customFn)})

		if c.SpanStatusFn == nil {
			t.Fatal("expected SpanStatusFn to be configured, got nil")
		}

		code, msg := c.SpanStatusFn(t.Context(), "/pkg.Service/Method", nil)
		assert.Equal(t, expectedCode, code)
		assert.Equal(t, expectedMsg, msg)
	})

	t.Run("nil fn is ignored", func(t *testing.T) {
		c := newConfig([]Option{WithSpanStatusFn(nil)})
		assert.Nil(t, c.SpanStatusFn)
	})

	t.Run("receives fullMethod and grpcStatus", func(t *testing.T) {
		var gotMethod string
		var gotStatus *grpc_status.Status

		customFn := func(_ context.Context, fullMethod string, grpcStatus *grpc_status.Status) (codes.Code, string) {
			gotMethod = fullMethod
			gotStatus = grpcStatus
			return codes.Unset, ""
		}

		c := newConfig([]Option{WithSpanStatusFn(customFn)})
		s := grpc_status.New(0 /* OK */, "")
		c.SpanStatusFn(t.Context(), "/mypackage.MyService/MyMethod", s)

		assert.Equal(t, "/mypackage.MyService/MyMethod", gotMethod)
		assert.Equal(t, s, gotStatus)
	})
}
