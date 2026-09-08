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

func TestAppEngineServiceName(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{
		Vars: map[string]string{
			gaeServiceEnv: "my-service",
		},
	})
	serviceName, err := d.AppEngineServiceName()
	assert.NoError(t, err)
	assert.Equal(t, "my-service", serviceName)
}

func TestAppEngineServiceNameErr(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{
		Vars: map[string]string{},
	})
	name, err := d.AppEngineServiceName()
	assert.Error(t, err)
	assert.Empty(t, name)
}

func TestAppEngineServiceVersion(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{
		Vars: map[string]string{
			gaeVersionEnv: "my-version",
		},
	})
	version, err := d.AppEngineServiceVersion()
	assert.NoError(t, err)
	assert.Equal(t, "my-version", version)
}

func TestAppEngineServiceVersionErr(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{
		Vars: map[string]string{},
	})
	version, err := d.AppEngineServiceVersion()
	assert.Error(t, err)
	assert.Empty(t, version)
}

func TestAppEngineServiceInstance(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{
		Vars: map[string]string{
			gaeInstanceEnv: "instance-123",
		},
	})
	instance, err := d.AppEngineServiceInstance()
	assert.NoError(t, err)
	assert.Equal(t, "instance-123", instance)
}

func TestAppEngineServiceInstanceErr(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{
		Vars: map[string]string{},
	})
	instance, err := d.AppEngineServiceInstance()
	assert.Error(t, err)
	assert.Empty(t, instance)
}

func TestAppEngineStandardAvailabilityZone(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t,
		"instance/zone", "us16",
	), &FakeOSProvider{})
	zone, err := d.AppEngineStandardAvailabilityZone()
	assert.NoError(t, err)
	assert.Equal(t, "us16", zone)
}

func TestAppEngineStandardAvailabilityZoneErr(t *testing.T) {
	d := NewTestDetector(&FakeMetadataTransport{
		Err: fmt.Errorf("fake error"),
	}, &FakeOSProvider{})
	zone, err := d.AppEngineStandardAvailabilityZone()
	assert.Error(t, err)
	assert.Empty(t, zone)
}

func TestAppEngineStandardCloudRegion(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t,
		regionMetadataAttr, "/projects/123/regions/us-central1",
	), &FakeOSProvider{})
	instance, err := d.AppEngineStandardCloudRegion()
	assert.NoError(t, err)
	assert.Equal(t, "us-central1", instance)
}

func TestAppEngineStandardCloudRegionErr(t *testing.T) {
	d := NewTestDetector(&FakeMetadataTransport{
		Err: fmt.Errorf("fake error"),
	}, &FakeOSProvider{})
	instance, err := d.AppEngineStandardCloudRegion()
	assert.Error(t, err)
	assert.Empty(t, instance)
}

func TestAppEngineFlexAvailabilityZoneAndRegion(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{})
	zone, region, err := d.AppEngineFlexAvailabilityZoneAndRegion()
	assert.NoError(t, err)
	assert.Equal(t, fakeZone, zone)
	assert.Equal(t, fakeRegion, region)
}

func TestAppEngineFlexAvailabilityZoneAndRegionMalformedZone(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t,
		"instance/zone", "us-central1",
	), &FakeOSProvider{})
	zone, region, err := d.AppEngineFlexAvailabilityZoneAndRegion()
	assert.Error(t, err)
	assert.Empty(t, zone)
	assert.Empty(t, region)
}

func TestAppEngineFlexAvailabilityZoneAndRegionNoZone(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t,
		"instance/zone", "",
	), &FakeOSProvider{})
	zone, region, err := d.AppEngineFlexAvailabilityZoneAndRegion()
	assert.Error(t, err)
	assert.Empty(t, zone)
	assert.Empty(t, region)
}

func TestAppEngineFlexAvailabilityZoneAndRegionErr(t *testing.T) {
	d := NewTestDetector(&FakeMetadataTransport{
		Err: fmt.Errorf("fake error"),
	}, &FakeOSProvider{})
	zone, region, err := d.AppEngineFlexAvailabilityZoneAndRegion()
	assert.Error(t, err)
	assert.Empty(t, zone)
	assert.Empty(t, region)
}
