// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelecho // import "go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// config is used to configure the mux middleware.
type config struct {
	TracerProvider        oteltrace.TracerProvider
	MeterProvider         metric.MeterProvider
	Propagators           propagation.TextMapPropagator
	Skipper               middleware.Skipper
	MetricAttributeFn     MetricAttributeFn
	EchoMetricAttributeFn EchoMetricAttributeFn
	OnError               OnErrorFn
}

// MetricAttributeFn is used to extract additional attributes from the http.Request
// and return them as a slice of attribute.KeyValue.
type MetricAttributeFn func(*http.Request) []attribute.KeyValue

// EchoMetricAttributeFn is used to extract additional attributes from the echo.Context
// and return them as a slice of attribute.KeyValue.
type EchoMetricAttributeFn func(*echo.Context) []attribute.KeyValue

// OnErrorFn is used to specify how errors are handled in the middleware.
type OnErrorFn func(*echo.Context, error)

// defaultOnError is the default function called when an error occurs during request processing.
// In Echo v5, errors are handled by the framework's HTTPErrorHandler automatically.
var defaultOnError = func(_ *echo.Context, _ error) {
	// In Echo v5, errors are propagated and handled by HTTPErrorHandler
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

// WithMeterProvider specifies a meter provider to use for creating a meter.
// If none is specified, the global provider is used.
func WithMeterProvider(provider metric.MeterProvider) Option {
	return optionFunc(func(cfg *config) {
		if provider != nil {
			cfg.MeterProvider = provider
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

// WithMetricAttributeFn specifies a function that extracts additional attributes from the http.Request
// and returns them as a slice of attribute.KeyValue.
//
// If attributes are duplicated between this method and `WithEchoMetricAttributeFn`, the attributes in this method will be overridden.
func WithMetricAttributeFn(f MetricAttributeFn) Option {
	return optionFunc(func(cfg *config) {
		cfg.MetricAttributeFn = f
	})
}

// WithEchoMetricAttributeFn specifies a function that extracts additional attributes from the echo.Context
// and returns them as a slice of attribute.KeyValue.
//
// If attributes are duplicated between this method and `WithMetricAttributeFn`, the attributes in this method will be used.
func WithEchoMetricAttributeFn(f EchoMetricAttributeFn) Option {
	return optionFunc(func(cfg *config) {
		cfg.EchoMetricAttributeFn = f
	})
}

// WithOnError specifies a function that is called when an error occurs during request processing.
//
// In Echo v5, errors are automatically handled by the HTTPErrorHandler after the middleware returns.
// This callback allows you to perform additional actions when an error occurs.
func WithOnError(f OnErrorFn) Option {
	return optionFunc(func(cfg *config) {
		if f != nil {
			cfg.OnError = f
		}
	})
}
