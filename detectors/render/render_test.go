// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package render

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

func TestDetect(t *testing.T) {
	full := map[string]string{
		"RENDER_SERVICE_ID":   "srv-1",
		"RENDER_INSTANCE_ID":  "srv-1-abc",
		"RENDER_SERVICE_NAME": "web",
		"RENDER_GIT_COMMIT":   "abc123",
	}
	tests := []struct {
		name string
		env  map[string]string
		opts []Option
		want []attribute.KeyValue
	}{
		{name: "not on render"},
		{
			name: "all variables",
			env:  full,
			want: []attribute.KeyValue{
				semconv.ServiceName("web"),
				semconv.ServiceVersion("abc123"),
			},
		},
		{
			name: "service id only",
			env:  map[string]string{"RENDER_SERVICE_ID": "srv-1"},
			want: []attribute.KeyValue{semconv.ServiceName("srv-1")},
		},
		{
			name: "attribute filter",
			env:  full,
			opts: []Option{WithAttributeFilter(func(kv attribute.KeyValue) bool {
				return kv.Key == semconv.ServiceVersionKey
			})},
			want: []attribute.KeyValue{semconv.ServiceVersion("abc123")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k := range full {
				t.Setenv(k, "")
			}
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			res, err := NewResourceDetector(tt.opts...).Detect(t.Context())
			require.NoError(t, err)
			if tt.want == nil {
				assert.Equal(t, resource.Empty(), res)
				return
			}
			assert.Equal(t, resource.NewWithAttributes(semconv.SchemaURL, tt.want...), res)
		})
	}
}

func TestDetectIgnoresEmptyValues(t *testing.T) {
	t.Setenv("RENDER_SERVICE_ID", "")
	res, err := NewResourceDetector().Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)

	t.Setenv("RENDER_SERVICE_ID", "srv-1")
	t.Setenv("RENDER_SERVICE_NAME", "")
	t.Setenv("RENDER_GIT_COMMIT", "")
	res, err = NewResourceDetector().Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceName("srv-1")), res)
}
