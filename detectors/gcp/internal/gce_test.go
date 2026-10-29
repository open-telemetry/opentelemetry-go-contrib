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

func TestGCEHostType(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{})
	hostType, err := d.GCEHostType()
	assert.NoError(t, err)
	assert.Equal(t, fakeInstanceMachineType, hostType)
}

func TestGCEHostTypeErr(t *testing.T) {
	d := NewTestDetector(&FakeMetadataTransport{
		Err: fmt.Errorf("fake error"),
	}, &FakeOSProvider{})
	hostType, err := d.GCEHostType()
	assert.Error(t, err)
	assert.Empty(t, hostType)
}

func TestGCEHostID(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{})
	instanceID, err := d.GCEHostID()
	assert.NoError(t, err)
	assert.Equal(t, fakeInstanceID, instanceID)
}

func TestGCEHostIDErr(t *testing.T) {
	d := NewTestDetector(&FakeMetadataTransport{
		Err: fmt.Errorf("fake error"),
	}, &FakeOSProvider{})
	instanceID, err := d.GCEHostID()
	assert.Error(t, err)
	assert.Empty(t, instanceID)
}

func TestGCEHostName(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{})
	hostName, err := d.GCEHostName()
	assert.NoError(t, err)
	assert.Equal(t, fakeInstanceName, hostName)
}

func TestGCEInstanceName(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{})
	hostName, err := d.GCEInstanceName()
	assert.NoError(t, err)
	assert.Equal(t, fakeInstanceName, hostName)
}

func TestGCEInstanceHostname(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{})
	hostName, err := d.GCEInstanceHostname()
	assert.NoError(t, err)
	assert.Equal(t, fakeInstanceHostname, hostName)
}

func TestGCEInstanceCustomHostname(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t,
		"instance/name", "my-host-123",
		"instance/hostname", "custom-dns.fakevm.example",
	), &FakeOSProvider{})
	hostName, err := d.GCEInstanceHostname()
	assert.NoError(t, err)
	assert.Equal(t, "custom-dns.fakevm.example", hostName)
}

func TestGCEHostNameErr(t *testing.T) {
	d := NewTestDetector(&FakeMetadataTransport{
		Err: fmt.Errorf("fake error"),
	}, &FakeOSProvider{})
	hostName, err := d.GCEHostName()
	assert.Error(t, err)
	assert.Empty(t, hostName)
}

func TestGCEAvailabilityZoneAndRegion(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{})
	zone, region, err := d.GCEAvailabilityZoneAndRegion()
	assert.NoError(t, err)
	assert.Equal(t, fakeZone, zone)
	assert.Equal(t, fakeRegion, region)
}

func TestGCEAvailabilityZoneAndRegionMalformedZone(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t,
		"instance/zone", "us-central1",
	), &FakeOSProvider{})
	zone, region, err := d.GCEAvailabilityZoneAndRegion()
	assert.Error(t, err)
	assert.Empty(t, zone)
	assert.Empty(t, region)
}

func TestGCEAvailabilityZoneAndRegionNoZone(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t,
		"instance/zone", "",
	), &FakeOSProvider{})
	zone, region, err := d.GCEAvailabilityZoneAndRegion()
	assert.Error(t, err)
	assert.Empty(t, zone)
	assert.Empty(t, region)
}

func TestGCEAvailabilityZoneAndRegionErr(t *testing.T) {
	d := NewTestDetector(&FakeMetadataTransport{
		Err: fmt.Errorf("fake error"),
	}, &FakeOSProvider{})
	zone, region, err := d.GCEAvailabilityZoneAndRegion()
	assert.Error(t, err)
	assert.Empty(t, zone)
	assert.Empty(t, region)
}

func TestGCEManagedInstanceGroup(t *testing.T) {
	for _, test := range []struct {
		createdBy string
		mig       *ManagedInstanceGroup
	}{
		{
			"projects/123456789012/zones/us-central1-f/instanceGroupManagers/igm-metadata",
			&ManagedInstanceGroup{
				Name:     "igm-metadata",
				Location: "us-central1-f",
				Type:     Zone,
			},
		},
		{
			"projects/123456789012/regions/us-central1-f/instanceGroupManagers/igm-metadata",
			&ManagedInstanceGroup{
				Name:     "igm-metadata",
				Location: "us-central1-f",
				Type:     Region,
			},
		},
		{
			"projects/123456789012/zones/us-central1-f/someOtherInstanceGroup/igm-metadata",
			&ManagedInstanceGroup{},
		},
		{
			"",
			&ManagedInstanceGroup{},
		},
		{
			"",
			nil,
		},
	} {
		t.Run(test.createdBy, func(t *testing.T) {
			fmpi := newFakeMetadataTransport(t)
			if test.createdBy != "" {
				fmpi.Metadata["instance/attributes/created-by"] = test.createdBy
			}
			if test.mig == nil {
				fmpi.Err = fmt.Errorf("fake error")
			}
			d := NewTestDetector(fmpi, &FakeOSProvider{})
			mig, err := d.GCEManagedInstanceGroup()
			if test.mig == nil {
				assert.Error(t, err)
			} else {
				assert.Equal(t, *test.mig, mig)
			}
		})
	}
}
