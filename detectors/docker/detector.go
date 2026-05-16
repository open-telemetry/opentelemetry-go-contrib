// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package docker // import "go.opentelemetry.io/contrib/detectors/docker"

import (
	"context"
	"fmt"

	"github.com/moby/moby/client"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

type resourceDetector struct {
	newProvider func(...client.Opt) (provider, error)
}

// Detect implements [resource.Detector].
func (r *resourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	dockerProvider, err := r.newProvider()
	if err != nil {
		return resource.Empty(), err
	}

	osType, err := dockerProvider.OSType(ctx)
	if err != nil {
		return resource.Empty(), fmt.Errorf("failed to fetch Docker OS type: %w", err)
	}

	hostname, err := dockerProvider.Hostname(ctx)
	if err != nil {
		return resource.Empty(), fmt.Errorf("failed getting OS hostname: %w", err)
	}

	containerInfo, err := dockerProvider.ContainerInfo(ctx)
	if err != nil {
		return resource.Empty(), fmt.Errorf("failed getting container info: %w", err)
	}

	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.HostName(hostname),
		semconv.OSTypeKey.String(osType),
		semconv.ContainerName(containerInfo.Name),
		semconv.ContainerImageName(containerInfo.Image),
	), nil
}

func NewResourceDetector() resource.Detector {
	return &resourceDetector{newProvider: newProvider}
}
