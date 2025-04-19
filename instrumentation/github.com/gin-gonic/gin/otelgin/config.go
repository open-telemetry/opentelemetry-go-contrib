// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Based on https://github.com/DataDog/dd-trace-go/blob/8fb554ff7cf694267f9077ae35e27ce4689ed8b6/contrib/gin-gonic/gin/option.go

package otelgin // import "go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

import (
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type config struct {
	TracerProvider    oteltrace.TracerProvider
	Propagators       propagation.TextMapPropagator
	Filters           []Filter
	SpanNameFormatter SpanNameFormatter
	MeterProvider     metric.MeterProvider
	MetricAttributeFn MetricAttributeFn
}

// defaultSpanNameFormatter is the default span name formatter.
var defaultSpanNameFormatter SpanNameFormatter = func(c *gin.Context) string {
	method := strings.ToUpper(c.Request.Method)
	if !slices.Contains([]string{
		http.MethodGet, http.MethodHead,
		http.MethodPost, http.MethodPut,
		http.MethodPatch, http.MethodDelete,
		http.MethodConnect, http.MethodOptions,
		http.MethodTrace,
	}, method) {
		method = "HTTP"
	}

	if path := c.FullPath(); path != "" {
		return method + " " + path
	}

	return method
}

// Filter filters an [net/http.Request] based on content of a [gin.Context].
type Filter func(*gin.Context) bool

// SpanNameFormatter is used by `WithSpanNameFormatter` to customize the request's span name.
type SpanNameFormatter func(*gin.Context) string

// MetricAttributeFn is used to extract additional attributes from the gin.Context
// and return them as a slice of attribute.KeyValue.
type MetricAttributeFn func(*gin.Context) []attribute.KeyValue

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

// WithFilter adds a gin filter to the list of filters used by the handler.
func WithFilter(f ...Filter) Option {
	return optionFunc(func(c *config) {
		c.Filters = append(c.Filters, f...)
	})
}

// WithSpanNameFormatter takes a function that will be called on every
// request and the returned string will become the Span Name.
func WithSpanNameFormatter(f SpanNameFormatter) Option {
	return optionFunc(func(c *config) {
		c.SpanNameFormatter = f
	})
}

// WithMeterProvider specifies a meter provider to use for creating a meter.
// If none is specified, the global provider is used.
func WithMeterProvider(mp metric.MeterProvider) Option {
	return optionFunc(func(c *config) {
		c.MeterProvider = mp
	})
}

// WithMetricAttributeFn specifies a function that extracts additional attributes from the gin.Context
// and returns them as a slice of attribute.KeyValue.
//
// If attributes are duplicated between this method and `WithMetricAttributeFn`, the attributes in this method will be used.
func WithMetricAttributeFn(f MetricAttributeFn) Option {
	return optionFunc(func(c *config) {
		c.MetricAttributeFn = f
	})
}
