// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scaleway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

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

// newTestDetector returns a detector querying endpoints instead of the real
// link-local metadata service.
func newTestDetector(endpoints []string, opts ...Option) *ResourceDetector {
	d := NewResourceDetector(opts...)
	d.endpoints = endpoints
	return d
}

// serveMetadata starts a test server running handler and returns a detector
// pointed at it. The server is closed via t.Cleanup.
func serveMetadata(t *testing.T, handler http.HandlerFunc, opts ...Option) *ResourceDetector {
	t.Helper()
	return newTestDetector([]string{newServer(t, handler)}, opts...)
}

// newServer starts a test server running handler and returns its URL. The
// server is closed via t.Cleanup.
func newServer(t *testing.T, handler http.HandlerFunc) string {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv.URL
}

// unreachableURL returns the URL of a server that has already been closed.
func unreachableURL(t *testing.T) string {
	t.Helper()

	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()
	return url
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
	d := serveMetadata(t, metadataHandler(metadataSample))

	res, err := d.Detect(t.Context())
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

// Nothing answers at the addresses of the metadata service: the process is not
// running on a Scaleway Instance.
func TestDetect_ConnectionRefused(t *testing.T) {
	d := newTestDetector([]string{unreachableURL(t)})

	res, err := d.Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

// Something answered at the address of the metadata service but did not serve
// the document. Reporting that as an absent instance would hide it.
func TestDetect_ServerError(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusInternalServerError, http.StatusForbidden} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			d := serveMetadata(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
			})

			res, err := d.Detect(t.Context())
			require.Error(t, err)
			assert.Contains(t, err.Error(), strconv.Itoa(status))
			assert.Nil(t, res)
		})
	}
}

func TestDetect_MalformedJSON(t *testing.T) {
	d := serveMetadata(t, metadataHandler("not json"))

	res, err := d.Detect(t.Context())
	require.Error(t, err)
	assert.Nil(t, res)
}

func TestDetect_PartialFailure(t *testing.T) {
	// An instance is reported, but without any of the detected fields.
	d := serveMetadata(t, metadataHandler(`{}`))

	res, err := d.Detect(t.Context())
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
	// The handler never answers, so only the canceled context can end the wait.
	d := serveMetadata(t, func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	})

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	res, err := d.Detect(ctx)
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, res)
}

// The request itself is canceled, not only the wait for it: it must not be left
// running after Detect returns.
func TestDetect_CancelsInFlightRequest(t *testing.T) {
	aborted := make(chan struct{})
	d := serveMetadata(t, func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
		close(aborted)
	})

	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		<-time.After(10 * time.Millisecond)
		cancel()
	}()

	res, err := d.Detect(ctx)
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, res)

	select {
	case <-aborted:
	case <-time.After(time.Second):
		t.Fatal("the metadata request outlived Detect")
	}
}

// The first address of the metadata service is not always the one that answers.
func TestDetect_SecondEndpoint(t *testing.T) {
	d := newTestDetector([]string{
		unreachableURL(t),
		newServer(t, metadataHandler(metadataSample)),
	})

	res, err := d.Detect(t.Context())
	require.NoError(t, err)

	val, ok := res.Set().Value(semconv.HostIDKey)
	require.True(t, ok)
	assert.Equal(t, attribute.StringValue("daa2ea5a-0ee6-4cdc-9f1a-e0d1cb4e6d86"), val)
}

// One deadline bounds the whole detection: it must not be restarted for every
// address of the metadata service.
func TestDetect_OneDeadlineAcrossEndpoints(t *testing.T) {
	// Block until the client gives up. Blocking on anything the test closes
	// would deadlock: the server is closed before any cleanup registered here.
	blocking := func(_ http.ResponseWriter, r *http.Request) { <-r.Context().Done() }
	d := newTestDetector([]string{
		newServer(t, blocking),
		newServer(t, blocking),
	})

	const timeout = 100 * time.Millisecond
	ctx, cancel := context.WithTimeout(t.Context(), timeout)
	defer cancel()

	start := time.Now()
	_, err := d.Detect(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)

	// Restarting the deadline for the second address would take at least twice
	// as long.
	assert.Less(t, time.Since(start), 2*timeout)
}

// The metadata service is on a link-local address that the process must reach
// directly. A proxy configured for outbound traffic must not be used for it.
//
// The configuration is asserted rather than the behavior: a test server listens
// on a loopback address, which [http.ProxyFromEnvironment] never proxies, so a
// proxied request cannot be provoked here.
func TestClientBypassesProxy(t *testing.T) {
	transport, ok := NewResourceDetector().client.Transport.(*http.Transport)
	require.True(t, ok)
	assert.Nil(t, transport.Proxy)
}

func TestDetect_WithAttributeFilter(t *testing.T) {
	filter := attribute.NewDenyKeysFilter(semconv.CloudPlatformKey, semconv.HostImageNameKey)
	d := serveMetadata(t, metadataHandler(metadataSample), WithAttributeFilter(filter))

	res, err := d.Detect(t.Context())
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

// The detector owns the client it queries the metadata service with. It must
// not disturb the default HTTP client of the process.
func TestDetect_LeavesDefaultHTTPClientAlone(t *testing.T) {
	orig := http.DefaultClient.Timeout
	t.Cleanup(func() { http.DefaultClient.Timeout = orig })
	http.DefaultClient.Timeout = 42 * time.Second

	d := serveMetadata(t, metadataHandler(metadataSample))

	_, err := d.Detect(t.Context())
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
