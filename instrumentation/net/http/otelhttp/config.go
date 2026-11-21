// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelhttp

import (
	"context"
	"net/http"
	"net/http/httptrace"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// ScopeName is the instrumentation scope name.
const ScopeName = "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

type baseConfig struct {
	Tracer             trace.Tracer
	Meter              metric.Meter
	Propagators        propagation.TextMapPropagator
	SpanStartOptions   []trace.SpanStartOption
	Filters            []Filter
	SpanNameFormatter  func(string, *http.Request) string
	TracerProvider     trace.TracerProvider
	MeterProvider      metric.MeterProvider
	MetricAttributesFn func(*http.Request) []attribute.KeyValue
}

func (c *baseConfig) init() {
	// Tracer is only initialized if manually specified. Otherwise, can be passed with the tracing context.
	if c.TracerProvider != nil {
		c.Tracer = newTracer(c.TracerProvider)
	}

	c.Meter = c.MeterProvider.Meter(
		ScopeName,
		metric.WithInstrumentationVersion(Version),
	)
}

// handlerConfig represents the configuration options available for HTTP handlers.
type handlerConfig struct {
	baseConfig

	ServerName       string
	PublicEndpointFn func(*http.Request) bool
	ReadEvent        bool
	WriteEvent       bool
	RouteTracking    bool
}

// transportConfig represents the configuration options available for the transport
// returned by NewTransport.
type transportConfig struct {
	baseConfig

	ClientTrace func(context.Context) *httptrace.ClientTrace
}

// HandlerOption interface used for setting optional handler config properties.
type HandlerOption interface {
	applyHandler(*handlerConfig)
}

// TransportOption interface used for setting optional transport config properties.
type TransportOption interface {
	applyTransport(*transportConfig)
}

// Option interface used for setting optional config properties shared by both
// handlers and transports.
type Option interface {
	HandlerOption
	TransportOption
}

type optionFunc func(*baseConfig)

func (o optionFunc) applyHandler(c *handlerConfig) {
	o(&c.baseConfig)
}

func (o optionFunc) applyTransport(c *transportConfig) {
	o(&c.baseConfig)
}

type handlerOptionFunc func(*handlerConfig)

func (o handlerOptionFunc) applyHandler(c *handlerConfig) {
	o(c)
}

type transportOptionFunc func(*transportConfig)

func (o transportOptionFunc) applyTransport(c *transportConfig) {
	o(c)
}

// newHandlerConfig creates a new handlerConfig struct and applies opts to it.
func newHandlerConfig(opts ...HandlerOption) *handlerConfig {
	c := &handlerConfig{
		baseConfig: baseConfig{
			Propagators:   otel.GetTextMapPropagator(),
			MeterProvider: otel.GetMeterProvider(),
		},
	}
	for _, opt := range opts {
		opt.applyHandler(c)
	}
	c.init()
	return c
}

// newTransportConfig creates a new transportConfig struct and applies opts to it.
func newTransportConfig(opts ...TransportOption) *transportConfig {
	c := &transportConfig{
		baseConfig: baseConfig{
			Propagators:   otel.GetTextMapPropagator(),
			MeterProvider: otel.GetMeterProvider(),
		},
	}
	for _, opt := range opts {
		opt.applyTransport(c)
	}
	c.init()
	return c
}

// WithTracerProvider specifies a tracer provider to use for creating a tracer.
// If none is specified, the global provider is used.
func WithTracerProvider(provider trace.TracerProvider) Option {
	return optionFunc(func(cfg *baseConfig) {
		if provider != nil {
			cfg.TracerProvider = provider
		}
	})
}

// WithMeterProvider specifies a meter provider to use for creating a meter.
// If none is specified, the global provider is used.
func WithMeterProvider(provider metric.MeterProvider) Option {
	return optionFunc(func(cfg *baseConfig) {
		if provider != nil {
			cfg.MeterProvider = provider
		}
	})
}

// WithPublicEndpointFn runs with every request, and allows conditionally
// configuring the Handler to link the span with an incoming span context. If
// this option is not provided or returns false, then the association is a
// child association instead of a link.
func WithPublicEndpointFn(fn func(*http.Request) bool) HandlerOption {
	return handlerOptionFunc(func(c *handlerConfig) {
		c.PublicEndpointFn = fn
	})
}

// WithPropagators configures specific propagators. If this
// option isn't specified, then the global TextMapPropagator is used.
func WithPropagators(ps propagation.TextMapPropagator) Option {
	return optionFunc(func(c *baseConfig) {
		if ps != nil {
			c.Propagators = ps
		}
	})
}

// WithSpanOptions configures an additional set of
// trace.SpanOptions, which are applied to each new span.
func WithSpanOptions(opts ...trace.SpanStartOption) Option {
	return optionFunc(func(c *baseConfig) {
		c.SpanStartOptions = append(c.SpanStartOptions, opts...)
	})
}

// WithFilter adds a filter to the list of filters used by the handler or transport.
// If any filter indicates to exclude a request then the request will not be
// traced. All filters must allow a request to be traced for a Span to be created.
// If no filters are provided then all requests are traced.
// Filters will be invoked for each processed request, it is advised to make them
// simple and fast.
func WithFilter(f Filter) Option {
	return optionFunc(func(c *baseConfig) {
		c.Filters = append(c.Filters, f)
	})
}

// Event represents message event types for [WithMessageEvents].
type Event int

// Different types of events that can be recorded, see WithMessageEvents.
const (
	unspecifiedEvents Event = iota
	ReadEvents
	WriteEvents
)

// WithMessageEvents configures the Handler to record the specified events
// (span.AddEvent) on spans. By default, no read or write events are recorded.
//
// Valid events are:
//   - ReadEvents: Record a "read" span event after every http.Request.Body.Read.
//   - WriteEvents: Record a "write" span event after every http.ResponseWriter.Write.
//
// The events do not include per-call byte-count attributes. Total request and
// response body sizes are recorded on the span according to HTTP semantic
// conventions.
func WithMessageEvents(events ...Event) HandlerOption {
	return handlerOptionFunc(func(c *handlerConfig) {
		for _, e := range events {
			switch e {
			case ReadEvents:
				c.ReadEvent = true
			case WriteEvents:
				c.WriteEvent = true
			}
		}
	})
}

// WithSpanNameFormatter takes a function that will be called on every
// request and the returned string will become the Span Name.
//
// When using [http.ServeMux] (or any middleware that sets the Pattern of [http.Request]),
// the span name formatter will run twice. Once when the span is created, and
// second time after the middleware, so the pattern can be used.
func WithSpanNameFormatter(f func(operation string, r *http.Request) string) Option {
	return optionFunc(func(c *baseConfig) {
		c.SpanNameFormatter = f
	})
}

// WithClientTrace takes a function that returns client trace instance that will be
// applied to the requests sent through the transport returned by NewTransport.
func WithClientTrace(f func(context.Context) *httptrace.ClientTrace) TransportOption {
	return transportOptionFunc(func(c *transportConfig) {
		c.ClientTrace = f
	})
}

// WithServerName returns a HandlerOption that sets the name of the (virtual) server
// handling requests.
//
// It is the only source of the server.address and server.port attributes on
// server metrics. The client-supplied Host header is never used for them.
func WithServerName(server string) HandlerOption {
	return handlerOptionFunc(func(c *handlerConfig) {
		c.ServerName = server
	})
}

// WithMetricAttributesFn returns an Option to set a function that maps an HTTP request to a slice of attribute.KeyValue.
// These attributes will be included in metrics for every request.
//
// Deprecated: WithMetricAttributesFn is deprecated and will be removed in a
// future release. Use [Labeler] instead.
func WithMetricAttributesFn(metricAttributesFn func(r *http.Request) []attribute.KeyValue) Option {
	return optionFunc(func(c *baseConfig) {
		c.MetricAttributesFn = metricAttributesFn
	})
}

// WithRouteTracking enables [RouteMiddleware] to record matched routes on the
// server span and metrics created by [NewHandler] or [NewMiddleware] when
// intervening middleware copies the request.
func WithRouteTracking() HandlerOption {
	return handlerOptionFunc(func(c *handlerConfig) {
		c.RouteTracking = true
	})
}
