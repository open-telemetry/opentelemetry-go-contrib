// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/semconv/v1.43.0/goconv"

	"go.opentelemetry.io/contrib/instrumentation/runtime/internal/x"
)

// config contains optional settings for reporting runtime metrics.
type config struct {
	// MinimumRefreshInterval sets the minimum interval between refreshes of
	// the runtime metrics. Values less than or equal to zero are ignored.
	MinimumRefreshInterval time.Duration

	// MeterProvider sets the metric.MeterProvider.  If nil, the global
	// Provider will be used.
	MeterProvider metric.MeterProvider
}

// optInMetric is a runtime metric that is only produced when listed in the
// OTEL_GO_X_RUNTIME_METRICS_OPTIN environment variable.
type optInMetric struct {
	name string
}

var (
	optInGCCycles = optInMetric{name: goconv.MemoryGCCyclesObservable{}.Name()}
	optInCPUTime  = optInMetric{name: goconv.CPUTimeObservable{}.Name()}
	optInGCPauses = optInMetric{name: goconv.MemoryGCPauseDuration{}.Name()}
)

func (m optInMetric) enabled() bool { return x.OptInMetrics.Enabled(m.name) }

// Option supports configuring optional settings for runtime metrics.
type Option interface {
	apply(*config)
}

// ProducerOption supports configuring optional settings for runtime metrics using a
// metric producer in addition to standard instrumentation.
type ProducerOption interface {
	Option
	applyProducer(*config)
}

// DefaultMinimumRefreshInterval is the default minimum interval between
// refreshes of the runtime metrics. Use the [WithMinimumRefreshInterval] option
// to modify this setting in [Start] and [NewProducer].
const DefaultMinimumRefreshInterval time.Duration = 15 * time.Second

// DefaultMinimumReadMemStatsInterval is the default minimum interval between
// refreshes of the runtime metrics. Use the [WithMinimumRefreshInterval] option
// to modify this setting in [Start] and [NewProducer].
//
// Deprecated: Use [DefaultMinimumRefreshInterval] instead.
const DefaultMinimumReadMemStatsInterval time.Duration = DefaultMinimumRefreshInterval

// WithMinimumRefreshInterval sets a minimum interval between refreshes of the
// runtime metrics. This setting is ignored when d is less than or equal to zero.
func WithMinimumRefreshInterval(d time.Duration) ProducerOption {
	return minimumRefreshIntervalOption(d)
}

type minimumRefreshIntervalOption time.Duration

func (o minimumRefreshIntervalOption) apply(c *config) {
	if o > 0 {
		c.MinimumRefreshInterval = time.Duration(o)
	}
}

func (o minimumRefreshIntervalOption) applyProducer(c *config) { o.apply(c) }

// WithMinimumReadMemStatsInterval sets a minimum interval between refreshes of
// the runtime metrics. This setting is ignored when d is less than or equal to
// zero.
//
// Deprecated: Use [WithMinimumRefreshInterval] instead.
func WithMinimumReadMemStatsInterval(d time.Duration) Option {
	return WithMinimumRefreshInterval(d)
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
		MeterProvider: otel.GetMeterProvider(),
	}
	for _, opt := range opts {
		opt.apply(&c)
	}
	if c.MinimumRefreshInterval <= 0 {
		c.MinimumRefreshInterval = DefaultMinimumRefreshInterval
	}
	return c
}

// newProducerConfig computes a config from the supplied ProducerOptions.
func newProducerConfig(opts ...ProducerOption) config {
	c := config{}
	for _, opt := range opts {
		opt.applyProducer(&c)
	}
	if c.MinimumRefreshInterval <= 0 {
		c.MinimumRefreshInterval = DefaultMinimumRefreshInterval
	}
	return c
}
