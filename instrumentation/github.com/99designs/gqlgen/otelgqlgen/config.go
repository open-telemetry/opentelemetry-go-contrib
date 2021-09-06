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

package otelgqlgen

import (
	"go.opentelemetry.io/otel/trace"
)

// config is used to configure the mongo tracer.
type config struct {
	TracerProvider          trace.TracerProvider
	Tracer                  trace.Tracer
	ComplexityExtensionName string
}

// Option specifies instrumentation configuration options.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (o optionFunc) apply(c *config) {
	o(c)
}

// WithTracerProvider specifies a tracer provider to use for creating a tracer.
// If none is specified, the global provider is used.
func WithTracerProvider(provider trace.TracerProvider) Option {
	return optionFunc(func(cfg *config) {
		cfg.TracerProvider = provider
	})
}

// WithComplexityExtensionName specifies complexity extension name
func WithComplexityExtensionName(complexityExtensionName string) Option {
	return optionFunc(func(cfg *config) {
		cfg.ComplexityExtensionName = complexityExtensionName
	})
}
