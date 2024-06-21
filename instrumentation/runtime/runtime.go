// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package runtime // import "go.opentelemetry.io/contrib/instrumentation/runtime"

import (
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/contrib/instrumentation/runtime/internal/deprecatedruntime"
	"go.opentelemetry.io/contrib/instrumentation/runtime/internal/x"
)

// ScopeName is the instrumentation scope name.
const ScopeName = "go.opentelemetry.io/contrib/instrumentation/runtime"

// config contains optional settings for reporting runtime metrics.
type config struct {
	// MinimumReadMemStatsInterval sets the minimum interval
	// between calls to runtime.ReadMemStats().  Negative values
	// are ignored.
	MinimumReadMemStatsInterval time.Duration

	// MeterProvider sets the metric.MeterProvider.  If nil, the global
	// Provider will be used.
	MeterProvider metric.MeterProvider
}

// Option supports configuring optional settings for runtime metrics.
type Option interface {
	apply(*config)
}

// DefaultMinimumReadMemStatsInterval is the default minimum interval
// between calls to runtime.ReadMemStats().  Use the
// WithMinimumReadMemStatsInterval() option to modify this setting in
// Start().
const DefaultMinimumReadMemStatsInterval time.Duration = 15 * time.Second

// WithMinimumReadMemStatsInterval sets a minimum interval between calls to
// runtime.ReadMemStats(), which is a relatively expensive call to make
// frequently.  This setting is ignored when `d` is negative.
func WithMinimumReadMemStatsInterval(d time.Duration) Option {
	return minimumReadMemStatsIntervalOption(d)
}

type minimumReadMemStatsIntervalOption time.Duration

func (o minimumReadMemStatsIntervalOption) apply(c *config) {
	if o >= 0 {
		c.MinimumReadMemStatsInterval = time.Duration(o)
	}
}

// WithMeterProvider sets the Metric implementation to use for
// reporting.  If this option is not used, the global metric.MeterProvider
// will be used.  `provider` must be non-nil.
func WithMeterProvider(provider metric.MeterProvider) Option {
	return metricProviderOption{provider}
}

type metricProviderOption struct{ metric.MeterProvider }

func (o metricProviderOption) apply(c *config) {
	if o.MeterProvider != nil {
		c.MeterProvider = o.MeterProvider
	}
}

// newConfig computes a config from the supplied Options.
func newConfig(opts ...Option) config {
	c := config{
		MeterProvider:               otel.GetMeterProvider(),
		MinimumReadMemStatsInterval: DefaultMinimumReadMemStatsInterval,
	}
	for _, opt := range opts {
		opt.apply(&c)
	}
	return c
}

// Start initializes reporting of runtime metrics using the supplied config.
func Start(opts ...Option) error {
	c := newConfig(opts...)
	if c.MinimumReadMemStatsInterval < 0 {
		c.MinimumReadMemStatsInterval = DefaultMinimumReadMemStatsInterval
	}
	if c.MeterProvider == nil {
		c.MeterProvider = otel.GetMeterProvider()
	}
	meter := c.MeterProvider.Meter(
		ScopeName,
		metric.WithInstrumentationVersion(Version()),
	)
	if x.DeprecatedRuntimeMetrics.Enabled() {
		return deprecatedruntime.Start(meter, c.MinimumReadMemStatsInterval)
	}
	// TODO (#5655) Implement new runtime conventions
	return nil
}
