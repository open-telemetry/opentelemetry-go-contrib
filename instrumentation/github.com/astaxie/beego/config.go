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
	tracer      trace.Tracer
	meter       metric.Meter
	propagators propagation.Propagators
	filters     []Filter
	formatter   SpanNameFormatter
}

// Option applies a configuration to the give Config.
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

// WithTracer set the tracer to be used by the middleware for
// creating spans.
// Defaults to global.Tracer("go.opentelemetry.io/contrib/instrumentation/github.com/astaxie/beego").
func WithTracer(tracer trace.Tracer) OptionFunc {
	return OptionFunc(func(c *Config) {
		c.tracer = tracer
	})
}

// WithMeter sets the meter to be used to create the instruments
// used in the middleware.
// Defaults to global.Meter("go.opentelemetry.io/contrib/instrumentation/github.com/astaxie/beego").
func WithMeter(meter metric.Meter) OptionFunc {
	return OptionFunc(func(c *Config) {
		c.meter = meter
	})
}

// WithPropagators sets the propagators used in the middleware.
// Defaults to global.Propagators().
func WithPropagators(propagators propagation.Propagators) OptionFunc {
	return OptionFunc(func(c *Config) {
		c.propagators = propagators
	})
}

// WithFilter adds the given filter
// as a filter used for tracing in the middleware.
// Defaults to no filters.
func WithFilter(f Filter) OptionFunc {
	return OptionFunc(func(c *Config) {
		c.filters = append(c.filters, f)
	})
}

// WithSpanNameFormatter sets the formatter to be used for format
// span names. Defaults to http.Request.URL.Path.
func WithSpanNameFormatter(f SpanNameFormatter) OptionFunc {
	return OptionFunc(func(c *Config) {
		c.formatter = f
	})
}

// ------------------------------------------ Private Functions

func configure(options ...Option) *Config {
	config := &Config{
		tracer:      global.Tracer(packageName),
		meter:       global.Meter(packageName),
		propagators: global.Propagators(),
		filters:     []Filter{},
		formatter:   defaultSpanNameFormatter,
	}
	for _, option := range options {
		option.Apply(config)
	}
	return config
}
