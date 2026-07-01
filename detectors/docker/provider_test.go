// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/system"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// roundTripFunc lets a func satisfy http.RoundTripper, so tests can fake the
// Docker daemon's HTTP responses without a live daemon or a mock production
// interface.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func jsonResponse(t *testing.T, status int, body any) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewReader(b)),
		Header:     make(http.Header),
	}
}

// newTestProvider builds a dockerProviderImpl backed by a real *client.Client
// whose HTTP transport is faked, so tests exercise the actual moby client
// (URL/version construction, JSON decoding) instead of a hand-rolled mock
// interface. Pinning the API version skips the implicit /_ping
// version-negotiation request the client otherwise makes on first use.
func newTestProvider(t *testing.T, rt roundTripFunc) *dockerProviderImpl {
	t.Helper()
	cli, err := client.New(
		client.WithAPIVersion(client.MaxAPIVersion),
		client.WithHTTPClient(&http.Client{Transport: rt}),
	)
	require.NoError(t, err)
	return &dockerProviderImpl{dockerClient: cli}
}

func TestDockerProviderImpl_Info(t *testing.T) {
	p := newTestProvider(t, func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, "/v"+client.MaxAPIVersion+"/info", req.URL.Path)
		return jsonResponse(t, http.StatusOK, system.Info{Name: "docker-host", OSType: "linux"}), nil
	})

	info, err := p.Info(t.Context())
	require.NoError(t, err)
	assert.Equal(t, hostInfo{Name: "docker-host", OSType: "linux"}, info)
}

func TestDockerProviderImpl_Info_Error(t *testing.T) {
	p := newTestProvider(t, func(req *http.Request) (*http.Response, error) {
		return jsonResponse(t, http.StatusInternalServerError, map[string]string{"message": "boom"}), nil
	})

	_, err := p.Info(t.Context())
	assert.Error(t, err)
}

func TestDockerProviderImpl_ContainerInfo(t *testing.T) {
	hostname, err := os.Hostname()
	require.NoError(t, err)

	p := newTestProvider(t, func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, "/v"+client.MaxAPIVersion+"/containers/"+hostname+"/json", req.URL.Path)
		return jsonResponse(t, http.StatusOK, container.InspectResponse{
			Name:   "my-container",
			Config: &container.Config{Image: "golang:1.25"},
		}), nil
	})

	info, err := p.ContainerInfo(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "my-container", info.Name)
	require.NotNil(t, info.Image)
	assert.Equal(t, "golang:1.25", *info.Image)
}

func TestDockerProviderImpl_ContainerInfo_NilConfig(t *testing.T) {
	p := newTestProvider(t, func(req *http.Request) (*http.Response, error) {
		return jsonResponse(t, http.StatusOK, container.InspectResponse{Name: "my-container"}), nil
	})

	info, err := p.ContainerInfo(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "my-container", info.Name)
	assert.Nil(t, info.Image)
}

func TestDockerProviderImpl_ContainerInfo_Error(t *testing.T) {
	p := newTestProvider(t, func(req *http.Request) (*http.Response, error) {
		return jsonResponse(t, http.StatusNotFound, map[string]string{"message": "no such container"}), nil
	})

	_, err := p.ContainerInfo(t.Context())
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to fetch container information")
}
