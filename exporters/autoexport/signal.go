// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package autoexport // import "go.opentelemetry.io/contrib/exporters/autoexport"

import (
	"context"
	"os"
)

type signal[T any] struct {
	envKey   string
	registry *registry[T]
}

func newSignal[T any](envKey string) signal[T] {
	return signal[T]{
		envKey: envKey,
		registry: &registry[T]{
			names: make(map[string]func(context.Context) (T, error)),
		},
	}
}

func (s signal[T]) create(ctx context.Context, opts ...option[T]) (T, error) {
	var cfg config[T]
	for _, opt := range opts {
		opt.apply(&cfg)
	}

	expType := os.Getenv(s.envKey)
	if expType == "" {
		if cfg.fallbackFactory != nil {
			return cfg.fallbackFactory(ctx)
		}
		expType = "otlp"
	}

	return s.registry.load(ctx, expType)
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
