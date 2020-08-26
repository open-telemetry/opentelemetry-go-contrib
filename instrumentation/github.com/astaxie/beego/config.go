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

package beego

import (
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/propagation"
	"go.opentelemetry.io/otel/api/trace"
)

// Config provides configuration for the beego OpenTelemetry
// middleware. Configuration is modified using the provided Options.
type Config struct {
	traceProvider trace.Provider
	meterProvider metric.Provider
	propagators   propagation.Propagators
	filters       []Filter
	formatter     SpanNameFormatter
}

// Option applies a configuration to the given Config.
type Option interface {
	Apply(*Config)
}

// OptionFunc is a function type that applies a particular
// configuration to the beego middleware in question.
type OptionFunc func(c *Config)

// Apply will apply the option to the Config, c.
func (o OptionFunc) Apply(c *Config) {
	o(c)
}

// ------------------------------------------ Options

// WithTracerProvider specifies a tracer provider to use for creating a tracer.
// If none is specified, the global provider is used.
func WithTracerProvider(traceProvider trace.Provider) Option {
	return OptionFunc(func(cfg *Config) {
		cfg.traceProvider = traceProvider
	})
}

// WithMeterProvider specifies a meter provider to use for creating a meter.
// If none is specified, the global provider is used.
func WithMeterProvider(provider metric.Provider) Option {
	return OptionFunc(func(cfg *Config) {
		cfg.meterProvider = provider
	})
}

// WithPropagators sets the propagators used in the middleware.
// Defaults to global.Propagators().
func WithPropagators(propagators propagation.Propagators) OptionFunc {
	return OptionFunc(func(c *Config) {
		c.propagators = propagators
	})
}

// WithFilter adds the given filter for use in the middleware.
// Defaults to no filters.
func WithFilter(f Filter) OptionFunc {
	return OptionFunc(func(c *Config) {
		c.filters = append(c.filters, f)
	})
}

// WithSpanNameFormatter sets the formatter to be used to format
// span names. Defaults to the path template.
func WithSpanNameFormatter(f SpanNameFormatter) OptionFunc {
	return OptionFunc(func(c *Config) {
		c.formatter = f
	})
}

// ------------------------------------------ Private Functions

func configure(options ...Option) *Config {
	config := &Config{
		traceProvider: global.TraceProvider(),
		meterProvider: global.MeterProvider(),
		propagators:   global.Propagators(),
		filters:       []Filter{},
		formatter:     defaultSpanNameFormatter,
	}
	for _, option := range options {
		option.Apply(config)
	}
	return config
}
