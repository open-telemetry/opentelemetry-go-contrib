// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package hetzner

import (
	"context"
	"fmt"
	"strconv"

	hcloudmeta "github.com/hetznercloud/hcloud-go/v2/hcloud/metadata"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.42.0"
)

// Compile-time interface assertion.
var _ resource.Detector = (*ResourceDetector)(nil)

// newHcloudClient is the factory for the hcloud metadata client.
// It is a package-level variable so tests can substitute a fake server.
var newHcloudClient = func() *hcloudmeta.Client {
	return hcloudmeta.NewClient()
}

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

// ResourceDetector collects resource information of Hetzner Cloud servers.
type ResourceDetector struct {
	client *hcloudmeta.Client
	cfg    config
}

// NewResourceDetector returns a [resource.Detector] that detects resource
// attributes on Hetzner Cloud servers.
func NewResourceDetector(opts ...Option) *ResourceDetector {
	var cfg config
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	return &ResourceDetector{client: newHcloudClient(), cfg: cfg}
}

// Detect detects resource attributes of the Hetzner Cloud server the process
// is running on. It returns an empty resource and no error when not running on
// a Hetzner Cloud server. If the process is running on a Hetzner Cloud server
// but some attributes cannot be retrieved, a partial resource is returned
// together with [resource.ErrPartialResource].
func (d *ResourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	if !d.client.IsHcloudServerWithContext(ctx) {
		return resource.Empty(), nil
	}

	attrs := []attribute.KeyValue{
		semconv.CloudProviderHetzner,
		semconv.CloudPlatformHetznerCloudServer,
	}

	var errs []error

	id, err := d.client.InstanceIDWithContext(ctx)
	if err != nil {
		errs = append(errs, fmt.Errorf("instance ID: %w", err))
	} else {
		attrs = append(attrs, semconv.HostID(strconv.FormatInt(id, 10)))
	}

	hostname, err := d.client.HostnameWithContext(ctx)
	if err != nil {
		errs = append(errs, fmt.Errorf("hostname: %w", err))
	} else {
		attrs = append(attrs, semconv.HostName(hostname))
	}

	region, err := d.client.RegionWithContext(ctx)
	if err != nil {
		errs = append(errs, fmt.Errorf("region: %w", err))
	} else {
		attrs = append(attrs, semconv.CloudRegion(region))
	}

	az, err := d.client.AvailabilityZoneWithContext(ctx)
	if err != nil {
		errs = append(errs, fmt.Errorf("availability zone: %w", err))
	} else {
		attrs = append(attrs, semconv.CloudAvailabilityZone(az))
	}

	if d.cfg.filter != nil {
		filtered := attrs[:0]
		for _, kv := range attrs {
			if d.cfg.filter(kv) {
				filtered = append(filtered, kv)
			}
		}
		attrs = filtered
	}

	res := resource.NewWithAttributes(semconv.SchemaURL, attrs...)

	if len(errs) > 0 {
		return res, fmt.Errorf("%w: %v", resource.ErrPartialResource, errs)
	}
	return res, nil
}
