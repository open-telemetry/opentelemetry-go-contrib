// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package alibabacloud

import (
	"io"
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

// fakeMetadata describes the responses of a fake metadata service.
type fakeMetadata struct {
	// token is served by the token endpoint. An empty token makes the token
	// endpoint return 404 and lets metadata requests succeed without a token,
	// like instances without hardened mode or ECI containers.
	token string
	// fields maps a metadata item path suffix to its value.
	fields map[string]string
	// statuses forces a response status for a metadata item path suffix.
	statuses map[string]int
}

// newFakeServer starts an httptest server serving the fakeMetadata and
// returns it together with a log of every request it received. The server is
// closed via t.Cleanup.
func newFakeServer(t *testing.T, f fakeMetadata) (*httptest.Server, *[]*http.Request) {
	t.Helper()

	var mu sync.Mutex
	var reqs []*http.Request

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		reqs = append(reqs, r.Clone(t.Context()))
		mu.Unlock()

		switch {
		case r.Method == http.MethodPut && r.URL.Path == tokenEndpoint:
			if f.token == "" {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			_, _ = io.WriteString(w, f.token)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, metaPrefix):
			if f.token != "" && r.Header.Get(tokenHeader) != f.token {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			item := strings.TrimPrefix(r.URL.Path, metaPrefix)
			if status, ok := f.statuses[item]; ok {
				http.Error(w, "boom", status)
				return
			}
			v, ok := f.fields[item]
			if !ok {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			_, _ = io.WriteString(w, v)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &reqs
}

// newTestDetector returns a detector pointed at url instead of the real
// link-local metadata endpoint.
func newTestDetector(url string, opts ...Option) *ResourceDetector {
	d := NewResourceDetector(opts...)
	d.endpoint = url
	return d
}

// fullFixture covers every metadata item the detector reads.
func fullFixture() fakeMetadata {
	return fakeMetadata{
		token: "test-token",
		fields: map[string]string{
			"instance-id":            "i-bp13znx0m0me8cquu1234",
			"region-id":              "cn-hangzhou",
			"zone-id":                "cn-hangzhou-i",
			"owner-account-id":       "1609982529920000",
			"hostname":               "iZbp13znx0m0me8cquu1234Z",
			"instance/instance-type": "ecs.g6e.large",
			"image-id":               "aliyun_3_x64_20G_alibase_20210425.vhd",
		},
	}
}

func TestNewResourceDetector(t *testing.T) {
	d := NewResourceDetector()
	require.NotNil(t, d)
	assert.Equal(t, defaultEndpoint, d.endpoint)
	assert.NotNil(t, d.client)
	assert.Nil(t, d.cfg.filter)

	filter := attribute.NewAllowKeysFilter(semconv.HostIDKey)
	d = NewResourceDetector(WithAttributeFilter(filter))
	require.NotNil(t, d)
	assert.NotNil(t, d.cfg.filter)
}

func TestDetect_IMDSv2(t *testing.T) {
	srv, reqs := newFakeServer(t, fullFixture())

	res, err := newTestDetector(srv.URL).Detect(t.Context())
	require.NoError(t, err)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderAlibabaCloud,
		semconv.CloudPlatformAlibabaCloudECS,
		semconv.HostID("i-bp13znx0m0me8cquu1234"),
		semconv.CloudRegion("cn-hangzhou"),
		semconv.CloudAvailabilityZone("cn-hangzhou-i"),
		semconv.CloudAccountID("1609982529920000"),
		semconv.HostName("iZbp13znx0m0me8cquu1234Z"),
		semconv.HostType("ecs.g6e.large"),
		semconv.HostImageID("aliyun_3_x64_20G_alibase_20210425.vhd"),
	)
	assert.Equal(t, expected, res)

	var sawTokenRequest bool
	for _, r := range *reqs {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == tokenEndpoint:
			sawTokenRequest = true
			assert.Equal(t, tokenTTL, r.Header.Get(tokenTTLHeader))
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, metaPrefix):
			assert.Equal(t, "test-token", r.Header.Get(tokenHeader),
				"metadata request for %q must carry the token", r.URL.Path)
		}
	}
	assert.True(t, sawTokenRequest, "expected a token request")
}

func TestDetect_IMDSv1Fallback(t *testing.T) {
	// An empty token makes the token endpoint return 404, e.g. on ECI
	// containers. Metadata must then be read without a token.
	f := fullFixture()
	f.token = ""
	srv, reqs := newFakeServer(t, f)

	res, err := newTestDetector(srv.URL).Detect(t.Context())
	require.NoError(t, err)

	val, ok := res.Set().Value(semconv.HostIDKey)
	require.True(t, ok)
	assert.Equal(t, "i-bp13znx0m0me8cquu1234", val.AsString())

	for _, r := range *reqs {
		assert.Empty(t, r.Header.Get(tokenHeader),
			"no request must carry a token in IMDSv1 mode")
	}
}

func TestDetect_HardenedModeWithoutToken(t *testing.T) {
	// The token endpoint is unavailable and metadata rejects unauthenticated
	// reads: not usable as Alibaba Cloud metadata.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	t.Cleanup(srv.Close)

	res, err := newTestDetector(srv.URL).Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetect_NotOnAlibabaCloud(t *testing.T) {
	// Closed server → connection refused → not on Alibaba Cloud → empty
	// resource, no error.
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close()

	res, err := newTestDetector(url).Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetect_TokenEndpointUnreachable(t *testing.T) {
	// A transport failure on the token request means the metadata service
	// itself is unreachable: detection must stop without issuing any metadata
	// request, instead of doubling the timeout on a second doomed request.
	var mu sync.Mutex
	var metadataRequests int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			// Abort the connection without a response: the client observes a
			// transport error, not an HTTP status.
			conn, _, err := w.(http.Hijacker).Hijack()
			if err == nil {
				_ = conn.Close()
			}
			return
		}
		mu.Lock()
		metadataRequests++
		mu.Unlock()
	}))
	t.Cleanup(srv.Close)

	res, err := newTestDetector(srv.URL).Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)

	mu.Lock()
	defer mu.Unlock()
	assert.Zero(t, metadataRequests, "no metadata request must follow an unreachable token endpoint")
}

func TestDetect_InstanceIDServerError(t *testing.T) {
	// The metadata service answered but failed on the anchor item: surface the
	// error instead of silently reporting "not on Alibaba Cloud".
	f := fullFixture()
	f.statuses = map[string]int{"instance-id": http.StatusInternalServerError}
	srv, _ := newFakeServer(t, f)

	res, err := newTestDetector(srv.URL).Detect(t.Context())
	assert.Error(t, err)
	assert.Nil(t, res)
}

func TestDetect_EmptyInstanceID(t *testing.T) {
	f := fullFixture()
	f.fields["instance-id"] = " "
	srv, _ := newFakeServer(t, f)

	_, err := newTestDetector(srv.URL).Detect(t.Context())
	require.Error(t, err)
	assert.ErrorContains(t, err, "empty instance-id")
}

func TestDetect_OptionalItemsMissing(t *testing.T) {
	// ECI containers do not expose every item: missing items are skipped
	// without an error.
	f := fullFixture()
	delete(f.fields, "instance/instance-type")
	delete(f.fields, "image-id")
	srv, _ := newFakeServer(t, f)

	res, err := newTestDetector(srv.URL).Detect(t.Context())
	require.NoError(t, err)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderAlibabaCloud,
		semconv.CloudPlatformAlibabaCloudECS,
		semconv.HostID("i-bp13znx0m0me8cquu1234"),
		semconv.CloudRegion("cn-hangzhou"),
		semconv.CloudAvailabilityZone("cn-hangzhou-i"),
		semconv.CloudAccountID("1609982529920000"),
		semconv.HostName("iZbp13znx0m0me8cquu1234Z"),
	)
	assert.Equal(t, expected, res)
}

func TestDetect_PartialFailure(t *testing.T) {
	// Two items fail while the rest succeed: return the partial resource
	// together with [resource.ErrPartialResource], wrapping the underlying
	// errors so callers can inspect them.
	f := fullFixture()
	f.statuses = map[string]int{
		"region-id": http.StatusInternalServerError,
		"zone-id":   http.StatusServiceUnavailable,
	}
	srv, _ := newFakeServer(t, f)

	res, err := newTestDetector(srv.URL).Detect(t.Context())
	require.Error(t, err)
	assert.ErrorIs(t, err, resource.ErrPartialResource)
	assert.ErrorContains(t, err, "region-id")

	// host.id and host.type must still be present.
	for _, k := range []attribute.Key{
		semconv.HostIDKey,
		semconv.HostTypeKey,
	} {
		_, ok := res.Set().Value(k)
		assert.True(t, ok, "expected %s to be present", k)
	}

	// The failed items must be absent.
	for _, k := range []attribute.Key{
		semconv.CloudRegionKey,
		semconv.CloudAvailabilityZoneKey,
	} {
		_, ok := res.Set().Value(k)
		assert.False(t, ok, "expected %s to be absent", k)
	}
}

func TestDetect_WithAttributeFilter(t *testing.T) {
	url := newFakeServerURL(t, fullFixture())

	filter := attribute.NewDenyKeysFilter(semconv.CloudPlatformKey)
	res, err := newTestDetector(url, WithAttributeFilter(filter)).Detect(t.Context())
	require.NoError(t, err)

	_, ok := res.Set().Value(semconv.CloudPlatformKey)
	assert.False(t, ok, "expected cloud.platform to be absent")

	for _, kv := range []attribute.KeyValue{
		semconv.CloudProviderAlibabaCloud,
		semconv.HostID("i-bp13znx0m0me8cquu1234"),
		semconv.HostName("iZbp13znx0m0me8cquu1234Z"),
		semconv.HostType("ecs.g6e.large"),
	} {
		val, ok := res.Set().Value(kv.Key)
		assert.True(t, ok, "expected %s to be present", kv.Key)
		assert.Equal(t, kv.Value, val)
	}
}

// newFakeServerURL is a convenience wrapper around newFakeServer for tests
// that only need the server URL.
func newFakeServerURL(t *testing.T, f fakeMetadata) string {
	t.Helper()
	srv, _ := newFakeServer(t, f)
	return srv.URL
}
