// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package oraclecloud

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

func TestNewResourceDetector(t *testing.T) {
	d := NewResourceDetector()
	require.NotNil(t, d)
	assert.Equal(t, defaultEndpoint, d.endpoint)
	assert.Equal(t, 2*time.Second, d.client.Timeout)

	transport, ok := d.client.Transport.(*http.Transport)
	require.True(t, ok)
	assert.Nil(t, transport.Proxy)
	assert.NotNil(t, d.client.CheckRedirect)
}

func TestDetectOKComputeUsesHostname(t *testing.T) {
	d := newTestDetector(t, http.StatusOK, `{
		"id":"ocid1.instance.oc1.phx.example",
		"hostname":"instance-hostname",
		"availabilityDomain":"Uocm:PHX-AD-1",
		"canonicalRegionName":"us-phoenix-1",
		"shape":"VM.Standard.E3.Flex",
		"regionInfo":{"realmKey":"oc1"},
		"metadata":{"realm":"ignored-realm"}
	}`)

	res, err := d.Detect(t.Context())
	require.NoError(t, err)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderOracleCloud,
		semconv.CloudPlatformOracleCloudCompute,
		semconv.CloudRegion("us-phoenix-1"),
		semconv.CloudAvailabilityZone("Uocm:PHX-AD-1"),
		semconv.CloudResourceID("ocid1.instance.oc1.phx.example"),
		semconv.HostID("ocid1.instance.oc1.phx.example"),
		semconv.HostName("instance-hostname"),
		semconv.HostType("VM.Standard.E3.Flex"),
		semconv.OracleCloudRealm("oc1"),
	)
	assert.Equal(t, expected, res)
	_, ok := res.Set().Value(semconv.CloudAccountIDKey)
	assert.False(t, ok)
	assertKV(t, res, semconv.CloudResourceID("ocid1.instance.oc1.phx.example"))
}

func TestDetectOKOKEAddsClusterName(t *testing.T) {
	d := newTestDetector(t, http.StatusOK, `{
		"id":"ocid1.instance.oc1.phx.example",
		"hostname":"worker-1",
		"availabilityDomain":"Uocm:PHX-AD-1",
		"canonicalRegionName":"us-phoenix-1",
		"regionInfo":{"realmKey":"oc1"},
		"metadata":{"oke-cluster-display-name":"prod-oke"}
	}`)

	res, err := d.Detect(t.Context())
	require.NoError(t, err)

	assertKV(t, res, semconv.CloudPlatformOracleCloudOKE)
	assertKV(t, res, semconv.K8SClusterName("prod-oke"))
}

func TestDetectMissingIDReturnsPartialResource(t *testing.T) {
	d := newTestDetector(t, http.StatusOK, `{
		"hostname":"instance-hostname",
		"availabilityDomain":"Uocm:PHX-AD-1",
		"canonicalRegionName":"us-phoenix-1",
		"shape":"VM.Standard.E3.Flex",
		"regionInfo":{"realmKey":"oc1"}
	}`)

	res, err := d.Detect(t.Context())
	require.Error(t, err)
	assert.ErrorIs(t, err, resource.ErrPartialResource)

	assertKV(t, res, semconv.CloudProviderOracleCloud)
	assertKV(t, res, semconv.CloudPlatformOracleCloudCompute)
	assertKV(t, res, semconv.HostName("instance-hostname"))
	assertKV(t, res, semconv.CloudRegion("us-phoenix-1"))

	_, ok := res.Set().Value(semconv.HostIDKey)
	assert.False(t, ok)
}

func TestDetectWithAttributeFilterAppliesBeforeReturningPartial(t *testing.T) {
	d := newTestDetector(t, http.StatusOK, `{
		"hostname":"instance-hostname",
		"availabilityDomain":"Uocm:PHX-AD-1",
		"canonicalRegionName":"us-phoenix-1",
		"shape":"VM.Standard.E3.Flex",
		"regionInfo":{"realmKey":"oc1"}
	}`, WithAttributeFilter(attribute.NewDenyKeysFilter(semconv.HostNameKey)))

	res, err := d.Detect(t.Context())
	require.Error(t, err)
	assert.ErrorIs(t, err, resource.ErrPartialResource)

	_, ok := res.Set().Value(semconv.HostNameKey)
	assert.False(t, ok)
	assertKV(t, res, semconv.CloudProviderOracleCloud)
}

func TestDetectWithAttributeFilterSelectsResourceIDRepresentation(t *testing.T) {
	d := newTestDetector(t, http.StatusOK, `{
		"id":"ocid1.instance.oc1.phx.example"
	}`, WithAttributeFilter(attribute.NewDenyKeysFilter(semconv.HostIDKey)))

	res, err := d.Detect(t.Context())
	require.NoError(t, err)

	_, ok := res.Set().Value(semconv.HostIDKey)
	assert.False(t, ok)
	assertKV(t, res, semconv.CloudResourceID("ocid1.instance.oc1.phx.example"))
}

func TestDetectUnreachableReturnsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close()

	res, err := newDetectorAt(url).Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetectCallerContextCancelledReturnsContextError(t *testing.T) {
	d := newTestDetector(t, http.StatusOK, `{"id":"ocid1.instance.oc1.phx.example"}`)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	res, err := d.Detect(ctx)
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, res)
}

func TestDetect4xxReturnsEmpty(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusNotFound} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			d := newTestDetector(t, status, `nope`)
			res, err := d.Detect(t.Context())
			require.NoError(t, err)
			assert.Equal(t, resource.Empty(), res)
		})
	}
}

func TestDetect429ReturnsError(t *testing.T) {
	d := newTestDetector(t, http.StatusTooManyRequests, `slow down`)
	res, err := d.Detect(t.Context())
	require.Error(t, err)
	assert.Nil(t, res)
}

func TestDetect5xxReturnsError(t *testing.T) {
	d := newTestDetector(t, http.StatusServiceUnavailable, `unavailable`)
	res, err := d.Detect(t.Context())
	require.Error(t, err)
	assert.Nil(t, res)
}

func TestDetectRedirectReturnsError(t *testing.T) {
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/somewhere", http.StatusFound)
	}))
	t.Cleanup(redirect.Close)

	res, err := newDetectorAt(redirect.URL).Detect(t.Context())
	require.Error(t, err)
	assert.Nil(t, res)
}

func TestDetectOversizedResponseReturnsError(t *testing.T) {
	body := `{"id":"` + strings.Repeat("x", maxBodySize) + `"}`
	d := newTestDetector(t, http.StatusOK, body)

	res, err := d.Detect(t.Context())
	require.Error(t, err)
	assert.Nil(t, res)
}

func TestDetectMalformedJSONReturnsError(t *testing.T) {
	d := newTestDetector(t, http.StatusOK, `{`)
	res, err := d.Detect(t.Context())
	require.Error(t, err)
	assert.Nil(t, res)
}

func TestDetectRealmFallsBackToMetadata(t *testing.T) {
	d := newTestDetector(t, http.StatusOK, `{
		"id":"ocid1.instance.oc1.phx.example",
		"hostname":"instance-hostname",
		"metadata":{"realm":"oc2"}
	}`)

	res, err := d.Detect(t.Context())
	require.NoError(t, err)
	assertKV(t, res, semconv.OracleCloudRealm("oc2"))
}

func newTestDetector(t *testing.T, status int, body string, opts ...Option) *ResourceDetector {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/opc/v2/instance/", r.URL.Path)
		assert.Equal(t, "Bearer Oracle", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return newDetectorAt(srv.URL+"/opc/v2/instance/", opts...)
}

func newDetectorAt(url string, opts ...Option) *ResourceDetector {
	d := NewResourceDetector(opts...)
	d.endpoint = url
	return d
}

func assertKV(t *testing.T, res *resource.Resource, want attribute.KeyValue) {
	t.Helper()
	got, ok := res.Set().Value(want.Key)
	require.True(t, ok, "missing attribute %s", want.Key)
	assert.Equal(t, want.Value, got)
}
