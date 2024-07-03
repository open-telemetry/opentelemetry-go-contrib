// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package gcp // import "go.opentelemetry.io/contrib/detectors/gcp"

import (
	"context"
	"fmt"
	"testing"

	"github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp"
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func TestDetect(t *testing.T) {
	// Set this before all tests to ensure metadata.onGCE() returns true
	t.Setenv("GCE_METADATA_HOST", "169.254.169.254")

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
				semconv.CloudAccountID("my-project"),
				semconv.CloudPlatformGCPKubernetesEngine,
				semconv.K8SClusterName("my-cluster"),
				semconv.CloudAvailabilityZone("us-central1-c"),
				semconv.HostID("1472385723456792345"),
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
				semconv.CloudAccountID("my-project"),
				semconv.CloudPlatformGCPKubernetesEngine,
				semconv.K8SClusterName("my-cluster"),
				semconv.CloudRegion("us-central1"),
				semconv.HostID("1472385723456792345"),
			),
		},
		{
			desc: "GCE",
			detector: &detector{detector: &fakeGCPDetector{
				projectID:              "my-project",
				cloudPlatform:          gcp.GCE,
				gceHostID:              "1472385723456792345",
				gceHostName:            "my-gke-node-1234",
				gceHostType:            "n1-standard1",
				gceAvailabilityZone:    "us-central1-c",
				gceRegion:              "us-central1",
				gcpGceInstanceName:     "my-gke-node-1234",
				gcpGceInstanceHostname: "hostname",
			}},
			expectedResource: resource.NewWithAttributes(semconv.SchemaURL,
				semconv.CloudProviderGCP,
				semconv.CloudAccountID("my-project"),
				semconv.CloudPlatformGCPComputeEngine,
				semconv.HostID("1472385723456792345"),
				semconv.HostName("my-gke-node-1234"),
				semconv.GCPGceInstanceNameKey.String("my-gke-node-1234"),
				semconv.GCPGceInstanceHostnameKey.String("hostname"),
				semconv.HostType("n1-standard1"),
				semconv.CloudRegion("us-central1"),
				semconv.CloudAvailabilityZone("us-central1-c"),
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
				semconv.CloudAccountID("my-project"),
				semconv.CloudPlatformGCPCloudRun,
				semconv.CloudRegion("us-central1"),
				semconv.FaaSName("my-service"),
				semconv.FaaSVersion("123456"),
				semconv.FaaSInstance("1472385723456792345"),
			),
		},
		{
			desc: "Cloud Run Job",
			detector: &detector{detector: &fakeGCPDetector{
				projectID:            "my-project",
				cloudPlatform:        gcp.CloudRunJob,
				faaSID:               "1472385723456792345",
				faaSCloudRegion:      "us-central1",
				faaSName:             "my-service",
				cloudRunJobExecution: "my-service-ekdih",
				cloudRunJobTaskIndex: "0",
			}},
			expectedResource: resource.NewWithAttributes(semconv.SchemaURL,
				semconv.CloudProviderGCP,
				semconv.CloudAccountID("my-project"),
				semconv.CloudPlatformGCPCloudRun,
				semconv.CloudRegion("us-central1"),
				semconv.FaaSName("my-service"),
				semconv.GCPCloudRunJobExecution("my-service-ekdih"),
				semconv.GCPCloudRunJobTaskIndex(0),
				semconv.FaaSInstance("1472385723456792345"),
			),
		},
		{
			desc: "Cloud Run Job Bad Index",
			detector: &detector{detector: &fakeGCPDetector{
				projectID:            "my-project",
				cloudPlatform:        gcp.CloudRunJob,
				faaSID:               "1472385723456792345",
				faaSCloudRegion:      "us-central1",
				faaSName:             "my-service",
				cloudRunJobExecution: "my-service-ekdih",
				cloudRunJobTaskIndex: "bad-value",
			}},
			expectedResource: resource.NewWithAttributes(semconv.SchemaURL,
				semconv.CloudProviderGCP,
				semconv.CloudAccountID("my-project"),
				semconv.CloudPlatformGCPCloudRun,
				semconv.CloudRegion("us-central1"),
				semconv.FaaSName("my-service"),
				semconv.GCPCloudRunJobExecution("my-service-ekdih"),
				semconv.FaaSInstance("1472385723456792345"),
			),
			expectErr: true,
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
				semconv.CloudAccountID("my-project"),
				semconv.CloudPlatformGCPCloudFunctions,
				semconv.CloudRegion("us-central1"),
				semconv.FaaSName("my-service"),
				semconv.FaaSVersion("123456"),
				semconv.FaaSInstance("1472385723456792345"),
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
				semconv.CloudAccountID("my-project"),
				semconv.CloudPlatformGCPAppEngine,
				semconv.CloudRegion("us-central1"),
				semconv.CloudAvailabilityZone("us-central1-c"),
				semconv.FaaSName("my-service"),
				semconv.FaaSVersion("123456"),
				semconv.FaaSInstance("1472385723456792345"),
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
				semconv.CloudAccountID("my-project"),
				semconv.CloudPlatformGCPAppEngine,
				semconv.CloudRegion("us-central1"),
				semconv.CloudAvailabilityZone("us-central1-c"),
				semconv.FaaSName("my-service"),
				semconv.FaaSVersion("123456"),
				semconv.FaaSInstance("1472385723456792345"),
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
				semconv.CloudAccountID("my-project"),
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
	gcpGceInstanceName        string
	gcpGceInstanceHostname    string
	cloudRunJobExecution      string
	cloudRunJobTaskIndex      string
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

func (f *fakeGCPDetector) GCEInstanceName() (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.gcpGceInstanceName, nil
}

func (f *fakeGCPDetector) GCEInstanceHostname() (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.gcpGceInstanceHostname, nil
}

func (f *fakeGCPDetector) CloudRunJobExecution() (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.cloudRunJobExecution, nil
}

func (f *fakeGCPDetector) CloudRunJobTaskIndex() (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.cloudRunJobTaskIndex, nil
}
