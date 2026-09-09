// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package akamai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	linodemeta "github.com/linode/go-metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

const testToken = "0123456789abcdef"

func testInstance() linodemeta.InstanceData {
	return linodemeta.InstanceData{
		ID:           4242,
		Label:        "linode-4242",
		Region:       "us-southeast",
		Type:         "g6-standard-4",
		HostUUID:     "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
		Tags:         []string{"environment:testing"},
		Image:        linodemeta.InstanceImageData{ID: "linode/ubuntu24.04", Label: "Ubuntu 24.04 LTS"},
		AccountEUUID: "ABCD1234-5678-90EF-GHIJ-KLMNOPQRSTUV",
	}
}

// counters records how often each metadata endpoint was called.
type counters struct {
	token    atomic.Int32
	instance atomic.Int32
}

// serverConfig customizes the fake metadata service. A nil body function makes
// the endpoint answer with the default successful response.
type serverConfig struct {
	tokenHandler    http.HandlerFunc
	instanceHandler http.HandlerFunc
	counters        *counters
}

// newFakeServer starts a fake Akamai instance metadata service. It mirrors the
// real service: the token endpoint requires the expiry header and answers with a
// JSON array holding one token, and the instance endpoint requires that token.
func newFakeServer(t *testing.T, cfg serverConfig) string {
	t.Helper()

	// Encode the bodies up front: the handlers run on their own goroutines,
	// where a failed assertion could not stop the test.
	tokenBody, err := json.Marshal([]string{testToken})
	require.NoError(t, err)
	instanceBody, err := json.Marshal(testInstance())
	require.NoError(t, err)

	mux := http.NewServeMux()
	mux.HandleFunc("PUT /v1/token", func(w http.ResponseWriter, r *http.Request) {
		if cfg.counters != nil {
			cfg.counters.token.Add(1)
		}
		if cfg.tokenHandler != nil {
			cfg.tokenHandler(w, r)
			return
		}
		if r.Header.Get("Metadata-Token-Expiry-Seconds") == "" {
			http.Error(w, "Missing Metadata-Token-Expiry-Seconds header", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(tokenBody)
	})
	mux.HandleFunc("GET /v1/instance", func(w http.ResponseWriter, r *http.Request) {
		if cfg.counters != nil {
			cfg.counters.instance.Add(1)
		}
		if r.Header.Get("Metadata-Token") != testToken {
			http.Error(w, "Invalid or missing Metadata-Token", http.StatusUnauthorized)
			return
		}
		if cfg.instanceHandler != nil {
			cfg.instanceHandler(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(instanceBody)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv.URL
}

func newTestDetector(url string, opts ...Option) *ResourceDetector {
	d := NewResourceDetector(opts...)
	d.baseURL = url
	return d
}

// expectedResource is the resource a successful detection of testInstance yields.
func expectedResource() *resource.Resource {
	inst := testInstance()
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderAkamaiCloud,
		semconv.CloudPlatformAkamaiCloudCompute,
		semconv.CloudAccountID(inst.AccountEUUID),
		semconv.CloudRegion(inst.Region),
		semconv.HostID("4242"),
		semconv.HostName(inst.Label),
		semconv.HostType(inst.Type),
		semconv.HostImageID(inst.Image.ID),
		semconv.HostImageName(inst.Image.Label),
	)
}

func TestNewResourceDetector(t *testing.T) {
	d := NewResourceDetector()
	require.NotNil(t, d)
	assert.Equal(t, defaultBaseURL, d.baseURL)
	assert.Equal(t, requestTimeout, d.client.Timeout)
}

func TestDetect(t *testing.T) {
	d := newTestDetector(newFakeServer(t, serverConfig{}))

	res, err := d.Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, expectedResource(), res)
}

// TestDetectFetchesEachDocumentOnce pins the request count. The collector's
// implementation mints a token before every request because of an upstream
// token expiry bug; this package must not.
func TestDetectFetchesEachDocumentOnce(t *testing.T) {
	var c counters
	d := newTestDetector(newFakeServer(t, serverConfig{counters: &c}))

	_, err := d.Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, int32(1), c.token.Load(), "one token request per detection")
	assert.Equal(t, int32(1), c.instance.Load(), "one instance request per detection")

	_, err = d.Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, int32(2), c.token.Load(), "tokens are not reused across detections")
	assert.Equal(t, int32(2), c.instance.Load())
}

func TestDetectNotOnAkamai(t *testing.T) {
	// Something other than the Akamai metadata service answered.
	d := newTestDetector(newFakeServer(t, serverConfig{
		tokenHandler: func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		},
	}))

	res, err := d.Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetectConnectionRefused(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()

	d := newTestDetector(url)

	res, err := d.Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetectTokenServerError(t *testing.T) {
	d := newTestDetector(newFakeServer(t, serverConfig{
		tokenHandler: func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		},
	}))

	res, err := d.Detect(t.Context())
	require.Error(t, err)
	assert.Nil(t, res)
}

func TestDetectTokenMalformedJSON(t *testing.T) {
	d := newTestDetector(newFakeServer(t, serverConfig{
		tokenHandler: func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("not json"))
		},
	}))

	res, err := d.Detect(t.Context())
	require.Error(t, err)
	assert.Nil(t, res)
}

func TestDetectTokenEmpty(t *testing.T) {
	d := newTestDetector(newFakeServer(t, serverConfig{
		tokenHandler: func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("[]"))
		},
	}))

	res, err := d.Detect(t.Context())
	require.Error(t, err)
	assert.Nil(t, res)
}

// TestDetectInstanceUnauthorized covers a plain-text 4xx from the instance
// endpoint. github.com/linode/go-metadata reports that as a successful,
// zero-valued instance document, which would otherwise be emitted as a resource
// with host.id "0" and empty region, name and type.
func TestDetectInstanceUnauthorized(t *testing.T) {
	d := newTestDetector(newFakeServer(t, serverConfig{
		tokenHandler: func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_, _ = w.Write([]byte(`["stale-token"]`))
		},
	}))

	res, err := d.Detect(t.Context())
	require.Error(t, err)
	assert.Nil(t, res)
}

func TestDetectInstanceServerError(t *testing.T) {
	d := newTestDetector(newFakeServer(t, serverConfig{
		instanceHandler: func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		},
	}))

	res, err := d.Detect(t.Context())
	require.Error(t, err)
	assert.Nil(t, res)
}

func TestDetectInstanceMalformedJSON(t *testing.T) {
	d := newTestDetector(newFakeServer(t, serverConfig{
		instanceHandler: func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("not json"))
		},
	}))

	res, err := d.Detect(t.Context())
	require.Error(t, err)
	assert.Nil(t, res)
}

func TestDetectPartialFailure(t *testing.T) {
	d := newTestDetector(newFakeServer(t, serverConfig{
		instanceHandler: func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("{}"))
		},
	}))

	res, err := d.Detect(t.Context())
	require.Error(t, err)
	assert.ErrorIs(t, err, resource.ErrPartialResource)
	require.NotNil(t, res)

	set := res.Set()
	for _, k := range []attribute.Key{semconv.CloudProviderKey, semconv.CloudPlatformKey} {
		_, ok := set.Value(k)
		assert.True(t, ok, "%s should still be reported", k)
	}
	for _, k := range []attribute.Key{
		semconv.CloudAccountIDKey,
		semconv.CloudRegionKey,
		semconv.HostIDKey,
		semconv.HostNameKey,
		semconv.HostTypeKey,
		semconv.HostImageIDKey,
		semconv.HostImageNameKey,
	} {
		_, ok := set.Value(k)
		assert.False(t, ok, "%s should not be reported", k)
	}
}

// collectorFixture is the instance document served by the fake metadata service
// in the Akamai e2e test of opentelemetry-collector-contrib.
const collectorFixture = `{
  "id": 12345678,
  "host_uuid": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "label": "test-akamai-instance",
  "region": "us-east",
  "type": "g6-standard-2",
  "tags": ["environment:testing", "team:observability"],
  "specs": {"vcpus": 2, "memory": 4096, "disk": 81920, "transfer": 4000, "gpus": 0},
  "backups": {"enabled": true, "status": "completed"},
  "account_euuid": "ABCD1234-5678-90EF-GHIJ-KLMNOPQRSTUV",
  "image": {"id": "linode/ubuntu22.04", "label": "Ubuntu 22.04 LTS"}
}`

// TestDetectMatchesCollectorDetector pins this port to the resource that
// resourcedetectionprocessor emits for the same instance document, so the two
// implementations cannot drift.
func TestDetectMatchesCollectorDetector(t *testing.T) {
	d := newTestDetector(newFakeServer(t, serverConfig{
		instanceHandler: func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(collectorFixture))
		},
	}))

	res, err := d.Detect(t.Context())
	require.NoError(t, err)

	got := map[string]string{}
	for _, kv := range res.Attributes() {
		got[string(kv.Key)] = kv.Value.AsString()
	}

	// Verbatim from processor/resourcedetectionprocessor/testdata/e2e/akamai/expected.yaml.
	assert.Equal(t, map[string]string{
		"cloud.account.id": "ABCD1234-5678-90EF-GHIJ-KLMNOPQRSTUV",
		"cloud.platform":   "akamai_cloud.compute",
		"cloud.provider":   "akamai_cloud",
		"cloud.region":     "us-east",
		"host.id":          "12345678",
		"host.image.id":    "linode/ubuntu22.04",
		"host.image.name":  "Ubuntu 22.04 LTS",
		"host.name":        "test-akamai-instance",
		"host.type":        "g6-standard-2",
	}, got)
}

func TestDetectWithAttributeFilter(t *testing.T) {
	d := newTestDetector(
		newFakeServer(t, serverConfig{}),
		WithAttributeFilter(attribute.NewDenyKeysFilter(semconv.CloudPlatformKey)),
	)

	res, err := d.Detect(t.Context())
	require.NoError(t, err)

	set := res.Set()
	_, ok := set.Value(semconv.CloudPlatformKey)
	assert.False(t, ok, "cloud.platform should be filtered out")

	v, ok := set.Value(semconv.HostIDKey)
	require.True(t, ok)
	assert.Equal(t, "4242", v.AsString())
	v, ok = set.Value(semconv.CloudProviderKey)
	require.True(t, ok)
	assert.Equal(t, "akamai_cloud", v.AsString())
}

// TestComposition_MergeWithDefault guards against schema URL drift between
// this detector and the SDK. [resource.Merge] reports
// [resource.ErrSchemaURLConflict] and drops the schema URL when the two
// disagree, so this fails as soon as the semconv version here and the one
// behind [resource.Default] diverge.
func TestComposition_MergeWithDefault(t *testing.T) {
	d := newTestDetector(newFakeServer(t, serverConfig{}))

	detected, err := d.Detect(t.Context())
	require.NoError(t, err)

	merged, err := resource.Merge(resource.Default(), detected)
	require.NoError(t, err)
	assert.NotErrorIs(t, err, resource.ErrSchemaURLConflict)
	assert.Equal(t, resource.Default().SchemaURL(), merged.SchemaURL())
}

// TestComposition_WithCoreDetectors asserts this detector composes with
// go.opentelemetry.io/otel/sdk's own built-in host detector.
func TestComposition_WithCoreDetectors(t *testing.T) {
	d := newTestDetector(newFakeServer(t, serverConfig{}))

	res, err := resource.New(t.Context(),
		resource.WithDetectors(d),
		resource.WithHost(),
	)
	require.NoError(t, err)
	assert.Equal(t, resource.Default().SchemaURL(), res.SchemaURL())
}
