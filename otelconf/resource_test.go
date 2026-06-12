// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
)

func TestNewResource(t *testing.T) {
	tests := []struct {
		name          string
		config        *Resource
		wantSchemaURL string
		wantAttrs     []attribute.KeyValue
		wantErrT      error
	}{
		{
			name:          "no-resource-configuration",
			wantSchemaURL: resource.Default().SchemaURL(),
		},
		{
			name:          "resource-no-attributes",
			config:        &Resource{},
			wantSchemaURL: "",
		},
		{
			name: "resource-with-schema",
			config: &Resource{
				SchemaUrl: ptr(semconv.SchemaURL),
			},
			wantSchemaURL: semconv.SchemaURL,
		},
		{
			name: "resource-with-attributes",
			config: &Resource{
				Attributes: []AttributeNameValue{
					{Name: string(semconv.ServiceNameKey), Value: "service-a"},
				},
			},
			wantSchemaURL: "",
			wantAttrs:     []attribute.KeyValue{semconv.ServiceName("service-a")},
		},
		{
			name: "resource-with-attributes-and-schema",
			config: &Resource{
				Attributes: []AttributeNameValue{
					{Name: string(semconv.ServiceNameKey), Value: "service-a"},
				},
				SchemaUrl: ptr(semconv.SchemaURL),
			},
			wantSchemaURL: semconv.SchemaURL,
			wantAttrs:     []attribute.KeyValue{semconv.ServiceName("service-a")},
		},
		{
			name: "resource-with-additional-attributes-and-schema",
			config: &Resource{
				Attributes: []AttributeNameValue{
					{Name: string(semconv.ServiceNameKey), Value: "service-a"},
					{Name: "attr-bool", Value: true},
				},
				SchemaUrl: ptr(semconv.SchemaURL),
			},
			wantSchemaURL: semconv.SchemaURL,
			wantAttrs: []attribute.KeyValue{
				semconv.ServiceName("service-a"),
				attribute.Bool("attr-bool", true),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newResource(tt.config)
			require.ErrorIs(t, err, tt.wantErrT)

			assert.Equal(t, tt.wantSchemaURL, got.SchemaURL())
			assert.Truef(t, got.Set().HasValue(semconv.ServiceNameKey), "should have %q attribute", semconv.ServiceNameKey)
			assert.Truef(t, got.Set().HasValue(semconv.TelemetrySDKNameKey), "should have %q attribute", semconv.TelemetrySDKNameKey)
			assert.Truef(t, got.Set().HasValue(semconv.TelemetrySDKLanguageKey), "should have %q attribute", semconv.TelemetrySDKLanguageKey)
			assert.Truef(t, got.Set().HasValue(semconv.TelemetrySDKVersionKey), "should have %q attribute", semconv.TelemetrySDKVersionKey)
			for _, want := range tt.wantAttrs {
				gotValue, ok := got.Set().Value(want.Key)
				if assert.Truef(t, ok, "should have %q attribute", want.Key) {
					assert.Equalf(t, want.Value, gotValue, "%q attribute value mismatch", want.Key)
				}
			}
		})
	}
}
