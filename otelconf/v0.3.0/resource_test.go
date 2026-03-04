// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.39.0"
)

func TestNewResource(t *testing.T) {
	tests := []struct {
		name         string
		config       *Resource
		wantResource *resource.Resource
	}{
		{
			name:         "no-resource-configuration",
			wantResource: resource.Default(),
		},
		{
			name:         "resource-no-attributes",
			config:       &Resource{},
			wantResource: resource.NewSchemaless(),
		},
		{
			name: "resource-with-schema",
			config: &Resource{
				SchemaUrl: ptr(semconv.SchemaURL),
			},
			wantResource: resource.NewWithAttributes(semconv.SchemaURL),
		},
		{
			name: "resource-with-attributes",
			config: &Resource{
				Attributes: []AttributeNameValue{
					{Name: "service.name", Value: "service-a"},
				},
			},
			wantResource: resource.NewWithAttributes("",
				semconv.ServiceName("service-a"),
			),
		},
		{
			name: "resource-with-attributes-and-schema",
			config: &Resource{
				Attributes: []AttributeNameValue{
					{Name: "service.name", Value: "service-a"},
				},
				SchemaUrl: ptr(semconv.SchemaURL),
			},
			wantResource: resource.NewWithAttributes(semconv.SchemaURL,
				semconv.ServiceName("service-a"),
			),
		},
		{
			name: "resource-with-additional-attributes-and-schema",
			config: &Resource{
				Attributes: []AttributeNameValue{
					{Name: "service.name", Value: "service-a"},
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
			got := newResource(tt.config)
			assert.Equal(t, tt.wantResource, got)
		})
	}
}
