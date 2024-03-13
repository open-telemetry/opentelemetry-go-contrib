// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgrpc // import "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	// ScopeName is the instrumentation scope name.
	ScopeName = "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	// GRPCStatusCodeKey is convention for numeric status code of a gRPC request.
	GRPCStatusCodeKey = attribute.Key("rpc.grpc.status_code")
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

	rpcDurationAdditionalOptions        []metric.Float64HistogramOption
	rpcRequestSizeAdditionalOptions     []metric.Int64HistogramOption
	rpcResponseSizeAdditionalOptions    []metric.Int64HistogramOption
	rpcRequestsPerRPCAdditionalOptions  []metric.Int64HistogramOption
	rpcResponsesPerRPCAdditionalOptions []metric.Int64HistogramOption

	rpcDuration        metric.Float64Histogram
	rpcRequestSize     metric.Int64Histogram
	rpcResponseSize    metric.Int64Histogram
	rpcRequestsPerRPC  metric.Int64Histogram
	rpcResponsesPerRPC metric.Int64Histogram
}

// Option applies an option value for a config.
type Option interface {
	apply(*config)
}

// newConfig returns a config configured with all the passed Options.
func newConfig(opts []Option, role string) *config {
	c := &config{
		Propagators:    otel.GetTextMapPropagator(),
		TracerProvider: otel.GetTracerProvider(),
		MeterProvider:  otel.GetMeterProvider(),
	}
	for _, o := range opts {
		o.apply(c)
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
		append([]metric.Float64HistogramOption{
			metric.WithDescription("Measures the duration of inbound RPC."),
			metric.WithUnit("ms"),
		}, c.rpcDurationAdditionalOptions...)...,
	)
	if err != nil {
		otel.Handle(err)
		if c.rpcDuration == nil {
			c.rpcDuration = noop.Float64Histogram{}
		}
	}

	c.rpcRequestSize, err = c.meter.Int64Histogram("rpc."+role+".request.size",
		append([]metric.Int64HistogramOption{
			metric.WithDescription("Measures size of RPC request messages (uncompressed)."),
			metric.WithUnit("By"),
		}, c.rpcRequestSizeAdditionalOptions...)...)
	if err != nil {
		otel.Handle(err)
		if c.rpcRequestSize == nil {
			c.rpcRequestSize = noop.Int64Histogram{}
		}
	}

	c.rpcResponseSize, err = c.meter.Int64Histogram("rpc."+role+".response.size",
		append([]metric.Int64HistogramOption{
			metric.WithDescription("Measures size of RPC response messages (uncompressed)."),
			metric.WithUnit("By"),
		}, c.rpcResponseSizeAdditionalOptions...)...)
	if err != nil {
		otel.Handle(err)
		if c.rpcResponseSize == nil {
			c.rpcResponseSize = noop.Int64Histogram{}
		}
	}

	c.rpcRequestsPerRPC, err = c.meter.Int64Histogram("rpc."+role+".requests_per_rpc",
		append([]metric.Int64HistogramOption{
			metric.WithDescription("Measures the number of messages received per RPC. Should be 1 for all non-streaming RPCs."),
			metric.WithUnit("{count}"),
		}, c.rpcRequestsPerRPCAdditionalOptions...)...)
	if err != nil {
		otel.Handle(err)
		if c.rpcRequestsPerRPC == nil {
			c.rpcRequestsPerRPC = noop.Int64Histogram{}
		}
	}

	c.rpcResponsesPerRPC, err = c.meter.Int64Histogram("rpc."+role+".responses_per_rpc",
		append([]metric.Int64HistogramOption{
			metric.WithDescription("Measures the number of messages received per RPC. Should be 1 for all non-streaming RPCs."),
			metric.WithUnit("{count}"),
		}, c.rpcResponsesPerRPCAdditionalOptions...)...)
	if err != nil {
		otel.Handle(err)
		if c.rpcResponsesPerRPC == nil {
			c.rpcResponsesPerRPC = noop.Int64Histogram{}
		}
	}

	return c
}

type propagatorsOption struct{ p propagation.TextMapPropagator }

func (o propagatorsOption) apply(c *config) {
	if o.p != nil {
		c.Propagators = o.p
	}
}

// WithPropagators returns an Option to use the Propagators when extracting
// and injecting trace context from requests.
func WithPropagators(p propagation.TextMapPropagator) Option {
	return propagatorsOption{p: p}
}

type tracerProviderOption struct{ tp trace.TracerProvider }

func (o tracerProviderOption) apply(c *config) {
	if o.tp != nil {
		c.TracerProvider = o.tp
	}
}

// WithInterceptorFilter returns an Option to use the request filter.
//
// Deprecated: Use stats handlers instead.
func WithInterceptorFilter(f Filter) Option {
	return interceptorFilterOption{f: f}
}

type interceptorFilterOption struct {
	f Filter
}

func (o interceptorFilterOption) apply(c *config) {
	if o.f != nil {
		c.Filter = o.f
	}
}

// WithTracerProvider returns an Option to use the TracerProvider when
// creating a Tracer.
func WithTracerProvider(tp trace.TracerProvider) Option {
	return tracerProviderOption{tp: tp}
}

type meterProviderOption struct{ mp metric.MeterProvider }

func (o meterProviderOption) apply(c *config) {
	if o.mp != nil {
		c.MeterProvider = o.mp
	}
}

// WithMeterProvider returns an Option to use the MeterProvider when
// creating a Meter. If this option is not provide the global MeterProvider will be used.
func WithMeterProvider(mp metric.MeterProvider) Option {
	return meterProviderOption{mp: mp}
}

// Event type that can be recorded, see WithMessageEvents.
type Event int

// Different types of events that can be recorded, see WithMessageEvents.
const (
	ReceivedEvents Event = iota
	SentEvents
)

type messageEventsProviderOption struct {
	events []Event
}

func (m messageEventsProviderOption) apply(c *config) {
	for _, e := range m.events {
		switch e {
		case ReceivedEvents:
			c.ReceivedEvent = true
		case SentEvents:
			c.SentEvent = true
		}
	}
}

// WithMessageEvents configures the Handler to record the specified events
// (span.AddEvent) on spans. By default only summary attributes are added at the
// end of the request.
//
// Valid events are:
//   - ReceivedEvents: Record the number of bytes read after every gRPC read operation.
//   - SentEvents: Record the number of bytes written after every gRPC write operation.
func WithMessageEvents(events ...Event) Option {
	return messageEventsProviderOption{events: events}
}

type spanStartOption struct{ opts []trace.SpanStartOption }

func (o spanStartOption) apply(c *config) {
	c.SpanStartOptions = append(c.SpanStartOptions, o.opts...)
}

// WithSpanOptions configures an additional set of
// trace.SpanOptions, which are applied to each new span.
func WithSpanOptions(opts ...trace.SpanStartOption) Option {
	return spanStartOption{opts}
}

// WithRPCDurationBucketBoundaries configures the
// RPC duration histogram bucket boundaries.
func WithRPCDurationBucketBoundaries(boundaries []float64) Option {
	return rpcDurationBucketBoundaries(boundaries)
}

type rpcDurationBucketBoundaries []float64

func (o rpcDurationBucketBoundaries) apply(c *config) {
	c.rpcDurationAdditionalOptions = append(c.rpcDurationAdditionalOptions, metric.WithExplicitBucketBoundaries(o...))
}

// WithRPCRequestSizeBucketBoundaries configures the
// RPC request size histogram bucket boundaries.
func WithRPCRequestSizeBucketBoundaries(boundaries []float64) Option {
	return rpcRequestSizeBucketBoundaries(boundaries)
}

type rpcRequestSizeBucketBoundaries []float64

func (o rpcRequestSizeBucketBoundaries) apply(c *config) {
	c.rpcRequestSizeAdditionalOptions = append(c.rpcRequestSizeAdditionalOptions, metric.WithExplicitBucketBoundaries(o...))
}

// WithRPCResponseSizeBucketBoundaries configures the
// RPC response size histogram bucket boundaries.
func WithRPCResponseSizeBucketBoundaries(boundaries []float64) Option {
	return rpcResponseSizeBucketBoundaries(boundaries)
}

type rpcResponseSizeBucketBoundaries []float64

func (o rpcResponseSizeBucketBoundaries) apply(c *config) {
	c.rpcResponseSizeAdditionalOptions = append(c.rpcResponseSizeAdditionalOptions, metric.WithExplicitBucketBoundaries(o...))
}

// WithRPCRequestsPerRPCBucketBoundaries configures the
// RPC requests per RPC histogram bucket boundaries.
func WithRPCRequestsPerRPCBucketBoundaries(boundaries []float64) Option {
	return rpcRequestsPerRPCBucketBoundaries(boundaries)
}

type rpcRequestsPerRPCBucketBoundaries []float64

func (o rpcRequestsPerRPCBucketBoundaries) apply(c *config) {
	c.rpcRequestsPerRPCAdditionalOptions = append(c.rpcRequestsPerRPCAdditionalOptions, metric.WithExplicitBucketBoundaries(o...))
}

// WithRPCResponsesPerRPCBucketBoundaries configures the
// RPC responses per RPC histogram bucket boundaries.
func WithRPCResponsesPerRPCBucketBoundaries(boundaries []float64) Option {
	return rpcResponsesPerRPCBucketBoundaries(boundaries)
}

type rpcResponsesPerRPCBucketBoundaries []float64

func (o rpcResponsesPerRPCBucketBoundaries) apply(c *config) {
	c.rpcResponsesPerRPCAdditionalOptions = append(c.rpcResponsesPerRPCAdditionalOptions, metric.WithExplicitBucketBoundaries(o...))
}
