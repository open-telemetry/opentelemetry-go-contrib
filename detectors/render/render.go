// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package render provides a resource detector for Render.com services.
package render // import "go.opentelemetry.io/contrib/detectors/render"

import (
	"context"
	"os"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
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

// ResourceDetector collects resource information of a Render.com service.
type ResourceDetector struct {
	cfg config
}

// NewResourceDetector returns a [resource.Detector] that detects resource
// attributes on Render.com services.
func NewResourceDetector(opts ...Option) *ResourceDetector {
	var cfg config
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	return &ResourceDetector{cfg: cfg}
}

// Detect detects resource attributes of the Render.com service the process
// is running on. It returns an empty resource and no error when not running
// on Render.
func (d *ResourceDetector) Detect(_ context.Context) (*resource.Resource, error) {
	serviceID, ok := os.LookupEnv("RENDER_SERVICE_ID")
	if !ok {
		return resource.Empty(), nil
	}

	var attrs []attribute.KeyValue

	if v, ok := os.LookupEnv("RENDER_INSTANCE_ID"); ok {
		attrs = append(attrs, semconv.ServiceInstanceID(v))
	} else {
		attrs = append(attrs, semconv.ServiceInstanceID(serviceID))
	}

	if v, ok := os.LookupEnv("RENDER_SERVICE_NAME"); ok {
		attrs = append(attrs, semconv.ServiceName(v))
	} else {
		attrs = append(attrs, semconv.ServiceName(serviceID))
	}

	if v, ok := os.LookupEnv("RENDER_GIT_COMMIT"); ok {
		attrs = append(attrs, semconv.ServiceVersion(v))
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

	return resource.NewWithAttributes(semconv.SchemaURL, attrs...), nil
}
