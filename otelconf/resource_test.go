// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.39.0"
)

func TestNewResource(t *testing.T) {
	tests := []struct {
		name         string
		config       *Resource
		wantResource *resource.Resource
		wantErrT     error
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
					{Name: string(semconv.ServiceNameKey), Value: "service-a"},
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
			config: &Resource{
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

func TestResourceOptsWitDetectors(t *testing.T) {
	tests := []struct {
		name                 string
		detectors            []ExperimentalResourceDetector
		wantHostAttributes   bool
		wantOSAttributes     bool
		wantHostIDAttribute  bool
		wantProcessAttribute bool
	}{
		{
			name:      "no-detectors",
			detectors: []ExperimentalResourceDetector{},
		},
		{
			name: "host-detector-enabled",
			detectors: []ExperimentalResourceDetector{
				{Host: ExperimentalHostResourceDetector{}},
			},
			wantHostAttributes:  true,
			wantOSAttributes:    true,
			wantHostIDAttribute: true,
		},
		{
			name: "process-detector-only",
			detectors: []ExperimentalResourceDetector{
				{Process: ExperimentalProcessResourceDetector{}},
			},
			wantProcessAttribute: true,
		},
		{
			name: "all-detectors",
			detectors: []ExperimentalResourceDetector{
				{Container: ExperimentalContainerResourceDetector{}},
				{Host: ExperimentalHostResourceDetector{}},
				{Process: ExperimentalProcessResourceDetector{}},
			},
			wantHostAttributes:   true,
			wantOSAttributes:     true,
			wantHostIDAttribute:  true,
			wantProcessAttribute: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Resource{
				DetectionDevelopment: &ExperimentalResourceDetection{
					Detectors: tt.detectors,
				},
			}
			got, err := newResource(config)
			require.NoError(t, err)
			require.NotNil(t, got)

			attrs := got.Attributes()
			attrMap := make(map[attribute.Key]attribute.Value)
			for _, attr := range attrs {
				attrMap[attr.Key] = attr.Value
			}

			// Check for host.name attribute
			_, ok := attrMap[semconv.HostNameKey]
			require.Equal(t, tt.wantHostAttributes, ok)

			// Check for os.type attribute (from WithOS())
			_, ok = attrMap[semconv.OSTypeKey]
			require.Equal(t, tt.wantOSAttributes, ok)

			// Check for host.id attribute
			_, ok = attrMap[semconv.HostIDKey]
			require.Equal(t, tt.wantHostIDAttribute, ok)

			// Check for process.pid attribute
			_, ok = attrMap[semconv.ProcessPIDKey]
			require.Equal(t, tt.wantProcessAttribute, ok)
		})
	}
}
