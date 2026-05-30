// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"context"
	"errors"
	"testing"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

type mockProvider struct {
	hostname         string
	hostnameErr      error
	osType           string
	osTypeErr        error
	containerName    string
	containerImage   string
	containerInfoErr error
}

func (m *mockProvider) Hostname(_ context.Context) (string, error) {
	return m.hostname, m.hostnameErr
}

func (m *mockProvider) OSType(_ context.Context) (string, error) {
	return m.osType, m.osTypeErr
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
		hostname:       "docker-host",
		osType:         "linux",
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

func TestDockerResourceDetectorOSTypeError(t *testing.T) {
	detector := newMockDetector(&mockProvider{
		osTypeErr:      errors.New("daemon unavailable"),
		hostname:       "docker-host",
		containerName:  "my-container",
		containerImage: "golang:1.25",
	})

	res, err := detector.Detect(context.Background())
	require.ErrorIs(t, err, resource.ErrPartialResource)

	attrs := res.Set()
	_, hasOSType := attrs.Value(semconv.OSTypeKey)
	assert.False(t, hasOSType)

	hostname, _ := attrs.Value(semconv.HostNameKey)
	assert.Equal(t, "docker-host", hostname.AsString())
}

func TestDockerResourceDetectorHostnameError(t *testing.T) {
	detector := newMockDetector(&mockProvider{
		osType:         "linux",
		hostnameErr:    errors.New("daemon unavailable"),
		containerName:  "my-container",
		containerImage: "golang:1.25",
	})

	res, err := detector.Detect(context.Background())
	require.ErrorIs(t, err, resource.ErrPartialResource)

	attrs := res.Set()
	_, hasHostname := attrs.Value(semconv.HostNameKey)
	assert.False(t, hasHostname)

	osType, _ := attrs.Value(semconv.OSTypeKey)
	assert.Equal(t, "linux", osType.AsString())
}

func TestDockerResourceDetectorContainerInfoError(t *testing.T) {
	detector := newMockDetector(&mockProvider{
		osType:           "linux",
		hostname:         "docker-host",
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
