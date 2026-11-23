// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelaws

import (
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type config struct {
	TracerProvider    trace.TracerProvider
	MeterProvider     metric.MeterProvider
	TextMapPropagator propagation.TextMapPropagator
	AttributeBuilders []AttributeBuilder
	MetricRecorders   []MetricRecorder
}

// Option applies an option value.
type Option interface {
	apply(*config)
}

// optionFunc provides a convenience wrapper for simple Options
// that can be represented as functions.
type optionFunc func(*config)

func (o optionFunc) apply(c *config) {
	o(c)
}

// WithTracerProvider specifies a tracer provider to use for creating a tracer.
// If none is specified, the global TracerProvider is used.
func WithTracerProvider(provider trace.TracerProvider) Option {
	return optionFunc(func(cfg *config) {
		if provider != nil {
			cfg.TracerProvider = provider
		}
	})
}

// WithMeterProvider specifies a meter provider to use for creating a meter.
// If none is specified, the global MeterProvider is used.
func WithMeterProvider(provider metric.MeterProvider) Option {
	return optionFunc(func(cfg *config) {
		if provider != nil {
			cfg.MeterProvider = provider
		}
	})
}

// WithTextMapPropagator specifies a Text Map Propagator to use when propagating context.
// If none is specified, the global TextMapPropagator is used.
func WithTextMapPropagator(propagator propagation.TextMapPropagator) Option {
	return optionFunc(func(cfg *config) {
		if propagator != nil {
			cfg.TextMapPropagator = propagator
		}
	})
}

// WithAttributeBuilder specifies an attribute setter function for setting service specific attributes.
// If none is specified, the service will be determined by the DefaultAttributeBuilder function and the corresponding attributes will be included.
func WithAttributeBuilder(attributeBuilders ...AttributeBuilder) Option {
	return optionFunc(func(cfg *config) {
		cfg.AttributeBuilders = append(cfg.AttributeBuilders, attributeBuilders...)
	})
}

// WithMetricRecorder specifies metric recorders called once each AWS SDK
// operation completes, for operations of every service. Use them to record
// custom metrics with instruments created by the caller.
// If none is specified, the built-in service metrics are recorded. Like
// WithAttributeBuilder, specifying one replaces them: include a metric
// recorder created with NewDefaultMetricRecorder to keep them.
// Nil metric recorders are ignored.
func WithMetricRecorder(metricRecorders ...MetricRecorder) Option {
	return optionFunc(func(cfg *config) {
		for _, recorder := range metricRecorders {
			if recorder != nil {
				cfg.MetricRecorders = append(cfg.MetricRecorders, recorder)
			}
		}
	})
}
