// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package otelconf provides an OpenTelemetry declarative configuration SDK.
package otelconf // import "go.opentelemetry.io/contrib/otelconf"

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/funcr"
	"go.opentelemetry.io/otel/log"
	nooplog "go.opentelemetry.io/otel/log/noop"
	"go.opentelemetry.io/otel/metric"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
	yaml "go.yaml.in/yaml/v3"

	"go.opentelemetry.io/contrib/otelconf/internal/provider"
)

const (
	envVarConfigFileDeprecated = "OTEL_EXPERIMENTAL_CONFIG_FILE"
	envVarConfigFile           = "OTEL_CONFIG_FILE"
)

// SDK is a struct that contains all the providers
// configured via the configuration model.
type SDK struct {
	meterProvider  metric.MeterProvider
	tracerProvider trace.TracerProvider
	loggerProvider log.LoggerProvider
	logger         logr.Logger
	propagator     propagation.TextMapPropagator
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

// Logger returns a [logr.Logger] whose verbosity is derived from the
// log_level field in the configuration.
//
// Scope: this logger is intended for the OTel SDK internal diagnostics
// logger only (the one set via [go.opentelemetry.io/otel.SetLogger]).
// It does NOT control application-level or per-component logging.
//
// When log_level is omitted from the configuration (nil), Logger returns
// a no-op logger so that any previously installed logger is left untouched.
// Callers should check before overriding:
//
//	if l := sdk.Logger(); l.GetSink() != nil {
//	    otel.SetLogger(l)
//	}
//
// Calling [go.opentelemetry.io/otel.SetLogger] unconditionally will
// replace a user-provided logger.
//
// The severity-to-verbosity mapping follows the OTel SDK internal
// logging conventions:
//
//	trace, debug (1–8)  → V(8)  (most verbose)
//	info         (9–12) → V(4)
//	warn         (13–16) → V(1)
//	error, fatal (17–24) → V(0)  (errors only)
//
// Numeric suffixes (debug2, warn4, …) map to their base level because
// [logr] does not model sub-levels.
func (s *SDK) Logger() logr.Logger {
	return s.logger
}

// Propagator returns a configured propagation.TextMapPropagator.
func (s *SDK) Propagator() propagation.TextMapPropagator {
	return s.propagator
}

// Shutdown calls shutdown on all configured providers.
func (s *SDK) Shutdown(ctx context.Context) error {
	return s.shutdown(ctx)
}

var (
	noopSDK = SDK{
		loggerProvider: nooplog.LoggerProvider{},
		meterProvider:  noopmetric.MeterProvider{},
		tracerProvider: nooptrace.TracerProvider{},
		propagator:     propagation.NewCompositeTextMapPropagator(),
		shutdown:       func(context.Context) error { return nil },
	}
	errDeprecatedEnvVarUsed = errors.New("OTEL_EXPERIMENTAL_CONFIG_FILE is no longer supported, use OTEL_CONFIG_FILE instead")
)

func parseConfigFileFromEnvironment(filename string) (ConfigurationOption, error) {
	b, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// Parse a configuration file into an OpenTelemetryConfiguration model.
	c, err := ParseYAML(b)
	if err != nil {
		return nil, err
	}

	// Create SDK components with the parsed configuration.
	return WithOpenTelemetryConfiguration(*c), nil
}

// NewSDK creates SDK providers based on the configuration model. It checks the local environment and
// uses the file set in the variable `OTEL_CONFIG_FILE` to configure the SDK automatically.
// Any file defined by `OTEL_CONFIG_FILE` will supersede all files passed with
// [WithOpenTelemetryConfiguration].
func NewSDK(opts ...ConfigurationOption) (SDK, error) {
	_, ok := os.LookupEnv(envVarConfigFileDeprecated)
	if ok {
		return noopSDK, errDeprecatedEnvVarUsed
	}
	filename, ok := os.LookupEnv(envVarConfigFile)
	if ok {
		opt, err := parseConfigFileFromEnvironment(filename)
		if err != nil {
			return noopSDK, err
		}
		opts = append(opts, opt)
	}
	o := configOptions{
		ctx: context.Background(),
	}
	for _, opt := range opts {
		o = opt.apply(o)
	}
	if o.opentelemetryConfig.Disabled != nil && *o.opentelemetryConfig.Disabled {
		return noopSDK, nil
	}

	l, err := newLogger(o.opentelemetryConfig.LogLevel)
	if err != nil {
		return noopSDK, err
	}

	r, err := newResource(o.opentelemetryConfig.Resource)
	if err != nil {
		return noopSDK, err
	}

	p, err := newPropagator(o.opentelemetryConfig.Propagator)
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
		logger:         l,
		propagator:     p,
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

// newLogger creates a [logr.Logger] for the OTel SDK internal diagnostics.
//
// When logLevel is nil (not configured), a zero-value [logr.Logger] is
// returned (nil sink) so that the caller can detect "not configured" via
// GetSink() == nil and avoid overriding a user-provided logger.
func newLogger(logLevel *SeverityNumber) (logr.Logger, error) {
	if logLevel == nil {
		return logr.Logger{}, nil
	}

	v, err := severityToVerbosity(*logLevel)
	if err != nil {
		return logr.Logger{}, err
	}

	// Use funcr which supports per-instance verbosity and avoids the
	// process-global stdr.SetVerbosity.
	l := funcr.NewJSON(func(obj string) {
		fmt.Fprintln(os.Stderr, obj)
	}, funcr.Options{
		Verbosity: v,
	})
	return l, nil
}

func severityToVerbosity(s SeverityNumber) (int, error) {
	base := s
	if len(s) > 0 {
		last := s[len(s)-1]
		if last >= '2' && last <= '4' {
			base = s[:len(s)-1]
		}
	}
	switch base {
	case SeverityNumberTrace, SeverityNumberDebug:
		return 8, nil
	case SeverityNumberInfo:
		return 4, nil
	case SeverityNumberWarn:
		return 1, nil
	case SeverityNumberError, SeverityNumberFatal:
		return 0, nil
	default:
		return 0, newErrInvalid(fmt.Sprintf("unsupported log_level %q", s))
	}
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
