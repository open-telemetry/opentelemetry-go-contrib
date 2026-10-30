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
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestGCEInstanceLabels(t *testing.T) {
	var requestPath string
	computeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		_, _ = w.Write([]byte(`{"labels":{"env":"prod","team":"instrumentation"}}`))
	}))
	defer computeServer.Close()

	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{})
	d.httpClient = computeServer.Client()
	d.computeBaseURL = computeServer.URL

	labels, err := d.GCEInstanceLabels(t.Context())
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{
		"env":  "prod",
		"team": "instrumentation",
	}, labels)
	assert.Equal(t, "/projects/my-project/zones/us-central1-a/instances/my-instance", requestPath)
}

func TestGCEInstanceLabelsEmpty(t *testing.T) {
	computeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"machineType":"n1-standard-1"}`))
	}))
	defer computeServer.Close()

	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{})
	d.httpClient = computeServer.Client()
	d.computeBaseURL = computeServer.URL

	labels, err := d.GCEInstanceLabels(t.Context())
	assert.NoError(t, err)
	assert.Empty(t, labels)
}

func TestGCEInstanceLabelsErr(t *testing.T) {
	computeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer computeServer.Close()

	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{})
	d.httpClient = computeServer.Client()
	d.computeBaseURL = computeServer.URL

	labels, err := d.GCEInstanceLabels(t.Context())
	assert.Error(t, err)
	assert.Nil(t, labels)
}

func TestGCEInstanceLabelsMetadataErr(t *testing.T) {
	for _, missingKey := range []string{
		"project/project-id",
		"instance/zone",
		"instance/name",
	} {
		t.Run(missingKey, func(t *testing.T) {
			fmpi := newFakeMetadataTransport(t)
			delete(fmpi.Metadata, missingKey)
			d := NewTestDetector(fmpi, &FakeOSProvider{})

			labels, err := d.GCEInstanceLabels(t.Context())
			assert.Error(t, err)
			assert.Nil(t, labels)
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestGCEInstanceLabelsDefaultBaseURL(t *testing.T) {
	var requestURL string
	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{})
	d.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			requestURL = req.URL.String()
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"labels":{"env":"prod"}}`)),
			}, nil
		}),
	}

	labels, err := d.GCEInstanceLabels(t.Context())
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{"env": "prod"}, labels)
	assert.Equal(t, "https://compute.googleapis.com/compute/v1/projects/my-project/zones/us-central1-a/instances/my-instance", requestURL)
}

func TestGCEInstanceLabelsInvalidJSON(t *testing.T) {
	computeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{invalid`))
	}))
	defer computeServer.Close()

	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{})
	d.httpClient = computeServer.Client()
	d.computeBaseURL = computeServer.URL

	labels, err := d.GCEInstanceLabels(t.Context())
	assert.Error(t, err)
	assert.Nil(t, labels)
}

func TestGCEInstanceLabelsInvalidBaseURL(t *testing.T) {
	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{})
	d.httpClient = &http.Client{}
	d.computeBaseURL = "://invalid-url"

	labels, err := d.GCEInstanceLabels(t.Context())
	assert.Error(t, err)
	assert.Nil(t, labels)
}

func TestGCEInstanceLabelsDefaultClientErr(t *testing.T) {
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/nonexistent/credentials.json")
	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{})

	labels, err := d.GCEInstanceLabels(t.Context())
	assert.Error(t, err)
	assert.Nil(t, labels)
}

func TestGCEInstanceLabelsContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	computeServer := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		cancel()
		<-r.Context().Done()
	}))
	defer computeServer.Close()

	d := NewTestDetector(newFakeMetadataTransport(t), &FakeOSProvider{})
	d.httpClient = computeServer.Client()
	d.computeBaseURL = computeServer.URL

	labels, err := d.GCEInstanceLabels(ctx)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, labels)
}
