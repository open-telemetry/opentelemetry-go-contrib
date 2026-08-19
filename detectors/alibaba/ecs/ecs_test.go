// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ecs

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

const testToken = "test-metadata-token"

// fullMetadata is the set of metadata paths a healthy ECS instance serves.
func fullMetadata() map[string]string {
	return map[string]string{
		"owner-account-id":       "1234567890123456",
		"region-id":              "cn-hangzhou",
		"zone-id":                "cn-hangzhou-a",
		"instance-id":            "i-abcdef123456",
		"hostname":               "ecs-instance-01",
		"image-id":               "m-abcdef123456",
		"instance/instance-type": "ecs.g6.large",
	}
}

// fakeService is an httptest stand-in for the ECS instance metadata service. It
// serves the token dance and the metadata paths in meta, and records what was
// requested.
type fakeService struct {
	mu       sync.Mutex
	requests map[string]int

	meta map[string]string
	// tokenStatus, when non-zero, is returned for the token request instead of
	// serving a token.
	tokenStatus int
	// tokenBody overrides the served token body.
	tokenBody *string
	// missing paths are answered with 404 even when present in meta.
	missing map[string]bool
	// hijacked paths have their connection closed without a response.
	hijacked map[string]bool
	// tokenSeen records the token each metadata request carried.
	tokenSeen []string
}

func (f *fakeService) count(key string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.requests[key]
}

func (f *fakeService) record(key string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.requests == nil {
		f.requests = make(map[string]int)
	}
	f.requests[key]++
}

func (f *fakeService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == tokenPath {
		f.record(tokenPath)
		if r.Method != http.MethodPut {
			http.Error(w, "expected PUT", http.StatusMethodNotAllowed)
			return
		}
		if ttl := r.Header.Get(tokenTTLHeader); ttl == "" {
			http.Error(w, "missing TTL header", http.StatusBadRequest)
			return
		}
		if f.tokenStatus != 0 {
			http.Error(w, "token unavailable", f.tokenStatus)
			return
		}
		body := testToken
		if f.tokenBody != nil {
			body = *f.tokenBody
		}
		_, _ = w.Write([]byte(body))
		return
	}

	path, ok := strings.CutPrefix(r.URL.Path, metadataBasePath)
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	f.record(path)

	f.mu.Lock()
	f.tokenSeen = append(f.tokenSeen, r.Header.Get(tokenHeader))
	f.mu.Unlock()

	if f.hijacked[path] {
		conn, _, err := w.(http.Hijacker).Hijack()
		if err == nil {
			_ = conn.Close()
		}
		return
	}
	if f.missing[path] {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	val, ok := f.meta[path]
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	_, _ = w.Write([]byte(val))
}

// newFakeService starts srv backed by f and returns its URL. The server is
// closed via t.Cleanup.
func newFakeService(t *testing.T, f *fakeService) string {
	t.Helper()
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	return srv.URL
}

// newTestDetector returns a detector pointed at url instead of the real
// instance metadata endpoint.
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
	url := newFakeService(t, &fakeService{meta: fullMetadata()})

	res, err := newTestDetector(url).Detect(t.Context())
	require.NoError(t, err)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderAlibabaCloud,
		semconv.CloudPlatformAlibabaCloudECS,
		semconv.CloudAccountID("1234567890123456"),
		semconv.CloudRegion("cn-hangzhou"),
		semconv.CloudAvailabilityZone("cn-hangzhou-a"),
		semconv.HostID("i-abcdef123456"),
		semconv.HostName("ecs-instance-01"),
		semconv.HostImageID("m-abcdef123456"),
		semconv.HostType("ecs.g6.large"),
	)
	assert.Equal(t, expected, res)
}

func TestDetect_ValuesAreTrimmed(t *testing.T) {
	// The metadata service is not guaranteed to omit a trailing newline.
	meta := fullMetadata()
	meta["hostname"] = "ecs-instance-01\n"
	url := newFakeService(t, &fakeService{meta: meta})

	res, err := newTestDetector(url).Detect(t.Context())
	require.NoError(t, err)

	val, ok := res.Set().Value(semconv.HostNameKey)
	require.True(t, ok)
	assert.Equal(t, attribute.StringValue("ecs-instance-01"), val)
}

func TestDetect_TokenIsSentOnEveryRequest(t *testing.T) {
	f := &fakeService{meta: fullMetadata()}
	url := newFakeService(t, f)

	_, err := newTestDetector(url).Detect(t.Context())
	require.NoError(t, err)

	require.Len(t, f.tokenSeen, len(metadataFields))
	for _, got := range f.tokenSeen {
		assert.Equal(t, testToken, got)
	}
}

func TestDetect_RequestCount(t *testing.T) {
	// One token request and exactly one request per metadata path: no value is
	// ever fetched twice.
	f := &fakeService{meta: fullMetadata()}
	url := newFakeService(t, f)

	_, err := newTestDetector(url).Detect(t.Context())
	require.NoError(t, err)

	assert.Equal(t, 1, f.count(tokenPath), "expected a single token request")
	for _, field := range metadataFields {
		assert.Equal(t, 1, f.count(field.path), "expected a single request for %s", field.path)
	}
}

func TestDetect_NotOnECS(t *testing.T) {
	// A client error means something other than the ECS metadata service
	// answered the request: not on ECS, so no error is reported.
	url := newFakeService(t, &fakeService{tokenStatus: http.StatusNotFound})

	res, err := newTestDetector(url).Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetect_ConnectionRefused(t *testing.T) {
	// Closed server → connection refused → not on ECS → empty resource, no error.
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close()

	res, err := newTestDetector(url).Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetect_TokenServerError(t *testing.T) {
	// The metadata service answered but failed: surface the error instead of
	// silently reporting "not on ECS".
	url := newFakeService(t, &fakeService{tokenStatus: http.StatusInternalServerError})

	res, err := newTestDetector(url).Detect(t.Context())
	require.Error(t, err)
	assert.Nil(t, res)
}

func TestDetect_EmptyToken(t *testing.T) {
	// The metadata service answered the token request with an empty body. It
	// responded, so this is a failure rather than "not on ECS".
	empty := ""
	url := newFakeService(t, &fakeService{meta: fullMetadata(), tokenBody: &empty})

	res, err := newTestDetector(url).Detect(t.Context())
	require.Error(t, err)
	assert.Nil(t, res)
}

func TestDetect_PartialFailure(t *testing.T) {
	// image-id is not served and hostname comes back empty. Both attributes are
	// dropped, the rest of the resource is still reported.
	meta := fullMetadata()
	meta["hostname"] = ""
	url := newFakeService(t, &fakeService{
		meta:    meta,
		missing: map[string]bool{"image-id": true},
	})

	res, err := newTestDetector(url).Detect(t.Context())
	require.Error(t, err)
	assert.ErrorIs(t, err, resource.ErrPartialResource)

	for _, kv := range []attribute.KeyValue{
		semconv.CloudProviderAlibabaCloud,
		semconv.CloudPlatformAlibabaCloudECS,
		semconv.CloudAccountID("1234567890123456"),
		semconv.CloudRegion("cn-hangzhou"),
		semconv.CloudAvailabilityZone("cn-hangzhou-a"),
		semconv.HostID("i-abcdef123456"),
		semconv.HostType("ecs.g6.large"),
	} {
		val, ok := res.Set().Value(kv.Key)
		assert.True(t, ok, "expected %s to be present", kv.Key)
		assert.Equal(t, kv.Value, val)
	}

	for _, k := range []attribute.Key{
		semconv.HostNameKey,
		semconv.HostImageIDKey,
	} {
		_, ok := res.Set().Value(k)
		assert.False(t, ok, "expected %s to be absent", k)
	}
}

func TestDetect_MetadataTransportError(t *testing.T) {
	// The connection is closed without a response for one metadata path. A
	// token was obtained, so the process is on ECS: the attribute is dropped
	// and the rest of the resource is still reported.
	url := newFakeService(t, &fakeService{
		meta:     fullMetadata(),
		hijacked: map[string]bool{"zone-id": true},
	})

	res, err := newTestDetector(url).Detect(t.Context())
	require.Error(t, err)
	assert.ErrorIs(t, err, resource.ErrPartialResource)

	_, ok := res.Set().Value(semconv.CloudAvailabilityZoneKey)
	assert.False(t, ok, "expected cloud.availability_zone to be absent")

	val, ok := res.Set().Value(semconv.HostIDKey)
	require.True(t, ok)
	assert.Equal(t, attribute.StringValue("i-abcdef123456"), val)
}

func TestDetect_WithAttributeFilter(t *testing.T) {
	url := newFakeService(t, &fakeService{meta: fullMetadata()})

	filter := attribute.NewDenyKeysFilter(semconv.CloudPlatformKey, semconv.HostImageIDKey)
	res, err := newTestDetector(url, WithAttributeFilter(filter)).Detect(t.Context())
	require.NoError(t, err)

	for _, k := range []attribute.Key{
		semconv.CloudPlatformKey,
		semconv.HostImageIDKey,
	} {
		_, ok := res.Set().Value(k)
		assert.False(t, ok, "expected %s to be absent", k)
	}

	for _, kv := range []attribute.KeyValue{
		semconv.CloudProviderAlibabaCloud,
		semconv.CloudAccountID("1234567890123456"),
		semconv.CloudRegion("cn-hangzhou"),
		semconv.CloudAvailabilityZone("cn-hangzhou-a"),
		semconv.HostID("i-abcdef123456"),
		semconv.HostName("ecs-instance-01"),
		semconv.HostType("ecs.g6.large"),
	} {
		val, ok := res.Set().Value(kv.Key)
		assert.True(t, ok, "expected %s to be present", kv.Key)
		assert.Equal(t, kv.Value, val)
	}
}
