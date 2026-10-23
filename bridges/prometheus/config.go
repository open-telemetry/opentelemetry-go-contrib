// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
)

// config contains options for the producer.
type config struct {
	gatherers []prometheus.Gatherer
}

// newConfig creates a validated config configured with options.
func newConfig(opts ...Option) config {
	cfg := config{}
	for _, opt := range opts {
		cfg = opt.apply(cfg)
	}

	if len(cfg.gatherers) == 0 {
		cfg.gatherers = []prometheus.Gatherer{prometheus.DefaultGatherer}
	}

	return cfg
}

// Option sets producer option values.
type Option interface {
	apply(config) config
}

type optionFunc func(config) config

func (fn optionFunc) apply(cfg config) config {
	return fn(cfg)
}

// WithGatherer adds a Prometheus Gatherer for the bridge to collect metrics
// from. If no WithGatherer option is provided, the bridge uses
// prometheus.DefaultGatherer. Multiple WithGatherer options may be provided
// to collect metrics from multiple gatherers.
func WithGatherer(gatherer prometheus.Gatherer) Option {
	return optionFunc(func(cfg config) config {
		cfg.gatherers = append(cfg.gatherers, gatherer)
		return cfg
	})
}
