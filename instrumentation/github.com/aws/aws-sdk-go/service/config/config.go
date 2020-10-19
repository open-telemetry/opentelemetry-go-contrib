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

package config

import (
	"go.opentelemetry.io/otel"
	otelmetric "go.opentelemetry.io/otel/api/metric"
	oteltrace "go.opentelemetry.io/otel/api/trace"
)

// Config is used to configure the aws sdk instrumentation.
type Config struct {
	TracerProvider           oteltrace.TracerProvider
	MetricProvider           otelmetric.MeterProvider
	Propagators              otel.TextMapPropagator
	SpanCorrelationInMetrics bool
}

// Option specifies instrumentation configuration options.
type Option func(*Config)

// WithPropagators specifies propagators to use for extracting
// information from the HTTP requests. If none are specified, global
// ones will be used.
func WithPropagators(propagators otel.TextMapPropagator) Option {
	return func(cfg *Config) {
		cfg.Propagators = propagators
	}
}

// WithTracerProvider specifies a tracer provider to use for creating a tracer.
// If none is specified, the global provider is used.
func WithTracerProvider(provider oteltrace.TracerProvider) Option {
	return func(cfg *Config) {
		cfg.TracerProvider = provider
	}
}

// WithMetricProvider specifies a metric provider to use for creating a tracer.
// If none is specified, the global provider is used.
func WithMetricProvider(provider otelmetric.MeterProvider) Option {
	return func(cfg *Config) {
		cfg.MetricProvider = provider
	}
}

// WithSpanCorrelationInMetrics specifies whether span id and trace id should be attached to metrics as labels
func WithSpanCorrelationInMetrics(v bool) Option {
	return func(cfg *Config) {
		cfg.SpanCorrelationInMetrics = v
	}
}
