// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelmongo // import "go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/v2/mongo/otelmongo"

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// ScopeName is the instrumentation scope name.
const ScopeName = "go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/v2/mongo/otelmongo"

// config is used to configure the mongo tracer.
// when you want to disable metrics or traces, use the [WithMeterProvider] options
// to set a [go.opentelemetry.io/otel/metric/noop.MeterProvider] or
// [WithTracerProvider] options to set a [go.opentelemetry.io/otel/trace/noop.TracerProvider]
type config struct {
	MeterProvider  metric.MeterProvider
	TracerProvider trace.TracerProvider

	Meter  metric.Meter
	Tracer trace.Tracer

	CommandAttributeDisabled bool
}

// newConfig returns a config with all Options set.
func newConfig(opts ...Option) config {
	cfg := config{
		MeterProvider:            otel.GetMeterProvider(),
		TracerProvider:           otel.GetTracerProvider(),
		CommandAttributeDisabled: true,
	}
	for _, opt := range opts {
		opt.apply(&cfg)
	}

	cfg.Meter = cfg.MeterProvider.Meter(
		ScopeName,
		metric.WithInstrumentationVersion(Version()),
	)

	cfg.Tracer = cfg.TracerProvider.Tracer(
		ScopeName,
		trace.WithInstrumentationVersion(Version()),
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

// WithMeterProvider specifies a meter provider to use for creating a meter.
// If none is specified, the global provider is used.
func WithMeterProvider(provider metric.MeterProvider) Option {
	return optionFunc(func(cfg *config) {
		if provider != nil {
			cfg.MeterProvider = provider
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
