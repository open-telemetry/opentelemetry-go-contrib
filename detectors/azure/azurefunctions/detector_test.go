// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azurefunctions

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

func TestNewResourceDetector(t *testing.T) {
	assert.IsType(t, &ResourceDetector{}, NewResourceDetector())
}

func TestReturnsEmptyIfNoFunctionsMarker(t *testing.T) {
	t.Setenv(functionsWorkerRuntimeEnvVar, "")
	t.Setenv(functionsExtensionVersionEnvVar, "")
	t.Setenv(websiteSiteNameEnvVar, "my-function-app")

	detector := ResourceDetector{}
	res, err := detector.Detect(t.Context())

	assert.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetectGatesOnExtensionVersion(t *testing.T) {
	t.Setenv(functionsWorkerRuntimeEnvVar, "")
	t.Setenv(functionsExtensionVersionEnvVar, "~4")
	t.Setenv(websiteSiteNameEnvVar, "my-function-app")

	detector := ResourceDetector{}
	res, err := detector.Detect(t.Context())

	assert.NoError(t, err)
	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderAzure,
		semconv.CloudPlatformAzureFunctions,
		semconv.ServiceName("my-function-app"),
	)
	assert.Equal(t, expected, res)
}

func TestDetectFullAttributeSet(t *testing.T) {
	t.Setenv(functionsWorkerRuntimeEnvVar, "dotnet-isolated")
	t.Setenv(regionNameEnvVar, "eastus")
	t.Setenv(websiteSiteNameEnvVar, "my-function-app")
	t.Setenv(websiteResourceGroupEnvVar, "my-rg")
	t.Setenv(websiteOwnerNameEnvVar, "11111111-1111-1111-1111-111111111111+my-rg-EastUSwebspace")
	t.Setenv(websiteInstanceIDEnvVar, "a1b2c3")
	t.Setenv(websiteSlotNameEnvVar, "staging")

	detector := ResourceDetector{}
	res, err := detector.Detect(t.Context())

	assert.NoError(t, err)
	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderAzure,
		semconv.CloudPlatformAzureFunctions,
		semconv.CloudRegion("eastus"),
		semconv.ServiceName("my-function-app"),
		semconv.AzureResourceGroupName("my-rg"),
		semconv.CloudAccountID("11111111-1111-1111-1111-111111111111"),
		semconv.CloudResourceID("/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/my-rg/providers/Microsoft.Web/sites/my-function-app"),
		semconv.FaaSInstance("a1b2c3"),
		semconv.DeploymentEnvironmentNameKey.String("staging"),
	)
	assert.Equal(t, expected, res)
}

func TestDetectOwnerNameWithoutPlus(t *testing.T) {
	t.Setenv(functionsWorkerRuntimeEnvVar, "dotnet-isolated")
	t.Setenv(websiteSiteNameEnvVar, "my-function-app")
	t.Setenv(websiteResourceGroupEnvVar, "my-rg")
	t.Setenv(websiteOwnerNameEnvVar, "11111111-1111-1111-1111-111111111111")

	detector := ResourceDetector{}
	res, err := detector.Detect(t.Context())

	assert.NoError(t, err)
	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderAzure,
		semconv.CloudPlatformAzureFunctions,
		semconv.ServiceName("my-function-app"),
		semconv.AzureResourceGroupName("my-rg"),
		semconv.CloudAccountID("11111111-1111-1111-1111-111111111111"),
		semconv.CloudResourceID("/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/my-rg/providers/Microsoft.Web/sites/my-function-app"),
	)
	assert.Equal(t, expected, res)
}

func TestDetectFlexConsumptionMissingResourceGroup(t *testing.T) {
	// Flex Consumption does not expose WEBSITE_RESOURCE_GROUP, so
	// cloud.resource_id and azure.resource_group.name must be omitted
	// rather than emitted with a partial value; cloud.account.id is still
	// derived from WEBSITE_OWNER_NAME independently.
	t.Setenv(functionsWorkerRuntimeEnvVar, "dotnet-isolated")
	t.Setenv(websiteSiteNameEnvVar, "my-function-app")
	t.Setenv(websiteOwnerNameEnvVar, "11111111-1111-1111-1111-111111111111+my-rg-EastUSwebspace")
	t.Setenv(websitePodNameEnvVar, "pod-abc123")
	t.Setenv(containerNameEnvVar, "should-not-be-used")

	detector := ResourceDetector{}
	res, err := detector.Detect(t.Context())

	assert.NoError(t, err)
	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderAzure,
		semconv.CloudPlatformAzureFunctions,
		semconv.ServiceName("my-function-app"),
		semconv.CloudAccountID("11111111-1111-1111-1111-111111111111"),
		semconv.FaaSInstance("pod-abc123"),
	)
	assert.Equal(t, expected, res)
}

func TestFunctionsInstanceIDFallsBackToContainerName(t *testing.T) {
	t.Setenv(functionsWorkerRuntimeEnvVar, "dotnet-isolated")
	t.Setenv(websiteSiteNameEnvVar, "my-function-app")
	t.Setenv(containerNameEnvVar, "container-xyz")

	detector := ResourceDetector{}
	res, err := detector.Detect(t.Context())

	assert.NoError(t, err)
	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderAzure,
		semconv.CloudPlatformAzureFunctions,
		semconv.ServiceName("my-function-app"),
		semconv.FaaSInstance("container-xyz"),
	)
	assert.Equal(t, expected, res)
}

func TestWithAttributeFilter(t *testing.T) {
	t.Setenv(functionsWorkerRuntimeEnvVar, "dotnet-isolated")
	t.Setenv(websiteSiteNameEnvVar, "my-function-app")
	t.Setenv(regionNameEnvVar, "eastus")

	filter := func(kv attribute.KeyValue) bool {
		return strings.HasPrefix(string(kv.Key), "cloud.")
	}
	detector := NewResourceDetector(WithAttributeFilter(filter))
	res, err := detector.Detect(t.Context())

	assert.NoError(t, err)
	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderAzure,
		semconv.CloudPlatformAzureFunctions,
		semconv.CloudRegion("eastus"),
	)
	assert.Equal(t, expected, res)
}
