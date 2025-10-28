// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf // import "go.opentelemetry.io/contrib/otelconf/v1.0.0-rc.1"

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

	var attrs []attribute.KeyValue
	for _, v := range r.Attributes {
		attrs = append(attrs, kv.FromNameValue(v.Name, v.Value))
	}

	if r.SchemaUrl == nil {
		return resource.NewSchemaless(attrs...), nil
	}
	return resource.NewWithAttributes(*r.SchemaUrl, attrs...), nil
}
