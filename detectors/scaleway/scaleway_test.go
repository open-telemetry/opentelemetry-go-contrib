// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scaleway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	instance "github.com/scaleway/scaleway-sdk-go/api/instance/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

// metadataSample is a response of the metadata service of a Scaleway Instance.
const metadataSample = `{
  "arch": "x86_64",
  "commercial_type": "STARDUST1-S",
  "hostname": "scw-interesting-bassi",
  "id": "daa2ea5a-0ee6-4cdc-9f1a-e0d1cb4e6d86",
  "image": {
    "arch": "x86_64",
    "id": "01fe25a9-2e95-41e2-9f3f-f7d3a00604be",
    "name": "Ubuntu 24.04 Noble Numbat",
    "organization": "51b656e3-4865-41e8-adbc-0c45bdd780db",
    "project": "51b656e3-4865-41e8-adbc-0c45bdd780db",
    "public": true,
    "state": "available",
    "zone": "nl-ams-1"
  },
  "location": {
    "cluster_id": "15",
    "hypervisor_id": "402",
    "node_id": "134",
    "platform_id": "23",
    "zone_id": "nl-ams-1"
  },
  "mac_address": "de:00:00:58:4f:8f",
  "name": "scw-interesting-bassi",
  "organization": "10542306-0c75-4265-9c2c-1fbfe4ea0bf0",
  "project": "10542306-0c75-4265-9c2c-1fbfe4ea0bf0",
  "private_ip": null,
  "state": "running",
  "state_detail": "booted",
  "tags": [],
  "zone": "nl-ams-1"
}`

// serveMetadata points the detector at a test server running handler. The
// server is closed and the client factory restored via t.Cleanup.
func serveMetadata(t *testing.T, handler http.HandlerFunc) {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	useMetadataURL(t, srv.URL)
}

// useMetadataURL makes the detector query url instead of the real link-local
// metadata service.
func useMetadataURL(t *testing.T, url string) {
	t.Helper()

	orig := newMetadataAPI
	t.Cleanup(func() { newMetadataAPI = orig })

	newMetadataAPI = func() *instance.MetadataAPI {
		api := instance.NewMetadataAPI()
		api.MetadataURL = &url
		return api
	}
}

// metadataHandler serves body from the metadata endpoint of the Scaleway API.
func metadataHandler(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/conf") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}
}

func TestDetect_OK(t *testing.T) {
	serveMetadata(t, metadataHandler(metadataSample))

	res, err := NewResourceDetector().Detect(t.Context())
	require.NoError(t, err)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderKey.String("scaleway_cloud"),
		semconv.CloudPlatformKey.String("scaleway_cloud_compute"),
		semconv.CloudAccountID("10542306-0c75-4265-9c2c-1fbfe4ea0bf0"),
		semconv.CloudAvailabilityZone("nl-ams-1"),
		semconv.CloudRegion("nl-ams"),
		semconv.HostID("daa2ea5a-0ee6-4cdc-9f1a-e0d1cb4e6d86"),
		semconv.HostImageID("01fe25a9-2e95-41e2-9f3f-f7d3a00604be"),
		semconv.HostImageName("Ubuntu 24.04 Noble Numbat"),
		semconv.HostName("scw-interesting-bassi"),
		semconv.HostType("STARDUST1-S"),
	)
	assert.Equal(t, expected, res)
}

func TestDetect_NotOnScaleway(t *testing.T) {
	// Nothing but 404s: the link-local address was answered by something that
	// is not the metadata service.
	serveMetadata(t, http.NotFound)

	res, err := NewResourceDetector().Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetect_ConnectionRefused(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()

	useMetadataURL(t, url)

	res, err := NewResourceDetector().Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetect_MalformedJSON(t *testing.T) {
	// The client reports a failure without the status that caused it, so a
	// body it cannot decode is indistinguishable from an absent metadata
	// service and is reported the same way.
	serveMetadata(t, metadataHandler("not json"))

	res, err := NewResourceDetector().Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetect_PartialFailure(t *testing.T) {
	// An instance is reported, but without any of the detected fields.
	serveMetadata(t, metadataHandler(`{}`))

	res, err := NewResourceDetector().Detect(t.Context())
	require.Error(t, err)
	assert.ErrorIs(t, err, resource.ErrPartialResource)

	// cloud.provider and cloud.platform must still be present.
	for _, kv := range []attribute.KeyValue{
		semconv.CloudProviderKey.String("scaleway_cloud"),
		semconv.CloudPlatformKey.String("scaleway_cloud_compute"),
	} {
		val, ok := res.Set().Value(kv.Key)
		assert.True(t, ok, "expected %s to be present", kv.Key)
		assert.Equal(t, kv.Value, val)
	}

	// Everything read from the metadata must be absent.
	for _, k := range []attribute.Key{
		semconv.CloudAccountIDKey,
		semconv.CloudAvailabilityZoneKey,
		semconv.CloudRegionKey,
		semconv.HostIDKey,
		semconv.HostImageIDKey,
		semconv.HostImageNameKey,
		semconv.HostNameKey,
		semconv.HostTypeKey,
	} {
		_, ok := res.Set().Value(k)
		assert.False(t, ok, "expected %s to be absent", k)
	}
}

func TestDetect_ContextCanceled(t *testing.T) {
	// The handler blocks until the test is done, so only the canceled context
	// can end the wait.
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })

	serveMetadata(t, func(http.ResponseWriter, *http.Request) {
		<-release
	})

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	res, err := NewResourceDetector().Detect(ctx)
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, res)
}

func TestDetect_WithAttributeFilter(t *testing.T) {
	serveMetadata(t, metadataHandler(metadataSample))

	filter := attribute.NewDenyKeysFilter(semconv.CloudPlatformKey, semconv.HostImageNameKey)
	res, err := NewResourceDetector(WithAttributeFilter(filter)).Detect(t.Context())
	require.NoError(t, err)

	for _, k := range []attribute.Key{semconv.CloudPlatformKey, semconv.HostImageNameKey} {
		_, ok := res.Set().Value(k)
		assert.False(t, ok, "expected %s to be absent", k)
	}

	for _, kv := range []attribute.KeyValue{
		semconv.CloudProviderKey.String("scaleway_cloud"),
		semconv.CloudAccountID("10542306-0c75-4265-9c2c-1fbfe4ea0bf0"),
		semconv.CloudRegion("nl-ams"),
		semconv.HostID("daa2ea5a-0ee6-4cdc-9f1a-e0d1cb4e6d86"),
		semconv.HostType("STARDUST1-S"),
	} {
		val, ok := res.Set().Value(kv.Key)
		assert.True(t, ok, "expected %s to be present", kv.Key)
		assert.Equal(t, kv.Value, val)
	}
}

// useUnreachableMetadataURLs points the detector at an address nothing answers,
// keeping the real client factory.
func useUnreachableMetadataURLs(t *testing.T) {
	t.Helper()

	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()

	orig := metadataURLs
	t.Cleanup(func() { metadataURLs = orig })
	metadataURLs = []string{url}
}

func TestDetect_UnreachableMetadataService(t *testing.T) {
	// The client discovers the address of the metadata service by probing it
	// with the default HTTP client of the process, and panics when nothing
	// answers. Detect must set the address itself to never reach that code.
	useUnreachableMetadataURLs(t)

	require.NotPanics(t, func() {
		res, err := NewResourceDetector().Detect(t.Context())
		require.NoError(t, err)
		assert.Equal(t, resource.Empty(), res)
	})
}

func TestDetect_LeavesDefaultHTTPClientAlone(t *testing.T) {
	// Discovering the address of the metadata service sets a timeout on the
	// default HTTP client of the process. Detect must not disturb it.
	useUnreachableMetadataURLs(t)

	orig := http.DefaultClient.Timeout
	t.Cleanup(func() { http.DefaultClient.Timeout = orig })
	http.DefaultClient.Timeout = 42 * time.Second

	_, err := NewResourceDetector().Detect(t.Context())
	require.NoError(t, err)

	assert.Equal(t, 42*time.Second, http.DefaultClient.Timeout)
}

func TestZoneToRegion(t *testing.T) {
	tests := []struct {
		zone string
		want string
	}{
		{"nl-ams-1", "nl-ams"},
		{"fr-par-2", "fr-par"},
		{"pl-waw-3", "pl-waw"},
		{"fr-par", "fr"},
		{"unknown", ""},
		{"", ""},
		{"-1", ""},
	}
	for _, tt := range tests {
		t.Run(tt.zone, func(t *testing.T) {
			assert.Equal(t, tt.want, zoneToRegion(tt.zone))
		})
	}
}
