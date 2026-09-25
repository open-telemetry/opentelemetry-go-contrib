// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package oraclecloud

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

// newFakeServer starts an httptest server serving meta as JSON and returns its
// URL. The server is closed via t.Cleanup.
func newFakeServer(t *testing.T, meta computeMetadata) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, authHeader, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(meta)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

// newTestDetector returns a detector pointed at url instead of the real
// link-local metadata endpoint.
func newTestDetector(url string, opts ...Option) *ResourceDetector {
	d := NewResourceDetector(opts...)
	d.endpoint = url
	return d
}

func TestNewResourceDetector(t *testing.T) {
	d := NewResourceDetector()
	require.NotNil(t, d)
	assert.Equal(t, defaultEndpoint, d.endpoint)
}

func TestNewResourceDetectorDisablesProxy(t *testing.T) {
	proxy := "http://" + net.JoinHostPort("127.0.0.1", "1")
	t.Setenv("HTTP_PROXY", proxy)
	t.Setenv("HTTPS_PROXY", proxy)

	detector := NewResourceDetector()

	transport, ok := detector.client.Transport.(*http.Transport)
	require.True(t, ok)
	assert.Nil(t, transport.Proxy)
}

func TestDetect_Success(t *testing.T) {
	url := newFakeServer(t, computeMetadata{
		HostID:             "ocid1.instance.oc1..aaaaaaa",
		HostDisplayName:    "my-instance",
		HostType:           "VM.Standard.E4.Flex",
		CanonicalRegionID:  "us-ashburn-1",
		AvailabilityDomain: "AD-1",
		Metadata: instanceMetadata{
			OKEClusterDisplayName: "my-oke-cluster",
			Realm:                 "oc1",
		},
	})

	res, err := newTestDetector(url).Detect(t.Context())
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
	url := newFakeServer(t, computeMetadata{
		HostID:             "ocid1.instance.oc1..aaaaaaa",
		HostDisplayName:    "my-instance",
		HostType:           "VM.Standard.E4.Flex",
		CanonicalRegionID:  "us-ashburn-1",
		AvailabilityDomain: "AD-1",
		Metadata: instanceMetadata{
			Realm: "oc1",
		},
	})

	res, err := newTestDetector(url).Detect(t.Context())
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
		semconv.OracleCloudRealm("oc1"),
	)

	assert.Equal(t, expected, res)
}

func TestDetect_NotOracleCloud(t *testing.T) {
	// Closed server → connection refused → not on Oracle Cloud → empty resource, no error.
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	url := srv.URL
	srv.Close()

	res, err := newTestDetector(url).Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetect_ClientError404_NotOracleCloud(t *testing.T) {
	// A client error means something other than Oracle Cloud metadata service answered:
	// not on Oracle Cloud, so no error is reported.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	res, err := newTestDetector(srv.URL).Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetect_ServerError(t *testing.T) {
	// The metadata service answered but failed: surface the error instead of
	// silently reporting "not on Oracle Cloud".
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	res, err := newTestDetector(srv.URL).Detect(t.Context())
	require.Error(t, err)
	assert.Nil(t, res)
	assert.Contains(t, err.Error(), "received non-OK response from Oracle Cloud IMDS")
}

func TestDetect_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{ invalid json `))
	}))
	t.Cleanup(srv.Close)

	res, err := newTestDetector(srv.URL).Detect(t.Context())
	require.Error(t, err)
	assert.Nil(t, res)
	assert.Contains(t, err.Error(), "failed to decode Oracle Cloud IMDS response")
}

func TestDetect_PartialResource(t *testing.T) {
	// Serve empty JSON object so missing host.id, hostname, host.type, region, and availability domain branches are all covered.
	url := newFakeServer(t, computeMetadata{})

	res, err := newTestDetector(url).Detect(t.Context())
	require.ErrorIs(t, err, resource.ErrPartialResource)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderOracleCloud,
		semconv.CloudPlatformOracleCloudCompute,
	)

	assert.Equal(t, expected, res)
}

func TestDetect_WithAttributeFilter(t *testing.T) {
	url := newFakeServer(t, computeMetadata{
		HostID:             "ocid1.instance.oc1..aaaaaaa",
		HostDisplayName:    "my-instance",
		HostType:           "VM.Standard.E4.Flex",
		CanonicalRegionID:  "us-ashburn-1",
		AvailabilityDomain: "AD-1",
	})

	filter := func(kv attribute.KeyValue) bool {
		return kv.Key == semconv.CloudProviderKey || kv.Key == semconv.HostIDKey
	}

	res, err := newTestDetector(url, WithAttributeFilter(filter)).Detect(t.Context())
	require.NoError(t, err)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderOracleCloud,
		semconv.HostID("ocid1.instance.oc1..aaaaaaa"),
	)

	assert.Equal(t, expected, res)
}

func TestDetect_RegionFallback(t *testing.T) {
	url := newFakeServer(t, computeMetadata{
		HostID:             "ocid1.instance.oc1..aaaaaaa",
		HostDisplayName:    "my-instance",
		HostType:           "VM.Standard.E4.Flex",
		RegionID:           "us-phoenix-1",
		AvailabilityDomain: "AD-1",
	})

	res, err := newTestDetector(url).Detect(t.Context())
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
	url := newFakeServer(t, computeMetadata{
		HostID:             "ocid1.instance.oc1..aaaaaaa",
		HostDisplayName:    "my-instance",
		HostType:           "VM.Standard.E4.Flex",
		CanonicalRegionID:  "us-ashburn-1",
		RegionID:           "iad",
		AvailabilityDomain: "AD-1",
	})

	res, err := newTestDetector(url).Detect(t.Context())
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
