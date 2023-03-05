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

package otelresty // import "go.opentelemetry.io/contrib/instrumentation/github.com/go-resty/resty/otelresty"

import (
	"github.com/go-resty/resty/v2"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// defaultSkipper provides default behaviour, which won't skip span creation.
func defaultSkipper(*resty.Request) bool {
	return false
}

func defaultSpanNameFormatter(_ string, req *resty.Request) string {
	return "http " + req.Method
}

// config is used to configure the go-resty middleware.
type config struct {
	TracerProvider    oteltrace.TracerProvider
	Propagators       propagation.TextMapPropagator
	SpanNameFormatter func(string, *resty.Request) string
	SpanStartOptions  []oteltrace.SpanStartOption
	Skipper           func(*resty.Request) bool
}

func newConfig(options ...Option) *config {
	cfg := &config{
		Propagators:    otel.GetTextMapPropagator(),
		TracerProvider: otel.GetTracerProvider(),
		Skipper:        defaultSkipper,
	}

	defaultOpts := []Option{
		WithSpanOptions(oteltrace.WithSpanKind(oteltrace.SpanKindClient)),
		WithSpanNameFormatter(defaultSpanNameFormatter),
	}

	options = append(defaultOpts, options...)

	for _, opt := range options {
		opt.apply(cfg)
	}

	return cfg
}

// Option applies a configuration value.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (o optionFunc) apply(c *config) {
	o(c)
}

// WithSkipper specifies a skipper function to determine if the middleware
// should not create a span for a determined request. If not specified,
// a span will always be created.
func WithSkipper(skipper func(r *resty.Request) bool) Option {
	return optionFunc(func(c *config) {
		if skipper != nil {
			c.Skipper = skipper
		}
	})
}

// WithPropagators specifies propagators to use for extracting
// information from the HTTP requests. If none are specified, global
// ones will be used.
func WithPropagators(propagators propagation.TextMapPropagator) Option {
	return optionFunc(func(cfg *config) {
		if propagators != nil {
			cfg.Propagators = propagators
		}
	})
}

// WithTracerProvider specifies a tracer provider to use for creating a tracer.
// If none is specified, the global provider is used.
func WithTracerProvider(provider oteltrace.TracerProvider) Option {
	return optionFunc(func(cfg *config) {
		if provider != nil {
			cfg.TracerProvider = provider
		}
	})
}

// WithSpanOptions configures an additional set of
// trace.SpanOptions, which are applied to each new span.
func WithSpanOptions(opts ...trace.SpanStartOption) Option {
	return optionFunc(func(c *config) {
		c.SpanStartOptions = append(c.SpanStartOptions, opts...)
	})
}

// WithSpanNameFormatter takes a function that will be called on every
// request and the returned string will become the Span Name.
func WithSpanNameFormatter(f func(operation string, r *resty.Request) string) Option {
	return optionFunc(func(c *config) {
		c.SpanNameFormatter = f
	})
}
