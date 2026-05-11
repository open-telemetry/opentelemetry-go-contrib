// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package azurecontainerapps provides a resource detector for Azure Container Apps.
package azurecontainerapps // import "go.opentelemetry.io/contrib/detectors/azure/azurecontainerapps"

import (
	"context"
	"os"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

// For a complete list of reserved environment variables in Azure Container Apps, see:
// https://learn.microsoft.com/en-us/azure/container-apps/environment-variables?tabs=portal#built-in-environment-variables
const (
	containerAppNameEnvVar        = "CONTAINER_APP_NAME"
	containerAppReplicaNameEnvVar = "CONTAINER_APP_REPLICA_NAME"
)

// ResourceDetector collects resource information from Azure Container Apps environment.
type ResourceDetector struct{}

// Compile time assertion that ResourceDetector implements the resource.Detector interface.
var _ resource.Detector = (*ResourceDetector)(nil)

// NewResourceDetector returns a resource detector that detects Azure Container Apps resources.
func NewResourceDetector() resource.Detector {
	return &ResourceDetector{}
}

// Detect collects resource attributes available when running on Azure Container Apps.
// It returns an empty resource when not running on Azure Container Apps.
func (*ResourceDetector) Detect(context.Context) (*resource.Resource, error) {
	appName := os.Getenv(containerAppNameEnvVar)
	if appName == "" {
		return resource.Empty(), nil
	}

	attrs := []attribute.KeyValue{
		semconv.CloudProviderAzure,
		semconv.CloudPlatformAzureContainerApps,
		semconv.ServiceName(appName),
	}

	if replicaName := os.Getenv(containerAppReplicaNameEnvVar); replicaName != "" {
		attrs = append(attrs, semconv.ServiceInstanceID(replicaName))
	}

	return resource.NewWithAttributes(semconv.SchemaURL, attrs...), nil
}
