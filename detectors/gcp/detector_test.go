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
	"os"
	"testing"

	"github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp"
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
)

func TestDetect(t *testing.T) {
	// Set this before all tests to ensure metadata.onGCE() returns true
	err := os.Setenv("GCE_METADATA_HOST", "169.254.169.254")
	assert.NoError(t, err)

	for _, tc := range []struct {
		desc             string
		detector         resource.Detector
		expectErr        bool
		expectedResource *resource.Resource
	}{
		{
			desc: "zonal GKE cluster",
			detector: &detector{detector: &fakeGCPDetector{
				projectID:           "my-project",
				cloudPlatform:       gcp.GKE,
				gkeHostID:           "1472385723456792345",
				gkeClusterName:      "my-cluster",
				gkeAvailabilityZone: "us-central1-c",
			}},
			expectedResource: resource.NewWithAttributes(semconv.SchemaURL,
				semconv.CloudProviderGCP,
				semconv.CloudAccountIDKey.String("my-project"),
				semconv.CloudPlatformGCPKubernetesEngine,
				semconv.K8SClusterNameKey.String("my-cluster"),
				semconv.CloudAvailabilityZoneKey.String("us-central1-c"),
				semconv.HostIDKey.String("1472385723456792345"),
			),
		},
		{
			desc: "regional GKE cluster",
			detector: &detector{detector: &fakeGCPDetector{
				projectID:      "my-project",
				cloudPlatform:  gcp.GKE,
				gkeHostID:      "1472385723456792345",
				gkeClusterName: "my-cluster",
				gkeRegion:      "us-central1",
			}},
			expectedResource: resource.NewWithAttributes(semconv.SchemaURL,
				semconv.CloudProviderGCP,
				semconv.CloudAccountIDKey.String("my-project"),
				semconv.CloudPlatformGCPKubernetesEngine,
				semconv.K8SClusterNameKey.String("my-cluster"),
				semconv.CloudRegionKey.String("us-central1"),
				semconv.HostIDKey.String("1472385723456792345"),
			),
		},
		{
			desc: "GCE",
			detector: &detector{detector: &fakeGCPDetector{
				projectID:           "my-project",
				cloudPlatform:       gcp.GCE,
				gceHostID:           "1472385723456792345",
				gceHostName:         "my-gke-node-1234",
				gceHostType:         "n1-standard1",
				gceAvailabilityZone: "us-central1-c",
				gceRegion:           "us-central1",
			}},
			expectedResource: resource.NewWithAttributes(semconv.SchemaURL,
				semconv.CloudProviderGCP,
				semconv.CloudAccountIDKey.String("my-project"),
				semconv.CloudPlatformGCPComputeEngine,
				semconv.HostIDKey.String("1472385723456792345"),
				semconv.HostNameKey.String("my-gke-node-1234"),
				semconv.HostTypeKey.String("n1-standard1"),
				semconv.CloudRegionKey.String("us-central1"),
				semconv.CloudAvailabilityZoneKey.String("us-central1-c"),
			),
		},
		{
			desc: "Cloud Run",
			detector: &detector{detector: &fakeGCPDetector{
				projectID:       "my-project",
				cloudPlatform:   gcp.CloudRun,
				faaSID:          "1472385723456792345",
				faaSCloudRegion: "us-central1",
				faaSName:        "my-service",
				faaSVersion:     "123456",
			}},
			expectedResource: resource.NewWithAttributes(semconv.SchemaURL,
				semconv.CloudProviderGCP,
				semconv.CloudAccountIDKey.String("my-project"),
				semconv.CloudPlatformGCPCloudRun,
				semconv.CloudRegionKey.String("us-central1"),
				semconv.FaaSNameKey.String("my-service"),
				semconv.FaaSVersionKey.String("123456"),
				semconv.FaaSIDKey.String("1472385723456792345"),
			),
		},
		{
			desc: "Cloud Functions",
			detector: &detector{detector: &fakeGCPDetector{
				projectID:       "my-project",
				cloudPlatform:   gcp.CloudFunctions,
				faaSID:          "1472385723456792345",
				faaSCloudRegion: "us-central1",
				faaSName:        "my-service",
				faaSVersion:     "123456",
			}},
			expectedResource: resource.NewWithAttributes(semconv.SchemaURL,
				semconv.CloudProviderGCP,
				semconv.CloudAccountIDKey.String("my-project"),
				semconv.CloudPlatformGCPCloudFunctions,
				semconv.CloudRegionKey.String("us-central1"),
				semconv.FaaSNameKey.String("my-service"),
				semconv.FaaSVersionKey.String("123456"),
				semconv.FaaSIDKey.String("1472385723456792345"),
			),
		},
		{
			desc: "App Engine Flex",
			detector: &detector{detector: &fakeGCPDetector{
				projectID:                 "my-project",
				cloudPlatform:             gcp.AppEngineFlex,
				appEngineServiceInstance:  "1472385723456792345",
				appEngineAvailabilityZone: "us-central1-c",
				appEngineRegion:           "us-central1",
				appEngineServiceName:      "my-service",
				appEngineServiceVersion:   "123456",
			}},
			expectedResource: resource.NewWithAttributes(semconv.SchemaURL,
				semconv.CloudProviderGCP,
				semconv.CloudAccountIDKey.String("my-project"),
				semconv.CloudPlatformGCPAppEngine,
				semconv.CloudRegionKey.String("us-central1"),
				semconv.CloudAvailabilityZoneKey.String("us-central1-c"),
				semconv.FaaSNameKey.String("my-service"),
				semconv.FaaSVersionKey.String("123456"),
				semconv.FaaSIDKey.String("1472385723456792345"),
			),
		},
		{
			desc: "App Engine Standard",
			detector: &detector{detector: &fakeGCPDetector{
				projectID:                 "my-project",
				cloudPlatform:             gcp.AppEngineStandard,
				appEngineServiceInstance:  "1472385723456792345",
				appEngineAvailabilityZone: "us-central1-c",
				appEngineRegion:           "us-central1",
				appEngineServiceName:      "my-service",
				appEngineServiceVersion:   "123456",
			}},
			expectedResource: resource.NewWithAttributes(semconv.SchemaURL,
				semconv.CloudProviderGCP,
				semconv.CloudAccountIDKey.String("my-project"),
				semconv.CloudPlatformGCPAppEngine,
				semconv.CloudRegionKey.String("us-central1"),
				semconv.CloudAvailabilityZoneKey.String("us-central1-c"),
				semconv.FaaSNameKey.String("my-service"),
				semconv.FaaSVersionKey.String("123456"),
				semconv.FaaSIDKey.String("1472385723456792345"),
			),
		},
		{
			desc: "Unknown Platform",
			detector: &detector{detector: &fakeGCPDetector{
				projectID:     "my-project",
				cloudPlatform: gcp.UnknownPlatform,
			}},
			expectedResource: resource.NewWithAttributes(semconv.SchemaURL,
				semconv.CloudProviderGCP,
				semconv.CloudAccountIDKey.String("my-project"),
			),
		},
		{
			desc: "error",
			detector: &detector{detector: &fakeGCPDetector{
				err: fmt.Errorf("failed to get metadata"),
			}},
			expectErr: true,
			expectedResource: resource.NewWithAttributes(semconv.SchemaURL,
				semconv.CloudProviderGCP,
			),
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			res, err := tc.detector.Detect(context.TODO())
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.expectedResource, res, "Resource object returned is incorrect")
		})
	}
}

// fakeGCPDetector implements gcpDetector and uses fake values.
type fakeGCPDetector struct {
	err                       error
	projectID                 string
	cloudPlatform             gcp.Platform
	gkeAvailabilityZone       string
	gkeRegion                 string
	gkeClusterName            string
	gkeHostID                 string
	gkeHostName               string
	faaSName                  string
	faaSVersion               string
	faaSID                    string
	faaSCloudRegion           string
	appEngineAvailabilityZone string
	appEngineRegion           string
	appEngineServiceName      string
	appEngineServiceVersion   string
	appEngineServiceInstance  string
	gceAvailabilityZone       string
	gceRegion                 string
	gceHostType               string
	gceHostID                 string
	gceHostName               string
}

func (f *fakeGCPDetector) ProjectID() (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.projectID, nil
}

func (f *fakeGCPDetector) CloudPlatform() gcp.Platform {
	return f.cloudPlatform
}

func (f *fakeGCPDetector) GKEAvailabilityZoneOrRegion() (string, gcp.LocationType, error) {
	if f.err != nil {
		return "", gcp.UndefinedLocation, f.err
	}
	if f.gkeAvailabilityZone != "" {
		return f.gkeAvailabilityZone, gcp.Zone, nil
	}
	return f.gkeRegion, gcp.Region, nil
}

func (f *fakeGCPDetector) GKEClusterName() (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.gkeClusterName, nil
}

func (f *fakeGCPDetector) GKEHostID() (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.gkeHostID, nil
}

func (f *fakeGCPDetector) GKEHostName() (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.gkeHostName, nil
}

func (f *fakeGCPDetector) FaaSName() (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.faaSName, nil
}

func (f *fakeGCPDetector) FaaSVersion() (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.faaSVersion, nil
}

func (f *fakeGCPDetector) FaaSID() (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.faaSID, nil
}

func (f *fakeGCPDetector) FaaSCloudRegion() (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.faaSCloudRegion, nil
}

func (f *fakeGCPDetector) AppEngineFlexAvailabilityZoneAndRegion() (string, string, error) {
	if f.err != nil {
		return "", "", f.err
	}
	return f.appEngineAvailabilityZone, f.appEngineRegion, nil
}

func (f *fakeGCPDetector) AppEngineStandardAvailabilityZone() (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.appEngineAvailabilityZone, nil
}

func (f *fakeGCPDetector) AppEngineStandardCloudRegion() (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.appEngineRegion, nil
}

func (f *fakeGCPDetector) AppEngineServiceName() (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.appEngineServiceName, nil
}

func (f *fakeGCPDetector) AppEngineServiceVersion() (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.appEngineServiceVersion, nil
}

func (f *fakeGCPDetector) AppEngineServiceInstance() (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.appEngineServiceInstance, nil
}

func (f *fakeGCPDetector) GCEAvailabilityZoneAndRegion() (string, string, error) {
	if f.err != nil {
		return "", "", f.err
	}
	return f.gceAvailabilityZone, f.gceRegion, nil
}

func (f *fakeGCPDetector) GCEHostType() (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.gceHostType, nil
}

func (f *fakeGCPDetector) GCEHostID() (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.gceHostID, nil
}

func (f *fakeGCPDetector) GCEHostName() (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.gceHostName, nil
}
