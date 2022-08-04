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

package otelecho // import "go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"

import (
	"context"
	"net/http"
	"net/http/httptrace"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// WithTracerProvider specifies a tracer provider to use for creating a tracer.
// If none is specified, the global provider is used.
func WithTracerProvider(provider trace.TracerProvider) Option {
	return optionFunc(func(cfg *config) {
		cfg.otelhttpOptions = append(cfg.otelhttpOptions, otelhttp.WithTracerProvider(provider))
	})
}

// WithMeterProvider specifies a meter provider to use for creating a meter.
// If none is specified, the global provider is used.
func WithMeterProvider(provider metric.MeterProvider) Option {
	return optionFunc(func(cfg *config) {
		cfg.otelhttpOptions = append(cfg.otelhttpOptions, otelhttp.WithMeterProvider(provider))
	})
}

// WithPublicEndpoint configures the Handler to link the span with an incoming
// span context. If this option is not provided, then the association is a child
// association instead of a link.
func WithPublicEndpoint() Option {
	return optionFunc(func(c *config) {
		c.otelhttpOptions = append(c.otelhttpOptions, otelhttp.WithPublicEndpoint())
	})
}

// WithPropagators configures specific propagators. If this
// option isn't specified, then the global TextMapPropagator is used.
func WithPropagators(ps propagation.TextMapPropagator) Option {
	return optionFunc(func(c *config) {
		c.otelhttpOptions = append(c.otelhttpOptions, otelhttp.WithPropagators(ps))
	})
}

// WithSpanOptions configures an additional set of
// trace.SpanOptions, which are applied to each new span.
func WithSpanOptions(opts ...trace.SpanStartOption) Option {
	return optionFunc(func(c *config) {
		c.otelhttpOptions = append(c.otelhttpOptions, otelhttp.WithSpanOptions(opts...))
	})
}

// WithFilter adds a filter to the list of filters used by the handler.
// If any filter indicates to exclude a request then the request will not be
// traced. All filters must allow a request to be traced for a Span to be created.
// If no filters are provided then all requests are traced.
// Filters will be invoked for each processed request, it is advised to make them
// simple and fast.
func WithFilter(f func(*http.Request) bool) Option {
	return optionFunc(func(c *config) {
		c.hasFilter = true
		c.otelhttpOptions = append(c.otelhttpOptions, otelhttp.WithFilter(f))
	})
}

type event int

// Different types of events that can be recorded, see WithMessageEvents.
const (
	ReadEvents event = iota
	WriteEvents
)

// WithMessageEvents configures the Handler to record the specified events
// (span.AddEvent) on spans. By default only summary attributes are added at the
// end of the request.
//
// Valid events are:
//     * ReadEvents: Record the number of bytes read after every http.Request.Body.Read
//       using the ReadBytesKey
//     * WriteEvents: Record the number of bytes written after every http.ResponeWriter.Write
//       using the WriteBytesKey
func WithMessageEvents(events ...event) Option {
	return optionFunc(func(c *config) {
		for _, e := range events {
			switch e {
			case ReadEvents:
				c.otelhttpOptions = append(c.otelhttpOptions, otelhttp.WithMessageEvents(otelhttp.ReadEvents))
			case WriteEvents:
				c.otelhttpOptions = append(c.otelhttpOptions, otelhttp.WithMessageEvents(otelhttp.WriteEvents))
			}
		}
	})
}

// WithSpanNameFormatter takes a function that will be called on every
// request and the returned string will become the Span Name.
func WithSpanNameFormatter(f func(operation string, r *http.Request) string) Option {
	return optionFunc(func(c *config) {
		c.hasSpanNameFormatter = true
		c.otelhttpOptions = append(c.otelhttpOptions, otelhttp.WithSpanNameFormatter(f))
	})
}

// WithClientTrace takes a function that returns client trace instance that will be
// applied to the requests sent through the otelhttp Transport.
func WithClientTrace(f func(context.Context) *httptrace.ClientTrace) Option {
	return optionFunc(func(c *config) {
		c.otelhttpOptions = append(c.otelhttpOptions, otelhttp.WithClientTrace(f))
	})
}
