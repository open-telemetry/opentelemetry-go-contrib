// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf // import "go.opentelemetry.io/contrib/otelconf"

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/contrib/otelconf/internal/kv"
)

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
		resource.WithAttributes(attrs...),
		resource.WithSchemaURL(schema),
	}

	return build(ctx, opts...)
}
