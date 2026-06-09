// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"context"
	"errors"
	"testing"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/system"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
)

type mockProvider struct {
	info             system.Info
	infoErr          error
	containerInfo    container.InspectResponse
	containerInfoErr error
}

func (m *mockProvider) Info(_ context.Context) (system.Info, error) {
	return m.info, m.infoErr
}

func (m *mockProvider) ContainerInfo(_ context.Context) (container.InspectResponse, error) {
	return m.containerInfo, m.containerInfoErr
}

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
	detector := newMockDetector(&mockProvider{
		info:          system.Info{Name: "docker-host", OSType: "linux"},
		containerInfo: container.InspectResponse{Name: "my-container", Config: &container.Config{Image: "golang:1.25"}},
	})

	res, err := detector.Detect(t.Context())
	require.NoError(t, err)

	attrs := res.Set()
	hostname, _ := attrs.Value(semconv.HostNameKey)
	assert.Equal(t, "docker-host", hostname.AsString())

	osType, _ := attrs.Value(semconv.OSTypeKey)
	assert.Equal(t, "linux", osType.AsString())

	containerName, _ := attrs.Value(semconv.ContainerNameKey)
	assert.Equal(t, "my-container", containerName.AsString())

	containerImage, _ := attrs.Value(semconv.ContainerImageNameKey)
	assert.Equal(t, "golang:1.25", containerImage.AsString())
}

func TestInfoError(t *testing.T) {
	detector := newMockDetector(&mockProvider{
		infoErr:       errors.New("daemon unavailable"),
		containerInfo: container.InspectResponse{Name: "my-container", Config: &container.Config{Image: "golang:1.25"}},
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
	assert.Equal(t, "golang:1.25", containerImage.AsString())
}

func TestContainerInfoError(t *testing.T) {
	detector := newMockDetector(&mockProvider{
		info:             system.Info{Name: "docker-host", OSType: "linux"},
		containerInfoErr: errors.New("no such container"),
	})

	res, err := detector.Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestWithAttributeFilter(t *testing.T) {
	detector := newMockDetector(
		&mockProvider{
			info:          system.Info{Name: "docker-host", OSType: "linux"},
			containerInfo: container.InspectResponse{Name: "my-container", Config: &container.Config{Image: "golang:1.25"}},
		},
		WithAttributeFilter(attribute.NewDenyKeysFilter(semconv.HostNameKey)),
	)

	res, err := detector.Detect(t.Context())
	require.NoError(t, err)

	// host.name must be absent.
	_, ok := res.Set().Value(semconv.HostNameKey)
	assert.False(t, ok, "expected host.name to be absent")

	// The other three attributes must be present.
	presentAttrs := []attribute.KeyValue{
		semconv.OSTypeKey.String("linux"),
		semconv.ContainerName("my-container"),
		semconv.ContainerImageName("golang:1.25"),
	}
	for _, kv := range presentAttrs {
		val, ok := res.Set().Value(kv.Key)
		assert.True(t, ok, "expected attribute %s to be present", kv.Key)
		assert.Equal(t, kv.Value, val)
	}
}
