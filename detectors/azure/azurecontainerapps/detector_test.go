// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azurecontainerapps

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
)

func TestNewResourceDetector(t *testing.T) {
	assert.IsType(t, &ResourceDetector{}, NewResourceDetector())
}

func TestWithAttributeFilter(t *testing.T) {
	t.Setenv("CONTAINER_APP_NAME", "my-app")
	t.Setenv("CONTAINER_APP_REPLICA_NAME", "my-app--abc123-0")

	filter := func(kv attribute.KeyValue) bool {
		return strings.HasPrefix(string(kv.Key), "service.")
	}
	detector := NewResourceDetector(WithAttributeFilter(filter))
	res, err := detector.Detect(t.Context())

	assert.NoError(t, err)
	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName("my-app"),
		semconv.ServiceInstanceID("my-app--abc123-0"),
	)
	assert.Equal(t, expected, res)
}

// Successfully return resource when process is running in an Azure Container Apps environment.
func TestDetectSuccess(t *testing.T) {
	t.Setenv("CONTAINER_APP_NAME", "my-app")
	t.Setenv("CONTAINER_APP_REPLICA_NAME", "my-app--abc123-0")

	attributes := []attribute.KeyValue{
		semconv.CloudProviderAzure,
		semconv.CloudPlatformAzureContainerApps,
		semconv.ServiceName("my-app"),
		semconv.ServiceInstanceID("my-app--abc123-0"),
	}
	expectedResource := resource.NewWithAttributes(semconv.SchemaURL, attributes...)
	detector := ResourceDetector{}
	res, err := detector.Detect(t.Context())

	assert.NoError(t, err, "Detector unexpectedly returned error")
	assert.Equal(t, expectedResource, res, "Resource returned is incorrect")
}

// Return partial resource with ErrPartialResource when replica name is not set.
func TestDetectMissingReplicaName(t *testing.T) {
	t.Setenv("CONTAINER_APP_NAME", "my-app")

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderAzure,
		semconv.CloudPlatformAzureContainerApps,
		semconv.ServiceName("my-app"),
	)
	detector := ResourceDetector{}
	res, err := detector.Detect(t.Context())

	assert.ErrorIs(t, err, resource.ErrPartialResource)
	assert.Equal(t, expected, res)
}

// Return empty resource when not running in an Azure Container Apps environment.
func TestReturnsIfNoEnvVars(t *testing.T) {
	t.Setenv(containerAppNameEnvVar, "")
	t.Setenv(containerAppReplicaNameEnvVar, "")
	detector := ResourceDetector{}
	res, err := detector.Detect(t.Context())

	assert.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}
