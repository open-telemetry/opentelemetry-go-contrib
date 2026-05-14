// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package vultr_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"

	"go.opentelemetry.io/contrib/detectors/vultr"
)

// vultrMetadata mirrors the JSON shape returned by the Vultr metadata service.
type vultrMetadata struct {
	Hostname     string `json:"hostname"`
	InstanceID   string `json:"instanceid"`
	InstanceV2ID string `json:"instance-v2-id"`
	Region       struct {
		RegionCode string `json:"regioncode"`
	} `json:"region"`
}

// newFakeServer starts an httptest server serving meta as JSON and returns its
// URL. The server is closed via t.Cleanup.
func newFakeServer(t *testing.T, meta vultrMetadata) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(meta)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestNewResourceDetector(t *testing.T) {
	d := vultr.NewResourceDetector()
	assert.NotNil(t, d)
}

func TestDetect_OK(t *testing.T) {
	url := newFakeServer(t, vultrMetadata{
		Hostname:     "srv-abc",
		InstanceID:   "legacy-id",
		InstanceV2ID: "550e8400-e29b-41d4-a716-446655440000",
		Region: struct {
			RegionCode string `json:"regioncode"`
		}{RegionCode: "ewr"},
	})

	res, err := vultr.NewResourceDetector(vultr.WithEndpoint(url)).Detect(t.Context())
	require.NoError(t, err)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderVultr,
		semconv.CloudPlatformVultrCloudCompute,
		semconv.HostID("550e8400-e29b-41d4-a716-446655440000"),
		semconv.HostName("srv-abc"),
		semconv.CloudRegion("ewr"),
	)
	assert.Equal(t, expected, res)
}

func TestDetect_OK_LegacyInstanceID(t *testing.T) {
	// instance-v2-id absent — should fall back to instanceid.
	url := newFakeServer(t, vultrMetadata{
		Hostname:   "srv-legacy",
		InstanceID: "legacy-only-id",
		Region: struct {
			RegionCode string `json:"regioncode"`
		}{RegionCode: "sjc"},
	})

	res, err := vultr.NewResourceDetector(vultr.WithEndpoint(url)).Detect(t.Context())
	require.NoError(t, err)

	val, ok := res.Set().Value(semconv.HostIDKey)
	assert.True(t, ok)
	assert.Equal(t, attribute.StringValue("legacy-only-id"), val)
}

func TestDetect_NotOnVultr(t *testing.T) {
	// Non-200 response → not on Vultr → empty resource, no error.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	res, err := vultr.NewResourceDetector(vultr.WithEndpoint(srv.URL)).Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetect_MalformedJSON(t *testing.T) {
	// 200 OK with a body that isn't valid JSON. The detector treats this the
	// same as any other fetch failure: empty resource, no error.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not json"))
	}))
	t.Cleanup(srv.Close)

	res, err := vultr.NewResourceDetector(vultr.WithEndpoint(srv.URL)).Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetect_ConnectionRefused(t *testing.T) {
	// Closed server → connection refused → not on Vultr → empty resource, no error.
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close()

	res, err := vultr.NewResourceDetector(vultr.WithEndpoint(url)).Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetect_PartialFailure(t *testing.T) {
	// Serve JSON with hostname, instanceid, instance-v2-id, and regioncode all absent.
	url := newFakeServer(t, vultrMetadata{})

	res, err := vultr.NewResourceDetector(vultr.WithEndpoint(url)).Detect(t.Context())
	require.Error(t, err)
	assert.ErrorIs(t, err, resource.ErrPartialResource)

	// cloud.provider and cloud.platform must still be present.
	for _, kv := range []attribute.KeyValue{
		semconv.CloudProviderVultr,
		semconv.CloudPlatformVultrCloudCompute,
	} {
		val, ok := res.Set().Value(kv.Key)
		assert.True(t, ok, "expected %s to be present", kv.Key)
		assert.Equal(t, kv.Value, val)
	}

	// host.id, host.name, cloud.region must be absent.
	for _, k := range []attribute.Key{
		semconv.HostIDKey,
		semconv.HostNameKey,
		semconv.CloudRegionKey,
	} {
		_, ok := res.Set().Value(k)
		assert.False(t, ok, "expected %s to be absent", k)
	}
}

func TestDetect_WithAttributeFilter(t *testing.T) {
	url := newFakeServer(t, vultrMetadata{
		Hostname:     "srv-filter",
		InstanceV2ID: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		Region: struct {
			RegionCode string `json:"regioncode"`
		}{RegionCode: "lax"},
	})

	filter := attribute.NewDenyKeysFilter(semconv.CloudPlatformKey)
	res, err := vultr.NewResourceDetector(
		vultr.WithEndpoint(url),
		vultr.WithAttributeFilter(filter),
	).Detect(t.Context())
	require.NoError(t, err)

	_, ok := res.Set().Value(semconv.CloudPlatformKey)
	assert.False(t, ok, "expected cloud.platform to be absent")

	for _, kv := range []attribute.KeyValue{
		semconv.CloudProviderVultr,
		semconv.HostID("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"),
		semconv.HostName("srv-filter"),
		semconv.CloudRegion("lax"),
	} {
		val, ok := res.Set().Value(kv.Key)
		assert.True(t, ok, "expected %s to be present", kv.Key)
		assert.Equal(t, kv.Value, val)
	}
}
