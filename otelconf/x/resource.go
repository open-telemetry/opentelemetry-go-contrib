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

	if r.DetectionDevelopment != nil {
		opts = append(opts, resourceOpts(r.DetectionDevelopment.Detectors)...)
	}

	return resource.New(context.Background(), opts...)
}
