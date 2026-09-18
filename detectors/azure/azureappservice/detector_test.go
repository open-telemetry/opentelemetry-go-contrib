// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azureappservice

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

// setAppServiceEnv sets the full set of App Service environment variables a
// scaled-out web app exposes.
func setAppServiceEnv(t *testing.T) {
	t.Helper()
	t.Setenv(siteNameEnvVar, "example-app-name")
	t.Setenv(resourceGroupEnvVar, "my-rg")
	t.Setenv(ownerNameEnvVar, "8c56d827-5f07-45ce-8f2b-6c5001db5c6f+my-rg-eastuswebspace")
	t.Setenv(regionEnvVar, "eastus")
	t.Setenv(slotNameEnvVar, "staging")
	t.Setenv(instanceIDEnvVar, "a1b2c3d4e5f67890abcdef1234567890abcdef1234567890abcdef1234567890")
	// Not a Function; keep the success paths deterministic regardless of the
	// ambient environment.
	t.Setenv(functionsWorkerRuntimeEnvVar, "")
}

func fullExpectedAttrs() []attribute.KeyValue {
	return []attribute.KeyValue{
		semconv.CloudProviderAzure,
		semconv.CloudPlatformAzureAppService,
		semconv.ServiceName("example-app-name"),
		semconv.CloudAccountID("8c56d827-5f07-45ce-8f2b-6c5001db5c6f"),
		semconv.CloudResourceID("/subscriptions/8c56d827-5f07-45ce-8f2b-6c5001db5c6f/resourceGroups/my-rg/providers/Microsoft.Web/sites/example-app-name"),
		semconv.AzureResourceGroupName("my-rg"),
		semconv.CloudRegion("eastus"),
		semconv.DeploymentEnvironmentNameKey.String("staging"),
		instanceIDKey.String("a1b2c3d4e5f67890abcdef1234567890abcdef1234567890abcdef1234567890"),
	}
}

func TestNewResourceDetector(t *testing.T) {
	assert.IsType(t, &ResourceDetector{}, NewResourceDetector())
}

func TestDetectSuccess(t *testing.T) {
	setAppServiceEnv(t)

	expected := resource.NewWithAttributes(semconv.SchemaURL, fullExpectedAttrs()...)
	res, err := (&ResourceDetector{}).Detect(t.Context())

	assert.NoError(t, err)
	assert.Equal(t, expected, res)
}

func TestDetectOwnerNameWithoutSubscriptionSeparator(t *testing.T) {
	setAppServiceEnv(t)
	// When WEBSITE_OWNER_NAME has no '+' the whole value is the subscription ID.
	t.Setenv(ownerNameEnvVar, "8c56d827-5f07-45ce-8f2b-6c5001db5c6f")

	res, err := (&ResourceDetector{}).Detect(t.Context())

	assert.NoError(t, err)
	assert.Contains(t, res.Attributes(), semconv.CloudAccountID("8c56d827-5f07-45ce-8f2b-6c5001db5c6f"))
	assert.Contains(t, res.Attributes(), semconv.CloudResourceID("/subscriptions/8c56d827-5f07-45ce-8f2b-6c5001db5c6f/resourceGroups/my-rg/providers/Microsoft.Web/sites/example-app-name"))
}

func TestDetectMissingInstanceID(t *testing.T) {
	setAppServiceEnv(t)
	t.Setenv(instanceIDEnvVar, "")

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderAzure,
		semconv.CloudPlatformAzureAppService,
		semconv.ServiceName("example-app-name"),
		semconv.CloudAccountID("8c56d827-5f07-45ce-8f2b-6c5001db5c6f"),
		semconv.CloudResourceID("/subscriptions/8c56d827-5f07-45ce-8f2b-6c5001db5c6f/resourceGroups/my-rg/providers/Microsoft.Web/sites/example-app-name"),
		semconv.AzureResourceGroupName("my-rg"),
		semconv.CloudRegion("eastus"),
		semconv.DeploymentEnvironmentNameKey.String("staging"),
	)
	res, err := (&ResourceDetector{}).Detect(t.Context())

	assert.ErrorIs(t, err, resource.ErrPartialResource)
	assert.Equal(t, expected, res)
}

func TestDetectMissingOptionalEnvVars(t *testing.T) {
	t.Setenv(siteNameEnvVar, "example-app-name")
	t.Setenv(resourceGroupEnvVar, "my-rg")
	t.Setenv(ownerNameEnvVar, "8c56d827-5f07-45ce-8f2b-6c5001db5c6f+my-rg-eastuswebspace")
	t.Setenv(regionEnvVar, "")
	t.Setenv(slotNameEnvVar, "")
	t.Setenv(instanceIDEnvVar, "")
	t.Setenv(functionsWorkerRuntimeEnvVar, "")

	res, err := (&ResourceDetector{}).Detect(t.Context())

	assert.ErrorIs(t, err, resource.ErrPartialResource)
	attrs := res.Attributes()
	assert.NotContains(t, attrs, semconv.CloudRegion("eastus"))
	assert.Contains(t, attrs, semconv.ServiceName("example-app-name"))
}

func TestWithAttributeFilter(t *testing.T) {
	setAppServiceEnv(t)

	filter := func(kv attribute.KeyValue) bool {
		return strings.HasPrefix(string(kv.Key), "cloud.")
	}
	res, err := NewResourceDetector(WithAttributeFilter(filter)).Detect(t.Context())

	assert.NoError(t, err)
	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderAzure,
		semconv.CloudPlatformAzureAppService,
		semconv.CloudAccountID("8c56d827-5f07-45ce-8f2b-6c5001db5c6f"),
		semconv.CloudResourceID("/subscriptions/8c56d827-5f07-45ce-8f2b-6c5001db5c6f/resourceGroups/my-rg/providers/Microsoft.Web/sites/example-app-name"),
		semconv.CloudRegion("eastus"),
	)
	assert.Equal(t, expected, res)
}

func TestReturnsEmptyIfNotOnAppService(t *testing.T) {
	t.Setenv(siteNameEnvVar, "")
	t.Setenv(resourceGroupEnvVar, "")
	t.Setenv(ownerNameEnvVar, "")

	res, err := (&ResourceDetector{}).Detect(t.Context())

	assert.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestReturnsEmptyOnAzureFunctions(t *testing.T) {
	// A Function sets the same WEBSITE_* gate variables plus
	// FUNCTIONS_WORKER_RUNTIME; the App Service detector must defer to the
	// Functions detector rather than claim it.
	setAppServiceEnv(t)
	t.Setenv(functionsWorkerRuntimeEnvVar, "dotnet-isolated")

	res, err := (&ResourceDetector{}).Detect(t.Context())

	assert.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestReturnsEmptyIfGateVarPartiallySet(t *testing.T) {
	t.Setenv(siteNameEnvVar, "example-app-name")
	t.Setenv(resourceGroupEnvVar, "")
	t.Setenv(ownerNameEnvVar, "8c56d827-5f07-45ce-8f2b-6c5001db5c6f+my-rg-eastuswebspace")

	res, err := (&ResourceDetector{}).Detect(t.Context())

	assert.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}
