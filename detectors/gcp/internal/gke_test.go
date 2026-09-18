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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	fakeClusterName     = "my-cluster"
	fakeClusterLocation = "us-central1-c" // note this is different from fakeZone
)

func newGKEFakeMetadataTransport(t *testing.T, keyValues ...string) *FakeMetadataTransport {
	return newFakeMetadataTransport(t,
		append([]string{
			"instance/attributes/cluster-name", fakeClusterName,
			"instance/attributes/cluster-location", fakeClusterLocation,
		}, keyValues...)...,
	)
}

func TestGKEHostID(t *testing.T) {
	d := NewTestDetector(newGKEFakeMetadataTransport(t), &FakeOSProvider{})
	instanceID, err := d.GKEHostID()
	assert.NoError(t, err)
	assert.Equal(t, fakeInstanceID, instanceID)
}

func TestGKEHostIDErr(t *testing.T) {
	d := NewTestDetector(&FakeMetadataTransport{
		Err: fmt.Errorf("fake error"),
	}, &FakeOSProvider{})
	instanceID, err := d.GKEHostID()
	assert.Error(t, err)
	assert.Empty(t, instanceID)
}

func TestGKEClusterName(t *testing.T) {
	d := NewTestDetector(newGKEFakeMetadataTransport(t), &FakeOSProvider{})
	clusterName, err := d.GKEClusterName()
	assert.NoError(t, err)
	assert.Equal(t, fakeClusterName, clusterName)
}

func TestGKEClusterNameErr(t *testing.T) {
	d := NewTestDetector(&FakeMetadataTransport{
		Err: fmt.Errorf("fake error"),
	}, &FakeOSProvider{})
	clusterName, err := d.GKEClusterName()
	assert.Error(t, err)
	assert.Empty(t, clusterName)
}

func TestGKEHostType(t *testing.T) {
	var requestPath string
	computeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		_, _ = w.Write([]byte(`{"machineType":"https://www.googleapis.com/compute/v1/projects/my-project/zones/us-central1-a/machineTypes/e2-standard-4"}`))
	}))
	defer computeServer.Close()

	d := NewTestDetector(newGKEFakeMetadataTransport(t), &FakeOSProvider{})
	d.httpClient = computeServer.Client()
	d.computeBaseURL = computeServer.URL

	hostType, err := d.GKEHostType()
	assert.NoError(t, err)
	assert.Equal(t, "e2-standard-4", hostType)
	assert.Equal(t, "/projects/my-project/zones/us-central1-a/instances/my-instance", requestPath)
}

func TestGKEHostTypeErr(t *testing.T) {
	computeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer computeServer.Close()

	d := NewTestDetector(newGKEFakeMetadataTransport(t), &FakeOSProvider{})
	d.httpClient = computeServer.Client()
	d.computeBaseURL = computeServer.URL

	hostType, err := d.GKEHostType()
	assert.Error(t, err)
	assert.Empty(t, hostType)
}

func TestGKEAvailabilityZoneOrRegionZonal(t *testing.T) {
	d := NewTestDetector(newGKEFakeMetadataTransport(t), &FakeOSProvider{})
	location, zoneOrRegion, err := d.GKEAvailabilityZoneOrRegion()
	assert.NoError(t, err)
	assert.Equal(t, Zone, zoneOrRegion)
	assert.Equal(t, "us-central1-c", location)
}

func TestGKEAvailabilityZoneOrRegionRegional(t *testing.T) {
	d := NewTestDetector(newGKEFakeMetadataTransport(t,
		"instance/attributes/cluster-location", "us-central1",
	), &FakeOSProvider{})
	location, zoneOrRegion, err := d.GKEAvailabilityZoneOrRegion()
	assert.NoError(t, err)
	assert.Equal(t, Region, zoneOrRegion)
	assert.Equal(t, "us-central1", location)
}

func TestGKEAvailabilityZoneOrRegionMalformed(t *testing.T) {
	d := NewTestDetector(newGKEFakeMetadataTransport(t,
		"instance/attributes/cluster-location", "uscentral1c",
	), &FakeOSProvider{})
	location, zoneOrRegion, err := d.GKEAvailabilityZoneOrRegion()
	assert.Error(t, err)
	assert.Equal(t, UndefinedLocation, zoneOrRegion)
	assert.Empty(t, location)
}

func TestGKEAvailabilityZoneOrRegionErr(t *testing.T) {
	d := NewTestDetector(&FakeMetadataTransport{
		Err: fmt.Errorf("fake error"),
	}, &FakeOSProvider{})
	location, zoneOrRegion, err := d.GKEAvailabilityZoneOrRegion()
	assert.Error(t, err)
	assert.Equal(t, UndefinedLocation, zoneOrRegion)
	assert.Empty(t, location)
}
