// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azurefunctions

import (
	"context"
	"fmt"
	"os"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.42.0"
)

const (
	functionsWorkerRuntimeEnvVar    = "FUNCTIONS_WORKER_RUNTIME"
	functionsExtensionVersionEnvVar = "FUNCTIONS_EXTENSION_VERSION"
	regionNameEnvVar                = "REGION_NAME"
	websiteOwnerNameEnvVar          = "WEBSITE_OWNER_NAME"
	websiteResourceGroupEnvVar      = "WEBSITE_RESOURCE_GROUP"
	websiteSiteNameEnvVar           = "WEBSITE_SITE_NAME"
	websiteInstanceIDEnvVar         = "WEBSITE_INSTANCE_ID"
	websitePodNameEnvVar            = "WEBSITE_POD_NAME"
	containerNameEnvVar             = "CONTAINER_NAME"
	websiteSlotNameEnvVar           = "WEBSITE_SLOT_NAME"

	azureResourceGroupNameKey = attribute.Key("azure.resource_group.name")
)

type config struct {
	filter attribute.Filter
}

// Option applies a configuration option to the ResourceDetector.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) { f(c) }

// WithAttributeFilter sets a filter to apply to all attributes detected by the ResourceDetector.
func WithAttributeFilter(filter attribute.Filter) Option {
	return optionFunc(func(c *config) { c.filter = filter })
}

// ResourceDetector detects resource attributes on Azure Functions.
type ResourceDetector struct {
	cfg config
}

var _ resource.Detector = (*ResourceDetector)(nil)

// NewResourceDetector creates a new ResourceDetector for Azure Functions.
func NewResourceDetector(opts ...Option) *ResourceDetector {
	var cfg config
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	return &ResourceDetector{cfg: cfg}
}

// Detect detects Azure Functions resource attributes found in the environment.
// It gates on the presence of a Functions marker (FUNCTIONS_WORKER_RUNTIME or
// FUNCTIONS_EXTENSION_VERSION); if neither is set, it returns an empty resource
// and no error.
func (d *ResourceDetector) Detect(context.Context) (*resource.Resource, error) {
	if os.Getenv(functionsWorkerRuntimeEnvVar) == "" && os.Getenv(functionsExtensionVersionEnvVar) == "" {
		return resource.Empty(), nil
	}

	attrs := []attribute.KeyValue{
		semconv.CloudProviderAzure,
		semconv.CloudPlatformAzureFunctions,
	}

	if region := os.Getenv(regionNameEnvVar); region != "" {
		attrs = append(attrs, semconv.CloudRegion(region))
	}

	siteName := os.Getenv(websiteSiteNameEnvVar)
	if siteName != "" {
		attrs = append(attrs, semconv.ServiceName(siteName))
	}

	resourceGroup := os.Getenv(websiteResourceGroupEnvVar)
	if resourceGroup != "" {
		attrs = append(attrs, azureResourceGroupNameKey.String(resourceGroup))
	}

	// WEBSITE_OWNER_NAME has the form "<subscription-id>+<resource-group>-<region>webspace";
	// the subscription ID is the segment before the first '+'.
	subscriptionID := os.Getenv(websiteOwnerNameEnvVar)
	if idx := strings.Index(subscriptionID, "+"); idx >= 0 {
		subscriptionID = subscriptionID[:idx]
	}
	if subscriptionID != "" {
		attrs = append(attrs, semconv.CloudAccountID(subscriptionID))
	}
	if siteName != "" && resourceGroup != "" && subscriptionID != "" {
		attrs = append(attrs, semconv.CloudResourceID(fmt.Sprintf(
			"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Web/sites/%s",
			subscriptionID, resourceGroup, siteName,
		)))
	}

	if instanceID := functionsInstanceID(); instanceID != "" {
		attrs = append(attrs, semconv.FaaSInstance(instanceID))
	}

	if slotName := os.Getenv(websiteSlotNameEnvVar); slotName != "" {
		attrs = append(attrs, semconv.DeploymentEnvironmentNameKey.String(slotName))
	}

	if d.cfg.filter != nil {
		filtered := attrs[:0]
		for _, kv := range attrs {
			if d.cfg.filter(kv) {
				filtered = append(filtered, kv)
			}
		}
		attrs = filtered
	}

	return resource.NewWithAttributes(semconv.SchemaURL, attrs...), nil
}

// functionsInstanceID resolves the platform instance id, branching by
// hosting plan the same way the Functions host's own GetInstanceId() does:
// WEBSITE_INSTANCE_ID on Windows Consumption / Elastic Premium / Dedicated,
// falling back to WEBSITE_POD_NAME then CONTAINER_NAME on Linux and Flex
// Consumption.
func functionsInstanceID() string {
	for _, envVar := range []string{websiteInstanceIDEnvVar, websitePodNameEnvVar, containerNameEnvVar} {
		if v := os.Getenv(envVar); v != "" {
			return v
		}
	}
	return ""
}
