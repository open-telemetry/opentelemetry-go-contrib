// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

func newConfig(opts ...Option) (*config, error) {
	defaultProxyEndpoint, err := url.Parse("http://127.0.0.1:2000")
	if err != nil {
		return nil, err
	}

	cfg := &config{
		endpoint:                     *defaultProxyEndpoint,
		samplingRulesPollingInterval: defaultPollingInterval * time.Second,
		logger:                       stdr.NewWithOptions(log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile), stdr.Options{LogCaller: stdr.Error}),
	}

	for _, option := range opts {
		option.apply(cfg)
	}

	if math.Signbit(float64(cfg.samplingRulesPollingInterval)) {
		return nil, fmt.Errorf("config validation error: samplingRulesPollingInterval should be positive number")
	}

	return cfg, nil
}
