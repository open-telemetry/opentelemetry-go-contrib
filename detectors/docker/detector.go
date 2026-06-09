// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package docker // import "go.opentelemetry.io/contrib/detectors/docker"

import (
	"context"
	"errors"
	"fmt"

	"github.com/moby/moby/client"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"

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
// error when the Docker daemon is unreachable (not a Docker environment). If
// the daemon is reachable but some attributes cannot be retrieved, a partial
// resource is returned together with [resource.ErrPartialResource].
func (r *ResourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
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

	attrs = append(attrs, semconv.ContainerName(containerInfo.Name), semconv.ContainerImageName(containerInfo.Config.Image))

	sysInfo, err := dockerProvider.Info(ctx)
	if err != nil {
		errs = append(errs, fmt.Errorf("docker info: %w", err))
	} else {
		attrs = append(attrs, semconv.HostName(sysInfo.Name), semconv.OSTypeKey.String(internal.GOOSToOSType(sysInfo.OSType)))
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
