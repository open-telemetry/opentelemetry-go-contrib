// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/moby/moby/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.42.0"
)

type mockProvider struct {
	info             hostInfo
	infoErr          error
	containerInfo    containerInfo
	containerInfoErr error
	closeCalls       int
}

func (m *mockProvider) Info(_ context.Context) (hostInfo, error) {
	return m.info, m.infoErr
}

func (m *mockProvider) ContainerInfo(_ context.Context) (containerInfo, error) {
	return m.containerInfo, m.containerInfoErr
}

func (m *mockProvider) Close() error {
	m.closeCalls++
	return nil
}

func ptr[T any](v T) *T { return &v }

func newMockDetector(m *mockProvider, opts ...Option) *ResourceDetector {
	d := NewResourceDetector(opts...)
	d.createProvider = func(...client.Opt) (provider, error) { return m, nil }
	return d
}

func TestNotDockerEnvironment(t *testing.T) {
	detector := &ResourceDetector{
		createProvider: func(...client.Opt) (provider, error) {
			return nil, errors.New("cannot connect to Docker daemon")
		},
	}

	res, err := detector.Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestSuccess(t *testing.T) {
	mock := &mockProvider{
		info: hostInfo{Name: "docker-host", OSType: "linux"},
		containerInfo: containerInfo{
			Name:      "my-container",
			ImageName: ptr("golang"),
			ImageID:   "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			Tags:      []string{"1.25"},
		},
	}
	detector := newMockDetector(mock)

	res, err := detector.Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 1, mock.closeCalls, "expected provider to be closed exactly once")

	attrs := res.Set()
	hostname, _ := attrs.Value(semconv.HostNameKey)
	assert.Equal(t, "docker-host", hostname.AsString())

	osType, _ := attrs.Value(semconv.OSTypeKey)
	assert.Equal(t, "linux", osType.AsString())

	containerName, _ := attrs.Value(semconv.ContainerNameKey)
	assert.Equal(t, "my-container", containerName.AsString())

	containerImage, _ := attrs.Value(semconv.ContainerImageNameKey)
	assert.Equal(t, "golang", containerImage.AsString())

	containerImageTags, _ := attrs.Value(semconv.ContainerImageTagsKey)
	assert.Equal(t, []string{"1.25"}, containerImageTags.AsStringSlice())

	containerImageID, _ := attrs.Value(semconv.ContainerImageIDKey)
	assert.Equal(t, "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", containerImageID.AsString())
}

func TestInfoError(t *testing.T) {
	detector := newMockDetector(&mockProvider{
		infoErr: errors.New("daemon unavailable"),
		containerInfo: containerInfo{
			Name:      "my-container",
			ImageName: ptr("golang"),
			ImageID:   "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			Tags:      []string{"1.25"},
		},
	})

	res, err := detector.Detect(t.Context())
	require.ErrorIs(t, err, resource.ErrPartialResource)

	attrs := res.Set()
	_, hasHostname := attrs.Value(semconv.HostNameKey)
	assert.False(t, hasHostname)

	_, hasOSType := attrs.Value(semconv.OSTypeKey)
	assert.False(t, hasOSType)

	containerName, _ := attrs.Value(semconv.ContainerNameKey)
	assert.Equal(t, "my-container", containerName.AsString())

	containerImage, _ := attrs.Value(semconv.ContainerImageNameKey)
	assert.Equal(t, "golang", containerImage.AsString())

	containerImageTags, _ := attrs.Value(semconv.ContainerImageTagsKey)
	assert.Equal(t, []string{"1.25"}, containerImageTags.AsStringSlice())
}

func TestSuccess_BareImageID(t *testing.T) {
	mock := &mockProvider{
		info: hostInfo{Name: "docker-host", OSType: "linux"},
		containerInfo: containerInfo{
			Name:    "my-container",
			ImageID: "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
	}
	detector := newMockDetector(mock)

	res, err := detector.Detect(t.Context())
	require.NoError(t, err, "a missing image name/tags from a bare-ID reference must not be a partial-resource error")

	attrs := res.Set()
	_, hasImageName := attrs.Value(semconv.ContainerImageNameKey)
	assert.False(t, hasImageName)

	_, hasTags := attrs.Value(semconv.ContainerImageTagsKey)
	assert.False(t, hasTags)

	containerImageID, _ := attrs.Value(semconv.ContainerImageIDKey)
	assert.Equal(t, "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", containerImageID.AsString())
}

func TestContainerInfoError(t *testing.T) {
	mock := &mockProvider{
		info:             hostInfo{Name: "docker-host", OSType: "linux"},
		containerInfoErr: errors.New("no such container"),
	}
	detector := newMockDetector(mock)

	res, err := detector.Detect(t.Context())
	require.Error(t, err)
	assert.NotErrorIs(t, err, resource.ErrPartialResource)
	assert.ErrorContains(t, err, "no such container")
	assert.Equal(t, resource.Empty(), res)
	assert.Equal(t, 1, mock.closeCalls, "expected provider to be closed exactly once")
}

// TestContainerInfoConnectionFailed and TestContainerInfoNotFound use a real
// dockerProviderImpl over a faked transport, rather than mockProvider, so
// that ContainerInfo returns genuine client.IsErrConnectionFailed /
// cerrdefs.IsNotFound errors instead of an arbitrary error string.

func TestContainerInfoConnectionFailed(t *testing.T) {
	p := newTestProvider(t, func(_ *http.Request) (*http.Response, error) {
		return nil, errors.New("connection refused")
	})
	detector := NewResourceDetector()
	detector.createProvider = func(...client.Opt) (provider, error) { return p, nil }

	res, err := detector.Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestContainerInfoNotFound(t *testing.T) {
	p := newTestProvider(t, func(_ *http.Request) (*http.Response, error) {
		return jsonResponse(t, http.StatusNotFound, map[string]string{"message": "no such container"}), nil
	})
	detector := NewResourceDetector()
	detector.createProvider = func(...client.Opt) (provider, error) { return p, nil }

	res, err := detector.Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestWithAttributeFilter(t *testing.T) {
	detector := newMockDetector(
		&mockProvider{
			info: hostInfo{Name: "docker-host", OSType: "linux"},
			containerInfo: containerInfo{
				Name:      "my-container",
				ImageName: ptr("golang"),
				ImageID:   "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				Tags:      []string{"1.25"},
			},
		},
		WithAttributeFilter(attribute.NewDenyKeysFilter(semconv.HostNameKey)),
	)

	res, err := detector.Detect(t.Context())
	require.NoError(t, err)

	// host.name must be absent.
	_, ok := res.Set().Value(semconv.HostNameKey)
	assert.False(t, ok, "expected host.name to be absent")

	// The other five attributes must be present.
	presentAttrs := []attribute.KeyValue{
		semconv.OSTypeKey.String("linux"),
		semconv.ContainerName("my-container"),
		semconv.ContainerImageName("golang"),
		semconv.ContainerImageTags("1.25"),
		semconv.ContainerImageID("sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"),
	}
	for _, kv := range presentAttrs {
		val, ok := res.Set().Value(kv.Key)
		assert.True(t, ok, "expected attribute %s to be present", kv.Key)
		assert.Equal(t, kv.Value, val)
	}
}
