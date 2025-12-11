// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf // import "go.opentelemetry.io/contrib/otelconf/v0.3.0"

import (
	"go.opentelemetry.io/otel/propagation"

	"go.opentelemetry.io/contrib/propagators/autoprop"
)

func newPropagator(prop OpenTelemetryConfigurationPropagator) (propagation.TextMapPropagator, error) {
	if prop == nil {
		return propagation.NewCompositeTextMapPropagator(), nil
	}

	p, ok := prop.(*PropagatorJson)
	if !ok {
		return nil, newErrInvalid("propagator")
	}

	n := len(p.Composite)
	if n == 0 {
		return propagation.NewCompositeTextMapPropagator(), nil
	}

	var names []string
	for _, propagator := range p.Composite {
		if propagator.B3 != nil {
			names = append(names, "b3")
		}
		if propagator.B3Multi != nil {
			names = append(names, "b3multi")
		}
		if propagator.Baggage != nil {
			names = append(names, "baggage")
		}
		if propagator.Jaeger != nil {
			names = append(names, "jaeger")
		}
		if propagator.Ottrace != nil {
			names = append(names, "ottrace")
		}
		if propagator.Tracecontext != nil {
			names = append(names, "tracecontext")
		}

		// TODO: support AdditionalProperties
	}
	if len(names) == 0 {
		return autoprop.NewTextMapPropagator(), nil
	}

	// TODO: handle composite list

	return autoprop.TextMapPropagator(names...)
}
