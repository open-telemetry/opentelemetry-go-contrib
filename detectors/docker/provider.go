// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package docker // import "go.opentelemetry.io/contrib/detectors/docker"

import (
	"context"
	"fmt"
	"os"

	"github.com/moby/moby/client"
)

type hostInfo struct {
	Name   string
	OSType string
}

type containerInfo struct {
	Name  string
	Image *string // for nil check if absent
}

type provider interface {
	Info(context.Context) (hostInfo, error)
	ContainerInfo(context.Context) (containerInfo, error)
}

type dockerProviderImpl struct {
	dockerClient *client.Client
}

func (d *dockerProviderImpl) Info(ctx context.Context) (hostInfo, error) {
	result, err := d.dockerClient.Info(ctx, client.InfoOptions{})
	if err != nil {
		return hostInfo{}, err
	}
	return hostInfo{Name: result.Info.Name, OSType: result.Info.OSType}, nil
}

func (d *dockerProviderImpl) ContainerInfo(ctx context.Context) (containerInfo, error) {
	// Docker sets the container hostname to the first 12 characters of the
	// container ID by default, which ContainerInspect can resolve. This breaks
	// when a custom hostname is set via --hostname or an orchestrator (e.g.
	// Kubernetes pod.spec.hostname), in which case ContainerInspect returns
	// "no such container" and container attributes will not be detected.
	hostname, err := os.Hostname()
	if err != nil {
		return containerInfo{}, err
	}
	result, err := d.dockerClient.ContainerInspect(ctx, hostname, client.ContainerInspectOptions{})
	if err != nil {
		return containerInfo{}, fmt.Errorf("failed to fetch container information: %w", err)
	}

	var image *string
	if result.Container.Config != nil {
		image = &result.Container.Config.Image
	}
	return containerInfo{Name: result.Container.Name, Image: image}, nil
}

func newProvider(opts ...client.Opt) (provider, error) {
	opts = append([]client.Opt{client.FromEnv}, opts...)
	cli, err := client.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("could not initialize Docker client: %w", err)
	}
	return &dockerProviderImpl{dockerClient: cli}, nil
}
