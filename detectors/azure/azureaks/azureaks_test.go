// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azureaks

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

// newFakeServer starts an httptest server serving meta as JSON and returns its
// URL along with a counter of the requests it received. The server is closed via
// t.Cleanup.
func newFakeServer(t *testing.T, meta aksMetadata) (string, *atomic.Int64) {
	t.Helper()
	var requests atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(meta)
	}))
	t.Cleanup(srv.Close)
	return srv.URL, &requests
}

// newTestDetector returns a detector pointed at url instead of the real
// link-local metadata endpoint.
func newTestDetector(url string, opts ...Option) *ResourceDetector {
	d := NewResourceDetector(opts...)
	d.endpoint = url
	return d
}

// onKubernetes makes the detector believe it is running inside a pod.
func onKubernetes(t *testing.T) {
	t.Helper()
	t.Setenv(kubernetesServiceHostEnvVar, "10.0.0.1")
}

func TestNewResourceDetector(t *testing.T) {
	d := NewResourceDetector()
	require.NotNil(t, d)
	assert.Equal(t, defaultEndpoint, d.endpoint)
}

func TestParseClusterName(t *testing.T) {
	tests := []struct {
		name          string
		resourceGroup string
		expected      string
	}{
		{
			name:          "generated group, returns cluster name",
			resourceGroup: "MC_myResourceGroup_AKSCluster_eastus",
			expected:      "AKSCluster",
		},
		{
			name:          "generated group with a lowercase prefix, returns cluster name",
			resourceGroup: "mc_myResourceGroup_AKSCluster_eastus",
			expected:      "AKSCluster",
		},
		{
			name:          "resource group contains underscores, falls back to the group name",
			resourceGroup: "MC_Resource_Group_AKSCluster_eastus",
			expected:      "MC_Resource_Group_AKSCluster_eastus",
		},
		{
			name:          "cluster name contains underscores, falls back to the group name",
			resourceGroup: "MC_myResourceGroup_AKS_Cluster_eastus",
			expected:      "MC_myResourceGroup_AKS_Cluster_eastus",
		},
		{
			name:          "custom infrastructure resource group, returns the group name",
			resourceGroup: "infra-group_name",
			expected:      "infra-group_name",
		},
		{
			name:          "custom infrastructure resource group with four segments, returns the group name",
			resourceGroup: "dev_infra_group_name",
			expected:      "dev_infra_group_name",
		},
		{
			// This case is unlikely because it would require the user to create
			// a custom infrastructure resource group with the MC prefix and the
			// correct number of underscores.
			name:          "custom infrastructure resource group with an MC prefix",
			resourceGroup: "MC_group_name_location",
			expected:      "name",
		},
		{
			name:          "empty resource group",
			resourceGroup: "",
			expected:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, parseClusterName(tt.resourceGroup))
		})
	}
}

func TestDetect_OK(t *testing.T) {
	onKubernetes(t)
	url, _ := newFakeServer(t, aksMetadata{
		ResourceGroupName: "MC_myResourceGroup_AKSCluster_eastus",
	})

	res, err := newTestDetector(url).Detect(t.Context())
	require.NoError(t, err)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderAzure,
		semconv.CloudPlatformAzureAKS,
		semconv.K8SClusterName("AKSCluster"),
	)
	assert.Equal(t, expected, res)
}

func TestDetect_CustomResourceGroup(t *testing.T) {
	// A custom infrastructure resource group does not encode the cluster name,
	// so it is reported unchanged.
	onKubernetes(t)
	url, _ := newFakeServer(t, aksMetadata{ResourceGroupName: "custom-infra-group"})

	res, err := newTestDetector(url).Detect(t.Context())
	require.NoError(t, err)

	val, ok := res.Set().Value(semconv.K8SClusterNameKey)
	require.True(t, ok)
	assert.Equal(t, attribute.StringValue("custom-infra-group"), val)
}

func TestDetect_NotOnKubernetes(t *testing.T) {
	// Without KUBERNETES_SERVICE_HOST the detector must not even query the
	// metadata service: a plain Azure VM is not AKS.
	t.Setenv(kubernetesServiceHostEnvVar, "")
	url, requests := newFakeServer(t, aksMetadata{ResourceGroupName: "MC_rg_cluster_eastus"})

	res, err := newTestDetector(url).Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
	assert.Equal(t, int64(0), requests.Load(), "expected no metadata request")
}

func TestDetect_SingleRequest(t *testing.T) {
	onKubernetes(t)
	url, requests := newFakeServer(t, aksMetadata{ResourceGroupName: "MC_rg_cluster_eastus"})

	_, err := newTestDetector(url).Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, int64(1), requests.Load(), "expected the metadata document to be fetched once")
}

func TestDetect_MetadataRequest(t *testing.T) {
	onKubernetes(t)

	var gotHeader, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("Metadata")
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(aksMetadata{ResourceGroupName: "MC_rg_cluster_eastus"})
	}))
	t.Cleanup(srv.Close)

	d := NewResourceDetector()
	d.endpoint = srv.URL + "?api-version=2021-12-13&format=json"

	_, err := d.Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "true", gotHeader)
	assert.Equal(t, "api-version=2021-12-13&format=json", gotQuery)
}

func TestDetect_NotOnAzure(t *testing.T) {
	// A client error means something other than the Azure Instance Metadata
	// Service answered the request: not on Azure, so no error is reported.
	onKubernetes(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	res, err := newTestDetector(srv.URL).Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetect_ConnectionRefused(t *testing.T) {
	// Closed server -> connection refused -> not on Azure -> empty resource, no error.
	onKubernetes(t)
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close()

	res, err := newTestDetector(url).Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetect_ServerError(t *testing.T) {
	// The metadata service answered but failed: surface the error instead of
	// silently reporting "not on Azure".
	onKubernetes(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	res, err := newTestDetector(srv.URL).Detect(t.Context())
	require.Error(t, err)
	assert.Nil(t, res)
}

func TestDetect_MalformedJSON(t *testing.T) {
	// 200 OK with a body that isn't valid JSON. The metadata service responded,
	// so this is a failure rather than "not on Azure".
	onKubernetes(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not json"))
	}))
	t.Cleanup(srv.Close)

	res, err := newTestDetector(srv.URL).Detect(t.Context())
	require.Error(t, err)
	assert.Nil(t, res)
}

func TestDetect_PartialFailure(t *testing.T) {
	// Serve JSON with resourceGroupName absent.
	onKubernetes(t)
	url, _ := newFakeServer(t, aksMetadata{})

	res, err := newTestDetector(url).Detect(t.Context())
	require.Error(t, err)
	assert.ErrorIs(t, err, resource.ErrPartialResource)

	// cloud.provider and cloud.platform must still be present.
	for _, kv := range []attribute.KeyValue{
		semconv.CloudProviderAzure,
		semconv.CloudPlatformAzureAKS,
	} {
		val, ok := res.Set().Value(kv.Key)
		assert.True(t, ok, "expected %s to be present", kv.Key)
		assert.Equal(t, kv.Value, val)
	}

	// k8s.cluster.name must be absent.
	_, ok := res.Set().Value(semconv.K8SClusterNameKey)
	assert.False(t, ok, "expected k8s.cluster.name to be absent")
}

func TestDetect_WithAttributeFilter(t *testing.T) {
	onKubernetes(t)
	url, _ := newFakeServer(t, aksMetadata{
		ResourceGroupName: "MC_myResourceGroup_AKSCluster_eastus",
	})

	filter := attribute.NewDenyKeysFilter(semconv.K8SClusterNameKey)
	res, err := newTestDetector(url, WithAttributeFilter(filter)).Detect(t.Context())
	require.NoError(t, err)

	_, ok := res.Set().Value(semconv.K8SClusterNameKey)
	assert.False(t, ok, "expected k8s.cluster.name to be absent")

	for _, kv := range []attribute.KeyValue{
		semconv.CloudProviderAzure,
		semconv.CloudPlatformAzureAKS,
	} {
		val, ok := res.Set().Value(kv.Key)
		assert.True(t, ok, "expected %s to be present", kv.Key)
		assert.Equal(t, kv.Value, val)
	}
}
