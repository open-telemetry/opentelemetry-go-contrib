// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package autoexport // import "go.opentelemetry.io/contrib/exporters/autoexport"

import (
	"context"
)

// executor allows different factories to be registered and executed.
type executor[T any] struct {
	// factories holds a list of exporter factory functions.
	factories []func(ctx context.Context) (T, error)
}

func newExecutor[T any]() *executor[T] {
	return &executor[T]{
		factories: make([]func(ctx context.Context) (T, error), 0),
	}
}

// Append appends the given factory to the executor.
func (f *executor[T]) Append(factory func(ctx context.Context) (T, error)) {
	f.factories = append(f.factories, factory)
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
