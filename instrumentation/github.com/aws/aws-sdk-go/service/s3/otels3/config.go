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

package otels3

import (
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// Config provides options for the AWS SDK instrumentation.
type config struct {
	TracerProvider  oteltrace.TracerProvider
	MetricProvider  otelmetric.MeterProvider
	Propagators     propagation.TextMapPropagator
	SpanCorrelation bool
}

// Option interface used for setting instrumentation configuration options.
type Option interface {
	apply(*config)
}

// optionFunc provides a wrapper for specifying options in function format
type optionFunc func(*config)

// Apply will set the option in the provided config.
func (o optionFunc) apply(cfg *config) {
	o(cfg)
}

// WithPropagators specifies propagators to use for extracting
// information from the HTTP requests. If none are specified, global
// ones will be used.
func WithPropagators(propagators propagation.TextMapPropagator) Option {
	return optionFunc(func(cfg *config) {
		cfg.Propagators = propagators
	})
}

// WithTracerProvider specifies a TracerProvider to use for creating a Tracer.
// If none is specified, the global provider is used.
func WithTracerProvider(provider oteltrace.TracerProvider) Option {
	return optionFunc(func(cfg *config) {
		cfg.TracerProvider = provider
	})
}

// WithMeterProvider specifies a MeterProvider to use for creating a Meter.
// If none is specified, the global provider is used.
func WithMeterProvider(provider otelmetric.MeterProvider) Option {
	return optionFunc(func(cfg *config) {
		cfg.MetricProvider = provider
	})
}

// WithSpanCorrelation specifies whether span ID and trace ID should be added to metric event as attributes.
func WithSpanCorrelation(v bool) Option {
	return optionFunc(func(cfg *config) {
		cfg.SpanCorrelation = v
	})
}
