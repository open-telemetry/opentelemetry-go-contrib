// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package semconv

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/attribute"
)

func TestStandardizeHTTPMethodMetric(t *testing.T) {
	testCases := []struct {
		method string
		want   attribute.KeyValue
	}{
		{
			method: "GET",
			want:   attribute.String("http.method", "GET"),
		},
		{
			method: "POST",
			want:   attribute.String("http.method", "POST"),
		},
		{
			method: "PUT",
			want:   attribute.String("http.method", "PUT"),
		},
		{
			method: "DELETE",
			want:   attribute.String("http.method", "DELETE"),
		},
		{
			method: "HEAD",
			want:   attribute.String("http.method", "HEAD"),
		},
		{
			method: "OPTIONS",
			want:   attribute.String("http.method", "OPTIONS"),
		},
		{
			method: "CONNECT",
			want:   attribute.String("http.method", "CONNECT"),
		},
		{
			method: "TRACE",
			want:   attribute.String("http.method", "TRACE"),
		},
		{
			method: "PATCH",
			want:   attribute.String("http.method", "PATCH"),
		},
		{
			method: "test",
			want:   attribute.String("http.method", "_OTHER"),
		},
	}
	for _, tt := range testCases {
		t.Run(tt.method, func(t *testing.T) {
			got := standardizeHTTPMethodMetric(tt.method)
			assert.Equal(t, tt.want, got)
		})
	}
}
