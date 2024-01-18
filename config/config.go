// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config // import "go.opentelemetry.io/contrib/config"

import (
	"context"
	"errors"
	"os"
	"regexp"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	"gopkg.in/yaml.v3"
)

const (
	protocolProtobufHTTP = "http/protobuf"
	protocolProtobufGRPC = "grpc/protobuf"

	compressionGzip = "gzip"
	compressionNone = "none"
)

type configOptions struct {
	ctx                 context.Context
	opentelemetryConfig OpenTelemetryConfiguration
}

type shutdownFunc func(context.Context) error

func noopShutdown(context.Context) error {
	return nil
}

// SDK is a struct that contains all the providers
// configured via the configuration model.
type SDK struct {
	meterProvider  metric.MeterProvider
	tracerProvider trace.TracerProvider
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

// Shutdown calls shutdown on all configured providers.
func (s *SDK) Shutdown(ctx context.Context) error {
	return s.shutdown(ctx)
}

// NewSDK creates SDK providers based on the configuration model.
//
// Caution: The implementation only returns noop providers.
func NewSDK(opts ...ConfigurationOption) (SDK, error) {
	o := configOptions{}
	for _, opt := range opts {
		o = opt.apply(o)
	}

	r, err := newResource(o.opentelemetryConfig.Resource)
	if err != nil {
		return SDK{}, err
	}

	mp, mpShutdown := initMeterProvider(o)
	tp, tpShutdown, err := tracerProvider(o, r)
	if err != nil {
		return SDK{}, err
	}

	return SDK{
		meterProvider:  mp,
		tracerProvider: tp,
		shutdown: func(ctx context.Context) error {
			err := mpShutdown(ctx)
			return errors.Join(err, tpShutdown(ctx))
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

// ParseYAML parses a YAML configuration file into an OpenTelemetryConfiguration.
func ParseYAML(file []byte) (*OpenTelemetryConfiguration, error) {
	re := regexp.MustCompile(`\$\{([a-zA-Z_][a-zA-Z0-9_]*)\}`)

	replaceEnvVars := func(input []byte) []byte {
		return re.ReplaceAllFunc(input, func(s []byte) []byte {
			match := re.FindSubmatch(s)
			if len(match) < 2 {
				return s
			}
			envVarName := string(match[1])
			envVarValue := os.Getenv(envVarName)
			return []byte(envVarValue)
		})
	}

	file = replaceEnvVars(file)

	var cfg OpenTelemetryConfiguration
	err := yaml.Unmarshal(file, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

// TODO: implement parsing functionality:
// - https://github.com/open-telemetry/opentelemetry-go-contrib/issues/4412

// TODO: create SDK from the model:
// - https://github.com/open-telemetry/opentelemetry-go-contrib/issues/4371

func newResource(res *Resource) (*resource.Resource, error) {
	if res == nil {
		return resource.Default(), nil
	}
	return resource.Merge(resource.Default(),
		resource.NewWithAttributes(semconv.SchemaURL,
			semconv.ServiceName(*res.Attributes.ServiceName),
		))
}
