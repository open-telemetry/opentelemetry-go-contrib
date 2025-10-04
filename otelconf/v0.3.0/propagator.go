// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf // import "go.opentelemetry.io/contrib/otelconf/v0.3.0"

import (
	"errors"

	"go.opentelemetry.io/otel/propagation"

	"go.opentelemetry.io/contrib/propagators/autoprop"
)

var errInvalidPropagatorNil = errors.New("invalid propagator name: nil")

func propagator(cfg configOptions) (propagation.TextMapPropagator, error) {
	if cfg.opentelemetryConfig.Propagator == nil {
		return autoprop.NewTextMapPropagator(), nil
	}

	n := len(cfg.opentelemetryConfig.Propagator.Composite)
	if n == 0 {
		return autoprop.NewTextMapPropagator(), nil
	}

	var names []string
	for _, name := range cfg.opentelemetryConfig.Propagator.Composite {
		if name == nil {
			return nil, errInvalidPropagatorNil
		}
		if *name == "" {
			continue
		}

		names = append(names, *name)
	}
	if len(names) == 0 {
		return autoprop.NewTextMapPropagator(), nil
	}

	return autoprop.TextMapPropagator(names...)
}
