// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package hetzner provides a resource detector for Hetzner Cloud servers.
package hetzner // import "go.opentelemetry.io/contrib/detectors/hetzner"

import (
	"context"
	"fmt"
	"strconv"

	hcloudmeta "github.com/hetznercloud/hcloud-go/v2/hcloud/metadata"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

// Compile-time interface assertion.
var _ resource.Detector = (*ResourceDetector)(nil)

// newHcloudClient is the factory for the hcloud metadata client.
// It is a package-level variable so tests can substitute a fake server.
var newHcloudClient = func() *hcloudmeta.Client {
	return hcloudmeta.NewClient()
}

// ResourceDetector collects resource information of Hetzner Cloud servers.
type ResourceDetector struct {
	client *hcloudmeta.Client
}

// NewResourceDetector returns a [resource.Detector] that detects resource
// attributes on Hetzner Cloud servers.
func NewResourceDetector() *ResourceDetector {
	return &ResourceDetector{client: newHcloudClient()}
}

// Detect detects resource attributes of the Hetzner Cloud server the process
// is running on. It returns an empty resource and no error when not running on
// a Hetzner Cloud server. If the process is running on a Hetzner Cloud server
// but some attributes cannot be retrieved, a partial resource is returned
// together with [resource.ErrPartialResource].
//
// All six attributes are always emitted when available. Callers that need
// per-attribute suppression (e.g. processor/resourcedetectionprocessor) should
// post-filter the returned [resource.Resource] according to their own
// configuration.
func (d *ResourceDetector) Detect(_ context.Context) (*resource.Resource, error) {
	if !d.client.IsHcloudServer() {
		return resource.Empty(), nil
	}

	attrs := []attribute.KeyValue{
		semconv.CloudProviderHetzner,
		semconv.CloudPlatformHetznerCloudServer,
	}

	var errs []error

	id, err := d.client.InstanceID()
	if err != nil {
		errs = append(errs, fmt.Errorf("instance ID: %w", err))
	} else {
		attrs = append(attrs, semconv.HostID(strconv.FormatInt(id, 10)))
	}

	hostname, err := d.client.Hostname()
	if err != nil {
		errs = append(errs, fmt.Errorf("hostname: %w", err))
	} else {
		attrs = append(attrs, semconv.HostName(hostname))
	}

	region, err := d.client.Region()
	if err != nil {
		errs = append(errs, fmt.Errorf("region: %w", err))
	} else {
		attrs = append(attrs, semconv.CloudRegion(region))
	}

	az, err := d.client.AvailabilityZone()
	if err != nil {
		errs = append(errs, fmt.Errorf("availability zone: %w", err))
	} else {
		attrs = append(attrs, semconv.CloudAvailabilityZone(az))
	}

	res := resource.NewWithAttributes(semconv.SchemaURL, attrs...)

	if len(errs) > 0 {
		return res, fmt.Errorf("%w: %v", resource.ErrPartialResource, errs)
	}
	return res, nil
}
