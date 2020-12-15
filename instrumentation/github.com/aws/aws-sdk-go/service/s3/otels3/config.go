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

// config is the configuration for the AWS SDK instrumentation.
type config struct {
	TracerProvider    oteltrace.TracerProvider
	MeterProvider     otelmetric.MeterProvider
	TextMapPropagator propagation.TextMapPropagator
	SpanCorrelation   bool
}

// Option sets instrumentation configuration options.
type Option interface {
	apply(*config)
}

// optionFunc provides a wrapper for specifying options as a function.
type optionFunc func(*config)

// apply sets the appropriate option in cfg.
func (o optionFunc) apply(cfg *config) {
	o(cfg)
}

// WithTextMapPropagator specifies the TextMapPropagator to extract
// information from the HTTP requests with. If none is provided, the global
// TextMapPropagator will be used.
func WithTextMapPropagator(textMapPropagator propagation.TextMapPropagator) Option {
	return optionFunc(func(cfg *config) {
		cfg.TextMapPropagator = textMapPropagator
	})
}

// WithTracerProvider specifies the TracerProvider used to create a Tracer for
// this instrumentation. If none is provided, the global TracerProvider is used.
func WithTracerProvider(provider oteltrace.TracerProvider) Option {
	return optionFunc(func(cfg *config) {
		cfg.TracerProvider = provider
	})
}

// WithMeterProvider specifies the MeterProvider used to create a Meter for
// this instrumentation. If none is provided, the global MeterProvider is used.
func WithMeterProvider(provider otelmetric.MeterProvider) Option {
	return optionFunc(func(cfg *config) {
		cfg.MeterProvider = provider
	})
}

// WithSpanCorrelation specifies whether span ID and trace ID should be added to metric events as attributes.
func WithSpanCorrelation(v bool) Option {
	return optionFunc(func(cfg *config) {
		cfg.SpanCorrelation = v
	})
}
