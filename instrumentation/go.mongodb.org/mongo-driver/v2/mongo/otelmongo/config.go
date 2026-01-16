// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelmongo // import "go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/v2/mongo/otelmongo"

import (
	"go.mongodb.org/mongo-driver/v2/event"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// ScopeName is the instrumentation scope name.
const ScopeName = "go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/v2/mongo/otelmongo"

// config is used to configure the mongo tracer.
type config struct {
	MeterProvider  metric.MeterProvider
	TracerProvider trace.TracerProvider

	Meter  metric.Meter
	Tracer trace.Tracer

	CommandAttributeDisabled bool

	SpanNameFormatter SpanNameFormatterFunc
}

// newConfig returns a config with all Options set.
func newConfig(opts ...Option) config {
	cfg := config{
		MeterProvider:            otel.GetMeterProvider(),
		TracerProvider:           otel.GetTracerProvider(),
		CommandAttributeDisabled: true,
	}

	cfg.SpanNameFormatter = func(event *event.CommandStartedEvent) string {
		collection, _ := extractCollection(event)
		if collection != "" {
			return collection + "." + event.CommandName
		}

		return event.CommandName
	}

	for _, opt := range opts {
		opt.apply(&cfg)
	}

	cfg.Meter = cfg.MeterProvider.Meter(
		ScopeName,
		metric.WithInstrumentationVersion(Version),
	)

	cfg.Tracer = cfg.TracerProvider.Tracer(
		ScopeName,
		trace.WithInstrumentationVersion(Version),
	)
	return cfg
}

// Option specifies instrumentation configuration options.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (o optionFunc) apply(c *config) {
	o(c)
}

// WithMeterProvider specifies a [metric.MeterProvider] to use for creating a Meter.
// If none is specified, the global MeterProvider is used.
func WithMeterProvider(provider metric.MeterProvider) Option {
	return optionFunc(func(cfg *config) {
		if provider != nil {
			cfg.MeterProvider = provider
		}
	})
}

// SpanNameFormatterFunc is a function that resolves the span name given an
// *event.CommandStartedEvent.
type SpanNameFormatterFunc func(e *event.CommandStartedEvent) string

// WithSpanNameFormatter specifies a function that resolves the span name given an
// *event.CommandStartedEvent. If none is specified, the default resolver is used,
// which returns "<collection>.<command>" if the collection is non-empty,
// and just "<command>" otherwise.
func WithSpanNameFormatter(resolver SpanNameFormatterFunc) Option {
	return optionFunc(func(cfg *config) {
		if resolver != nil {
			cfg.SpanNameFormatter = resolver
		}
	})
}

// WithTracerProvider specifies a tracer provider to use for creating a tracer.
// If none is specified, the global provider is used.
func WithTracerProvider(provider trace.TracerProvider) Option {
	return optionFunc(func(cfg *config) {
		if provider != nil {
			cfg.TracerProvider = provider
		}
	})
}

// WithCommandAttributeDisabled specifies if the MongoDB command is added as an attribute to Spans or not.
// This is disabled by default and the MongoDB command will not be added as an attribute
// to Spans if this option is not provided.
func WithCommandAttributeDisabled(disabled bool) Option {
	return optionFunc(func(cfg *config) {
		cfg.CommandAttributeDisabled = disabled
	})
}
