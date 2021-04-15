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

package otelhttp

import (
	"net/http"

	"go.opentelemetry.io/contrib"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	instrumentationName = "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Config represents the configuration options available for the http.Handler
// and http.Transport types.
type Config struct {
	tracer            trace.Tracer
	meter             metric.Meter
	propagators       propagation.TextMapPropagator
	spanStartOptions  []trace.SpanOption
	readEvent         bool
	writeEvent        bool
	filters           []Filter
	spanNameFormatter func(string, *http.Request) string

	tracerProvider trace.TracerProvider
	meterProvider  metric.MeterProvider
}

// Option Interface used for setting *optional* config properties
type Option interface {
	Apply(*Config)
}

// OptionFunc provides a convenience wrapper for simple Options
// that can be represented as functions.
type OptionFunc func(*Config)

func (o OptionFunc) Apply(c *Config) {
	o(c)
}

// newConfig creates a new config struct and applies opts to it.
func newConfig(opts ...Option) *Config {
	c := &Config{
		propagators:    otel.GetTextMapPropagator(),
		tracerProvider: otel.GetTracerProvider(),
		meterProvider:  global.GetMeterProvider(),
	}
	for _, opt := range opts {
		opt.Apply(c)
	}

	c.tracer = c.tracerProvider.Tracer(
		instrumentationName,
		trace.WithInstrumentationVersion(contrib.SemVersion()),
	)
	c.meter = c.meterProvider.Meter(
		instrumentationName,
		metric.WithInstrumentationVersion(contrib.SemVersion()),
	)

	return c
}

// WithTracerProvider specifies a tracer provider to use for creating a tracer.
// If none is specified, the global provider is used.
func WithTracerProvider(provider trace.TracerProvider) Option {
	return OptionFunc(func(cfg *Config) {
		cfg.tracerProvider = provider
	})
}

// WithMeterProvider specifies a meter provider to use for creating a meter.
// If none is specified, the global provider is used.
func WithMeterProvider(provider metric.MeterProvider) Option {
	return OptionFunc(func(cfg *Config) {
		cfg.meterProvider = provider
	})
}

// WithPublicEndpoint configures the Handler to link the span with an incoming
// span context. If this option is not provided, then the association is a child
// association instead of a link.
func WithPublicEndpoint() Option {
	return OptionFunc(func(c *Config) {
		c.spanStartOptions = append(c.spanStartOptions, trace.WithNewRoot())
	})
}

// WithPropagators configures specific propagators. If this
// option isn't specified then
func WithPropagators(ps propagation.TextMapPropagator) Option {
	return OptionFunc(func(c *Config) {
		c.propagators = ps
	})
}

// WithSpanOptions configures an additional set of
// trace.SpanOptions, which are applied to each new span.
func WithSpanOptions(opts ...trace.SpanOption) Option {
	return OptionFunc(func(c *Config) {
		c.spanStartOptions = append(c.spanStartOptions, opts...)
	})
}

// WithFilter adds a filter to the list of filters used by the handler.
// If any filter indicates to exclude a request then the request will not be
// traced. All filters must allow a request to be traced for a Span to be created.
// If no filters are provided then all requests are traced.
// Filters will be invoked for each processed request, it is advised to make them
// simple and fast.
func WithFilter(f Filter) Option {
	return OptionFunc(func(c *Config) {
		c.filters = append(c.filters, f)
	})
}

type event int

// Different types of events that can be recorded, see WithMessageEvents
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
	return OptionFunc(func(c *Config) {
		for _, e := range events {
			switch e {
			case ReadEvents:
				c.readEvent = true
			case WriteEvents:
				c.writeEvent = true
			}
		}
	})
}

// WithSpanNameFormatter takes a function that will be called on every
// request and the returned string will become the Span Name
func WithSpanNameFormatter(f func(operation string, r *http.Request) string) Option {
	return OptionFunc(func(c *Config) {
		c.spanNameFormatter = f
	})
}
