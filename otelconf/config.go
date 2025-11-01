// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package otelconf provides an OpenTelemetry declarative configuration SDK.
package otelconf // import "go.opentelemetry.io/contrib/otelconf"

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/log"
	nooplog "go.opentelemetry.io/otel/log/noop"
	"go.opentelemetry.io/otel/metric"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
	yaml "go.yaml.in/yaml/v3"

	"go.opentelemetry.io/contrib/otelconf/internal/provider"
)

// SDK is a struct that contains all the providers
// configured via the configuration model.
type SDK struct {
	meterProvider  metric.MeterProvider
	tracerProvider trace.TracerProvider
	loggerProvider log.LoggerProvider
	shutdown       shutdownFunc
}

// TracerProvider returns a configured trace.TracerProvider.
func (s *SDK) TracerProvider() trace.TracerProvider {
	return s.tracerProvider
}

// MeterProvider returns a configured metric.MeterProvider.
func (s *SDK) MeterProvider() metric.MeterProvider {
	return s.meterProvider
}

// LoggerProvider returns a configured log.LoggerProvider.
func (s *SDK) LoggerProvider() log.LoggerProvider {
	return s.loggerProvider
}

// Shutdown calls shutdown on all configured providers.
func (s *SDK) Shutdown(ctx context.Context) error {
	return s.shutdown(ctx)
}

var noopSDK = SDK{
	loggerProvider: nooplog.LoggerProvider{},
	meterProvider:  noopmetric.MeterProvider{},
	tracerProvider: nooptrace.TracerProvider{},
	shutdown:       func(context.Context) error { return nil },
}

// NewSDK creates SDK providers based on the configuration model.
func NewSDK(opts ...ConfigurationOption) (SDK, error) {
	o := configOptions{
		ctx: context.Background(),
	}
	for _, opt := range opts {
		o = opt.apply(o)
	}
	if o.opentelemetryConfig.Disabled != nil && *o.opentelemetryConfig.Disabled {
		return noopSDK, nil
	}

	r, err := newResource(o.opentelemetryConfig.Resource)
	if err != nil {
		return noopSDK, err
	}

	mp, mpShutdown, err := meterProvider(o, r)
	if err != nil {
		return noopSDK, err
	}

	tp, tpShutdown, err := tracerProvider(o, r)
	if err != nil {
		return noopSDK, err
	}

	lp, lpShutdown, err := loggerProvider(o, r)
	if err != nil {
		return noopSDK, err
	}

	return SDK{
		meterProvider:  mp,
		tracerProvider: tp,
		loggerProvider: lp,
		shutdown: func(ctx context.Context) error {
			return errors.Join(mpShutdown(ctx), tpShutdown(ctx), lpShutdown(ctx))
		},
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

// WithLoggerProviderOptions appends LoggerProviderOptions used for constructing
// the LoggerProvider. OpenTelemetryConfiguration takes precedence over these options.
func WithLoggerProviderOptions(opts ...sdklog.LoggerProviderOption) ConfigurationOption {
	return configurationOptionFunc(func(c configOptions) configOptions {
		c.loggerProviderOptions = append(c.loggerProviderOptions, opts...)
		return c
	})
}

// WithMeterProviderOptions appends metric.Options used for constructing the
// MeterProvider. OpenTelemetryConfiguration takes precedence over these options.
func WithMeterProviderOptions(opts ...sdkmetric.Option) ConfigurationOption {
	return configurationOptionFunc(func(c configOptions) configOptions {
		c.meterProviderOptions = append(c.meterProviderOptions, opts...)
		return c
	})
}

// WithTracerProviderOptions appends TracerProviderOptions used for constructing
// the TracerProvider. OpenTelemetryConfiguration takes precedence over these options.
func WithTracerProviderOptions(opts ...sdktrace.TracerProviderOption) ConfigurationOption {
	return configurationOptionFunc(func(c configOptions) configOptions {
		c.tracerProviderOptions = append(c.tracerProviderOptions, opts...)
		return c
	})
}

// ParseYAML parses a YAML configuration file into an OpenTelemetryConfiguration.
func ParseYAML(file []byte) (*OpenTelemetryConfiguration, error) {
	file, err := provider.ReplaceEnvVars(file)
	if err != nil {
		return nil, err
	}
	var cfg OpenTelemetryConfiguration
	err = yaml.Unmarshal(file, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
