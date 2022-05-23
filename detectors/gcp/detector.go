// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gcp // import "go.opentelemetry.io/contrib/detectors/gcp"

import (
	"context"
	"fmt"

	"cloud.google.com/go/compute/metadata"
	"github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
)

// NewDetector returns a resource detector which detects resource attributes on:
// * Google Compute Engine (GCE)
// * Google Kubernetes Engine (GKE)
// * Google App Engine (GAE)
// * Cloud Run
// * Cloud Functions
func NewDetector() resource.Detector {
	return &detector{detector: gcp.NewDetector()}
}

type detector struct {
	detector gcpDetector
}

// Detect detects associated resources when running on GCE, GKE, GAE,
// Cloud Run, and Cloud functions.
func (d *detector) Detect(ctx context.Context) (*resource.Resource, error) {
	if !metadata.OnGCE() {
		return nil, nil
	}
	projectID, err := d.detector.ProjectID()
	if err != nil {
		return nil, err
	}
	attributes := []attribute.KeyValue{semconv.CloudProviderGCP, semconv.CloudAccountIDKey.String(projectID)}

	switch d.detector.CloudPlatform() {
	case gcp.GKE:
		attributes = append(attributes, semconv.CloudPlatformGCPKubernetesEngine)
		v, locType, err := d.detector.GKEAvailabilityZoneOrRegion()
		if err != nil {
			return nil, err
		}
		switch locType {
		case gcp.Zone:
			attributes = append(attributes, semconv.CloudAvailabilityZoneKey.String(v))
		case gcp.Region:
			attributes = append(attributes, semconv.CloudRegionKey.String(v))
		default:
			return nil, fmt.Errorf("location must be zone or region. Got %v", locType)
		}
		return detectWithFuncs(attributes, map[attribute.Key]detectionFunc{
			semconv.K8SClusterNameKey: d.detector.GKEClusterName,
			semconv.HostIDKey:         d.detector.GKEHostID,
			semconv.HostNameKey:       d.detector.GKEHostName,
		})
	case gcp.CloudRun:
		attributes = append(attributes, semconv.CloudPlatformGCPCloudRun)
		return detectWithFuncs(attributes, map[attribute.Key]detectionFunc{
			semconv.FaaSNameKey:    d.detector.FaaSName,
			semconv.FaaSVersionKey: d.detector.FaaSVersion,
			semconv.FaaSIDKey:      d.detector.FaaSID,
			semconv.CloudRegionKey: d.detector.FaaSCloudRegion,
		})
	case gcp.CloudFunctions:
		attributes = append(attributes, semconv.CloudPlatformGCPCloudFunctions)
		return detectWithFuncs(attributes, map[attribute.Key]detectionFunc{
			semconv.FaaSNameKey:    d.detector.FaaSName,
			semconv.FaaSVersionKey: d.detector.FaaSVersion,
			semconv.FaaSIDKey:      d.detector.FaaSID,
			semconv.CloudRegionKey: d.detector.FaaSCloudRegion,
		})
	case gcp.AppEngine:
		attributes = append(attributes, semconv.CloudPlatformGCPAppEngine)
		zone, region, err := d.detector.AppEngineAvailabilityZoneAndRegion()
		if err != nil {
			return nil, err
		}
		attributes = append(attributes, semconv.CloudAvailabilityZoneKey.String(zone))
		attributes = append(attributes, semconv.CloudRegionKey.String(region))
		return detectWithFuncs(attributes, map[attribute.Key]detectionFunc{
			semconv.FaaSNameKey:    d.detector.AppEngineServiceName,
			semconv.FaaSVersionKey: d.detector.AppEngineServiceVersion,
			semconv.FaaSIDKey:      d.detector.AppEngineServiceInstance,
		})
	case gcp.GCE:
		attributes = append(attributes, semconv.CloudPlatformGCPComputeEngine)
		zone, region, err := d.detector.GCEAvailabilityZoneAndRegion()
		if err != nil {
			return nil, err
		}
		attributes = append(attributes, semconv.CloudAvailabilityZoneKey.String(zone))
		attributes = append(attributes, semconv.CloudRegionKey.String(region))
		return detectWithFuncs(attributes, map[attribute.Key]detectionFunc{
			semconv.HostTypeKey: d.detector.GCEHostType,
			semconv.HostIDKey:   d.detector.GCEHostID,
			semconv.HostNameKey: d.detector.GCEHostName,
		})
	default:
		// We don't support this platform yet, so just return with what we have
		return resource.NewWithAttributes(semconv.SchemaURL, attributes...), nil
	}
}

type detectionFunc func() (string, error)

// detectWithFuncs is a helper to reduce the amount of error handling code
func detectWithFuncs(attributes []attribute.KeyValue, funcs map[attribute.Key]detectionFunc) (*resource.Resource, error) {
	for key, detect := range funcs {
		v, err := detect()
		if err != nil {
			return nil, err
		}
		attributes = append(attributes, key.String(v))
	}
	return resource.NewWithAttributes(semconv.SchemaURL, attributes...), nil
}
