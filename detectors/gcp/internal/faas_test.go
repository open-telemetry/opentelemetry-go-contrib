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

func TestFaaSName(t *testing.T) {
	d := NewTestDetector(&FakeMetadataTransport{}, &FakeOSProvider{
		Vars: map[string]string{
			faasServiceEnv: "my-service",
		},
	})
	name, err := d.FaaSName()
	assert.NoError(t, err)
	assert.Equal(t, "my-service", name)
}

func TestFaaSJobsName(t *testing.T) {
	d := NewTestDetector(&FakeMetadataTransport{}, &FakeOSProvider{
		Vars: map[string]string{
			cloudRunJobsEnv: "my-service",
		},
	})
	name, err := d.FaaSName()
	assert.NoError(t, err)
	assert.Equal(t, "my-service", name)
}

func TestFaaSWorkerPoolName(t *testing.T) {
	d := NewTestDetector(&FakeMetadataTransport{}, &FakeOSProvider{
		Vars: map[string]string{
			cloudRunWorkerPoolEnv: "my-service",
		},
	})
	name, err := d.FaaSName()
	assert.NoError(t, err)
	assert.Equal(t, "my-service", name)
}

func TestFaaSNameErr(t *testing.T) {
	d := NewTestDetector(&FakeMetadataTransport{}, &FakeOSProvider{
		Vars: map[string]string{},
	})
	name, err := d.FaaSName()
	assert.Error(t, err)
	assert.Empty(t, name)
}

func TestFaaSVersion(t *testing.T) {
	d := NewTestDetector(&FakeMetadataTransport{}, &FakeOSProvider{
		Vars: map[string]string{
			faasRevisionEnv: "version-123",
		},
	})
	version, err := d.FaaSVersion()
	assert.NoError(t, err)
	assert.Equal(t, "version-123", version)
}

func TestFaaSWorkerPoolVersion(t *testing.T) {
	d := NewTestDetector(&FakeMetadataTransport{}, &FakeOSProvider{
		Vars: map[string]string{
			cloudRunWorkerPoolEnv: "my-pool",
			cloudRunRevisionEnv:   "version-123",
		},
	})
	version, err := d.FaaSVersion()
	assert.NoError(t, err)
	assert.Equal(t, "version-123", version)
}

func TestFaaSVersionErr(t *testing.T) {
	d := NewTestDetector(&FakeMetadataTransport{}, &FakeOSProvider{
		Vars: map[string]string{},
	})
	version, err := d.FaaSVersion()
	assert.Error(t, err)
	assert.Empty(t, version)
}

func TestFaaSJobExecution(t *testing.T) {
	d := NewTestDetector(&FakeMetadataTransport{}, &FakeOSProvider{
		Vars: map[string]string{
			cloudRunJobExecutionEnv: "version-123",
		},
	})
	version, err := d.CloudRunJobExecution()
	assert.NoError(t, err)
	assert.Equal(t, "version-123", version)
}

func TestFaaSJobExecutionErr(t *testing.T) {
	d := NewTestDetector(&FakeMetadataTransport{}, &FakeOSProvider{
		Vars: map[string]string{},
	})
	name, err := d.CloudRunJobExecution()
	assert.Error(t, err)
	assert.Empty(t, name)
}

func TestFaaSJobTaskIndex(t *testing.T) {
	d := NewTestDetector(&FakeMetadataTransport{}, &FakeOSProvider{
		Vars: map[string]string{
			cloudRunJobTaskIndexEnv: "5",
		},
	})
	version, err := d.CloudRunJobTaskIndex()
	assert.NoError(t, err)
	assert.Equal(t, "5", version)
}

func TestFaaSJobTaskIndexErr(t *testing.T) {
	d := NewTestDetector(&FakeMetadataTransport{}, &FakeOSProvider{
		Vars: map[string]string{},
	})
	name, err := d.CloudRunJobTaskIndex()
	assert.Error(t, err)
	assert.Empty(t, name)
}

func TestFaaSID(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{})
	instance, err := d.FaaSID()
	assert.NoError(t, err)
	assert.Equal(t, fakeInstanceID, instance)
}

func TestFaaSIDErr(t *testing.T) {
	d := NewTestDetector(&FakeMetadataTransport{
		Err: fmt.Errorf("fake error"),
	}, &FakeOSProvider{})
	instance, err := d.FaaSID()
	assert.Error(t, err)
	assert.Empty(t, instance)
}

func TestFaaSCloudRegion(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t,
		regionMetadataAttr, "/projects/123/regions/us-central1",
	), &FakeOSProvider{})
	instance, err := d.FaaSCloudRegion()
	assert.NoError(t, err)
	assert.Equal(t, "us-central1", instance)
}

func TestFaaSCloudRegionErr(t *testing.T) {
	d := NewTestDetector(&FakeMetadataTransport{
		Err: fmt.Errorf("fake error"),
	}, &FakeOSProvider{})
	instance, err := d.FaaSCloudRegion()
	assert.Error(t, err)
	assert.Empty(t, instance)
}
