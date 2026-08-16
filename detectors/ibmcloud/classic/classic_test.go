// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package classic

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

// fullMetadata is a complete, well-formed response set from the SoftLayer
// Resource Metadata service.
func fullMetadata() map[string]string {
	return map[string]string{
		idPath:               "123456",
		hostnamePath:         "test-classic-instance",
		datacenterPath:       "dal10",
		accountIDPath:        "3186058",
		globalIdentifierPath: "06220b70-9072-4f83-ba16-d62f03106c1c",
	}
}

// newFakeServer starts an httptest server that serves bodies for the given
// metadata paths as plain text and 404s anything else. The server is closed via
// t.Cleanup.
func newFakeServer(t *testing.T, bodies map[string]string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := bodies[strings.TrimPrefix(r.URL.Path, "/")]
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

// newTestDetector returns a detector pointed at url instead of the real
// SoftLayer metadata endpoint.
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
	url := newFakeServer(t, fullMetadata())

	res, err := newTestDetector(url).Detect(t.Context())
	require.NoError(t, err)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderIBMCloud,
		semconv.CloudPlatformKey.String("ibm_cloud.classic"),
		semconv.HostID("123456"),
		semconv.HostName("test-classic-instance"),
		semconv.CloudAvailabilityZone("dal10"),
		semconv.CloudAccountID("3186058"),
		semconv.CloudResourceID("06220b70-9072-4f83-ba16-d62f03106c1c"),
	)
	assert.Equal(t, expected, res)
}

func TestDetect_TrimsWhitespace(t *testing.T) {
	// The .txt endpoints return values with a trailing newline, and some
	// deployments pad them with spaces.
	url := newFakeServer(t, map[string]string{
		idPath:               "100\n",
		hostnamePath:         "  test-host  \n",
		datacenterPath:       "dal13\r\n",
		accountIDPath:        "\t3186058\t",
		globalIdentifierPath: " 06220b70-9072-4f83-ba16-d62f03106c1c \n",
	})

	res, err := newTestDetector(url).Detect(t.Context())
	require.NoError(t, err)

	for _, kv := range []attribute.KeyValue{
		semconv.HostID("100"),
		semconv.HostName("test-host"),
		semconv.CloudAvailabilityZone("dal13"),
		semconv.CloudAccountID("3186058"),
		semconv.CloudResourceID("06220b70-9072-4f83-ba16-d62f03106c1c"),
	} {
		val, ok := res.Set().Value(kv.Key)
		require.True(t, ok, "expected %s to be present", kv.Key)
		assert.Equal(t, kv.Value, val)
	}
}

func TestDetect_NotOnClassic(t *testing.T) {
	// A client error means the request was answered by something that is not
	// the SoftLayer metadata service: not on IBM Cloud Classic, so no error is
	// reported.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	res, err := newTestDetector(srv.URL).Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetect_ConnectionRefused(t *testing.T) {
	// Closed server -> connection refused -> not on IBM Cloud Classic ->
	// empty resource, no error.
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close()

	res, err := newTestDetector(url).Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetect_ServerError(t *testing.T) {
	// The metadata service answered the probe but failed: surface the error
	// instead of silently reporting "not on IBM Cloud Classic".
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	res, err := newTestDetector(srv.URL).Detect(t.Context())
	require.Error(t, err)
	assert.Nil(t, res)
}

func TestDetect_PartialFailure(t *testing.T) {
	// The probe succeeds, so this process is on IBM Cloud Classic, but two
	// fields are unusable: one 500s and one is served empty. The remaining
	// fields must still be reported.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch strings.TrimPrefix(r.URL.Path, "/") {
		case idPath:
			_, _ = w.Write([]byte("123456"))
		case hostnamePath:
			http.Error(w, "boom", http.StatusInternalServerError)
		case datacenterPath:
			_, _ = w.Write([]byte("\n"))
		case accountIDPath:
			_, _ = w.Write([]byte("3186058"))
		case globalIdentifierPath:
			_, _ = w.Write([]byte("06220b70-9072-4f83-ba16-d62f03106c1c"))
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	res, err := newTestDetector(srv.URL).Detect(t.Context())
	require.Error(t, err)
	assert.ErrorIs(t, err, resource.ErrPartialResource)

	for _, kv := range []attribute.KeyValue{
		semconv.CloudProviderIBMCloud,
		semconv.CloudPlatformKey.String("ibm_cloud.classic"),
		semconv.HostID("123456"),
		semconv.CloudAccountID("3186058"),
		semconv.CloudResourceID("06220b70-9072-4f83-ba16-d62f03106c1c"),
	} {
		val, ok := res.Set().Value(kv.Key)
		assert.True(t, ok, "expected %s to be present", kv.Key)
		assert.Equal(t, kv.Value, val)
	}

	for _, k := range []attribute.Key{
		semconv.HostNameKey,
		semconv.CloudAvailabilityZoneKey,
	} {
		_, ok := res.Set().Value(k)
		assert.False(t, ok, "expected %s to be absent", k)
	}
}

func TestDetect_EmptyValuesAreOmitted(t *testing.T) {
	// Every endpoint answers 200 with an empty body. The probe succeeded, so
	// the resource is partial; no attribute may be emitted with an empty value.
	url := newFakeServer(t, map[string]string{
		idPath:               "",
		hostnamePath:         "",
		datacenterPath:       "",
		accountIDPath:        "",
		globalIdentifierPath: "",
	})

	res, err := newTestDetector(url).Detect(t.Context())
	require.Error(t, err)
	assert.ErrorIs(t, err, resource.ErrPartialResource)

	for _, kv := range []attribute.KeyValue{
		semconv.CloudProviderIBMCloud,
		semconv.CloudPlatformKey.String("ibm_cloud.classic"),
	} {
		val, ok := res.Set().Value(kv.Key)
		assert.True(t, ok, "expected %s to be present", kv.Key)
		assert.Equal(t, kv.Value, val)
	}

	for _, k := range []attribute.Key{
		semconv.HostIDKey,
		semconv.HostNameKey,
		semconv.CloudAvailabilityZoneKey,
		semconv.CloudAccountIDKey,
		semconv.CloudResourceIDKey,
	} {
		_, ok := res.Set().Value(k)
		assert.False(t, ok, "expected %s to be absent", k)
	}
}

func TestDetect_WithAttributeFilter(t *testing.T) {
	url := newFakeServer(t, fullMetadata())

	filter := attribute.NewDenyKeysFilter(semconv.CloudPlatformKey, semconv.CloudAccountIDKey)
	res, err := newTestDetector(url, WithAttributeFilter(filter)).Detect(t.Context())
	require.NoError(t, err)

	for _, k := range []attribute.Key{
		semconv.CloudPlatformKey,
		semconv.CloudAccountIDKey,
	} {
		_, ok := res.Set().Value(k)
		assert.False(t, ok, "expected %s to be absent", k)
	}

	for _, kv := range []attribute.KeyValue{
		semconv.CloudProviderIBMCloud,
		semconv.HostID("123456"),
		semconv.HostName("test-classic-instance"),
		semconv.CloudAvailabilityZone("dal10"),
		semconv.CloudResourceID("06220b70-9072-4f83-ba16-d62f03106c1c"),
	} {
		val, ok := res.Set().Value(kv.Key)
		assert.True(t, ok, "expected %s to be present", kv.Key)
		assert.Equal(t, kv.Value, val)
	}
}

func TestDetect_FetchesEachEndpointOnce(t *testing.T) {
	// The SoftLayer metadata service has no aggregate document, so one request
	// per field is expected -- but no field may be fetched twice.
	var (
		mu     sync.Mutex
		counts = map[string]int{}
	)
	bodies := fullMetadata()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		mu.Lock()
		counts[path]++
		mu.Unlock()

		body, ok := bodies[path]
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	_, err := newTestDetector(srv.URL).Detect(t.Context())
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	assert.Len(t, counts, len(bodies), "unexpected set of requested paths: %v", counts)
	for path := range bodies {
		assert.Equal(t, 1, counts[path], "expected exactly one request for %s", path)
	}
}
