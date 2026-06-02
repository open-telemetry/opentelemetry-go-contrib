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
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
)

type mockProvider struct {
	info             system.Info
	infoErr          error
	containerName    string
	containerImage   string
	containerInfoErr error
}

func (m *mockProvider) Info(_ context.Context) (system.Info, error) {
	return m.info, m.infoErr
}

func (m *mockProvider) ContainerInfo(_ context.Context) (container.InspectResponse, error) {
	if m.containerInfoErr != nil {
		return container.InspectResponse{}, m.containerInfoErr
	}
	return container.InspectResponse{
		Name:   m.containerName,
		Config: &container.Config{Image: m.containerImage},
	}, nil
}

func newMockDetector(m *mockProvider) *resourceDetector {
	return &resourceDetector{
		createProvider: func(...client.Opt) (provider, error) { return m, nil },
	}
}

func TestDockerResourceDetectorNotDockerEnvironment(t *testing.T) {
	detector := &resourceDetector{
		createProvider: func(...client.Opt) (provider, error) {
			return nil, errors.New("cannot connect to Docker daemon")
		},
	}

	res, err := detector.Detect(context.Background())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDockerResourceDetectorSuccess(t *testing.T) {
	detector := newMockDetector(&mockProvider{
		info:           system.Info{Name: "docker-host", OSType: "linux"},
		containerName:  "my-container",
		containerImage: "golang:1.25",
	})

	res, err := detector.Detect(context.Background())
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

func TestDockerResourceDetectorInfoError(t *testing.T) {
	detector := newMockDetector(&mockProvider{
		infoErr:        errors.New("daemon unavailable"),
		containerName:  "my-container",
		containerImage: "golang:1.25",
	})

	res, err := detector.Detect(context.Background())
	require.ErrorIs(t, err, resource.ErrPartialResource)

	attrs := res.Set()
	_, hasHostname := attrs.Value(semconv.HostNameKey)
	assert.False(t, hasHostname)

	_, hasOSType := attrs.Value(semconv.OSTypeKey)
	assert.False(t, hasOSType)

	containerName, _ := attrs.Value(semconv.ContainerNameKey)
	assert.Equal(t, "my-container", containerName.AsString())
}

func TestDockerResourceDetectorContainerInfoError(t *testing.T) {
	detector := newMockDetector(&mockProvider{
		info:             system.Info{Name: "docker-host", OSType: "linux"},
		containerInfoErr: errors.New("no such container"),
	})

	res, err := detector.Detect(context.Background())
	require.ErrorIs(t, err, resource.ErrPartialResource)

	attrs := res.Set()
	_, hasContainerName := attrs.Value(semconv.ContainerNameKey)
	assert.False(t, hasContainerName)

	osType, _ := attrs.Value(semconv.OSTypeKey)
	assert.Equal(t, "linux", osType.AsString())

	hostname, _ := attrs.Value(semconv.HostNameKey)
	assert.Equal(t, "docker-host", hostname.AsString())
}
