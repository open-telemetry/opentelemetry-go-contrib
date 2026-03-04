// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package autodetect_test

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
)

const key = "my.key"

type MyDetector struct{}

func (MyDetector) Detect(context.Context) (*resource.Resource, error) {
	return resource.NewSchemaless(attribute.String(key, "value")), nil
}

var enc = keyEncoder{}

type keyEncoder struct{}

func (keyEncoder) Encode(iterator attribute.Iterator) string {
	var b strings.Builder

	iterator.Next()
	_, _ = b.WriteString(string(iterator.Attribute().Key))

	for iterator.Next() {
		_, _ = b.WriteRune(' ')
		_, _ = b.WriteString(string(iterator.Attribute().Key))
	}
	return b.String()
}
