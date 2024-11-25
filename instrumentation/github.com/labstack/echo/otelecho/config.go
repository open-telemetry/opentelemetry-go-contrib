// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelecho // import "go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"

import (
	"net/http"

	"github.com/labstack/echo/v4/middleware"

	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// defaultSpanNameFormatter is the default function used for formatting span names.
var defaultSpanNameFormatter = func(path string, _ *http.Request) string {
	return path
}

// SpanNameFormatter is a function that takes a path and an HTTP request and returns a span name.
type SpanNameFormatter func(string, *http.Request) string

// config is used to configure the mux middleware.
type config struct {
	TracerProvider    oteltrace.TracerProvider
	Propagators       propagation.TextMapPropagator
	Skipper           middleware.Skipper
	spanNameFormatter SpanNameFormatter
}

// Option specifies instrumentation configuration options.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (o optionFunc) apply(c *config) {
	o(c)
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

// WithSkipper specifies a skipper for allowing requests to skip generating spans.
func WithSkipper(skipper middleware.Skipper) Option {
	return optionFunc(func(cfg *config) {
		cfg.Skipper = skipper
	})
}

// WithSpanNameFormatter specifies a function to use for formatting span names.
func WithSpanNameFormatter(formatter SpanNameFormatter) Option {
	return optionFunc(func(cfg *config) {
		if formatter != nil {
			cfg.spanNameFormatter = formatter
		}
	})
}
