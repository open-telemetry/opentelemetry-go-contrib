// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xray // import "go.opentelemetry.io/contrib/samplers/aws/xray"

import (
	"fmt"
	"log"
	"math"
	"net/url"
	"os"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
)

const (
	defaultPollingInterval = 300
)

type config struct {
	endpoint                     url.URL
	samplingRulesPollingInterval time.Duration
	logger                       logr.Logger
}

// Option sets configuration on the sampler.
type Option interface {
	apply(*config) *config
}

type optionFunc func(*config) *config

func (f optionFunc) apply(cfg *config) *config {
	return f(cfg)
}

// WithEndpoint sets custom proxy endpoint.
// If this option is not provided the default endpoint used will be http://127.0.0.1:2000.
func WithEndpoint(endpoint url.URL) Option {
	return optionFunc(func(cfg *config) *config {
		cfg.endpoint = endpoint
		return cfg
	})
}

// WithSamplingRulesPollingInterval sets polling interval for sampling rules.
// If this option is not provided the default samplingRulesPollingInterval used will be 300 seconds.
func WithSamplingRulesPollingInterval(polingInterval time.Duration) Option {
	return optionFunc(func(cfg *config) *config {
		cfg.samplingRulesPollingInterval = polingInterval
		return cfg
	})
}

// WithLogger sets custom logging for remote sampling implementation.
// If this option is not provided the default logger used will be go-logr/stdr (https://github.com/go-logr/stdr).
func WithLogger(l logr.Logger) Option {
	return optionFunc(func(cfg *config) *config {
		cfg.logger = l
		return cfg
	})
}

var defaultLogger = stdr.NewWithOptions(log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile), stdr.Options{LogCaller: stdr.Error})

func newConfig(opts ...Option) (*config, error) {
	defaultProxyEndpoint, err := url.Parse("http://127.0.0.1:2000")
	if err != nil {
		return nil, err
	}

	cfg := &config{
		endpoint:                     *defaultProxyEndpoint,
		samplingRulesPollingInterval: defaultPollingInterval * time.Second,
		logger:                       defaultLogger,
	}

	for _, option := range opts {
		option.apply(cfg)
	}

	if math.Signbit(float64(cfg.samplingRulesPollingInterval)) {
		return nil, fmt.Errorf("config validation error: samplingRulesPollingInterval should be positive number")
	}

	return cfg, nil
}
