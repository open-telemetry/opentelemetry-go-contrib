// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"

	"go.opentelemetry.io/contrib/detectors/gcp/internal"
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
				cloudPlatform:       internal.GKE,
				gkeHostID:           "1472385723456792345",
				gkeClusterName:      "my-cluster",
				gkeAvailabilityZone: "us-central1-c",
			}},
			expectedResource: resource.NewWithAttributes(
				semconv.SchemaURL,
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
				cloudPlatform:  internal.GKE,
				gkeHostID:      "1472385723456792345",
				gkeClusterName: "my-cluster",
				gkeRegion:      "us-central1",
			}},
			expectedResource: resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.CloudProviderGCP,
				semconv.CloudAccountID("my-project"),
				semconv.CloudPlatformGCPKubernetesEngine,
				semconv.K8SClusterName("my-cluster"),
				semconv.CloudRegion("us-central1"),
				semconv.HostID("1472385723456792345"),
			),
		},
		{
			desc: "GCE without MIG",
			detector: &detector{detector: &fakeGCPDetector{
				projectID:              "my-project",
				cloudPlatform:          internal.GCE,
				gceHostID:              "1472385723456792345",
				gceHostName:            "my-gke-node-1234",
				gceHostType:            "n1-standard1",
				gceAvailabilityZone:    "us-central1-c",
				gceRegion:              "us-central1",
				gcpGceInstanceName:     "my-gke-node-1234",
				gcpGceInstanceHostname: "hostname",
			}},
			expectedResource: resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.CloudProviderGCP,
				semconv.CloudAccountID("my-project"),
				semconv.CloudPlatformGCPComputeEngine,
				semconv.HostID("1472385723456792345"),
				semconv.HostName("my-gke-node-1234"),
				semconv.GCPGCEInstanceNameKey.String("my-gke-node-1234"),
				semconv.GCPGCEInstanceHostnameKey.String("hostname"),
				semconv.HostType("n1-standard1"),
				semconv.CloudRegion("us-central1"),
				semconv.CloudAvailabilityZone("us-central1-c"),
			),
		},
		{
			desc: "Cloud Run",
			detector: &detector{detector: &fakeGCPDetector{
				projectID:       "my-project",
				cloudPlatform:   internal.CloudRun,
				faaSID:          "1472385723456792345",
				faaSCloudRegion: "us-central1",
				faaSName:        "my-service",
				faaSVersion:     "123456",
			}},
			expectedResource: resource.NewWithAttributes(
				semconv.SchemaURL,
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
				cloudPlatform:        internal.CloudRunJob,
				faaSID:               "1472385723456792345",
				faaSCloudRegion:      "us-central1",
				faaSName:             "my-service",
				cloudRunJobExecution: "my-service-ekdih",
				cloudRunJobTaskIndex: "0",
			}},
			expectedResource: resource.NewWithAttributes(
				semconv.SchemaURL,
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
				cloudPlatform:        internal.CloudRunJob,
				faaSID:               "1472385723456792345",
				faaSCloudRegion:      "us-central1",
				faaSName:             "my-service",
				cloudRunJobExecution: "my-service-ekdih",
				cloudRunJobTaskIndex: "bad-value",
			}},
			expectedResource: resource.NewWithAttributes(
				semconv.SchemaURL,
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
				cloudPlatform:   internal.CloudFunctions,
				faaSID:          "1472385723456792345",
				faaSCloudRegion: "us-central1",
				faaSName:        "my-service",
				faaSVersion:     "123456",
			}},
			expectedResource: resource.NewWithAttributes(
				semconv.SchemaURL,
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
				cloudPlatform:             internal.AppEngineFlex,
				appEngineServiceInstance:  "1472385723456792345",
				appEngineAvailabilityZone: "us-central1-c",
				appEngineRegion:           "us-central1",
				appEngineServiceName:      "my-service",
				appEngineServiceVersion:   "123456",
			}},
			expectedResource: resource.NewWithAttributes(
				semconv.SchemaURL,
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
				cloudPlatform:             internal.AppEngineStandard,
				appEngineServiceInstance:  "1472385723456792345",
				appEngineAvailabilityZone: "us-central1-c",
				appEngineRegion:           "us-central1",
				appEngineServiceName:      "my-service",
				appEngineServiceVersion:   "123456",
			}},
			expectedResource: resource.NewWithAttributes(
				semconv.SchemaURL,
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
			desc: "GCE with MIG",
			detector: &detector{detector: &fakeGCPDetector{
				projectID:              "my-project",
				cloudPlatform:          internal.GCE,
				gceHostID:              "1472385723456792345",
				gceHostName:            "my-gke-node-1234",
				gceHostType:            "n1-standard1",
				gceAvailabilityZone:    "us-central1-c",
				gceRegion:              "us-central1",
				gcpGceInstanceName:     "my-gke-node-1234",
				gcpGceInstanceHostname: "hostname",
				gcpGceManagedInstanceGroup: internal.ManagedInstanceGroup{
					Name:     "my-mig",
					Location: "us-central1",
					Type:     internal.Region,
				},
			}},
			expectedResource: resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.CloudProviderGCP,
				semconv.CloudAccountID("my-project"),
				semconv.CloudPlatformGCPComputeEngine,
				semconv.HostID("1472385723456792345"),
				semconv.HostName("my-gke-node-1234"),
				semconv.GCPGCEInstanceNameKey.String("my-gke-node-1234"),
				semconv.GCPGCEInstanceHostnameKey.String("hostname"),
				semconv.HostType("n1-standard1"),
				semconv.CloudRegion("us-central1"),
				semconv.CloudAvailabilityZone("us-central1-c"),
				semconv.GCPGCEInstanceGroupManagerName("my-mig"),
				semconv.GCPGCEInstanceGroupManagerRegion("us-central1"),
			),
		},
		{
			desc: "GCE with zonal MIG",
			detector: &detector{detector: &fakeGCPDetector{
				projectID:              "my-project",
				cloudPlatform:          internal.GCE,
				gceHostID:              "1472385723456792345",
				gceHostName:            "my-gke-node-1234",
				gceHostType:            "n1-standard1",
				gceAvailabilityZone:    "us-central1-c",
				gceRegion:              "us-central1",
				gcpGceInstanceName:     "my-gke-node-1234",
				gcpGceInstanceHostname: "hostname",
				gcpGceManagedInstanceGroup: internal.ManagedInstanceGroup{
					Name:     "my-mig",
					Location: "us-central1-c",
					Type:     internal.Zone,
				},
			}},
			expectedResource: resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.CloudProviderGCP,
				semconv.CloudAccountID("my-project"),
				semconv.CloudPlatformGCPComputeEngine,
				semconv.HostID("1472385723456792345"),
				semconv.HostName("my-gke-node-1234"),
				semconv.GCPGCEInstanceNameKey.String("my-gke-node-1234"),
				semconv.GCPGCEInstanceHostnameKey.String("hostname"),
				semconv.HostType("n1-standard1"),
				semconv.CloudRegion("us-central1"),
				semconv.CloudAvailabilityZone("us-central1-c"),
				semconv.GCPGCEInstanceGroupManagerName("my-mig"),
				semconv.GCPGCEInstanceGroupManagerZone("us-central1-c"),
			),
		},
		{
			desc: "GCE with MIG invalid type",
			detector: &detector{detector: &fakeGCPDetector{
				projectID:              "my-project",
				cloudPlatform:          internal.GCE,
				gceHostID:              "1472385723456792345",
				gceHostName:            "my-gke-node-1234",
				gceHostType:            "n1-standard1",
				gceAvailabilityZone:    "us-central1-c",
				gceRegion:              "us-central1",
				gcpGceInstanceName:     "my-gke-node-1234",
				gcpGceInstanceHostname: "hostname",
				gcpGceManagedInstanceGroup: internal.ManagedInstanceGroup{
					Name:     "my-mig",
					Location: "us-central1",
					Type:     internal.LocationType(99),
				},
			}},
			expectErr: true,
			expectedResource: resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.CloudProviderGCP,
				semconv.CloudAccountID("my-project"),
				semconv.CloudPlatformGCPComputeEngine,
				semconv.HostID("1472385723456792345"),
				semconv.HostName("my-gke-node-1234"),
				semconv.GCPGCEInstanceNameKey.String("my-gke-node-1234"),
				semconv.GCPGCEInstanceHostnameKey.String("hostname"),
				semconv.HostType("n1-standard1"),
				semconv.CloudRegion("us-central1"),
				semconv.CloudAvailabilityZone("us-central1-c"),
				semconv.GCPGCEInstanceGroupManagerName("my-mig"),
			),
		},
		{
			desc: "GCE with MIG error",
			detector: &detector{detector: &fakeGCPDetector{
				projectID:              "my-project",
				cloudPlatform:          internal.GCE,
				gceHostID:              "1472385723456792345",
				gceHostName:            "my-gke-node-1234",
				gceHostType:            "n1-standard1",
				gceAvailabilityZone:    "us-central1-c",
				gceRegion:              "us-central1",
				gcpGceInstanceName:     "my-gke-node-1234",
				gcpGceInstanceHostname: "hostname",
				migErr:                 fmt.Errorf("failed to get MIG"),
			}},
			expectErr: true,
			expectedResource: resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.CloudProviderGCP,
				semconv.CloudAccountID("my-project"),
				semconv.CloudPlatformGCPComputeEngine,
				semconv.HostID("1472385723456792345"),
				semconv.HostName("my-gke-node-1234"),
				semconv.GCPGCEInstanceNameKey.String("my-gke-node-1234"),
				semconv.GCPGCEInstanceHostnameKey.String("hostname"),
				semconv.HostType("n1-standard1"),
				semconv.CloudRegion("us-central1"),
				semconv.CloudAvailabilityZone("us-central1-c"),
			),
		},
		{
			desc: "Cloud Run Worker Pool",
			detector: &detector{detector: &fakeGCPDetector{
				projectID:       "my-project",
				cloudPlatform:   internal.CloudRunWorkerPool,
				faaSID:          "1472385723456792345",
				faaSCloudRegion: "us-central1",
				faaSName:        "my-service",
				faaSVersion:     "123456",
			}},
			expectedResource: resource.NewWithAttributes(
				semconv.SchemaURL,
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
			desc: "Bare Metal Solution",
			detector: &detector{detector: &fakeGCPDetector{
				projectID:                       "my-project",
				cloudPlatform:                   internal.BareMetalSolution,
				gcpBareMetalSolutionCloudRegion: "us-central1",
				gcpBareMetalSolutionInstanceID:  "1472385723456792345",
				gcpBareMetalSolutionProjectID:   "my-project",
			}},
			expectedResource: resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.CloudProviderGCP,
				semconv.CloudPlatformGCPBareMetalSolution,
				semconv.CloudAccountID("my-project"),
				semconv.HostID("1472385723456792345"),
				semconv.CloudRegion("us-central1"),
			),
		},
		{
			desc: "Unknown Platform",
			detector: &detector{detector: &fakeGCPDetector{
				projectID:     "my-project",
				cloudPlatform: internal.UnknownPlatform,
			}},
			expectedResource: resource.NewWithAttributes(
				semconv.SchemaURL,
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
			expectedResource: resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.CloudProviderGCP,
			),
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			res, err := tc.detector.Detect(t.Context())
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.expectedResource, res, "Resource object returned is incorrect")
		})
	}
}

func TestBareMetalSolutionEnv(t *testing.T) {
	expectedResource := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderGCP,
		semconv.CloudPlatformGCPBareMetalSolution,
		semconv.CloudAccountID("my-project"),
		semconv.HostID("my-instance-id"),
		semconv.CloudRegion("us-central1"),
	)

	t.Run("documented BMS_LOCATION env vars", func(t *testing.T) {
		t.Setenv("BMS_PROJECT_ID", "my-project")
		t.Setenv("BMS_LOCATION", "us-central1")
		t.Setenv("BMS_INSTANCE_ID", "my-instance-id")

		res, err := NewDetector().Detect(t.Context())
		assert.NoError(t, err)
		assert.Equal(t, expectedResource, res)
	})

	t.Run("legacy BMS_REGION alias", func(t *testing.T) {
		t.Setenv("BMS_PROJECT_ID", "my-project")
		t.Setenv("BMS_REGION", "us-central1")
		t.Setenv("BMS_INSTANCE_ID", "my-instance-id")

		res, err := NewDetector().Detect(t.Context())
		assert.NoError(t, err)
		assert.Equal(t, expectedResource, res)
	})

	t.Run("partial BMS env vars do not trigger BMS resource or extra CloudPlatform probe", func(t *testing.T) {
		fake := &fakeGCPDetector{
			projectID:                     "my-project",
			cloudPlatform:                 internal.UnknownPlatform,
			gcpBareMetalSolutionProjectID: "my-project",
		}
		d := &detector{detector: fake}

		res, err := d.Detect(t.Context())
		assert.NoError(t, err)
		assert.NotEqual(t, expectedResource, res)
		assert.Equal(t, 1, fake.cloudPlatformCalls, "CloudPlatform should only be called once after OnGCE")
	})
}

// fakeGCPDetector implements gcpDetector and uses fake values.
type fakeGCPDetector struct {
	err                             error
	migErr                          error
	cloudPlatformCalls              int
	projectID                       string
	cloudPlatform                   internal.Platform
	gkeAvailabilityZone             string
	gkeRegion                       string
	gkeClusterName                  string
	gkeHostID                       string
	gkeHostName                     string
	faaSName                        string
	faaSVersion                     string
	faaSID                          string
	faaSCloudRegion                 string
	appEngineAvailabilityZone       string
	appEngineRegion                 string
	appEngineServiceName            string
	appEngineServiceVersion         string
	appEngineServiceInstance        string
	gceAvailabilityZone             string
	gceRegion                       string
	gceHostType                     string
	gceHostID                       string
	gceHostName                     string
	gcpGceInstanceName              string
	gcpGceInstanceHostname          string
	gcpGceManagedInstanceGroup      internal.ManagedInstanceGroup
	gcpBareMetalSolutionCloudRegion string
	gcpBareMetalSolutionInstanceID  string
	gcpBareMetalSolutionProjectID   string
	cloudRunJobExecution            string
	cloudRunJobTaskIndex            string
}

func (f *fakeGCPDetector) ProjectID() (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.projectID, nil
}

func (f *fakeGCPDetector) CloudPlatform() internal.Platform {
	f.cloudPlatformCalls++
	return f.cloudPlatform
}

func (f *fakeGCPDetector) GKEAvailabilityZoneOrRegion() (string, internal.LocationType, error) {
	if f.err != nil {
		return "", internal.UndefinedLocation, f.err
	}
	if f.gkeAvailabilityZone != "" {
		return f.gkeAvailabilityZone, internal.Zone, nil
	}
	return f.gkeRegion, internal.Region, nil
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

func (f *fakeGCPDetector) GCEManagedInstanceGroup() (internal.ManagedInstanceGroup, error) {
	if f.migErr != nil {
		return internal.ManagedInstanceGroup{}, f.migErr
	}
	if f.err != nil {
		return internal.ManagedInstanceGroup{}, f.err
	}
	return f.gcpGceManagedInstanceGroup, nil
}

func (f *fakeGCPDetector) BareMetalSolutionInstanceID() (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.gcpBareMetalSolutionInstanceID, nil
}

func (f *fakeGCPDetector) BareMetalSolutionCloudRegion() (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.gcpBareMetalSolutionCloudRegion, nil
}

func (f *fakeGCPDetector) BareMetalSolutionProjectID() (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.gcpBareMetalSolutionProjectID, nil
}
