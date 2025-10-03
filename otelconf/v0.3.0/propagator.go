// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf // import "go.opentelemetry.io/contrib/otelconf/v0.3.0"

import (
	"errors"

	"go.opentelemetry.io/contrib/propagators/autoprop"
	"go.opentelemetry.io/otel/propagation"
)

var (
	errInvalidPropagatorEmpty = errors.New("invalid propagator name: empty")
	errInvalidPropagatorNil   = errors.New("invalid propagator name: nil")
)

func propagator(cfg configOptions) (propagation.TextMapPropagator, error) {
	if cfg.opentelemetryConfig.Propagator == nil {
		return autoprop.NewTextMapPropagator(), nil
	}

	n := len(cfg.opentelemetryConfig.Propagator.Composite)
	if n == 0 {
		return autoprop.NewTextMapPropagator(), nil
	}

	names := make([]string, 0, n)
	for _, name := range cfg.opentelemetryConfig.Propagator.Composite {
		if name == nil {
			return nil, errInvalidPropagatorNil
		}
		if *name == "" {
			return nil, errInvalidPropagatorEmpty
		}

		names = append(names, *name)
	}

	return autoprop.TextMapPropagator(names...)
}
