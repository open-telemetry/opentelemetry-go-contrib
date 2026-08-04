// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consul

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

// selfResponse is the shape of the /v1/agent/self response the detector reads.
type selfResponse map[string]map[string]any

// agentSelf returns a self response with the agent configuration the tests use.
func agentSelf() selfResponse {
	return selfResponse{
		"Config": {
			"NodeName":   "node-1",
			"Datacenter": "dc1",
			"NodeID":     "00000000-0000-0000-0000-000000000000",
		},
	}
}

// newFakeAgent starts an httptest server answering /v1/agent/self with self and
// returns its URL. The server is closed via t.Cleanup.
func newFakeAgent(t *testing.T, self selfResponse) string {
	t.Helper()
	return newRawFakeAgent(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/agent/self" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(self)
	})
}

// newRawFakeAgent starts an httptest server using h and returns its URL. The
// server is closed via t.Cleanup.
func newRawFakeAgent(t *testing.T, h http.HandlerFunc) string {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv.URL
}

// newTestDetector returns a detector pointed at the agent listening on addr.
// The client is built by Detect so that the context deadline applies.
func newTestDetector(t *testing.T, addr string, opts ...Option) *ResourceDetector {
	t.Helper()
	return NewResourceDetector(append([]Option{WithAddress(addr)}, opts...)...)
}

func TestNewResourceDetector(t *testing.T) {
	d := NewResourceDetector()
	require.NotNil(t, d)
	assert.Nil(t, d.cfg.client)
	assert.Nil(t, d.cfg.metaKeyFilter)
}

func TestDetect_OK(t *testing.T) {
	addr := newFakeAgent(t, agentSelf())

	res, err := newTestDetector(t, addr).Detect(t.Context())
	require.NoError(t, err)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.HostName("node-1"),
		semconv.CloudRegion("dc1"),
		semconv.HostID("00000000-0000-0000-0000-000000000000"),
	)
	assert.Equal(t, expected, res)
}

func TestDetect_NoMetaWithoutFilter(t *testing.T) {
	self := agentSelf()
	self["Meta"] = map[string]any{"rack": "r1"}

	res, err := newTestDetector(t, newFakeAgent(t, self)).Detect(t.Context())
	require.NoError(t, err)

	for _, k := range []attribute.Key{"rack", metaPrefix + "rack"} {
		_, ok := res.Set().Value(k)
		assert.False(t, ok, "expected %s to be absent without WithMetaKeyFilter", k)
	}
}

func TestDetect_MetaFiltered(t *testing.T) {
	self := agentSelf()
	self["Meta"] = map[string]any{"rack": "r1", "environment": "prod"}

	// The filter receives the raw Consul meta key, without the prefix.
	filter := func(key string) bool { return key == "rack" }
	res, err := newTestDetector(t, newFakeAgent(t, self), WithMetaKeyFilter(filter)).Detect(t.Context())
	require.NoError(t, err)

	val, ok := res.Set().Value(metaPrefix + "rack")
	require.True(t, ok, "expected consul.meta.rack to be present")
	assert.Equal(t, attribute.StringValue("r1"), val)

	_, ok = res.Set().Value("rack")
	assert.False(t, ok, "expected the unprefixed rack to be absent")

	_, ok = res.Set().Value(metaPrefix + "environment")
	assert.False(t, ok, "expected consul.meta.environment to be absent")
}

func TestDetect_MetaDoesNotOverrideDetectedAttribute(t *testing.T) {
	// Node meta keys are namespaced, so they cannot collide with a detected
	// attribute. Consul itself rejects a meta key like this one; it is used
	// here only to exercise the collision the prefix rules out.
	self := agentSelf()
	self["Meta"] = map[string]any{"host.name": "from-meta"}

	filter := func(string) bool { return true }
	res, err := newTestDetector(t, newFakeAgent(t, self), WithMetaKeyFilter(filter)).Detect(t.Context())
	require.NoError(t, err)

	val, ok := res.Set().Value(semconv.HostNameKey)
	require.True(t, ok)
	assert.Equal(t, attribute.StringValue("node-1"), val)

	val, ok = res.Set().Value(metaPrefix + "host.name")
	require.True(t, ok, "expected consul.meta.host.name to be present")
	assert.Equal(t, attribute.StringValue("from-meta"), val)
}

func TestDetect_NonStringMetaValue(t *testing.T) {
	// A non-string meta value must be skipped, not panic.
	self := agentSelf()
	self["Meta"] = map[string]any{"rack": 42, "zone": "z1"}

	filter := func(string) bool { return true }
	res, err := newTestDetector(t, newFakeAgent(t, self), WithMetaKeyFilter(filter)).Detect(t.Context())
	require.NoError(t, err)

	_, ok := res.Set().Value(metaPrefix + "rack")
	assert.False(t, ok, "expected non-string rack to be absent")

	val, ok := res.Set().Value(metaPrefix + "zone")
	require.True(t, ok)
	assert.Equal(t, attribute.StringValue("z1"), val)
}

func TestDetect_ConnectionRefused(t *testing.T) {
	// Closed server → connection refused → empty resource, no error.
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	addr := srv.URL
	srv.Close()

	res, err := newTestDetector(t, addr).Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestIsDialFailure(t *testing.T) {
	// Errors are constructed, not provoked, so the test does not depend on DNS.
	dial := func(err error) error {
		return &url.Error{
			Op:  "Get",
			URL: "http://consul.example:8500/v1/agent/self",
			Err: &net.OpError{Op: "dial", Net: "tcp", Err: err},
		}
	}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"connection refused", dial(errors.New("connect: connection refused")), true},
		{"no such host", dial(&net.DNSError{Err: "no such host", IsNotFound: true}), true},
		{"read timeout", &url.Error{Op: "Get", Err: &net.OpError{Op: "read"}}, false},
		{"status error", api.StatusError{Code: 500, Body: "boom"}, false},
		{"context canceled", context.Canceled, false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isDialFailure(tt.err))
		})
	}
}

func TestDetect_ServerError(t *testing.T) {
	addr := newRawFakeAgent(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})

	res, err := newTestDetector(t, addr).Detect(t.Context())
	require.Error(t, err)
	assert.Nil(t, res)
}

func TestDetect_MalformedJSON(t *testing.T) {
	addr := newRawFakeAgent(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not json"))
	})

	res, err := newTestDetector(t, addr).Detect(t.Context())
	require.Error(t, err)
	assert.Nil(t, res)
}

func TestDetect_MissingConfigSection(t *testing.T) {
	// A null Config is served raw: selfResponse cannot encode a null value.
	tests := map[string]string{
		"absent": `{}`,
		"null":   `{"Config": null}`,
	}

	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			addr := newRawFakeAgent(t, func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(body))
			})

			res, err := newTestDetector(t, addr).Detect(t.Context())
			require.Error(t, err)
			assert.NotErrorIs(t, err, resource.ErrPartialResource)
			assert.Nil(t, res)
		})
	}
}

func TestDetect_PartialFailure(t *testing.T) {
	// Datacenter and NodeID absent, NodeName present.
	self := selfResponse{"Config": {"NodeName": "node-1"}}

	res, err := newTestDetector(t, newFakeAgent(t, self)).Detect(t.Context())
	require.Error(t, err)
	assert.ErrorIs(t, err, resource.ErrPartialResource)

	val, ok := res.Set().Value(semconv.HostNameKey)
	require.True(t, ok, "expected host.name to be present")
	assert.Equal(t, attribute.StringValue("node-1"), val)

	for _, k := range []attribute.Key{semconv.CloudRegionKey, semconv.HostIDKey} {
		_, ok := res.Set().Value(k)
		assert.False(t, ok, "expected %s to be absent", k)
	}
}

func TestDetect_NonStringConfigValue(t *testing.T) {
	self := selfResponse{"Config": {
		"NodeName":   "node-1",
		"Datacenter": 1,
		"NodeID":     "00000000-0000-0000-0000-000000000000",
	}}

	res, err := newTestDetector(t, newFakeAgent(t, self)).Detect(t.Context())
	require.Error(t, err)
	assert.ErrorIs(t, err, resource.ErrPartialResource)

	_, ok := res.Set().Value(semconv.CloudRegionKey)
	assert.False(t, ok, "expected cloud.region to be absent")
}

func TestDetect_ContextCanceled(t *testing.T) {
	addr := newRawFakeAgent(t, func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	})

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	res, err := newTestDetector(t, addr).Detect(ctx)
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, res)
}

func TestDetect_ContextDeadline(t *testing.T) {
	// The caller's deadline bounds the request: there is no separate timeout.
	released := make(chan struct{})
	t.Cleanup(func() { close(released) })
	addr := newRawFakeAgent(t, func(_ http.ResponseWriter, r *http.Request) {
		select {
		case <-released:
		case <-r.Context().Done():
		}
	})

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
	t.Cleanup(cancel)

	res, err := newTestDetector(t, addr).Detect(ctx)
	require.Error(t, err)
	assert.Nil(t, res)
}

func TestDetect_WithAttributeFilter(t *testing.T) {
	filter := attribute.NewDenyKeysFilter(semconv.HostIDKey)
	res, err := newTestDetector(t, newFakeAgent(t, agentSelf()), WithAttributeFilter(filter)).Detect(t.Context())
	require.NoError(t, err)

	_, ok := res.Set().Value(semconv.HostIDKey)
	assert.False(t, ok, "expected host.id to be absent")

	for _, kv := range []attribute.KeyValue{
		semconv.HostName("node-1"),
		semconv.CloudRegion("dc1"),
	} {
		val, ok := res.Set().Value(kv.Key)
		assert.True(t, ok, "expected %s to be present", kv.Key)
		assert.Equal(t, kv.Value, val)
	}
}

func TestDetect_WithClientIgnoresConnectionOptions(t *testing.T) {
	cfg := api.DefaultConfig()
	cfg.Address = newFakeAgent(t, agentSelf())
	client, err := api.NewClient(cfg)
	require.NoError(t, err)

	// WithAddress points at a closed server; WithClient must win.
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	dead := srv.URL
	srv.Close()

	d := NewResourceDetector(WithAddress(dead), WithClient(client))
	res, err := d.Detect(t.Context())
	require.NoError(t, err)

	val, ok := res.Set().Value(semconv.HostNameKey)
	require.True(t, ok)
	assert.Equal(t, attribute.StringValue("node-1"), val)
}

func TestAPIConfig_TokenFile(t *testing.T) {
	tokenFile := filepath.Join(t.TempDir(), "token")
	require.NoError(t, os.WriteFile(tokenFile, []byte("secret-token-value"), 0o600))

	cfg := NewResourceDetector(WithTokenFile(tokenFile)).apiConfig()

	// TokenFile must be set on api.Config.TokenFile so the consul client reads
	// the file. It must NOT be set on api.Config.Token, which would use the
	// path as a literal token.
	assert.Equal(t, tokenFile, cfg.TokenFile)
	assert.Empty(t, cfg.Token)
}

func TestAPIConfig_Token(t *testing.T) {
	cfg := NewResourceDetector(WithToken("my-secret-token")).apiConfig()

	assert.Equal(t, "my-secret-token", cfg.Token)
	assert.Empty(t, cfg.TokenFile)
}

func TestAPIConfig_AllFields(t *testing.T) {
	cfg := NewResourceDetector(
		WithAddress("http://consul.local:8500"),
		WithDatacenter("dc2"),
		WithNamespace("team-a"),
		WithToken("direct-token"),
	).apiConfig()

	assert.Equal(t, "http://consul.local:8500", cfg.Address)
	assert.Equal(t, "dc2", cfg.Datacenter)
	assert.Equal(t, "team-a", cfg.Namespace)
	assert.Equal(t, "direct-token", cfg.Token)
}

func TestAPIConfig_Defaults(t *testing.T) {
	// Without options the consul client defaults must be preserved, which is
	// what resolves the CONSUL_* environment variables.
	cfg := NewResourceDetector().apiConfig()

	defaultCfg := api.DefaultConfig()
	assert.Equal(t, defaultCfg.Address, cfg.Address)
	assert.Equal(t, defaultCfg.Datacenter, cfg.Datacenter)
	assert.Equal(t, defaultCfg.Namespace, cfg.Namespace)
	assert.Equal(t, defaultCfg.Token, cfg.Token)
	assert.Equal(t, defaultCfg.TokenFile, cfg.TokenFile)
}
