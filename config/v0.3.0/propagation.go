// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config // import "go.opentelemetry.io/contrib/config/v0.3.0"

import (
	"go.opentelemetry.io/contrib/propagators/autoprop"
	"go.opentelemetry.io/otel/propagation"
)

func propagator(cfg configOptions) (propagation.TextMapPropagator, error) {
	if cfg.opentelemetryConfig.Propagator == nil {
		return autoprop.NewTextMapPropagator(), nil
	}

	names := make([]string, 0, len(cfg.opentelemetryConfig.Propagator.Composite))
	for _, name := range cfg.opentelemetryConfig.Propagator.Composite {
		if name != nil && *name != "" {
			names = append(names, *name)
		}
	}

	return autoprop.TextMapPropagator(names...)
}
