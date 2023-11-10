// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config // import "go.opentelemetry.io/contrib/config"

import (
	"context"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type configOptions struct {
	ctx                 context.Context
	opentelemetryConfig OpenTelemetryConfiguration
}

// SDK is a struct that contains all the providers
// configured via the configuration model.
type SDK struct {
	meterProvider  metric.MeterProvider
	tracerProvider trace.TracerProvider
}

// TracerProvider returns a configured trace.TracerProvider.
func (s *SDK) TracerProvider() trace.TracerProvider {
	return s.tracerProvider
}

// MeterProvider returns a configured metric.MeterProvider.
func (s *SDK) MeterProvider() metric.MeterProvider {
	return s.meterProvider
}

// NewSDK creates SDK providers based on the configuration model.
//
// Caution: The implementation only returns noop providers.
func NewSDK(opts ...ConfigurationOption) (SDK, error) {
	o := configOptions{}
	for _, opt := range opts {
		o = opt.apply(o)
	}

	return SDK{
		meterProvider:  initMeterProvider(o),
		tracerProvider: initTracerProvider(o),
	}, nil
}

// ConfigurationOption configures options for providers.
type ConfigurationOption interface {
	apply(configOptions) configOptions
}

type configurationOptionFunc func(configOptions) configOptions

func (fn configurationOptionFunc) apply(cfg configOptions) configOptions {
	return fn(cfg)
}

// WithContext sets the context.Context for the SDK.
func WithContext(ctx context.Context) ConfigurationOption {
	return configurationOptionFunc(func(c configOptions) configOptions {
		c.ctx = ctx
		return c
	})
}

// WithOpenTelemetryConfiguration sets the OpenTelemetryConfiguration used
// to produce the SDK.
func WithOpenTelemetryConfiguration(cfg OpenTelemetryConfiguration) ConfigurationOption {
	return configurationOptionFunc(func(c configOptions) configOptions {
		c.opentelemetryConfig = cfg
		return c
	})
}

// TODO: implement parsing functionality:
// - https://github.com/open-telemetry/opentelemetry-go-contrib/issues/4373
// - https://github.com/open-telemetry/opentelemetry-go-contrib/issues/4412

// TODO: create SDK from the model:
// - https://github.com/open-telemetry/opentelemetry-go-contrib/issues/4371
