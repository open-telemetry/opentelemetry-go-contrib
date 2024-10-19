// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config // import "go.opentelemetry.io/contrib/config"

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

type mockType struct{}

func TestNewResource(t *testing.T) {
	res, err := resource.Merge(resource.Default(),
		resource.NewWithAttributes(semconv.SchemaURL,
			semconv.ServiceName("service-a"),
		))
	other := mockType{}
	require.NoError(t, err)
	resWithAttrs, err := resource.Merge(resource.Default(),
		resource.NewWithAttributes(semconv.SchemaURL,
			semconv.ServiceName("service-a"),
			attribute.Bool("attr-bool", true),
			attribute.String("attr-uint64", fmt.Sprintf("%d", 164)),
			attribute.Int64("attr-int64", int64(-164)),
			attribute.Float64("attr-float64", float64(64.0)),
			attribute.Int64("attr-int8", int64(-18)),
			attribute.Int64("attr-uint8", int64(18)),
			attribute.Int64("attr-int16", int64(-116)),
			attribute.Int64("attr-uint16", int64(116)),
			attribute.Int64("attr-int32", int64(-132)),
			attribute.Int64("attr-uint32", int64(132)),
			attribute.Float64("attr-float32", float64(32.0)),
			attribute.Int64("attr-int", int64(-1)),
			attribute.String("attr-uint", fmt.Sprintf("%d", 1)),
			attribute.String("attr-string", "string-val"),
			attribute.String("attr-default", fmt.Sprintf("%v", other)),
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
				Attributes: []AttributeNameValue{
					{Name: "service.name", Value: "service-a"},
				},
			},
			wantResource: resource.NewSchemaless(res.Attributes()...),
			wantErr:      resource.ErrSchemaURLConflict,
		},
		{
			name: "resource-with-attributes-and-schema",
			config: &Resource{
				Attributes: []AttributeNameValue{
					{Name: "service.name", Value: "service-a"},
				},
				SchemaUrl: ptr(semconv.SchemaURL),
			},
			wantResource: res,
		},
		{
			name: "resource-with-additional-attributes-and-schema",
			config: &Resource{
				Attributes: []AttributeNameValue{
					{Name: "service.name", Value: "service-a"},
					{Name: "attr-bool", Value: true},
					{Name: "attr-int64", Value: int64(-164)},
					{Name: "attr-uint64", Value: uint64(164)},
					{Name: "attr-float64", Value: float64(64.0)},
					{Name: "attr-int8", Value: int8(-18)},
					{Name: "attr-uint8", Value: uint8(18)},
					{Name: "attr-int16", Value: int16(-116)},
					{Name: "attr-uint16", Value: uint16(116)},
					{Name: "attr-int32", Value: int32(-132)},
					{Name: "attr-uint32", Value: uint32(132)},
					{Name: "attr-float32", Value: float32(32.0)},
					{Name: "attr-int", Value: int(-1)},
					{Name: "attr-uint", Value: uint(1)},
					{Name: "attr-string", Value: "string-val"},
					{Name: "attr-default", Value: other},
				},
				SchemaUrl: ptr(semconv.SchemaURL),
			},
			wantResource: resWithAttrs,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newResource(tt.config)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.wantResource, got)
		})
	}
}
