// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package autoexport // import "go.opentelemetry.io/contrib/exporters/autoexport"

import (
	"context"
)

// factory is a type alias for a factory method to build a signal-specific exporter.
type factory[T any] func(ctx context.Context) (T, error)

// executor allows different factories to be registered and executed.
type executor[T any] struct {
	// factories holds a list of exporter factory functions.
	factories []factory[T]
}

func newExecutor[T any]() *executor[T] {
	return &executor[T]{
		factories: make([]factory[T], 0),
	}
}

// Append appends the given factory to the executor.
func (f *executor[T]) Append(fact factory[T]) {
	f.factories = append(f.factories, fact)
}

// Execute executes all the factories and returns the results.
// An error will be returned if at least one factory fails.
func (f *executor[T]) Execute(ctx context.Context) ([]T, error) {
	var results []T

	for _, registered := range f.factories {
		result, err := registered(ctx)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, nil
}
