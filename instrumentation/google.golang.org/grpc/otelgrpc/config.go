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

package otelgrpc // import "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	// ScopeName is the instrumentation scope name.
	ScopeName = "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

// Filter is a predicate used to determine whether a given request in
// interceptor info should be traced. A Filter must return true if
// the request should be traced.
type Filter func(*InterceptorInfo) bool

// config is a group of options for this instrumentation.
type config struct {
	Filter           Filter
	Propagators      propagation.TextMapPropagator
	TracerProvider   trace.TracerProvider
	MeterProvider    metric.MeterProvider
	SpanStartOptions []trace.SpanStartOption

	ReceivedEvent bool
	SentEvent     bool

	tracer trace.Tracer
	meter  metric.Meter

	rpcDuration        metric.Float64Histogram
	rpcRequestSize     metric.Int64Histogram
	rpcResponseSize    metric.Int64Histogram
	rpcRequestsPerRPC  metric.Int64Histogram
	rpcResponsesPerRPC metric.Int64Histogram
}

// Option applies an option value for a config.
type Option func(*config)

// newConfig returns a config configured with all the passed Options.
func newConfig(opts []Option, role string) *config {
	c := &config{
		Propagators:    otel.GetTextMapPropagator(),
		TracerProvider: otel.GetTracerProvider(),
		MeterProvider:  otel.GetMeterProvider(),
	}
	for _, fn := range opts {
		fn(c)
	}

	c.tracer = c.TracerProvider.Tracer(
		ScopeName,
		trace.WithInstrumentationVersion(SemVersion()),
	)

	c.meter = c.MeterProvider.Meter(
		ScopeName,
		metric.WithInstrumentationVersion(Version()),
		metric.WithSchemaURL(semconv.SchemaURL),
	)

	var err error
	c.rpcDuration, err = c.meter.Float64Histogram("rpc."+role+".duration",
		metric.WithDescription("Measures the duration of inbound RPC."),
		metric.WithUnit("ms"))
	if err != nil {
		otel.Handle(err)
	}

	c.rpcRequestSize, err = c.meter.Int64Histogram("rpc."+role+".request.size",
		metric.WithDescription("Measures size of RPC request messages (uncompressed)."),
		metric.WithUnit("By"))
	if err != nil {
		otel.Handle(err)
	}

	c.rpcResponseSize, err = c.meter.Int64Histogram("rpc."+role+".response.size",
		metric.WithDescription("Measures size of RPC response messages (uncompressed)."),
		metric.WithUnit("By"))
	if err != nil {
		otel.Handle(err)
	}

	c.rpcRequestsPerRPC, err = c.meter.Int64Histogram("rpc."+role+".requests_per_rpc",
		metric.WithDescription("Measures the number of messages received per RPC. Should be 1 for all non-streaming RPCs."),
		metric.WithUnit("{count}"))
	if err != nil {
		otel.Handle(err)
	}

	c.rpcResponsesPerRPC, err = c.meter.Int64Histogram("rpc."+role+".responses_per_rpc",
		metric.WithDescription("Measures the number of messages received per RPC. Should be 1 for all non-streaming RPCs."),
		metric.WithUnit("{count}"))
	if err != nil {
		otel.Handle(err)
	}

	return c
}

// WithPropagators returns an Option to use the Propagators when extracting
// and injecting trace context from requests.
func WithPropagators(p propagation.TextMapPropagator) Option {
	return func(c *config) {
		if p != nil {
			c.Propagators = p
		}
	}
}

// WithInterceptorFilter returns an Option to use the request filter.
//
// Deprecated: Use stats handlers instead.
func WithInterceptorFilter(f Filter) Option {
	return func(c *config) {
		if f != nil {
			c.Filter = f
		}
	}
}

// WithTracerProvider returns an Option to use the TracerProvider when
// creating a Tracer.
func WithTracerProvider(tp trace.TracerProvider) Option {
	return func(c *config) {
		if tp != nil {
			c.TracerProvider = tp
		}
	}
}

// WithMeterProvider returns an Option to use the MeterProvider when
// creating a Meter. If this option is not provide the global MeterProvider will be used.
func WithMeterProvider(mp metric.MeterProvider) Option {
	return func(c *config) {
		if mp != nil {
			c.MeterProvider = mp
		}
	}
}

// Event type that can be recorded, see WithMessageEvents.
type Event int

// Different types of events that can be recorded, see WithMessageEvents.
const (
	ReceivedEvents Event = iota
	SentEvents
)

// WithMessageEvents configures the Handler to record the specified events
// (span.AddEvent) on spans. By default only summary attributes are added at the
// end of the request.
//
// Valid events are:
//   - ReceivedEvents: Record the number of bytes read after every gRPC read operation.
//   - SentEvents: Record the number of bytes written after every gRPC write operation.
func WithMessageEvents(events ...Event) Option {
	return func(c *config) {
		for _, e := range events {
			switch e {
			case ReceivedEvents:
				c.ReceivedEvent = true
			case SentEvents:
				c.SentEvent = true
			}
		}
	}
}

// WithSpanOptions configures an additional set of
// trace.SpanOptions, which are applied to each new span.
func WithSpanOptions(opts ...trace.SpanStartOption) Option {
	return func(c *config) {
		c.SpanStartOptions = append(c.SpanStartOptions, opts...)
	}
}
