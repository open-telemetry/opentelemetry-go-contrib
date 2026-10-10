// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/contrib/instrumentation/runtime/internal/x"
)

// config contains optional settings for reporting runtime metrics.
type config struct {
	// MinimumReadMemStatsInterval sets the minimum interval
	// between calls to runtime.ReadMemStats().  Negative values
	// are ignored.
	MinimumReadMemStatsInterval time.Duration

	// MeterProvider sets the metric.MeterProvider.  If nil, the global
	// Provider will be used.
	MeterProvider metric.MeterProvider

	// OptInMetrics holds the opt-in metrics enabled with WithOptInMetrics.
	OptInMetrics map[OptInMetric]struct{}
}

// optInEnabled returns if m was enabled with WithOptInMetrics or the
// OTEL_GO_X_RUNTIME_METRICS_OPTIN environment variable.
func (c config) optInEnabled(m OptInMetric) bool {
	_, ok := c.OptInMetrics[m]
	return ok || x.OptInMetrics.Enabled(m.name)
}

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

func (o minimumReadMemStatsIntervalOption) applyProducer(c *config) { o.apply(c) }

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

// OptInMetric is a runtime metric that is only produced when enabled.
type OptInMetric struct {
	name string
}

// String returns the semantic convention name of the metric.
func (m OptInMetric) String() string { return m.name }

var (
	// MemoryGCCycles is the go.memory.gc.cycles metric, produced by [Start].
	MemoryGCCycles = OptInMetric{name: "go.memory.gc.cycles"}
	// CPUTime is the go.cpu.time metric, produced by [Start].
	CPUTime = OptInMetric{name: "go.cpu.time"}
	// MemoryGCPauseDuration is the go.memory.gc.pause.duration metric,
	// produced by [NewProducer].
	MemoryGCPauseDuration = OptInMetric{name: "go.memory.gc.pause.duration"}
)

// WithOptInMetrics enables the given opt-in metrics, in addition to any
// listed in the OTEL_GO_X_RUNTIME_METRICS_OPTIN environment variable. Metrics
// that the receiving [Start] or [NewProducer] does not produce are ignored.
func WithOptInMetrics(metrics ...OptInMetric) ProducerOption {
	return optInMetricsOption(metrics)
}

type optInMetricsOption []OptInMetric

func (o optInMetricsOption) apply(c *config) {
	if c.OptInMetrics == nil {
		c.OptInMetrics = make(map[OptInMetric]struct{}, len(o))
	}
	for _, m := range o {
		c.OptInMetrics[m] = struct{}{}
	}
}

func (o optInMetricsOption) applyProducer(c *config) { o.apply(c) }

// newConfig computes a config from the supplied Options.
func newConfig(opts ...Option) config {
	c := config{
		MeterProvider: otel.GetMeterProvider(),
	}
	for _, opt := range opts {
		opt.apply(&c)
	}
	if c.MinimumReadMemStatsInterval <= 0 {
		c.MinimumReadMemStatsInterval = DefaultMinimumReadMemStatsInterval
	}
	return c
}

// newProducerConfig computes a config from the supplied ProducerOptions.
func newProducerConfig(opts ...ProducerOption) config {
	c := config{}
	for _, opt := range opts {
		opt.applyProducer(&c)
	}
	if c.MinimumReadMemStatsInterval <= 0 {
		c.MinimumReadMemStatsInterval = DefaultMinimumReadMemStatsInterval
	}
	return c
}
