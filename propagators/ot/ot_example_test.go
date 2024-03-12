// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ot_test

import (
	"go.opentelemetry.io/contrib/propagators/ot"
	"go.opentelemetry.io/otel"
)

func ExampleOT() {
	otPropagator := ot.OT{}
	// register ot propagator
	otel.SetTextMapPropagator(otPropagator)
}
