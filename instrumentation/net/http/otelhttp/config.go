// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelhttp // import "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

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

// sharedConfig provides configuration fields shared between handler and transport.
// It contains settings for tracing, metrics, propagation, and filtering.
type sharedConfig struct {
	Tracer            trace.Tracer
	Meter             metric.Meter
	Propagators       propagation.TextMapPropagator
	SpanStartOptions  []trace.SpanStartOption
	Filters           []Filter
	SpanNameFormatter func(string, *http.Request) string

	TracerProvider     trace.TracerProvider
	MeterProvider      metric.MeterProvider
	MetricAttributesFn func(*http.Request) []attribute.KeyValue
}

// handlerConfig extends sharedConfig with HTTP handler-specific configuration.
type handlerConfig struct {
	sharedConfig
	ServerName       string
	PublicEndpointFn func(*http.Request) bool
	ReadEvent        bool
	WriteEvent       bool
}

// transportConfig extends sharedConfig with HTTP transport-specific configuration.
type transportConfig struct {
	sharedConfig
	ClientTrace func(context.Context) *httptrace.ClientTrace
}

// config is the legacy configuration struct used by the deprecated Option interface.

// Deprecated: Use handlerConfig or transportConfig instead. This struct will be removed in a future release.
type config struct {
	ServerName        string
	Tracer            trace.Tracer
	Meter             metric.Meter
	Propagators       propagation.TextMapPropagator
	SpanStartOptions  []trace.SpanStartOption
	PublicEndpointFn  func(*http.Request) bool
	ReadEvent         bool
	WriteEvent        bool
	Filters           []Filter
	SpanNameFormatter func(string, *http.Request) string
	ClientTrace       func(context.Context) *httptrace.ClientTrace

	TracerProvider     trace.TracerProvider
	MeterProvider      metric.MeterProvider
	MetricAttributesFn func(*http.Request) []attribute.KeyValue
}

// HandlerOption defines an option for configuring an HTTP handler.
type HandlerOption interface {
	Option
	applyHandler(*handlerConfig)
}

// TransportOption defines an option for configuring an HTTP transport.
type TransportOption interface {
	Option
	applyTransport(*transportConfig)
}

// CombinedOption is a combination of [HandlerOption[ and [TransportOption].
// It is implemented by options that are applicable for both [NewHandler] and [NewTransport].
type CombinedOption interface {
	HandlerOption
	TransportOption
}

// Option defines the legacy interface for setting optional configuration properties.
//
// Deprecated: Use [HandlerOption] or [TransportOption] instead. This interface will be removed in a future release.
type Option interface {
	apply(*config)
}

// sharedOptionFunc implements HandlerOption, TransportOption, and Option.
// It allows a single function to handle configuration for all three config types.
type sharedOptionFunc struct {
	handlerFunc   func(*handlerConfig)
	transportFunc func(*transportConfig)
	legacyFunc    func(*config)
}

// handlerOptionFunc implements HandlerOption and Option.
// It allows a function to handle handler and legacy configuration.
type handlerOptionFunc struct {
	handlerFunc func(*handlerConfig)
	legacyFunc  func(*config)
}

// transportOptionFunc implements TransportOption and Option.
// It allows a function to handle transport and legacy configuration.
type transportOptionFunc struct {
	transportFunc func(*transportConfig)
	legacyFunc    func(*config)
}

// applyHandler applies the handler function to the handlerConfig.
func (f sharedOptionFunc) applyHandler(c *handlerConfig) { f.handlerFunc(c) }

// applyTransport applies the transport function to the transportConfig.
func (f sharedOptionFunc) applyTransport(c *transportConfig) { f.transportFunc(c) }

// apply applies the legacy function to the deprecated config.
func (f sharedOptionFunc) apply(c *config) { f.legacyFunc(c) }

// applyHandler applies the handler function to the handlerConfig.
func (f handlerOptionFunc) applyHandler(c *handlerConfig) { f.handlerFunc(c) }

// apply applies the legacy function to the deprecated config.
func (f handlerOptionFunc) apply(c *config) { f.legacyFunc(c) }

// applyTransport applies the transport function to the transportConfig.
func (f transportOptionFunc) applyTransport(c *transportConfig) { f.transportFunc(c) }

// apply applies the legacy function to the deprecated config.
func (f transportOptionFunc) apply(c *config) { f.legacyFunc(c) }

// newHandlerConfig creates a new handlerConfig from the provided options.
// It initializes default values and applies each option in order.
func newHandlerConfig(opts ...Option) *handlerConfig {
	c := &handlerConfig{
		sharedConfig: sharedConfig{
			Propagators:   otel.GetTextMapPropagator(),
			MeterProvider: otel.GetMeterProvider(),
		},
	}
	for _, opt := range opts {
		if to, ok := opt.(HandlerOption); ok {
			to.applyHandler(c)
		} else {
			c.applyLegacy(opt)
		}
	}

	if c.TracerProvider != nil {
		c.Tracer = newTracer(c.TracerProvider)
	}

	c.Meter = c.MeterProvider.Meter(
		ScopeName,
		metric.WithInstrumentationVersion(Version),
	)

	return c
}

// applyLegacy applies a legacy Option to the handlerConfig by converting
// between config types and back, ensuring backward compatibility.
func (c *handlerConfig) applyLegacy(opt Option) {
	legacy := &config{
		ServerName:         c.ServerName,
		Tracer:             c.Tracer,
		Meter:              c.Meter,
		Propagators:        c.Propagators,
		SpanStartOptions:   c.SpanStartOptions,
		PublicEndpointFn:   c.PublicEndpointFn,
		ReadEvent:          c.ReadEvent,
		WriteEvent:         c.WriteEvent,
		Filters:            c.Filters,
		SpanNameFormatter:  c.SpanNameFormatter,
		TracerProvider:     c.TracerProvider,
		MeterProvider:      c.MeterProvider,
		MetricAttributesFn: c.MetricAttributesFn,
	}
	opt.apply(legacy)
	c.ServerName = legacy.ServerName
	c.Tracer = legacy.Tracer
	c.Meter = legacy.Meter
	c.Propagators = legacy.Propagators
	c.SpanStartOptions = legacy.SpanStartOptions
	c.PublicEndpointFn = legacy.PublicEndpointFn
	c.ReadEvent = legacy.ReadEvent
	c.WriteEvent = legacy.WriteEvent
	c.Filters = legacy.Filters
	c.SpanNameFormatter = legacy.SpanNameFormatter
	c.TracerProvider = legacy.TracerProvider
	c.MeterProvider = legacy.MeterProvider
	c.MetricAttributesFn = legacy.MetricAttributesFn
}

// newTransportConfig creates a new transportConfig from the provided options.
// It initializes default values and applies each option in order.
func newTransportConfig(opts ...Option) *transportConfig {
	c := &transportConfig{
		sharedConfig: sharedConfig{
			Propagators:   otel.GetTextMapPropagator(),
			MeterProvider: otel.GetMeterProvider(),
		},
	}
	for _, opt := range opts {
		if to, ok := opt.(TransportOption); ok {
			to.applyTransport(c)
		} else {
			c.applyLegacy(opt)
		}
	}

	if c.TracerProvider != nil {
		c.Tracer = newTracer(c.TracerProvider)
	}

	c.Meter = c.MeterProvider.Meter(
		ScopeName,
		metric.WithInstrumentationVersion(Version),
	)

	return c
}

// applyLegacy applies a legacy Option to the transportConfig by converting
// between config types and back, ensuring backward compatibility.
func (c *transportConfig) applyLegacy(opt Option) {
	legacy := &config{
		Tracer:             c.Tracer,
		Meter:              c.Meter,
		Propagators:        c.Propagators,
		SpanStartOptions:   c.SpanStartOptions,
		Filters:            c.Filters,
		SpanNameFormatter:  c.SpanNameFormatter,
		ClientTrace:        c.ClientTrace,
		TracerProvider:     c.TracerProvider,
		MeterProvider:      c.MeterProvider,
		MetricAttributesFn: c.MetricAttributesFn,
	}
	opt.apply(legacy)
	c.Tracer = legacy.Tracer
	c.Meter = legacy.Meter
	c.Propagators = legacy.Propagators
	c.SpanStartOptions = legacy.SpanStartOptions
	c.Filters = legacy.Filters
	c.SpanNameFormatter = legacy.SpanNameFormatter
	c.ClientTrace = legacy.ClientTrace
	c.TracerProvider = legacy.TracerProvider
	c.MeterProvider = legacy.MeterProvider
	c.MetricAttributesFn = legacy.MetricAttributesFn
}

// WithTracerProvider specifies a tracer provider to use for creating a tracer.
// If none is specified, the global provider is used.
func WithTracerProvider(provider trace.TracerProvider) CombinedOption {
	return sharedOptionFunc{
		handlerFunc: func(c *handlerConfig) {
			if provider != nil {
				c.TracerProvider = provider
			}
		},
		transportFunc: func(c *transportConfig) {
			if provider != nil {
				c.TracerProvider = provider
			}
		},
		legacyFunc: func(c *config) {
			if provider != nil {
				c.TracerProvider = provider
			}
		},
	}
}

// WithMeterProvider specifies a meter provider to use for creating a meter.
// If none is specified, the global provider is used.
func WithMeterProvider(provider metric.MeterProvider) CombinedOption {
	return sharedOptionFunc{
		handlerFunc: func(c *handlerConfig) {
			if provider != nil {
				c.MeterProvider = provider
			}
		},
		transportFunc: func(c *transportConfig) {
			if provider != nil {
				c.MeterProvider = provider
			}
		},
		legacyFunc: func(c *config) {
			if provider != nil {
				c.MeterProvider = provider
			}
		},
	}
}

// WithPublicEndpointFn runs with every request, and allows conditionally
// configuring the Handler to link the span with an incoming span context. If
// this option is not provided or returns false, then the association is a
// child association instead of a link.
func WithPublicEndpointFn(fn func(*http.Request) bool) HandlerOption {
	return handlerOptionFunc{
		handlerFunc: func(c *handlerConfig) { c.PublicEndpointFn = fn },
		legacyFunc:  func(c *config) { c.PublicEndpointFn = fn },
	}
}

// WithPropagators configures specific propagators. If this
// option isn't specified, then the global TextMapPropagator is used.
func WithPropagators(ps propagation.TextMapPropagator) CombinedOption {
	return sharedOptionFunc{
		handlerFunc: func(c *handlerConfig) {
			if ps != nil {
				c.Propagators = ps
			}
		},
		transportFunc: func(c *transportConfig) {
			if ps != nil {
				c.Propagators = ps
			}
		},
		legacyFunc: func(c *config) {
			if ps != nil {
				c.Propagators = ps
			}
		},
	}
}

// WithSpanOptions configures an additional set of
// trace.SpanOptions, which are applied to each new span.
func WithSpanOptions(opts ...trace.SpanStartOption) CombinedOption {
	return sharedOptionFunc{
		handlerFunc: func(c *handlerConfig) {
			if opts != nil {
				c.SpanStartOptions = append(c.SpanStartOptions, opts...)
			}
		},
		transportFunc: func(c *transportConfig) {
			if opts != nil {
				c.SpanStartOptions = append(c.SpanStartOptions, opts...)
			}
		},
		legacyFunc: func(c *config) {
			if opts != nil {
				c.SpanStartOptions = append(c.SpanStartOptions, opts...)
			}
		},
	}
}

// WithFilter adds a filter to the list of filters used by the handler.
// If any filter indicates to exclude a request then the request will not be
// traced. All filters must allow a request to be traced for a Span to be created.
// If no filters are provided then all requests are traced.
// Filters will be invoked for each processed request, it is advised to make them
// simple and fast.
func WithFilter(f Filter) CombinedOption {
	return sharedOptionFunc{
		handlerFunc:   func(c *handlerConfig) { c.Filters = append(c.Filters, f) },
		transportFunc: func(c *transportConfig) { c.Filters = append(c.Filters, f) },
		legacyFunc:    func(c *config) { c.Filters = append(c.Filters, f) },
	}
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
// (span.AddEvent) on spans. By default only summary attributes are added at the
// end of the request.
//
// Valid events are:
//   - ReadEvents: Record the number of bytes read after every http.Request.Body.Read
//     using the ReadBytesKey
//   - WriteEvents: Record the number of bytes written after every http.ResponeWriter.Write
//     using the WriteBytesKey
func WithMessageEvents(events ...Event) HandlerOption {
	return handlerOptionFunc{
		handlerFunc: func(c *handlerConfig) {
			for _, e := range events {
				switch e {
				case ReadEvents:
					c.ReadEvent = true
				case WriteEvents:
					c.WriteEvent = true
				}
			}
		},
		legacyFunc: func(c *config) {
			for _, e := range events {
				switch e {
				case ReadEvents:
					c.ReadEvent = true
				case WriteEvents:
					c.WriteEvent = true
				}
			}
		},
	}
}

// WithSpanNameFormatter takes a function that will be called on every
// request and the returned string will become the Span Name.
//
// When using [http.ServeMux] (or any middleware that sets the Pattern of [http.Request]),
// the span name formatter will run twice. Once when the span is created, and
// second time after the middleware, so the pattern can be used.
func WithSpanNameFormatter(f func(operation string, r *http.Request) string) CombinedOption {
	return sharedOptionFunc{
		handlerFunc: func(c *handlerConfig) {
			c.SpanNameFormatter = f
		},
		transportFunc: func(c *transportConfig) {
			c.SpanNameFormatter = f
		},
		legacyFunc: func(c *config) {
			c.SpanNameFormatter = f
		},
	}
}

// WithClientTrace takes a function that returns client trace instance that will be
// applied to the requests sent through the otelhttp Transport.
func WithClientTrace(f func(context.Context) *httptrace.ClientTrace) TransportOption {
	return transportOptionFunc{
		transportFunc: func(c *transportConfig) { c.ClientTrace = f },
		legacyFunc:    func(c *config) { c.ClientTrace = f },
	}
}

// WithServerName returns an Option that sets the name of the (virtual) server
// handling requests.
func WithServerName(server string) HandlerOption {
	return handlerOptionFunc{
		handlerFunc: func(c *handlerConfig) { c.ServerName = server },
		legacyFunc:  func(c *config) { c.ServerName = server },
	}
}

// WithMetricAttributesFn returns an Option to set a function that maps an HTTP request to a slice of attribute.KeyValue.
// These attributes will be included in metrics for every request.
//
// Deprecated: WithMetricAttributesFn is deprecated and will be removed in a
// future release. Use [Labeler] instead.
func WithMetricAttributesFn(metricAttributesFn func(r *http.Request) []attribute.KeyValue) CombinedOption {
	return sharedOptionFunc{
		handlerFunc: func(c *handlerConfig) {
			c.MetricAttributesFn = metricAttributesFn
		},
		transportFunc: func(c *transportConfig) {
			c.MetricAttributesFn = metricAttributesFn
		},
		legacyFunc: func(c *config) {
			c.MetricAttributesFn = metricAttributesFn
		},
	}
}
