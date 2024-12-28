// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelaws // import "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"

import (
	"context"

	"github.com/aws/smithy-go/middleware"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type config struct {
	TracerProvider    trace.TracerProvider
	TextMapPropagator propagation.TextMapPropagator
	AttributeBuilders []AttributeBuilder
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

// WithTextMapPropagator specifies a Text Map Propagator to use when propagating context.
// If none is specified, the global TextMapPropagator is used.
func WithTextMapPropagator(propagator propagation.TextMapPropagator) Option {
	return optionFunc(func(cfg *config) {
		if propagator != nil {
			cfg.TextMapPropagator = propagator
		}
	})
}

// WithAttributeSetter specifies an attribute setter function for setting service specific attributes.
// If none is specified, the service will be determined by the DefaultAttributeBuilder function and the corresponding attributes will be included.
func WithAttributeSetter(attributesetters ...AttributeSetter) Option {
	var attributeBuilders []AttributeBuilder
	for _, setter := range attributesetters {
		attributeBuilders = append(attributeBuilders, func(ctx context.Context, in middleware.InitializeInput, out middleware.InitializeOutput) []attribute.KeyValue {
			return setter(ctx, in)
		})
	}

	return WithAttributeBuilder(attributeBuilders...)
}

// WithAttributeBuilder specifies an attribute setter function for setting service specific attributes.
// If none is specified, the service will be determined by the DefaultAttributeBuilder function and the corresponding attributes will be included.
func WithAttributeBuilder(attributeBuilders ...AttributeBuilder) Option {
	return optionFunc(func(cfg *config) {
		cfg.AttributeBuilders = append(cfg.AttributeBuilders, attributeBuilders...)
	})
}
