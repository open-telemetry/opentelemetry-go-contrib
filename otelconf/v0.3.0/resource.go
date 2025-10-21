// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf // import "go.opentelemetry.io/contrib/otelconf/v0.3.0"

import (
	"go.opentelemetry.io/contrib/otelconf/internal/kv"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
)

func newResource(res *Resource) *resource.Resource {
	if res == nil {
		return resource.Default()
	}

	var attrs []attribute.KeyValue
	for _, v := range res.Attributes {
		attrs = append(attrs, kv.FromNameValue(v.Name, v.Value))
	}

	if res.SchemaUrl == nil {
		return resource.NewSchemaless(attrs...)
	}
	return resource.NewWithAttributes(*res.SchemaUrl, attrs...)
}
