// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azurecontainerapps // import "go.opentelemetry.io/contrib/detectors/azure/azurecontainerapps"

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
)

// For a complete list of reserved environment variables in Azure Container Apps, see:
// https://learn.microsoft.com/en-us/azure/container-apps/environment-variables?tabs=portal#built-in-environment-variables
const (
	containerAppNameEnvVar        = "CONTAINER_APP_NAME"
	containerAppReplicaNameEnvVar = "CONTAINER_APP_REPLICA_NAME"
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

// WithAttributeFilter sets a filter that controls which detected attributes are
// included in the returned resource. Only attributes for which filter returns
// true are included. By default all attributes are included.
func WithAttributeFilter(filter attribute.Filter) Option {
	return optionFunc(func(c *config) { c.filter = filter })
}

// ResourceDetector collects resource information from Azure Container Apps environment.
type ResourceDetector struct {
	cfg config
}

// Compile time assertion that ResourceDetector implements the resource.Detector interface.
var _ resource.Detector = (*ResourceDetector)(nil)

// NewResourceDetector returns a resource detector that detects Azure Container Apps resources.
func NewResourceDetector(opts ...Option) *ResourceDetector {
	var cfg config
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	return &ResourceDetector{cfg: cfg}
}

// Detect collects resource attributes available when running on Azure Container Apps.
// It returns an empty resource when not running on Azure Container Apps.
func (d *ResourceDetector) Detect(context.Context) (*resource.Resource, error) {
	appName := os.Getenv(containerAppNameEnvVar)
	if appName == "" {
		return resource.Empty(), nil
	}

	attrs := []attribute.KeyValue{
		semconv.CloudProviderAzure,
		semconv.CloudPlatformAzureContainerApps,
		semconv.ServiceName(appName),
	}

	var partialErr error
	replicaName := os.Getenv(containerAppReplicaNameEnvVar)
	if replicaName != "" {
		attrs = append(attrs, semconv.ServiceInstanceID(replicaName))
	} else {
		partialErr = fmt.Errorf("%w: %s not set", resource.ErrPartialResource, containerAppReplicaNameEnvVar)
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
