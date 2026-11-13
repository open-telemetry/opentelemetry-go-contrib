// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"

	"go.opentelemetry.io/contrib/detectors/gcp/internal"
)

// gcpDetector can detect attributes of GCP environments.
type gcpDetector interface {
	ProjectID(context.Context) (string, error)
	CloudPlatform(context.Context) internal.Platform
	GKEAvailabilityZoneOrRegion(context.Context) (string, internal.LocationType, error)
	GKEClusterName(context.Context) (string, error)
	GKEHostID(context.Context) (string, error)
	GKEHostType(context.Context) (string, error)
	FaaSName(context.Context) (string, error)
	FaaSVersion(context.Context) (string, error)
	FaaSID(context.Context) (string, error)
	FaaSCloudRegion(context.Context) (string, error)
	AppEngineFlexAvailabilityZoneAndRegion(context.Context) (string, string, error)
	AppEngineStandardAvailabilityZone(context.Context) (string, error)
	AppEngineStandardCloudRegion(context.Context) (string, error)
	AppEngineServiceName(context.Context) (string, error)
	AppEngineServiceVersion(context.Context) (string, error)
	AppEngineServiceInstance(context.Context) (string, error)
	GCEAvailabilityZoneAndRegion(context.Context) (string, string, error)
	GCEHostType(context.Context) (string, error)
	GCEHostID(context.Context) (string, error)
	GCEHostName(context.Context) (string, error)
	GCEInstanceHostname(context.Context) (string, error)
	GCEInstanceName(context.Context) (string, error)
	GCEInstanceLabels(context.Context) (map[string]string, error)
	CloudRunJobExecution(context.Context) (string, error)
	CloudRunJobTaskIndex(context.Context) (string, error)
	GCEManagedInstanceGroup(context.Context) (internal.ManagedInstanceGroup, error)
	BareMetalSolutionInstanceID(context.Context) (string, error)
	BareMetalSolutionCloudRegion(context.Context) (string, error)
	BareMetalSolutionProjectID(context.Context) (string, error)
}
