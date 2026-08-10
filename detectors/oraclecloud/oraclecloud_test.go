// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package oraclecloud

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

func TestDetect_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, authHeader, r.Header.Get("Authorization"))
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "ocid1.instance.oc1..aaaaaaa",
				"displayName": "my-instance",
				"shape": "VM.Standard.E4.Flex",
				"canonicalRegionName": "us-ashburn-1",
				"availabilityDomain": "AD-1",
				"metadata": {
					"oke-cluster-display-name": "my-oke-cluster",
					"realm": "oc1"
				}
			}`))
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer ts.Close()

	detector := NewResourceDetector()
	detector.endpoint = ts.URL

	res, err := detector.Detect(t.Context())
	require.NoError(t, err)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderOracleCloud,
		semconv.CloudPlatformOracleCloudOKE,
		semconv.HostID("ocid1.instance.oc1..aaaaaaa"),
		semconv.HostName("my-instance"),
		semconv.HostType("VM.Standard.E4.Flex"),
		semconv.CloudRegion("us-ashburn-1"),
		semconv.CloudAvailabilityZone("AD-1"),
		semconv.K8SClusterName("my-oke-cluster"),
		semconv.OracleCloudRealm("oc1"),
	)

	assert.Equal(t, expected, res)
}

func TestDetect_NonOKE(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "ocid1.instance.oc1..aaaaaaa",
				"displayName": "my-instance",
				"shape": "VM.Standard.E4.Flex",
				"canonicalRegionName": "us-ashburn-1",
				"availabilityDomain": "AD-1",
				"metadata": {
					"realm": "oc1"
				}
			}`))
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer ts.Close()

	detector := NewResourceDetector()
	detector.endpoint = ts.URL

	res, err := detector.Detect(t.Context())
	require.NoError(t, err)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderOracleCloud,
		semconv.HostID("ocid1.instance.oc1..aaaaaaa"),
		semconv.HostName("my-instance"),
		semconv.HostType("VM.Standard.E4.Flex"),
		semconv.CloudRegion("us-ashburn-1"),
		semconv.CloudAvailabilityZone("AD-1"),
		semconv.OracleCloudRealm("oc1"),
	)

	assert.Equal(t, expected, res)
}

func TestDetect_NotOracleCloud(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	detector := NewResourceDetector()
	detector.endpoint = ts.URL

	res, err := detector.Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetect_PartialResource(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			// Missing host.id and host.type
			_, _ = w.Write([]byte(`{
				"displayName": "my-instance",
				"canonicalRegionName": "us-ashburn-1",
				"availabilityDomain": "AD-1"
			}`))
			return
		}
	}))
	defer ts.Close()

	detector := NewResourceDetector()
	detector.endpoint = ts.URL

	res, err := detector.Detect(t.Context())
	require.ErrorIs(t, err, resource.ErrPartialResource)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderOracleCloud,
		semconv.HostName("my-instance"),
		semconv.CloudRegion("us-ashburn-1"),
		semconv.CloudAvailabilityZone("AD-1"),
	)

	assert.Equal(t, expected, res)
}

func TestDetect_WithAttributeFilter(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "ocid1.instance.oc1..aaaaaaa",
				"displayName": "my-instance",
				"shape": "VM.Standard.E4.Flex",
				"canonicalRegionName": "us-ashburn-1",
				"availabilityDomain": "AD-1"
			}`))
			return
		}
	}))
	defer ts.Close()

	filter := func(kv attribute.KeyValue) bool {
		return kv.Key == semconv.CloudProviderKey || kv.Key == semconv.HostIDKey
	}

	detector := NewResourceDetector(WithAttributeFilter(filter))
	detector.endpoint = ts.URL

	res, err := detector.Detect(t.Context())
	require.NoError(t, err)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderOracleCloud,
		semconv.HostID("ocid1.instance.oc1..aaaaaaa"),
	)

	assert.Equal(t, expected, res)
}

func TestDetect_RegionFallback(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "ocid1.instance.oc1..aaaaaaa",
				"displayName": "my-instance",
				"shape": "VM.Standard.E4.Flex",
				"region": "us-phoenix-1",
				"availabilityDomain": "AD-1"
			}`))
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer ts.Close()

	detector := NewResourceDetector()
	detector.endpoint = ts.URL

	res, err := detector.Detect(t.Context())
	require.NoError(t, err)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderOracleCloud,
		semconv.CloudPlatformOracleCloudCompute,
		semconv.HostID("ocid1.instance.oc1..aaaaaaa"),
		semconv.HostName("my-instance"),
		semconv.HostType("VM.Standard.E4.Flex"),
		semconv.CloudRegion("us-phoenix-1"),
		semconv.CloudAvailabilityZone("AD-1"),
	)

	assert.Equal(t, expected, res)
}

func TestDetect_CanonicalRegionPrecedence(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "ocid1.instance.oc1..aaaaaaa",
				"displayName": "my-instance",
				"shape": "VM.Standard.E4.Flex",
				"canonicalRegionName": "us-ashburn-1",
				"region": "iad",
				"availabilityDomain": "AD-1"
			}`))
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer ts.Close()

	detector := NewResourceDetector()
	detector.endpoint = ts.URL

	res, err := detector.Detect(t.Context())
	require.NoError(t, err)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderOracleCloud,
		semconv.CloudPlatformOracleCloudCompute,
		semconv.HostID("ocid1.instance.oc1..aaaaaaa"),
		semconv.HostName("my-instance"),
		semconv.HostType("VM.Standard.E4.Flex"),
		semconv.CloudRegion("us-ashburn-1"),
		semconv.CloudAvailabilityZone("AD-1"),
	)

	assert.Equal(t, expected, res)
}

