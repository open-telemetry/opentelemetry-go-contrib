// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cvm

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

// fullMetadata is a complete set of metadata documents, keyed by the path the
// metadata service serves them under.
func fullMetadata() map[string]string {
	return map[string]string{
		"app-id":                 "1250000000",
		"placement/region":       "ap-guangzhou",
		"placement/zone":         "ap-guangzhou-3",
		"instance-id":            "ins-abcd1234",
		"instance-name":          "cvm-test",
		"instance/image-id":      "img-9qabwvbn",
		"instance/instance-type": "S5.MEDIUM4",
	}
}

// newFakeServer starts an httptest server serving docs as plain text, keyed by
// metadata path. Paths absent from docs are answered with 404. The server is
// closed via t.Cleanup.
func newFakeServer(t *testing.T, docs map[string]string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		val, ok := docs[strings.TrimPrefix(r.URL.Path, "/")]
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(val))
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

// newTestDetector returns a detector pointed at url instead of the real
// metadata endpoint.
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
		semconv.CloudProviderTencentCloud,
		semconv.CloudPlatformTencentCloudCVM,
		semconv.CloudAccountID("1250000000"),
		semconv.CloudRegion("ap-guangzhou"),
		semconv.CloudAvailabilityZone("ap-guangzhou-3"),
		semconv.HostID("ins-abcd1234"),
		semconv.HostName("cvm-test"),
		semconv.HostImageID("img-9qabwvbn"),
		semconv.HostType("S5.MEDIUM4"),
	)
	assert.Equal(t, expected, res)
}

func TestDetect_TrimsWhitespace(t *testing.T) {
	// The metadata service may terminate a document with a newline.
	docs := fullMetadata()
	docs["instance-name"] = "cvm-test\n"
	docs["placement/region"] = "  ap-shanghai  "

	res, err := newTestDetector(newFakeServer(t, docs)).Detect(t.Context())
	require.NoError(t, err)

	val, ok := res.Set().Value(semconv.HostNameKey)
	require.True(t, ok)
	assert.Equal(t, attribute.StringValue("cvm-test"), val)

	val, ok = res.Set().Value(semconv.CloudRegionKey)
	require.True(t, ok)
	assert.Equal(t, attribute.StringValue("ap-shanghai"), val)
}

func TestDetect_NotOnTencentCloud(t *testing.T) {
	// A client error means something other than the Tencent Cloud metadata
	// service answered the request: not on Tencent Cloud, so no error is
	// reported.
	res, err := newTestDetector(newFakeServer(t, nil)).Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetect_ConnectionRefused(t *testing.T) {
	// Closed server -> connection refused -> not on Tencent Cloud -> empty
	// resource, no error.
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close()

	res, err := newTestDetector(url).Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetect_ServerError(t *testing.T) {
	// The metadata service answered but failed: surface the error instead of
	// silently reporting "not on Tencent Cloud".
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	res, err := newTestDetector(srv.URL).Detect(t.Context())
	require.Error(t, err)
	assert.Nil(t, res)
}

func TestDetect_SingleDocumentUnavailable(t *testing.T) {
	// Detection is all-or-nothing: a single missing document drops every
	// attribute rather than reporting a partial resource.
	docs := fullMetadata()
	delete(docs, "app-id")

	res, err := newTestDetector(newFakeServer(t, docs)).Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetect_OversizedResponse(t *testing.T) {
	docs := fullMetadata()
	docs["instance-name"] = strings.Repeat("a", maxResponseSize+1)

	res, err := newTestDetector(newFakeServer(t, docs)).Detect(t.Context())
	require.Error(t, err)
	assert.Nil(t, res)
}

func TestDetect_EmptyValues(t *testing.T) {
	// Every request is answered, but the documents are empty: the instance is
	// on Tencent Cloud, so report a partial resource.
	docs := make(map[string]string, len(metadataKeys))
	for _, key := range metadataKeys {
		docs[key.path] = ""
	}

	res, err := newTestDetector(newFakeServer(t, docs)).Detect(t.Context())
	require.Error(t, err)
	assert.ErrorIs(t, err, resource.ErrPartialResource)

	// cloud.provider and cloud.platform must still be present.
	for _, kv := range []attribute.KeyValue{
		semconv.CloudProviderTencentCloud,
		semconv.CloudPlatformTencentCloudCVM,
	} {
		val, ok := res.Set().Value(kv.Key)
		assert.True(t, ok, "expected %s to be present", kv.Key)
		assert.Equal(t, kv.Value, val)
	}

	// Every detected attribute must be absent.
	for _, k := range []attribute.Key{
		semconv.CloudAccountIDKey,
		semconv.CloudRegionKey,
		semconv.CloudAvailabilityZoneKey,
		semconv.HostIDKey,
		semconv.HostNameKey,
		semconv.HostImageIDKey,
		semconv.HostTypeKey,
	} {
		_, ok := res.Set().Value(k)
		assert.False(t, ok, "expected %s to be absent", k)
	}
}

func TestDetect_WithAttributeFilter(t *testing.T) {
	url := newFakeServer(t, fullMetadata())

	filter := attribute.NewDenyKeysFilter(semconv.CloudPlatformKey, semconv.HostImageIDKey)
	res, err := newTestDetector(url, WithAttributeFilter(filter)).Detect(t.Context())
	require.NoError(t, err)

	for _, k := range []attribute.Key{semconv.CloudPlatformKey, semconv.HostImageIDKey} {
		_, ok := res.Set().Value(k)
		assert.False(t, ok, "expected %s to be absent", k)
	}

	for _, kv := range []attribute.KeyValue{
		semconv.CloudProviderTencentCloud,
		semconv.CloudAccountID("1250000000"),
		semconv.CloudRegion("ap-guangzhou"),
		semconv.CloudAvailabilityZone("ap-guangzhou-3"),
		semconv.HostID("ins-abcd1234"),
		semconv.HostName("cvm-test"),
		semconv.HostType("S5.MEDIUM4"),
	} {
		val, ok := res.Set().Value(kv.Key)
		assert.True(t, ok, "expected %s to be present", kv.Key)
		assert.Equal(t, kv.Value, val)
	}
}

func TestDetect_RequestsEachDocumentOnce(t *testing.T) {
	docs := fullMetadata()

	var mu sync.Mutex
	requests := make(map[string]int, len(docs))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		mu.Lock()
		requests[path]++
		mu.Unlock()
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(docs[path]))
	}))
	t.Cleanup(srv.Close)

	_, err := newTestDetector(srv.URL).Detect(t.Context())
	require.NoError(t, err)

	expected := make(map[string]int, len(metadataKeys))
	for _, key := range metadataKeys {
		expected[key.path] = 1
	}
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, expected, requests)
}
