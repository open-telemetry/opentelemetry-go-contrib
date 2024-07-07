// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package autoexport // import "go.opentelemetry.io/contrib/exporters/autoexport"

import (
	"context"

	"go.opentelemetry.io/contrib/exporters/autoexport/utils/env"
	"go.opentelemetry.io/contrib/exporters/autoexport/utils/functional"
)

// signal represents a generic OpenTelemetry signal (logs, metrics and traces).
type signal[T any] struct {
	envKey   string
	registry *registry[T]
}

// newSignal initializes a new OpenTelemetry signal for the given type T.
func newSignal[T any](envKey string) signal[T] {
	return signal[T]{
		envKey: envKey,
		registry: &registry[T]{
			names: make(map[string]factory[T]),
		},
	}
}

func (s signal[T]) create(ctx context.Context, options ...functional.Option[config[T]]) ([]T, error) {
	cfg, executor := functional.ResolveOptions(options...), newExecutor[T]()

	exporters, err := env.WithStringList(s.envKey, ",")
	if err != nil {
		if cfg.fallbackFactory != nil {
			executor.Append(cfg.fallbackFactory)
			return executor.Execute(ctx)
		}
		exporters = append(exporters, "otlp")
	}

	for _, expType := range exporters {
		factory, err := s.registry.load(expType)
		if err != nil {
			return nil, err
		}
		executor.Append(factory)
	}

	return executor.Execute(ctx)
}

// config holds common configuration across the different
// supported signals (logs, traces and metrics).
type config[T any] struct {
	fallbackFactory factory[T]
}

// withFallbackFactory assigns a fallback factory for the current signal.
func withFallbackFactory[T any](factoryFn factory[T]) functional.Option[config[T]] {
	return func(s *config[T]) *config[T] {
		s.fallbackFactory = factoryFn
		return s
	}
}
