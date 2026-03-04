// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package jaeger_test

import (
	"go.opentelemetry.io/otel"

	"go.opentelemetry.io/contrib/propagators/jaeger"
)

func ExampleJaeger() {
	p := jaeger.Jaeger{}
	// register jaeger propagator
	otel.SetTextMapPropagator(p)
}
