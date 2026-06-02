// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package docker // import "go.opentelemetry.io/contrib/detectors/docker"

import (
	"context"
	"fmt"
	"os"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/system"
	"github.com/moby/moby/client"
)

type provider interface {
	Info(context.Context) (system.Info, error)
	ContainerInfo(context.Context) (container.InspectResponse, error)
}

type dockerProviderImpl struct {
	dockerClient *client.Client
}

func (d *dockerProviderImpl) Info(ctx context.Context) (system.Info, error) {
	result, err := d.dockerClient.Info(ctx, client.InfoOptions{})
	if err != nil {
		return system.Info{}, err
	}
	return result.Info, nil
}

func (d *dockerProviderImpl) ContainerInfo(ctx context.Context) (container.InspectResponse, error) {
	// Docker sets the container hostname to the first 12 characters of the
	// container ID by default, which ContainerInspect can resolve. This breaks
	// when a custom hostname is set via --hostname or an orchestrator (e.g.
	// Kubernetes pod.spec.hostname), in which case ContainerInspect returns
	// "no such container" and container attributes will not be detected.
	hostname, err := os.Hostname()
	if err != nil {
		return container.InspectResponse{}, err
	}
	containerInspectResult, err := d.dockerClient.ContainerInspect(ctx, hostname, client.ContainerInspectOptions{})
	if err != nil {
		return container.InspectResponse{}, fmt.Errorf("failed to fetch container information: %w", err)
	}
	return containerInspectResult.Container, nil
}

func newProvider(opts ...client.Opt) (provider, error) {
	opts = append([]client.Opt{client.FromEnv}, opts...)
	cli, err := client.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("could not initialize Docker client: %w", err)
	}
	return &dockerProviderImpl{dockerClient: cli}, nil
}
