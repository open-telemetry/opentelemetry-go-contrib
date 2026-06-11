// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package heroku // import "go.opentelemetry.io/contrib/detectors/heroku"

import (
	"context"
	"os"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
)

const (
	envAppID            = "HEROKU_APP_ID"
	envAppName          = "HEROKU_APP_NAME"
	envDynoID           = "HEROKU_DYNO_ID"
	envReleaseCreatedAt = "HEROKU_RELEASE_CREATED_AT"
	envReleaseVersion   = "HEROKU_RELEASE_VERSION"
	envSlugCommit       = "HEROKU_SLUG_COMMIT"
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

// ResourceDetector collects resource information of Heroku dynos.
type ResourceDetector struct {
	cfg config
}

// NewResourceDetector returns a [resource.Detector] that detects resource
// attributes on Heroku dynos.
func NewResourceDetector(opts ...Option) *ResourceDetector {
	var cfg config
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	return &ResourceDetector{cfg: cfg}
}

// Detect detects resource attributes of the Heroku dyno the process is running
// on. It returns an empty resource and no error when no Heroku attributes are
// available.
func (d *ResourceDetector) Detect(context.Context) (*resource.Resource, error) {
	var attrs []attribute.KeyValue

	if dynoID, ok := os.LookupEnv(envDynoID); ok {
		attrs = append(attrs, semconv.ServiceInstanceID(dynoID))
	}
	if v, ok := os.LookupEnv(envAppID); ok {
		attrs = append(attrs, semconv.CloudProviderHeroku, semconv.HerokuAppID(v))
	}
	if v, ok := os.LookupEnv(envAppName); ok {
		attrs = append(attrs, semconv.ServiceName(v))
	}
	if v, ok := os.LookupEnv(envReleaseCreatedAt); ok {
		attrs = append(attrs, semconv.HerokuReleaseCreationTimestamp(v))
	}
	if v, ok := os.LookupEnv(envReleaseVersion); ok {
		attrs = append(attrs, semconv.ServiceVersion(v))
	}
	if v, ok := os.LookupEnv(envSlugCommit); ok {
		attrs = append(attrs, semconv.HerokuReleaseCommit(v))
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

	if len(attrs) == 0 {
		return resource.Empty(), nil
	}
	return resource.NewWithAttributes(semconv.SchemaURL, attrs...), nil
}
