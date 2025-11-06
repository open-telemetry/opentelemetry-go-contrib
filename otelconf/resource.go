// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf // import "go.opentelemetry.io/contrib/otelconf"

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/contrib/otelconf/internal/kv"
)

func newResource(res OpenTelemetryConfigurationResource) (*resource.Resource, error) {
	if res == nil {
		return resource.Default(), nil
	}

	r, ok := res.(*ResourceJson)
	if !ok {
		return nil, newErrInvalid("resource")
	}

	attrs := make([]attribute.KeyValue, 0, len(r.Attributes))
	for _, v := range r.Attributes {
		attrs = append(attrs, kv.FromNameValue(v.Name, v.Value))
	}

	if r.SchemaUrl == nil {
		return resource.NewSchemaless(attrs...), nil
	}
	return resource.NewWithAttributes(*r.SchemaUrl, attrs...), nil
}
