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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCloudPlatformAppEngineFlex(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{
		Vars: map[string]string{
			gaeServiceEnv: "foo",
		},
	})
	platform := d.CloudPlatform()
	assert.Equal(t, AppEngineFlex, platform)
}

func TestCloudPlatformAppEngineStandard(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{
		Vars: map[string]string{
			gaeServiceEnv: "foo",
			gaeEnv:        "standard",
		},
	})
	platform := d.CloudPlatform()
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
	platform := d.CloudPlatform()
	assert.Equal(t, GKE, platform)
}

func TestCloudPlatformK8sNotGKE(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{
		Vars: map[string]string{
			k8sServiceHostEnv: "foo",
		},
	})
	platform := d.CloudPlatform()
	assert.Equal(t, GCE, platform)
}

func TestCloudPlatformUnknown(t *testing.T) {
	d := NewTestDetector(&FakeMetadataTransport{Err: fmt.Errorf("no metadata server")}, &FakeOSProvider{
		Vars: map[string]string{
			k8sServiceHostEnv: "foo",
		},
	})
	platform := d.CloudPlatform()
	assert.Equal(t, UnknownPlatform, platform)
}

func TestCloudPlatformGCE(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{
		Vars: map[string]string{},
	})
	platform := d.CloudPlatform()
	assert.Equal(t, GCE, platform)
}

func TestCloudPlatformCloudRun(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{
		Vars: map[string]string{
			cloudRunConfigurationEnv: "foo",
		},
	})
	platform := d.CloudPlatform()
	assert.Equal(t, CloudRun, platform)
}

func TestCloudPlatformCloudRunJobs(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{
		Vars: map[string]string{
			cloudRunJobsEnv: "foo",
		},
	})
	platform := d.CloudPlatform()
	assert.Equal(t, CloudRunJob, platform)
}

func TestCloudPlatformCloudRunWorkerPool(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{
		Vars: map[string]string{
			cloudRunWorkerPoolEnv: "foo",
		},
	})
	platform := d.CloudPlatform()
	assert.Equal(t, CloudRunWorkerPool, platform)
}

func TestCloudPlatformCloudFunctions(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{
		Vars: map[string]string{
			cloudFunctionsTargetEnv: "foo",
		},
	})
	platform := d.CloudPlatform()
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
	platform := d.CloudPlatform()
	assert.Equal(t, BareMetalSolution, platform)
}

func TestProjectID(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{})
	project, err := d.ProjectID()
	assert.NoError(t, err)
	assert.Equal(t, fakeProjectID, project)
}

func TestProjectIDErr(t *testing.T) {
	d := NewTestDetector(&FakeMetadataTransport{
		Err: fmt.Errorf("fake error"),
	}, &FakeOSProvider{})
	project, err := d.ProjectID()
	assert.Error(t, err)
	assert.Empty(t, project)
}
