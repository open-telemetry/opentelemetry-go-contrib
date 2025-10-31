// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

func TestNewResource(t *testing.T) {
	tests := []struct {
		name         string
		config       OpenTelemetryConfigurationResource
		wantResource *resource.Resource
		wantErrT     error
	}{
		{
			name:         "no-resource-configuration",
			wantResource: resource.Default(),
		},
		{
			name:         "invalid resource",
			config:       "",
			wantResource: nil,
			wantErrT:     newErrInvalid("resource"),
		},
		{
			name:         "resource-no-attributes",
			config:       &ResourceJson{},
			wantResource: resource.NewSchemaless(),
		},
		{
			name: "resource-with-schema",
			config: &ResourceJson{
				SchemaUrl: ptr(semconv.SchemaURL),
			},
			wantResource: resource.NewWithAttributes(semconv.SchemaURL),
		},
		{
			name: "resource-with-attributes",
			config: &ResourceJson{
				Attributes: []AttributeNameValue{
					{Name: string(semconv.ServiceNameKey), Value: "service-a"},
				},
			},
			wantResource: resource.NewWithAttributes("",
				semconv.ServiceName("service-a"),
			),
		},
		{
			name: "resource-with-attributes-and-schema",
			config: &ResourceJson{
				Attributes: []AttributeNameValue{
					{Name: string(semconv.ServiceNameKey), Value: "service-a"},
				},
				SchemaUrl: ptr(semconv.SchemaURL),
			},
			wantResource: resource.NewWithAttributes(semconv.SchemaURL,
				semconv.ServiceName("service-a"),
			),
		},
		{
			name: "resource-with-additional-attributes-and-schema",
			config: &ResourceJson{
				Attributes: []AttributeNameValue{
					{Name: string(semconv.ServiceNameKey), Value: "service-a"},
					{Name: "attr-bool", Value: true},
				},
				SchemaUrl: ptr(semconv.SchemaURL),
			},
			wantResource: resource.NewWithAttributes(semconv.SchemaURL,
				semconv.ServiceName("service-a"),
				attribute.Bool("attr-bool", true)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newResource(tt.config)
			require.ErrorIs(t, tt.wantErrT, err)
			assert.Equal(t, tt.wantResource, got)
		})
	}
}
