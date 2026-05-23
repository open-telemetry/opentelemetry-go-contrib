// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package hetzner

import (
	"net/http"
	"net/http/httptest"
	"testing"

	hcloudmeta "github.com/hetznercloud/hcloud-go/v2/hcloud/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
)

// withFakeMetaServer starts a fake Hetzner metadata HTTP server using the
// provided mux, overrides newHcloudClient so the detector points at it, and
// registers t.Cleanup to restore the original factory and close the server.
func withFakeMetaServer(t *testing.T, mux http.Handler) {
	t.Helper()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	orig := newHcloudClient
	newHcloudClient = func() *hcloudmeta.Client {
		return hcloudmeta.NewClient(hcloudmeta.WithEndpoint(srv.URL))
	}
	t.Cleanup(func() { newHcloudClient = orig })
}

func TestNewResourceDetector(t *testing.T) {
	d := NewResourceDetector()
	assert.NotNil(t, d)
}

func TestDetect_OK(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/hostname", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("srv-123"))
	})
	mux.HandleFunc("/instance-id", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("987654321"))
	})
	mux.HandleFunc("/region", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("nbg1"))
	})
	mux.HandleFunc("/availability-zone", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("nbg1-dc3"))
	})
	withFakeMetaServer(t, mux)

	res, err := NewResourceDetector().Detect(t.Context())
	require.NoError(t, err)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderHetzner,
		semconv.CloudPlatformHetznerCloudServer,
		semconv.HostID("987654321"),
		semconv.HostName("srv-123"),
		semconv.CloudRegion("nbg1"),
		semconv.CloudAvailabilityZone("nbg1-dc3"),
	)
	assert.Equal(t, expected, res)
}

func TestDetect_NotOnHetzner(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/hostname", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})
	withFakeMetaServer(t, mux)

	res, err := NewResourceDetector().Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetect_PartialFailure(t *testing.T) {
	// /hostname must succeed so IsHcloudServer() returns true, and the
	// subsequent Hostname() call (also to /hostname) also succeeds — so
	// host.name will be present. The three other endpoints fail.
	mux := http.NewServeMux()
	mux.HandleFunc("/hostname", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("srv-456"))
	})
	mux.HandleFunc("/instance-id", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusInternalServerError)
	})
	mux.HandleFunc("/region", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusInternalServerError)
	})
	mux.HandleFunc("/availability-zone", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusInternalServerError)
	})
	withFakeMetaServer(t, mux)

	res, err := NewResourceDetector().Detect(t.Context())
	require.Error(t, err)
	assert.ErrorIs(t, err, resource.ErrPartialResource)

	// cloud.provider, cloud.platform, and host.name must be present.
	presentAttrs := []attribute.KeyValue{
		semconv.CloudProviderHetzner,
		semconv.CloudPlatformHetznerCloudServer,
		semconv.HostName("srv-456"),
	}
	for _, kv := range presentAttrs {
		val, ok := res.Set().Value(kv.Key)
		assert.True(t, ok, "expected attribute %s to be present", kv.Key)
		assert.Equal(t, kv.Value, val)
	}

	// host.id, cloud.region, and cloud.availability_zone must be absent.
	absentKeys := []attribute.Key{
		semconv.HostIDKey,
		semconv.CloudRegionKey,
		semconv.CloudAvailabilityZoneKey,
	}
	for _, k := range absentKeys {
		_, ok := res.Set().Value(k)
		assert.False(t, ok, "expected attribute %s to be absent", k)
	}
}

func TestDetect_WithAttributeFilter(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/hostname", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("srv-789"))
	})
	mux.HandleFunc("/instance-id", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("111222333"))
	})
	mux.HandleFunc("/region", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("fsn1"))
	})
	mux.HandleFunc("/availability-zone", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("fsn1-dc14"))
	})
	withFakeMetaServer(t, mux)

	// Filter out cloud.platform.
	filter := attribute.NewDenyKeysFilter(semconv.CloudPlatformKey)
	res, err := NewResourceDetector(WithAttributeFilter(filter)).Detect(t.Context())
	require.NoError(t, err)

	// cloud.platform must be absent.
	_, ok := res.Set().Value(semconv.CloudPlatformKey)
	assert.False(t, ok, "expected cloud.platform to be absent")

	// The other five attributes must be present.
	presentAttrs := []attribute.KeyValue{
		semconv.CloudProviderHetzner,
		semconv.HostID("111222333"),
		semconv.HostName("srv-789"),
		semconv.CloudRegion("fsn1"),
		semconv.CloudAvailabilityZone("fsn1-dc14"),
	}
	for _, kv := range presentAttrs {
		val, ok := res.Set().Value(kv.Key)
		assert.True(t, ok, "expected attribute %s to be present", kv.Key)
		assert.Equal(t, kv.Value, val)
	}
}
