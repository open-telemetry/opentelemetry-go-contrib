// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testContextKey struct{}

func TestCloudPlatformAppEngineFlex(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{
		Vars: map[string]string{
			gaeServiceEnv: "foo",
		},
	})
	platform := d.CloudPlatform(t.Context())
	assert.Equal(t, AppEngineFlex, platform)
}

func TestCloudPlatformAppEngineStandard(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{
		Vars: map[string]string{
			gaeServiceEnv: "foo",
			gaeEnv:        "standard",
		},
	})
	platform := d.CloudPlatform(t.Context())
	assert.Equal(t, AppEngineStandard, platform)
}

func TestCloudPlatformGKE(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t,
		"instance/attributes/cluster-location", "us-central-1a",
	), &FakeOSProvider{
		Vars: map[string]string{
			k8sServiceHostEnv: "foo",
		},
	})
	platform := d.CloudPlatform(t.Context())
	assert.Equal(t, GKE, platform)
}

func TestCloudPlatformK8sNotGKE(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{
		Vars: map[string]string{
			k8sServiceHostEnv: "foo",
		},
	})
	platform := d.CloudPlatform(t.Context())
	assert.Equal(t, GCE, platform)
}

func TestCloudPlatformUnknown(t *testing.T) {
	d := NewTestDetector(&FakeMetadataTransport{Err: fmt.Errorf("no metadata server")}, &FakeOSProvider{
		Vars: map[string]string{
			k8sServiceHostEnv: "foo",
		},
	})
	platform := d.CloudPlatform(t.Context())
	assert.Equal(t, UnknownPlatform, platform)
}

func TestCloudPlatformGCE(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{
		Vars: map[string]string{},
	})
	platform := d.CloudPlatform(t.Context())
	assert.Equal(t, GCE, platform)
}

func TestCloudPlatformCloudRun(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{
		Vars: map[string]string{
			cloudRunConfigurationEnv: "foo",
		},
	})
	platform := d.CloudPlatform(t.Context())
	assert.Equal(t, CloudRun, platform)
}

func TestCloudPlatformCloudRunJobs(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{
		Vars: map[string]string{
			cloudRunJobsEnv: "foo",
		},
	})
	platform := d.CloudPlatform(t.Context())
	assert.Equal(t, CloudRunJob, platform)
}

func TestCloudPlatformCloudRunWorkerPool(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{
		Vars: map[string]string{
			cloudRunWorkerPoolEnv: "foo",
		},
	})
	platform := d.CloudPlatform(t.Context())
	assert.Equal(t, CloudRunWorkerPool, platform)
}

func TestCloudPlatformCloudFunctions(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{
		Vars: map[string]string{
			cloudFunctionsTargetEnv: "foo",
		},
	})
	platform := d.CloudPlatform(t.Context())
	assert.Equal(t, CloudFunctions, platform)
}

func TestCloudPlatformBareMetalSolution(t *testing.T) {
	d := NewTestDetector(&FakeMetadataTransport{Err: fmt.Errorf("no metadata server")}, &FakeOSProvider{
		Vars: map[string]string{
			bmsInstanceIDEnv: "foo",
			bmsProjectIDEnv:  "bar",
			bmsRegionEnv:     "qux",
		},
	})
	platform := d.CloudPlatform(t.Context())
	assert.Equal(t, BareMetalSolution, platform)
}

func TestCloudPlatformBareMetalSolutionLocation(t *testing.T) {
	d := NewTestDetector(&FakeMetadataTransport{Err: fmt.Errorf("no metadata server")}, &FakeOSProvider{
		Vars: map[string]string{
			bmsInstanceIDEnv: "foo",
			bmsProjectIDEnv:  "bar",
			bmsLocationEnv:   "qux",
		},
	})
	platform := d.CloudPlatform(t.Context())
	assert.Equal(t, BareMetalSolution, platform)
}

func TestProjectID(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{})
	project, err := d.ProjectID(t.Context())
	assert.NoError(t, err)
	assert.Equal(t, fakeProjectID, project)
}

func TestProjectIDErr(t *testing.T) {
	d := NewTestDetector(&FakeMetadataTransport{
		Err: fmt.Errorf("fake error"),
	}, &FakeOSProvider{})
	project, err := d.ProjectID(t.Context())
	assert.Error(t, err)
	assert.Empty(t, project)
}

func TestCloudPlatformContextPropagation(t *testing.T) {
	for _, tc := range []struct {
		name         string
		vars         map[string]string
		wantPlatform Platform
	}{
		{
			name: "GKE probe",
			vars: map[string]string{
				k8sServiceHostEnv: "foo",
			},
			wantPlatform: GKE,
		},
		{
			name:         "GCE probe",
			vars:         map[string]string{},
			wantPlatform: GCE,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			transport := newGKEFakeMetadataTransport(t)
			d := NewTestDetector(transport, &FakeOSProvider{Vars: tc.vars})

			ctx := context.WithValue(t.Context(), testContextKey{}, "ctx-val")
			assert.Equal(t, tc.wantPlatform, d.CloudPlatform(ctx))
			require.NotEmpty(t, transport.Contexts)
			for _, reqCtx := range transport.Contexts {
				assert.Equal(t, "ctx-val", reqCtx.Value(testContextKey{}))
			}

			transport.Contexts = nil
			canceledCtx, cancel := context.WithCancel(ctx)
			cancel()
			assert.Equal(t, UnknownPlatform, d.CloudPlatform(canceledCtx))
			require.NotEmpty(t, transport.Contexts)
			for _, reqCtx := range transport.Contexts {
				assert.ErrorIs(t, reqCtx.Err(), context.Canceled)
			}

			transport.Contexts = nil
			expiredCtx, cancelDeadline := context.WithDeadline(ctx, time.Now().Add(-time.Second))
			defer cancelDeadline()
			assert.Equal(t, UnknownPlatform, d.CloudPlatform(expiredCtx))
			require.NotEmpty(t, transport.Contexts)
			for _, reqCtx := range transport.Contexts {
				assert.ErrorIs(t, reqCtx.Err(), context.DeadlineExceeded)
			}
		})
	}
}

func TestMetadataGettersContextPropagation(t *testing.T) {
	for _, tc := range []struct {
		name string
		call func(ctx context.Context, d *Detector) error
	}{
		{
			name: "ProjectID",
			call: func(ctx context.Context, d *Detector) error {
				_, err := d.ProjectID(ctx)
				return err
			},
		},
		{
			name: "GCEHostType",
			call: func(ctx context.Context, d *Detector) error {
				_, err := d.GCEHostType(ctx)
				return err
			},
		},
		{
			name: "GCEHostID",
			call: func(ctx context.Context, d *Detector) error {
				_, err := d.GCEHostID(ctx)
				return err
			},
		},
		{
			name: "GCEHostName",
			call: func(ctx context.Context, d *Detector) error {
				_, err := d.GCEHostName(ctx)
				return err
			},
		},
		{
			name: "GCEInstanceName",
			call: func(ctx context.Context, d *Detector) error {
				_, err := d.GCEInstanceName(ctx)
				return err
			},
		},
		{
			name: "GCEInstanceHostname",
			call: func(ctx context.Context, d *Detector) error {
				_, err := d.GCEInstanceHostname(ctx)
				return err
			},
		},
		{
			name: "GCEAvailabilityZoneAndRegion",
			call: func(ctx context.Context, d *Detector) error {
				_, _, err := d.GCEAvailabilityZoneAndRegion(ctx)
				return err
			},
		},
		{
			name: "GCEManagedInstanceGroup",
			call: func(ctx context.Context, d *Detector) error {
				_, err := d.GCEManagedInstanceGroup(ctx)
				return err
			},
		},
		{
			name: "GKEHostID",
			call: func(ctx context.Context, d *Detector) error {
				_, err := d.GKEHostID(ctx)
				return err
			},
		},
		{
			name: "GKEClusterName",
			call: func(ctx context.Context, d *Detector) error {
				_, err := d.GKEClusterName(ctx)
				return err
			},
		},
		{
			name: "GKEAvailabilityZoneOrRegion",
			call: func(ctx context.Context, d *Detector) error {
				_, _, err := d.GKEAvailabilityZoneOrRegion(ctx)
				return err
			},
		},
		{
			name: "FaaSID",
			call: func(ctx context.Context, d *Detector) error {
				_, err := d.FaaSID(ctx)
				return err
			},
		},
		{
			name: "FaaSCloudRegion",
			call: func(ctx context.Context, d *Detector) error {
				_, err := d.FaaSCloudRegion(ctx)
				return err
			},
		},
		{
			name: "AppEngineStandardAvailabilityZone",
			call: func(ctx context.Context, d *Detector) error {
				_, err := d.AppEngineStandardAvailabilityZone(ctx)
				return err
			},
		},
		{
			name: "AppEngineStandardCloudRegion",
			call: func(ctx context.Context, d *Detector) error {
				_, err := d.AppEngineStandardCloudRegion(ctx)
				return err
			},
		},
		{
			name: "AppEngineFlexAvailabilityZoneAndRegion",
			call: func(ctx context.Context, d *Detector) error {
				_, _, err := d.AppEngineFlexAvailabilityZoneAndRegion(ctx)
				return err
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			transport := newGKEFakeMetadataTransport(t,
				regionMetadataAttr, "/projects/123/regions/us-central1",
				"instance/attributes/"+createdByInstanceAttr, "projects/123/zones/us-central1-a/instanceGroupManagers/my-mig",
			)
			d := NewTestDetector(transport, &FakeOSProvider{})

			ctx := context.WithValue(t.Context(), testContextKey{}, "ctx-val")
			require.NoError(t, tc.call(ctx, d))
			require.NotEmpty(t, transport.Contexts)
			for _, reqCtx := range transport.Contexts {
				assert.Equal(t, "ctx-val", reqCtx.Value(testContextKey{}))
			}

			canceledCtx, cancel := context.WithCancel(ctx)
			cancel()
			assert.ErrorIs(t, tc.call(canceledCtx, d), context.Canceled)

			expiredCtx, cancelDeadline := context.WithDeadline(ctx, time.Now().Add(-time.Second))
			defer cancelDeadline()
			assert.ErrorIs(t, tc.call(expiredCtx, d), context.DeadlineExceeded)
		})
	}
}
