// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package docker // import "go.opentelemetry.io/contrib/detectors/docker"

import (
	"context"
	"errors"
	"fmt"

	"github.com/moby/moby/client"
	"go.opentelemetry.io/contrib/detectors/internal"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
)

type resourceDetector struct {
	createProvider func(...client.Opt) (provider, error)
}

// Detect returns a [resource.Resource] containing Docker host and container
// attributes for the running container. It returns an empty resource and no
// error when the Docker daemon is unreachable (not a Docker environment). If
// the daemon is reachable but some attributes cannot be retrieved, a partial
// resource is returned together with [resource.ErrPartialResource].
func (r *resourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	// A provider failure means the Docker daemon is unreachable, which is
	// indistinguishable from not running in a Docker environment. Return an
	// empty resource with no error so autodetect can continue with other detectors.
	dockerProvider, err := r.createProvider()
	if err != nil {
		return resource.Empty(), nil
	}

	containerInfo, err := dockerProvider.ContainerInfo(ctx)
	if err != nil {
		return resource.Empty(), nil
	}

	var (
		attrs []attribute.KeyValue
		errs  []error
	)

	attrs = append(attrs, semconv.ContainerName(containerInfo.Name))
	attrs = append(attrs, semconv.ContainerImageName(containerInfo.Config.Image))

	sysInfo, err := dockerProvider.Info(ctx)
	if err != nil {
		errs = append(errs, fmt.Errorf("docker info: %w", err))
	} else {
		attrs = append(attrs, semconv.HostName(sysInfo.Name))
		attrs = append(attrs, semconv.OSTypeKey.String(internal.GOOSToOSType(sysInfo.OSType)))
	}

	res := resource.NewWithAttributes(semconv.SchemaURL, attrs...)

	if len(errs) > 0 {
		return res, fmt.Errorf("%w: %w", resource.ErrPartialResource, errors.Join(errs...))
	}
	return res, nil
}

// NewResourceDetector returns a [resource.Detector] that detects resource
// attributes for Docker containers using the local Docker daemon.
func NewResourceDetector() resource.Detector {
	return &resourceDetector{createProvider: newProvider}
}
