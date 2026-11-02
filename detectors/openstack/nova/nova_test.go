// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package nova

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

const (
	metadataPath     = "/openstack/latest/meta_data.json"
	instanceTypePath = "/latest/meta-data/instance-type"
)

// fakeServer is a metadata service serving both the Nova metadata document and
// the EC2 compatible instance type endpoint.
type fakeServer struct {
	url string
	// metadataRequests counts the requests made to the metadata document.
	metadataRequests atomic.Int64
}

// newFakeServer starts a metadata service serving meta as the metadata
// document and instanceType from the EC2 compatible endpoint. An empty
// instanceType makes that endpoint respond with 404, as deployments without EC2
// compatibility do. The server is closed via t.Cleanup.
func newFakeServer(t *testing.T, meta metadataResponse, instanceType string) *fakeServer {
	t.Helper()

	fs := &fakeServer{}

	mux := http.NewServeMux()
	mux.HandleFunc(metadataPath, func(w http.ResponseWriter, _ *http.Request) {
		fs.metadataRequests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(meta)
	})
	mux.HandleFunc(instanceTypePath, func(w http.ResponseWriter, _ *http.Request) {
		if instanceType == "" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(instanceType))
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	fs.url = srv.URL
	return fs
}

// newTestDetector returns a detector pointed at url instead of the real
// link-local metadata endpoints.
func newTestDetector(url string, opts ...Option) *ResourceDetector {
	d := NewResourceDetector(opts...)
	d.endpoint = url + metadataPath
	d.ec2Endpoint = url + instanceTypePath
	return d
}

func fullMetadata() metadataResponse {
	return metadataResponse{
		AvailabilityZone: "az1",
		Hostname:         "test-openstack-instance.example.com",
		Meta: map[string]string{
			"team":  "observability",
			"env":   "testing",
			"other": "ignored",
		},
		ProjectID: "test-project-id-12345",
		UUID:      "b221c929-59b8-4c45-a6fa-104ee0b8b175",
	}
}

func TestNewResourceDetector(t *testing.T) {
	d := NewResourceDetector()
	require.NotNil(t, d)
	assert.Equal(t, defaultEndpoint, d.endpoint)
	assert.Equal(t, defaultEC2Endpoint, d.ec2Endpoint)
}

func TestDetect_OK(t *testing.T) {
	srv := newFakeServer(t, fullMetadata(), "m1.medium")

	keys := regexp.MustCompile(`^(team|env)$`)
	res, err := newTestDetector(srv.url, WithMetaKeyFilter(keys.MatchString)).Detect(t.Context())
	require.NoError(t, err)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderKey.String("openstack"),
		semconv.CloudPlatformKey.String("openstack_nova"),
		semconv.CloudAccountID("test-project-id-12345"),
		semconv.CloudAvailabilityZone("az1"),
		semconv.HostID("b221c929-59b8-4c45-a6fa-104ee0b8b175"),
		semconv.HostName("test-openstack-instance.example.com"),
		semconv.HostType("m1.medium"),
		attribute.String("openstack.nova.meta.team", "observability"),
		attribute.String("openstack.nova.meta.env", "testing"),
	)
	assert.Equal(t, expected, res)
}

func TestDetect_NoMetaKeyFilter(t *testing.T) {
	// Without WithMetaKeyFilter no instance metadata entry is emitted.
	srv := newFakeServer(t, fullMetadata(), "m1.medium")

	res, err := newTestDetector(srv.url).Detect(t.Context())
	require.NoError(t, err)

	for _, k := range []attribute.Key{
		"openstack.nova.meta.team",
		"openstack.nova.meta.env",
		"openstack.nova.meta.other",
	} {
		_, ok := res.Set().Value(k)
		assert.False(t, ok, "expected %s to be absent", k)
	}
}

func TestDetect_SingleMetadataRequest(t *testing.T) {
	// The metadata document holds every attribute but host.type, so a single
	// request must be enough to detect them.
	srv := newFakeServer(t, fullMetadata(), "m1.medium")

	_, err := newTestDetector(srv.url).Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, int64(1), srv.metadataRequests.Load())
}

func TestDetect_HostTypeUnavailable(t *testing.T) {
	// Deployments without EC2 compatibility do not serve the instance type.
	// The attribute is omitted, but detection still succeeds.
	srv := newFakeServer(t, fullMetadata(), "")

	res, err := newTestDetector(srv.url).Detect(t.Context())
	require.NoError(t, err)

	_, ok := res.Set().Value(semconv.HostTypeKey)
	assert.False(t, ok, "expected host.type to be absent")

	val, ok := res.Set().Value(semconv.HostIDKey)
	require.True(t, ok)
	assert.Equal(t, attribute.StringValue("b221c929-59b8-4c45-a6fa-104ee0b8b175"), val)
}

func TestDetect_NotOnOpenStack(t *testing.T) {
	// A client error means something other than the Nova metadata service
	// answered the request: not on OpenStack, so no error is reported.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	res, err := newTestDetector(srv.URL).Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetect_ServerError(t *testing.T) {
	// The metadata service answered but failed: surface the error instead of
	// silently reporting "not on OpenStack".
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
	// so this is a failure rather than "not on OpenStack".
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not json"))
	}))
	t.Cleanup(srv.Close)

	res, err := newTestDetector(srv.URL).Detect(t.Context())
	require.Error(t, err)
	assert.Nil(t, res)
}

func TestDetect_ConnectionRefused(t *testing.T) {
	// Closed server → connection refused → not on OpenStack → empty resource,
	// no error.
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close()

	res, err := newTestDetector(url).Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetect_PartialFailure(t *testing.T) {
	// Serve a metadata document with every field absent.
	srv := newFakeServer(t, metadataResponse{}, "")

	res, err := newTestDetector(srv.url).Detect(t.Context())
	require.Error(t, err)
	assert.ErrorIs(t, err, resource.ErrPartialResource)

	// cloud.provider and cloud.platform must still be present.
	for _, kv := range []attribute.KeyValue{
		semconv.CloudProviderKey.String("openstack"),
		semconv.CloudPlatformKey.String("openstack_nova"),
	} {
		val, ok := res.Set().Value(kv.Key)
		assert.True(t, ok, "expected %s to be present", kv.Key)
		assert.Equal(t, kv.Value, val)
	}

	// Everything read from the metadata document must be absent.
	for _, k := range []attribute.Key{
		semconv.CloudAccountIDKey,
		semconv.CloudAvailabilityZoneKey,
		semconv.HostIDKey,
		semconv.HostNameKey,
		semconv.HostTypeKey,
	} {
		_, ok := res.Set().Value(k)
		assert.False(t, ok, "expected %s to be absent", k)
	}
}

func TestDetect_WithAttributeFilter(t *testing.T) {
	srv := newFakeServer(t, fullMetadata(), "m1.medium")

	keys := regexp.MustCompile(`^team$`)
	filter := attribute.NewDenyKeysFilter(semconv.CloudPlatformKey, "openstack.nova.meta.team")
	res, err := newTestDetector(
		srv.url,
		WithMetaKeyFilter(keys.MatchString),
		WithAttributeFilter(filter),
	).Detect(t.Context())
	require.NoError(t, err)

	// Denied keys are dropped, including the instance metadata ones.
	for _, k := range []attribute.Key{semconv.CloudPlatformKey, "openstack.nova.meta.team"} {
		_, ok := res.Set().Value(k)
		assert.False(t, ok, "expected %s to be absent", k)
	}

	for _, kv := range []attribute.KeyValue{
		semconv.CloudProviderKey.String("openstack"),
		semconv.CloudAccountID("test-project-id-12345"),
		semconv.CloudAvailabilityZone("az1"),
		semconv.HostID("b221c929-59b8-4c45-a6fa-104ee0b8b175"),
		semconv.HostName("test-openstack-instance.example.com"),
		semconv.HostType("m1.medium"),
	} {
		val, ok := res.Set().Value(kv.Key)
		assert.True(t, ok, "expected %s to be present", kv.Key)
		assert.Equal(t, kv.Value, val)
	}
}
