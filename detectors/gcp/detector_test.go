// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"
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
				gkeHostType:         "e2-standard-4",
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
			desc: "zonal GKE cluster with host.type enabled",
			detector: &detector{
				detector: &fakeGCPDetector{
					projectID:           "my-project",
					cloudPlatform:       internal.GKE,
					gkeHostID:           "1472385723456792345",
					gkeHostType:         "e2-standard-4",
					gkeClusterName:      "my-cluster",
					gkeAvailabilityZone: "us-central1-c",
				},
				cfg: config{gkeHostType: true},
			},
			expectedResource: resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.CloudProviderGCP,
				semconv.CloudAccountID("my-project"),
				semconv.CloudPlatformGCPKubernetesEngine,
				semconv.K8SClusterName("my-cluster"),
				semconv.CloudAvailabilityZone("us-central1-c"),
				semconv.HostID("1472385723456792345"),
				semconv.HostType("e2-standard-4"),
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
				gcpGceInstanceLabels: map[string]string{
					"env": "prod",
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
			),
		},
		{
			desc: "GCE with instance labels filter enabled",
			detector: &detector{
				detector: &fakeGCPDetector{
					projectID:              "my-project",
					cloudPlatform:          internal.GCE,
					gceHostID:              "1472385723456792345",
					gceHostName:            "my-gke-node-1234",
					gceHostType:            "n1-standard1",
					gceAvailabilityZone:    "us-central1-c",
					gceRegion:              "us-central1",
					gcpGceInstanceName:     "my-gke-node-1234",
					gcpGceInstanceHostname: "hostname",
					gcpGceInstanceLabels: map[string]string{
						"env":     "prod",
						"team":    "instrumentation",
						"ignored": "secret",
					},
				},
				cfg: config{
					gceLabelKeyFilter: func(k string) bool {
						return k == "env" || k == "team"
					},
				},
			},
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
				semconv.GCPGCEInstanceLabels("env", "prod"),
				semconv.GCPGCEInstanceLabels("team", "instrumentation"),
			),
		},
		{
			desc: "GCE with instance labels error",
			detector: &detector{
				detector: &fakeGCPDetector{
					projectID:              "my-project",
					cloudPlatform:          internal.GCE,
					gceHostID:              "1472385723456792345",
					gceHostName:            "my-gke-node-1234",
					gceHostType:            "n1-standard1",
					gceAvailabilityZone:    "us-central1-c",
					gceRegion:              "us-central1",
					gcpGceInstanceName:     "my-gke-node-1234",
					gcpGceInstanceHostname: "hostname",
					gceLabelsErr:           fmt.Errorf("failed to get GCE labels"),
				},
				cfg: config{
					gceLabelKeyFilter: func(string) bool { return true },
				},
			},
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

func TestNewDetectorWithOptions(t *testing.T) {
	filter := func(k string) bool { return k == "env" }
	d, ok := NewDetectorWithOptions(
		WithGKEHostType(),
		WithGCEInstanceLabelKeyFilter(filter),
	).(*detector)
	assert.True(t, ok)
	assert.True(t, d.cfg.gkeHostType)
	assert.NotNil(t, d.cfg.gceLabelKeyFilter)
	assert.True(t, d.cfg.gceLabelKeyFilter("env"))
	assert.False(t, d.cfg.gceLabelKeyFilter("other"))

	defaultDetector, ok := NewDetector().(*detector)
	assert.True(t, ok)
	assert.False(t, defaultDetector.cfg.gkeHostType)
	assert.Nil(t, defaultDetector.cfg.gceLabelKeyFilter)
}

func TestDetectGKEHostTypeContextCancellation(t *testing.T) {
	t.Setenv("GCE_METADATA_HOST", "169.254.169.254")

	fake := &fakeGCPDetector{
		projectID:           "my-project",
		cloudPlatform:       internal.GKE,
		gkeHostID:           "1472385723456792345",
		gkeHostType:         "e2-standard-4",
		gkeClusterName:      "my-cluster",
		gkeAvailabilityZone: "us-central1-c",
	}
	d := &detector{
		detector: fake,
		cfg:      config{gkeHostType: true},
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	res, err := d.Detect(ctx)
	assert.ErrorIs(t, err, resource.ErrPartialResource)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Same(t, ctx, fake.gkeHostTypeCtx)
	assert.Equal(t, resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderGCP,
		semconv.CloudAccountID("my-project"),
		semconv.CloudPlatformGCPKubernetesEngine,
		semconv.K8SClusterName("my-cluster"),
		semconv.CloudAvailabilityZone("us-central1-c"),
		semconv.HostID("1472385723456792345"),
	), res)
}

func TestDetectGCEInstanceLabelsContextCancellation(t *testing.T) {
	t.Setenv("GCE_METADATA_HOST", "169.254.169.254")

	fake := &fakeGCPDetector{
		projectID:              "my-project",
		cloudPlatform:          internal.GCE,
		gceHostID:              "1472385723456792345",
		gceHostName:            "my-gke-node-1234",
		gceHostType:            "n1-standard1",
		gceAvailabilityZone:    "us-central1-c",
		gceRegion:              "us-central1",
		gcpGceInstanceName:     "my-gke-node-1234",
		gcpGceInstanceHostname: "hostname",
		gcpGceInstanceLabels: map[string]string{
			"env": "prod",
		},
	}
	d := &detector{
		detector: fake,
		cfg: config{
			gceLabelKeyFilter: func(string) bool { return true },
		},
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	res, err := d.Detect(ctx)
	assert.ErrorIs(t, err, resource.ErrPartialResource)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Same(t, ctx, fake.gceLabelsCtx)
	assert.Equal(t, resource.NewWithAttributes(
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
	), res)
}

// fakeGCPDetector implements gcpDetector and uses fake values.
type fakeGCPDetector struct {
	err                             error
	migErr                          error
	gceLabelsErr                    error
	cloudPlatformCalls              int
	gkeHostTypeCtx                  context.Context
	gceLabelsCtx                    context.Context
	projectID                       string
	cloudPlatform                   internal.Platform
	gkeAvailabilityZone             string
	gkeRegion                       string
	gkeClusterName                  string
	gkeHostID                       string
	gkeHostType                     string
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
	gcpGceInstanceLabels            map[string]string
	gcpGceManagedInstanceGroup      internal.ManagedInstanceGroup
	gcpBareMetalSolutionCloudRegion string
	gcpBareMetalSolutionInstanceID  string
	gcpBareMetalSolutionProjectID   string
	cloudRunJobExecution            string
	cloudRunJobTaskIndex            string
}

func (f *fakeGCPDetector) ProjectID(_ context.Context) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.projectID, nil
}

func (f *fakeGCPDetector) CloudPlatform(_ context.Context) internal.Platform {
	f.cloudPlatformCalls++
	return f.cloudPlatform
}

func (f *fakeGCPDetector) GKEAvailabilityZoneOrRegion(_ context.Context) (string, internal.LocationType, error) {
	if f.err != nil {
		return "", internal.UndefinedLocation, f.err
	}
	if f.gkeAvailabilityZone != "" {
		return f.gkeAvailabilityZone, internal.Zone, nil
	}
	return f.gkeRegion, internal.Region, nil
}

func (f *fakeGCPDetector) GKEClusterName(_ context.Context) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.gkeClusterName, nil
}

func (f *fakeGCPDetector) GKEHostID(_ context.Context) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.gkeHostID, nil
}

func (f *fakeGCPDetector) GKEHostType(ctx context.Context) (string, error) {
	f.gkeHostTypeCtx = ctx
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if f.err != nil {
		return "", f.err
	}
	return f.gkeHostType, nil
}

func (f *fakeGCPDetector) GKEHostName() (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.gkeHostName, nil
}

func (f *fakeGCPDetector) FaaSName(_ context.Context) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.faaSName, nil
}

func (f *fakeGCPDetector) FaaSVersion(_ context.Context) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.faaSVersion, nil
}

func (f *fakeGCPDetector) FaaSID(_ context.Context) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.faaSID, nil
}

func (f *fakeGCPDetector) FaaSCloudRegion(_ context.Context) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.faaSCloudRegion, nil
}

func (f *fakeGCPDetector) AppEngineFlexAvailabilityZoneAndRegion(_ context.Context) (string, string, error) {
	if f.err != nil {
		return "", "", f.err
	}
	return f.appEngineAvailabilityZone, f.appEngineRegion, nil
}

func (f *fakeGCPDetector) AppEngineStandardAvailabilityZone(_ context.Context) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.appEngineAvailabilityZone, nil
}

func (f *fakeGCPDetector) AppEngineStandardCloudRegion(_ context.Context) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.appEngineRegion, nil
}

func (f *fakeGCPDetector) AppEngineServiceName(_ context.Context) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.appEngineServiceName, nil
}

func (f *fakeGCPDetector) AppEngineServiceVersion(_ context.Context) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.appEngineServiceVersion, nil
}

func (f *fakeGCPDetector) AppEngineServiceInstance(_ context.Context) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.appEngineServiceInstance, nil
}

func (f *fakeGCPDetector) GCEAvailabilityZoneAndRegion(_ context.Context) (string, string, error) {
	if f.err != nil {
		return "", "", f.err
	}
	return f.gceAvailabilityZone, f.gceRegion, nil
}

func (f *fakeGCPDetector) GCEHostType(_ context.Context) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.gceHostType, nil
}

func (f *fakeGCPDetector) GCEHostID(_ context.Context) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.gceHostID, nil
}

func (f *fakeGCPDetector) GCEHostName(_ context.Context) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.gceHostName, nil
}

func (f *fakeGCPDetector) GCEInstanceName(_ context.Context) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.gcpGceInstanceName, nil
}

func (f *fakeGCPDetector) GCEInstanceHostname(_ context.Context) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.gcpGceInstanceHostname, nil
}

func (f *fakeGCPDetector) GCEInstanceLabels(ctx context.Context) (map[string]string, error) {
	f.gceLabelsCtx = ctx
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if f.gceLabelsErr != nil {
		return nil, f.gceLabelsErr
	}
	if f.err != nil {
		return nil, f.err
	}
	return f.gcpGceInstanceLabels, nil
}

func (f *fakeGCPDetector) CloudRunJobExecution(_ context.Context) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.cloudRunJobExecution, nil
}

func (f *fakeGCPDetector) CloudRunJobTaskIndex(_ context.Context) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.cloudRunJobTaskIndex, nil
}

func (f *fakeGCPDetector) GCEManagedInstanceGroup(_ context.Context) (internal.ManagedInstanceGroup, error) {
	if f.migErr != nil {
		return internal.ManagedInstanceGroup{}, f.migErr
	}
	if f.err != nil {
		return internal.ManagedInstanceGroup{}, f.err
	}
	return f.gcpGceManagedInstanceGroup, nil
}

func (f *fakeGCPDetector) BareMetalSolutionInstanceID(_ context.Context) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.gcpBareMetalSolutionInstanceID, nil
}

func (f *fakeGCPDetector) BareMetalSolutionCloudRegion(_ context.Context) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.gcpBareMetalSolutionCloudRegion, nil
}

func (f *fakeGCPDetector) BareMetalSolutionProjectID(_ context.Context) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.gcpBareMetalSolutionProjectID, nil
}
