// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
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

func TestResourceOptsWithDetectors(t *testing.T) {
	tests := []struct {
		name                 string
		detectors            []ExperimentalResourceDetector
		wantHostAttributes   bool
		wantOSAttributes     bool
		wantProcessAttribute bool
		wantServiceAttribute bool
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
			wantHostAttributes: true,
			wantOSAttributes:   true,
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
				{Service: ExperimentalServiceResourceDetector{}},
			},
			wantHostAttributes:   true,
			wantOSAttributes:     true,
			wantProcessAttribute: true,
			wantServiceAttribute: true,
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
			attrSet := make(map[attribute.Key]bool)
			for _, attr := range attrs {
				attrSet[attr.Key] = true
			}

			assert.Equal(t, tt.wantHostAttributes, attrSet[semconv.HostNameKey], "should have host.name attribute")
			assert.Equal(t, tt.wantOSAttributes, attrSet[semconv.OSTypeKey], "should have os.type attribute (from WithOS()")
			assert.Equal(t, tt.wantProcessAttribute, attrSet[semconv.ProcessPIDKey], "should have process.pid attribute")
			assert.Equal(t, tt.wantServiceAttribute, attrSet[semconv.ServiceInstanceIDKey], "should have service.instance.id attribute")
		})
	}
}
