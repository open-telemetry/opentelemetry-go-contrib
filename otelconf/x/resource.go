// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package x // import "go.opentelemetry.io/contrib/otelconf/x"

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"

	ecsdetector "go.opentelemetry.io/contrib/detectors/aws/ecs"

	"go.opentelemetry.io/contrib/otelconf/internal/kv"
)

func resourceOpts(detectors []ExperimentalResourceDetector) []resource.Option {
	opts := []resource.Option{}
	for _, d := range detectors {
		if d.AWSECS != nil {
			opts = append(opts, resource.WithDetectors(ecsdetector.NewResourceDetector()))
		}
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

type resourceBuilder func(context.Context, ...resource.Option) (*resource.Resource, error)

func newResource(ctx context.Context, r *Resource) (*resource.Resource, error) {
	return newResourceWithBuilder(ctx, r, resource.New)
}

func newResourceWithBuilder(ctx context.Context, r *Resource, build resourceBuilder) (*resource.Resource, error) {
	if r == nil {
		return resource.DefaultWithContext(ctx), nil
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
		resource.WithAttributes(resource.DefaultWithContext(ctx).Attributes()...),
	}

	if r.DetectionDevelopment != nil {
		opts = append(opts, resourceOpts(r.DetectionDevelopment.Detectors)...)
	}

	opts = append(
		opts,
		resource.WithAttributes(attrs...),
		resource.WithSchemaURL(schema),
	)

	return build(ctx, opts...)
}
