// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config // import "go.opentelemetry.io/contrib/config"

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

func TestNewResource(t *testing.T) {
	res, err := resource.Merge(resource.Default(),
		resource.NewWithAttributes(semconv.SchemaURL,
			semconv.ServiceName("service-a"),
		))
	require.NoError(t, err)
	tests := []struct {
		name         string
		config       *Resource
		wantResource *resource.Resource
		wantErr      error
	}{
		{
			name:         "no-resource-configuration",
			wantResource: resource.Default(),
		},
		{
			name:         "resource-no-attributes",
			config:       &Resource{},
			wantResource: resource.Default(),
		},
		{
			name: "resource-with-attributes-invalid-schema",
			config: &Resource{
				SchemaUrl: ptr("https://opentelemetry.io/invalid-schema"),
				Attributes: &Attributes{
					ServiceName: ptr("service-a"),
				},
			},
			wantResource: resource.Empty(),
			wantErr:      errors.New("cannot merge resource due to conflicting Schema URL"),
		},
		{
			name: "resource-with-attributes-and-schema",
			config: &Resource{
				Attributes: &Attributes{
					ServiceName: ptr("service-a"),
				},
				SchemaUrl: ptr(semconv.SchemaURL),
			},
			wantResource: res,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newResource(tt.config)
			require.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.wantResource, got)
		})
	}
}
