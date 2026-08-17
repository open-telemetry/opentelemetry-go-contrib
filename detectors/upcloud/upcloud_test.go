// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upcloud

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
// URL. The server is closed via t.Cleanup.
func newFakeServer(t *testing.T, meta metadataResponse) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

func TestDetect_OK(t *testing.T) {
	url := newFakeServer(t, metadataResponse{
		CloudName:  "upcloud",
		Hostname:   "metadata.example.com",
		InstanceID: "00bf9504-a4cb-4839-88ff-124a2c95e169",
		Region:     "de-fra1",
	})

	res, err := newTestDetector(url).Detect(t.Context())
	require.NoError(t, err)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderKey.String("upcloud"),
		semconv.CloudRegion("de-fra1"),
		semconv.HostID("00bf9504-a4cb-4839-88ff-124a2c95e169"),
		semconv.HostName("metadata.example.com"),
	)
	assert.Equal(t, expected, res)
}

func TestDetect_NotOnUpCloud(t *testing.T) {
	// A client error means something other than the UpCloud metadata service
	// answered the request: not on UpCloud, so no error is reported.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	res, err := newTestDetector(srv.URL).Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetect_ConnectionRefused(t *testing.T) {
	// Closed server → connection refused → not on UpCloud → empty resource, no error.
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close()

	res, err := newTestDetector(url).Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetect_ServerError(t *testing.T) {
	// The metadata service answered but failed: surface the error instead of
	// silently reporting "not on UpCloud".
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
	// so this is a failure rather than "not on UpCloud".
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
	// Serve JSON with cloud_name, region, instance_id, and hostname all absent.
	url := newFakeServer(t, metadataResponse{})

	res, err := newTestDetector(url).Detect(t.Context())
	require.Error(t, err)
	assert.ErrorIs(t, err, resource.ErrPartialResource)

	for _, k := range []attribute.Key{
		semconv.CloudProviderKey,
		semconv.CloudRegionKey,
		semconv.HostIDKey,
		semconv.HostNameKey,
	} {
		_, ok := res.Set().Value(k)
		assert.False(t, ok, "expected %s to be absent", k)
	}
}

func TestDetect_PartialFailureMissingHostname(t *testing.T) {
	// Only the hostname is absent: the remaining attributes are still reported.
	url := newFakeServer(t, metadataResponse{
		CloudName:  "upcloud",
		InstanceID: "00bf9504-a4cb-4839-88ff-124a2c95e169",
		Region:     "de-fra1",
	})

	res, err := newTestDetector(url).Detect(t.Context())
	require.Error(t, err)
	assert.ErrorIs(t, err, resource.ErrPartialResource)

	for _, kv := range []attribute.KeyValue{
		semconv.CloudProviderKey.String("upcloud"),
		semconv.CloudRegion("de-fra1"),
		semconv.HostID("00bf9504-a4cb-4839-88ff-124a2c95e169"),
	} {
		val, ok := res.Set().Value(kv.Key)
		assert.True(t, ok, "expected %s to be present", kv.Key)
		assert.Equal(t, kv.Value, val)
	}

	_, ok := res.Set().Value(semconv.HostNameKey)
	assert.False(t, ok, "expected host.name to be absent")
}

func TestDetect_WithAttributeFilter(t *testing.T) {
	url := newFakeServer(t, metadataResponse{
		CloudName:  "upcloud",
		Hostname:   "srv-filter",
		InstanceID: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		Region:     "fi-hel2",
	})

	filter := attribute.NewDenyKeysFilter(semconv.HostNameKey)
	res, err := newTestDetector(url, WithAttributeFilter(filter)).Detect(t.Context())
	require.NoError(t, err)

	_, ok := res.Set().Value(semconv.HostNameKey)
	assert.False(t, ok, "expected host.name to be absent")

	for _, kv := range []attribute.KeyValue{
		semconv.CloudProviderKey.String("upcloud"),
		semconv.CloudRegion("fi-hel2"),
		semconv.HostID("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"),
	} {
		val, ok := res.Set().Value(kv.Key)
		assert.True(t, ok, "expected %s to be present", kv.Key)
		assert.Equal(t, kv.Value, val)
	}
}

func TestDetect_SingleRequest(t *testing.T) {
	// The whole metadata document is served by a single endpoint: one Detect
	// call must not fetch it more than once.
	var requests atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(metadataResponse{
			CloudName:  "upcloud",
			Hostname:   "srv-count",
			InstanceID: "00bf9504-a4cb-4839-88ff-124a2c95e169",
			Region:     "de-fra1",
		})
	}))
	t.Cleanup(srv.Close)

	_, err := newTestDetector(srv.URL).Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, int64(1), requests.Load())
}
