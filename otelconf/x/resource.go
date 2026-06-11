// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package x // import "go.opentelemetry.io/contrib/otelconf/x"

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/contrib/otelconf/internal/kv"
)

func resourceOpts(detectors []ExperimentalResourceDetector) []resource.Option {
	opts := []resource.Option{}
	for _, d := range detectors {
		if d.Container != nil {
			opts = append(opts, resource.WithContainer())
		}
		if d.Host != nil {
			opts = append(opts, resource.WithHost(), resource.WithOS())
		}
		if d.Process != nil {
			opts = append(opts, resource.WithProcess())
		}
		if d.Service != nil {
			opts = append(opts, resource.WithService())
		}
	}
	return opts
}

func newResource(r *Resource) (*resource.Resource, error) {
	if r == nil {
		return resource.Default(), nil
	}

	attrs := make([]attribute.KeyValue, 0, len(r.Attributes))
	for _, v := range r.Attributes {
		attrs = append(attrs, kv.FromNameValue(v.Name, v.Value))
	}

	var schema string
	if r.SchemaUrl != nil {
		schema = *r.SchemaUrl
	}
	opts := []resource.Option{
		resource.WithAttributes(attrs...),
		resource.WithSchemaURL(schema),
	}

	base, err := resource.New(context.Background(), opts...)
	if err != nil {
		return nil, err
	}

	if r.DetectionDevelopment == nil {
		return base, nil
	}

	detected, err := newDetectedResource(r.DetectionDevelopment)
	if err != nil {
		return nil, err
	}

	return resource.Merge(base, detected)
}

func newDetectedResource(detection *ExperimentalResourceDetection) (*resource.Resource, error) {
	filter, err := newIncludeExcludeFilter(detection.Attributes)
	if err != nil {
		return nil, err
	}

	opts := resourceOpts(detection.Detectors)
	if len(opts) == 0 {
		return resource.NewSchemaless(), nil
	}

	detected, err := resource.New(context.Background(), opts...)
	if err != nil {
		return nil, err
	}

	attrs := detected.Attributes()
	filtered := make([]attribute.KeyValue, 0, len(attrs))
	for _, attr := range attrs {
		if filter(attr) {
			filtered = append(filtered, attr)
		}
	}

	return resource.NewWithAttributes(detected.SchemaURL(), filtered...), nil
}
