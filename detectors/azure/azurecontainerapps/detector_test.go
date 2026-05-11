// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azurecontainerapps

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

func TestNewResourceDetector(t *testing.T) {
	d := NewResourceDetector()
	assert.IsType(t, &ResourceDetector{}, d)
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

// Return empty resource when not running in an Azure Container Apps environment.
func TestReturnsIfNoEnvVars(t *testing.T) {
	os.Clearenv()
	detector := ResourceDetector{}
	res, err := detector.Detect(t.Context())

	assert.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}
