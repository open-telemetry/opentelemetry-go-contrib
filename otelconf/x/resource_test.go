// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package x

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.42.0"
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
			name: "resource-with-empty-detection",
			config: &Resource{
				DetectionDevelopment: &ExperimentalResourceDetection{},
			},
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
			got, err := newResource(t.Context(), tt.config)
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

func TestNewResourceUsesContext(t *testing.T) {
	wantCtx := context.WithValue(t.Context(), ctxKey{}, "resource")
	want := resource.NewSchemaless(attribute.String("from", "builder"))
	got, err := newResourceWithBuilder(wantCtx, &Resource{}, func(ctx context.Context, _ ...resource.Option) (*resource.Resource, error) {
		assert.Same(t, wantCtx, ctx)
		return want, nil
	})
	require.NoError(t, err)
	assert.Same(t, want, got)
}

type ctxKey struct{}

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
				{AWSEKS: ExperimentalAWSEKSResourceDetector{}},
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
			got, err := newResource(t.Context(), config)
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

func TestNewResourceWithDetectionAttributesFilter(t *testing.T) {
	tests := []struct {
		name     string
		attrs    *IncludeExclude
		wantHost bool
		wantOS   bool
	}{
		{
			name:     "no-filter",
			wantHost: true,
			wantOS:   true,
		},
		{
			name: "include-host-only",
			attrs: &IncludeExclude{
				Included: []string{string(semconv.HostNameKey)},
			},
			wantHost: true,
		},
		{
			name: "exclude-os",
			attrs: &IncludeExclude{
				Excluded: []string{string(semconv.OSTypeKey)},
			},
			wantHost: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newResource(t.Context(), &Resource{
				DetectionDevelopment: &ExperimentalResourceDetection{
					Detectors: []ExperimentalResourceDetector{
						{Host: ExperimentalHostResourceDetector{}},
					},
					Attributes: tt.attrs,
				},
			})
			require.NoError(t, err)

			attrs := got.Attributes()
			attrSet := make(map[attribute.Key]bool)
			for _, attr := range attrs {
				attrSet[attr.Key] = true
			}

			assert.Equal(t, tt.wantHost, attrSet[semconv.HostNameKey])
			assert.Equal(t, tt.wantOS, attrSet[semconv.OSTypeKey])
		})
	}
}

func TestNewResourceWithDetectionAttributesFilterDoesNotApplyToConfiguredAttributes(t *testing.T) {
	got, err := newResource(t.Context(), &Resource{
		Attributes: []AttributeNameValue{
			{Name: "custom", Value: "value"},
		},
		DetectionDevelopment: &ExperimentalResourceDetection{
			Detectors: []ExperimentalResourceDetector{
				{Host: ExperimentalHostResourceDetector{}},
			},
			Attributes: &IncludeExclude{
				Included: []string{string(semconv.HostNameKey)},
				Excluded: []string{"custom"},
			},
		},
	})
	require.NoError(t, err)

	gotAttrs := attrMap(got.Attributes())
	assert.Contains(t, gotAttrs, attribute.Key("custom"))
	assert.Contains(t, gotAttrs, semconv.HostNameKey)
	assert.NotContains(t, gotAttrs, semconv.OSTypeKey)
}

func TestNewResourceWithDetectionAttributesFilterRemovesDetectedSchema(t *testing.T) {
	schemaURL := "https://example.com/schema"
	got, err := newResource(t.Context(), &Resource{
		SchemaUrl: ptr(schemaURL),
		DetectionDevelopment: &ExperimentalResourceDetection{
			Detectors: []ExperimentalResourceDetector{
				{Host: ExperimentalHostResourceDetector{}},
			},
			Attributes: &IncludeExclude{
				Included: []string{"unmatched.attribute"},
			},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, schemaURL, got.SchemaURL())
	assert.NotContains(t, attrMap(got.Attributes()), semconv.HostNameKey)
	assert.NotContains(t, attrMap(got.Attributes()), semconv.OSTypeKey)
}

func TestNewResourceWithDetectionDoesNotOverrideConfiguredAttributes(t *testing.T) {
	got, err := newResource(t.Context(), &Resource{
		Attributes: []AttributeNameValue{
			{Name: string(semconv.ServiceNameKey), Value: "configured-service"},
		},
		DetectionDevelopment: &ExperimentalResourceDetection{
			Detectors: []ExperimentalResourceDetector{
				{Service: ExperimentalServiceResourceDetector{}},
			},
		},
	})
	require.NoError(t, err)

	gotAttrs := attrMap(got.Attributes())
	assert.Equal(t, "configured-service", gotAttrs[semconv.ServiceNameKey].AsString())
	assert.Contains(t, gotAttrs, semconv.ServiceInstanceIDKey)
}

func TestNewResourceWithDetectionAttributesFilterError(t *testing.T) {
	got, err := newResource(t.Context(), &Resource{
		DetectionDevelopment: &ExperimentalResourceDetection{
			Detectors: []ExperimentalResourceDetector{
				{Host: ExperimentalHostResourceDetector{}},
			},
			Attributes: &IncludeExclude{
				Included: []string{"foo"},
				Excluded: []string{"foo"},
			},
		},
	})
	require.Equal(t, fmt.Errorf("attribute cannot be in both include and exclude list: foo"), err)
	assert.True(t, got.Set().HasValue(semconv.ServiceNameKey))
}

func attrMap(attrs []attribute.KeyValue) map[attribute.Key]attribute.Value {
	attrSet := make(map[attribute.Key]attribute.Value, len(attrs))
	for _, attr := range attrs {
		attrSet[attr.Key] = attr.Value
	}
	return attrSet
}
