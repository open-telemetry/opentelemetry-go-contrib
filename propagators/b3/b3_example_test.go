// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package b3_test

import (
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
)

func ExampleNew() {
	p := b3.New()
	// Register the B3 propagator globally.
	otel.SetTextMapPropagator(p)
}

func ExampleNew_injectEncoding() {
	// Create a B3 propagator configured to inject context with both multiple
	// and single header B3 HTTP encoding.
	p := b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader | b3.B3SingleHeader))
	otel.SetTextMapPropagator(p)
}
