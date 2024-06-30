// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package functional // import "go.opentelemetry.io/contrib/exporters/autoexport/utils/functional"

// Option is a type alias.
type Option[T any] func(*T) *T

// ResolveOptions applies the given options to a new T instance
// and return it once options has been applied.
func ResolveOptions[T any](opts ...Option[T]) *T {
	o := new(T)
	for _, opt := range opts {
		if opt != nil {
			opt(o)
		}
	}
	return o
}
