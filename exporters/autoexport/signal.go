// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package autoexport // import "go.opentelemetry.io/contrib/exporters/autoexport"

import (
	"context"

	"go.opentelemetry.io/contrib/exporters/autoexport/utils/env"
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

func (s signal[T]) create(ctx context.Context, opts ...option[T]) ([]T, error) {
	var cfg config[T]
	for _, opt := range opts {
		opt.apply(&cfg)
	}

	executor := newExecutor[T]()

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

type config[T any] struct {
	fallbackFactory func(ctx context.Context) (T, error)
}

type option[T any] interface {
	apply(cfg *config[T])
}

type optionFunc[T any] func(cfg *config[T])

//lint:ignore U1000 https://github.com/dominikh/go-tools/issues/1440
func (fn optionFunc[T]) apply(cfg *config[T]) {
	fn(cfg)
}

func withFallbackFactory[T any](fallbackFactory func(ctx context.Context) (T, error)) option[T] {
	return optionFunc[T](func(cfg *config[T]) {
		cfg.fallbackFactory = fallbackFactory
	})
}
