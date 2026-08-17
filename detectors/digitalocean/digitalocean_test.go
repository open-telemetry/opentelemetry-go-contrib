// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package digitalocean

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	do "github.com/digitalocean/go-metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

// metadataPath is the only path the detector requests.
const metadataPath = "/metadata/v1.json"

// metadataSample is a response of the metadata service of a DigitalOcean
// Droplet, trimmed to the fields the detector reads plus a few it ignores.
const metadataSample = `{
  "droplet_id": 2756294,
  "hostname": "sample-droplet",
  "region": "nyc3",
  "vendor_data": "#cloud-config",
  "public_keys": ["ssh-rsa AEXAMPLEKEY"],
  "tags": ["web", "env:prod"],
  "features": {
    "dhcp_enabled": true
  }
}`

// serveMetadata points the detector at a test server running handler. The
// server is closed and the client factory restored via t.Cleanup.
func serveMetadata(t *testing.T, handler http.HandlerFunc) {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	useBaseURL(t, srv.URL)
}

// useBaseURL makes the detector query rawURL instead of the real link-local
// metadata service.
func useBaseURL(t *testing.T, rawURL string) {
	t.Helper()

	base, err := url.Parse(rawURL)
	require.NoError(t, err)

	orig := newClient
	t.Cleanup(func() { newClient = orig })

	newClient = func(opts ...do.ClientOption) *do.Client {
		return orig(append([]do.ClientOption{do.WithBaseURL(base)}, opts...)...)
	}
}

// serveJSON serves body as JSON on the metadata path and 404s everything else.
func serveJSON(t *testing.T, body string) {
	t.Helper()

	serveMetadata(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != metadataPath {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	})
}

func TestNewResourceDetector(t *testing.T) {
	d := NewResourceDetector()
	require.NotNil(t, d)
	assert.Nil(t, d.cfg.filter)
}

func TestDetect(t *testing.T) {
	serveJSON(t, metadataSample)

	res, err := NewResourceDetector().Detect(t.Context())
	require.NoError(t, err)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		cloudProviderDigitalOcean,
		semconv.HostID("2756294"),
		semconv.HostName("sample-droplet"),
		semconv.CloudRegion("nyc3"),
	)
	assert.Equal(t, expected, res)
}

func TestDetectFetchesMetadataOnce(t *testing.T) {
	// All attributes come from a single document: the detector must not request
	// it more than once per Detect call.
	var requests atomic.Int64

	serveMetadata(t, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path != metadataPath {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(metadataSample))
	})

	_, err := NewResourceDetector().Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, int64(1), requests.Load())
}

func TestDetectNotOnDigitalOcean(t *testing.T) {
	// A client error means something other than the DigitalOcean metadata
	// service answered the request: not on a Droplet, so no error is reported.
	serveMetadata(t, func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	res, err := NewResourceDetector().Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetectConnectionRefused(t *testing.T) {
	// Closed server: connection refused, so nothing answered the link-local
	// address and the process is not on a Droplet.
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	rawURL := srv.URL
	srv.Close()

	useBaseURL(t, rawURL)

	res, err := NewResourceDetector().Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetectServerError(t *testing.T) {
	// The metadata service answered but failed: surface the error instead of
	// silently reporting "not on DigitalOcean".
	serveMetadata(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})

	res, err := NewResourceDetector().Detect(t.Context())
	require.Error(t, err)
	assert.Nil(t, res)
}

func TestDetectMalformedJSON(t *testing.T) {
	// 200 OK with a body that is not valid JSON. The metadata service
	// responded, so this is a failure rather than "not on DigitalOcean".
	serveJSON(t, "not json")

	res, err := NewResourceDetector().Detect(t.Context())
	require.Error(t, err)
	assert.Nil(t, res)
}

func TestDetectPartialResource(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		missing attribute.Key
		present []attribute.KeyValue
	}{
		{
			name:    "droplet_id",
			body:    `{"hostname": "sample-droplet", "region": "nyc3"}`,
			missing: semconv.HostIDKey,
			present: []attribute.KeyValue{
				semconv.HostName("sample-droplet"),
				semconv.CloudRegion("nyc3"),
			},
		},
		{
			name:    "hostname",
			body:    `{"droplet_id": 2756294, "region": "nyc3"}`,
			missing: semconv.HostNameKey,
			present: []attribute.KeyValue{
				semconv.HostID("2756294"),
				semconv.CloudRegion("nyc3"),
			},
		},
		{
			name:    "region",
			body:    `{"droplet_id": 2756294, "hostname": "sample-droplet"}`,
			missing: semconv.CloudRegionKey,
			present: []attribute.KeyValue{
				semconv.HostID("2756294"),
				semconv.HostName("sample-droplet"),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			serveJSON(t, test.body)

			res, err := NewResourceDetector().Detect(t.Context())
			require.Error(t, err)
			assert.ErrorIs(t, err, resource.ErrPartialResource)
			assert.Contains(t, err.Error(), test.name)

			_, ok := res.Set().Value(test.missing)
			assert.False(t, ok, "expected %s to be absent", test.missing)

			// cloud.provider and every other detected attribute must remain.
			for _, kv := range append(test.present, cloudProviderDigitalOcean) {
				val, ok := res.Set().Value(kv.Key)
				assert.True(t, ok, "expected %s to be present", kv.Key)
				assert.Equal(t, kv.Value, val)
			}
		})
	}
}

func TestDetectEmptyMetadata(t *testing.T) {
	// The metadata service answered with a document carrying none of the
	// detected fields: a partial resource holding only cloud.provider.
	serveJSON(t, `{}`)

	res, err := NewResourceDetector().Detect(t.Context())
	require.Error(t, err)
	assert.ErrorIs(t, err, resource.ErrPartialResource)

	expected := resource.NewWithAttributes(semconv.SchemaURL, cloudProviderDigitalOcean)
	assert.Equal(t, expected, res)
}

func TestDetectWithAttributeFilter(t *testing.T) {
	serveJSON(t, metadataSample)

	filter := attribute.NewDenyKeysFilter(semconv.HostNameKey)
	res, err := NewResourceDetector(WithAttributeFilter(filter)).Detect(t.Context())
	require.NoError(t, err)

	_, ok := res.Set().Value(semconv.HostNameKey)
	assert.False(t, ok, "expected host.name to be absent")

	for _, kv := range []attribute.KeyValue{
		cloudProviderDigitalOcean,
		semconv.HostID("2756294"),
		semconv.CloudRegion("nyc3"),
	} {
		val, ok := res.Set().Value(kv.Key)
		assert.True(t, ok, "expected %s to be present", kv.Key)
		assert.Equal(t, kv.Value, val)
	}
}

func TestDetectCanceledContext(t *testing.T) {
	serveJSON(t, metadataSample)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	res, err := NewResourceDetector().Detect(ctx)
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, res)
}
