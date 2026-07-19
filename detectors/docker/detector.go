// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"context"
	"errors"
	"fmt"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/moby/moby/client"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.42.0"

	"go.opentelemetry.io/contrib/detectors/docker/internal"
)

// Compile-time interface assertion.
var _ resource.Detector = (*ResourceDetector)(nil)

type config struct {
	filter attribute.Filter
}

// Option configures a [ResourceDetector].
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) { f(c) }

// WithAttributeFilter sets a filter that controls which detected attributes are
// included in the returned resource. Only attributes for which filter returns
// true are included. By default all attributes are included.
func WithAttributeFilter(filter attribute.Filter) Option {
	return optionFunc(func(c *config) { c.filter = filter })
}

// ResourceDetector detects resource attributes for Docker containers using the
// local Docker daemon.
type ResourceDetector struct {
	createProvider func(...client.Opt) (provider, error)
	cfg            config
}

// NewResourceDetector returns a [resource.Detector] that detects resource
// attributes for Docker containers using the local Docker daemon.
func NewResourceDetector(opts ...Option) *ResourceDetector {
	var cfg config
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	return &ResourceDetector{createProvider: newProvider, cfg: cfg}
}

// Detect returns a [resource.Resource] containing Docker host and container
// attributes for the running container. It returns an empty resource and no
// error when it cannot confirm this process is running inside a Docker
// container:
//
//   - the Docker daemon is unreachable, or
//   - the daemon is reachable but no container matches this process's hostname.
//
// If the container is identified but some attributes cannot be retrieved, a
// partial resource is returned with [resource.ErrPartialResource].
func (r *ResourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	dockerProvider, err := r.createProvider()
	if err != nil {
		return resource.Empty(), nil
	}
	defer func() { _ = dockerProvider.Close() }()

	// ContainerInfo is the only call tied to this process's own identity (via
	// hostname), so it is the sole signal for whether we're in a container; a
	// reachable daemon alone doesn't mean that (e.g. running on the Docker host).
	containerInfo, err := dockerProvider.ContainerInfo(ctx)
	if err != nil {
		if client.IsErrConnectionFailed(err) || cerrdefs.IsNotFound(err) {
			// Daemon unreachable, or no container matches our hostname
			return resource.Empty(), nil
		}
		return resource.Empty(), fmt.Errorf("docker container info: %w", err)
	}

	var (
		attrs []attribute.KeyValue
		errs  []error
	)

	attrs = append(attrs, semconv.ContainerName(containerInfo.Name))
	// container.image.name and container.image.tags are legitimately absent
	// when the container was referenced by bare image ID (e.g.
	// "docker run sha256:<id>"); container.image.id still identifies the image.
	if containerInfo.ImageName != nil {
		attrs = append(attrs, semconv.ContainerImageName(*containerInfo.ImageName))
	}
	if len(containerInfo.Tags) > 0 {
		attrs = append(attrs, semconv.ContainerImageTags(containerInfo.Tags...))
	}
	if containerInfo.ImageID != "" {
		attrs = append(attrs, semconv.ContainerImageID(containerInfo.ImageID))
	}

	hostInfo, err := dockerProvider.Info(ctx)
	if err != nil {
		errs = append(errs, fmt.Errorf("docker info: %w", err))
	} else {
		attrs = append(attrs, semconv.HostName(hostInfo.Name), semconv.OSTypeKey.String(internal.GOOSToOSType(hostInfo.OSType)))
	}

	if r.cfg.filter != nil {
		filtered := attrs[:0]
		for _, kv := range attrs {
			if r.cfg.filter(kv) {
				filtered = append(filtered, kv)
			}
		}
		attrs = filtered
	}

	res := resource.NewWithAttributes(semconv.SchemaURL, attrs...)

	if len(errs) > 0 {
		return res, fmt.Errorf("%w: %w", resource.ErrPartialResource, errors.Join(errs...))
	}
	return res, nil
}
