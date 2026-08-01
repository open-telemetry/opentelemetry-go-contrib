// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azureappservice

import (
	"context"
	"fmt"
	"os"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

const (
	siteNameEnvVar      = "WEBSITE_SITE_NAME"
	resourceGroupEnvVar = "WEBSITE_RESOURCE_GROUP"
	ownerNameEnvVar     = "WEBSITE_OWNER_NAME"
	regionEnvVar        = "REGION_NAME"
	slotNameEnvVar      = "WEBSITE_SLOT_NAME"
	instanceIDEnvVar    = "WEBSITE_INSTANCE_ID"

	// Some Azure Functions run on the same App Service infrastructure.
	// We need to distinguish Functions apps from App Service web apps.
	functionsWorkerRuntimeEnvVar = "FUNCTIONS_WORKER_RUNTIME"

	instanceIDKey = attribute.Key("azure.app_service.instance.id")
)

type config struct {
	filter attribute.Filter
}

// Option configures a [ResourceDetector].
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) { f(c) }

// WithAttributeFilter restricts the attributes the detector emits to those for
// which filter returns true.
func WithAttributeFilter(filter attribute.Filter) Option {
	return optionFunc(func(c *config) { c.filter = filter })
}

// ResourceDetector detects resource information for Azure App Service.
type ResourceDetector struct {
	cfg config
}

var _ resource.Detector = (*ResourceDetector)(nil)

// NewResourceDetector returns a resource detector that detects Azure App Service resources.
func NewResourceDetector(opts ...Option) *ResourceDetector {
	var cfg config
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	return &ResourceDetector{cfg: cfg}
}

// Detect returns a resource describing the Azure App Service the process is running on.
// It returns an empty resource when not running on Azure App Service.
func (d *ResourceDetector) Detect(context.Context) (*resource.Resource, error) {
	siteName := os.Getenv(siteNameEnvVar)
	resourceGroup := os.Getenv(resourceGroupEnvVar)
	ownerName := os.Getenv(ownerNameEnvVar)
	if siteName == "" || resourceGroup == "" || ownerName == "" {
		return resource.Empty(), nil
	}
	// Defer to the Functions detector.
	if os.Getenv(functionsWorkerRuntimeEnvVar) != "" {
		return resource.Empty(), nil
	}

	// WEBSITE_OWNER_NAME has the form "<subscription-id>+<resource-group>-<region>webspace";
	// the subscription ID is the segment before the first '+'.
	subscriptionID := ownerName
	if sub, _, ok := strings.Cut(ownerName, "+"); ok {
		subscriptionID = sub
	}

	attrs := []attribute.KeyValue{
		semconv.CloudProviderAzure,
		semconv.CloudPlatformAzureAppService,
		semconv.ServiceName(siteName),
		semconv.CloudAccountID(subscriptionID),
		semconv.CloudResourceID(fmt.Sprintf(
			"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Web/sites/%s",
			subscriptionID, resourceGroup, siteName,
		)),
		semconv.AzureResourceGroupName(resourceGroup),
	}

	if region := os.Getenv(regionEnvVar); region != "" {
		attrs = append(attrs, semconv.CloudRegion(region))
	}
	if slot := os.Getenv(slotNameEnvVar); slot != "" {
		attrs = append(attrs, semconv.DeploymentEnvironmentNameKey.String(slot))
	}

	var partialErr error
	if instanceID := os.Getenv(instanceIDEnvVar); instanceID != "" {
		attrs = append(attrs, instanceIDKey.String(instanceID))
	} else {
		partialErr = fmt.Errorf("%w: %s not set", resource.ErrPartialResource, instanceIDEnvVar)
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

	return resource.NewWithAttributes(semconv.SchemaURL, attrs...), partialErr
}
