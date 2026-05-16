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
	hostname       string
	osType         string
	containerName  string
	containerImage string
}

func (m *mockProvider) Hostname(_ context.Context) (string, error) {
	return m.hostname, nil
}

func (m *mockProvider) OSType(_ context.Context) (string, error) {
	return m.osType, nil
}

func (m *mockProvider) ContainerInfo(_ context.Context) (container.InspectResponse, error) {
	resp := container.InspectResponse{}
	resp.Name = m.containerName
	resp.Image = m.containerImage

	return resp, nil
}

func TestDockerResourceDetectorProviderError(t *testing.T) {
	detector := &resourceDetector{
		newProvider: func(...client.Opt) (provider, error) {
			return nil, errors.New("cannot connect to Docker daemon")
		},
	}

	res, err := detector.Detect(context.Background())
	require.Error(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDockerResourceDetectorSuccess(t *testing.T) {
	detector := &resourceDetector{
		newProvider: func(...client.Opt) (provider, error) {
			return &mockProvider{
				hostname:       "docker-host",
				osType:         "linux",
				containerName:  "/my-container",
				containerImage: "golang:1.25",
			}, nil
		},
	}

	res, err := detector.Detect(context.Background())
	require.NoError(t, err)

	attrs := res.Set()
	hostname, _ := attrs.Value(semconv.HostNameKey)
	assert.Equal(t, "docker-host", hostname.AsString())

	osType, _ := attrs.Value(semconv.OSTypeKey)
	assert.Equal(t, "linux", osType.AsString())

	containerName, _ := attrs.Value(semconv.ContainerNameKey)
	assert.Equal(t, "/my-container", containerName.AsString())

	containerImage, _ := attrs.Value(semconv.ContainerImageNameKey)
	assert.Equal(t, "golang:1.25", containerImage.AsString())
}

func TestDockerResourceDetectorZos(t *testing.T) {
	detector := &resourceDetector{
		newProvider: func(...client.Opt) (provider, error) {
			return &mockProvider{
				hostname:       "docker-host",
				osType:         "linux",
				containerName:  "/my-container",
				containerImage: "golang:1.25",
			}, nil
		},
	}

	res, err := detector.Detect(context.Background())
	require.NoError(t, err)

	attrs := res.Set()
	hostname, _ := attrs.Value(semconv.HostNameKey)
	assert.Equal(t, "docker-host", hostname.AsString())

	osType, _ := attrs.Value(semconv.OSTypeKey)
	assert.Equal(t, "linux", osType.AsString())
	assert.Equal(t, semconv.OSTypeLinux.Value.AsString(), osType.AsString())

	containerName, _ := attrs.Value(semconv.ContainerNameKey)
	assert.Equal(t, "/my-container", containerName.AsString())

	containerImage, _ := attrs.Value(semconv.ContainerImageNameKey)
	assert.Equal(t, "golang:1.25", containerImage.AsString())
}
