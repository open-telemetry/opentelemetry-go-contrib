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

package otelaws // import "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"

import (
	"go.opentelemetry.io/otel/trace"
)

type config struct {
	TracerProvider  trace.TracerProvider
	AttributeSetter []AttributeSetter
}

// Option applies an option value.
type Option interface {
	apply(*config)
}

// optionFunc provides a convenience wrapper for simple Options
// that can be represented as functions.
type optionFunc func(*config)

func (o optionFunc) apply(c *config) {
	o(c)
}

// WithTracerProvider specifies a tracer provider to use for creating a tracer.
// If none is specified, the global TracerProvider is used.
func WithTracerProvider(provider trace.TracerProvider) Option {
	return optionFunc(func(cfg *config) {
		if provider != nil {
			cfg.TracerProvider = provider
		}
	})
}

// WithAttributeSetter specifies an attribute setter function for setting service specific attributes.
// If none is specified, the service will be determined by the DefaultAttributeSetter function and the corresponding attributes will be included.
func WithAttributeSetter(attributesetters ...AttributeSetter) Option {
	return optionFunc(func(cfg *config) {
		cfg.AttributeSetter = append(cfg.AttributeSetter, attributesetters...)
	})
}
