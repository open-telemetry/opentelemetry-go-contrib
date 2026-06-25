// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package x

import (
	"context"
	"net/http"
	"net/http/httptest"
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
				{AWSEC2: ExperimentalAWSEC2ResourceDetector{}},
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
			for _, detector := range tt.detectors {
				if detector.AWSEC2 != nil {
					stubIMDS(t)
				}
			}
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

func TestNewResourceConfiguredAttributesOverrideDetectors(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "detected-service")

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

	value, ok := got.Set().Value(semconv.ServiceNameKey)
	require.True(t, ok)
	assert.Equal(t, "configured-service", value.AsString())
}

func stubIMDS(t *testing.T) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/latest/api/token":
			_, _ = w.Write([]byte("token"))
		case "/latest/dynamic/instance-identity/document":
			_, _ = w.Write([]byte(`{
				"accountId": "123456789012",
				"architecture": "x86_64",
				"availabilityZone": "us-west-2b",
				"imageId": "ami-5fb8c835",
				"instanceId": "i-1234567890abcdef0",
				"instanceType": "t2.micro",
				"pendingTime": "2016-11-19T16:32:11Z",
				"privateIp": "10.158.112.84",
				"region": "us-west-2",
				"version": "2017-09-30"
			}`))
		case "/latest/meta-data/hostname":
			_, _ = w.Write([]byte("ip-12-34-56-78.us-west-2.compute.internal"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	t.Setenv("AWS_EC2_METADATA_SERVICE_ENDPOINT", server.URL)
}
