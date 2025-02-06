// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config // import "go.opentelemetry.io/contrib/config/v0.3.0"

import (
	"errors"

	"go.opentelemetry.io/contrib/propagators/aws/xray"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/contrib/propagators/ot"
	"go.opentelemetry.io/otel/propagation"
)

func propagator(cfg configOptions) (propagation.TextMapPropagator, error) {
	if cfg.opentelemetryConfig.Propagator == nil {
		return nil, nil
	}

	var errs []error
	var ps []propagation.TextMapPropagator
	for _, name := range cfg.opentelemetryConfig.Propagator.Composite {
		if name == nil || *name == "" {
			continue
		}

		p, err := propagatorByName(*name)
		if err == nil {
			ps = append(ps, p)
		} else {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	if len(ps) == 0 {
		return nil, nil
	}

	res := propagation.NewCompositeTextMapPropagator(ps...)
	return res, nil
}

func propagatorByName(name string) (propagation.TextMapPropagator, error) {
	switch name {
	case "tracecontext":
		return propagation.TraceContext{}, nil
	case "baggage":
		return propagation.Baggage{}, nil
	case "b3":
		return b3.New(), nil
	case "b3multi":
		return b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader)), nil
	case "jaeger":
		return jaeger.Jaeger{}, nil
	case "xray":
		return xray.Propagator{}, nil
	case "ottrace":
		return ot.OT{}, nil
	default:
		return nil, errors.New("unsupported propagator")
	}
}
