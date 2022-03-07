// Copyright The OpenTelemetry Authors
// Copyright (c) 2021 The Jaeger Authors.
// Copyright (c) 2017 Uber Technologies, Inc.
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

package jaegerremote

import (
	"time"

	"go.opentelemetry.io/otel/sdk/trace"
)

// SamplerOption is a function that sets some option on the sampler
type SamplerOption func(options *samplerConfig)

type samplerConfig struct {
	sampler                 trace.Sampler
	samplingServerURL       string
	samplingRefreshInterval time.Duration
	samplingFetcher         samplingStrategyFetcher
	samplingParser          samplingStrategyParser
	updaters                []samplerUpdater
	posParams               perOperationSamplerParams
}

type config struct {
	c samplerConfig
}

// newConfig returns an appropriately configured config.
func newConfig(options ...Option) samplerConfig {
	c := new(config)
	for _, option := range options {
		option.apply(c)
	}
	if c.c.sampler == nil {
		c.c.sampler = newProbabilisticSampler(0.001)
	}
	if c.c.samplingServerURL == "" {
		c.c.samplingServerURL = defaultSamplingServerURL
	}
	if c.c.samplingRefreshInterval <= 0 {
		c.c.samplingRefreshInterval = defaultSamplingRefreshInterval
	}
	if c.c.samplingFetcher == nil {
		c.c.samplingFetcher = newHTTPSamplingStrategyFetcher(c.c.samplingServerURL)
	}
	if c.c.samplingParser == nil {
		c.c.samplingParser = new(samplingStrategyParserImpl)
	}
	if c.c.updaters == nil {
		c.c.updaters = []samplerUpdater{
			&perOperationSamplerUpdater{
				MaxOperations:            c.c.posParams.MaxOperations,
				OperationNameLateBinding: c.c.posParams.OperationNameLateBinding,
			},
			new(probabilisticSamplerUpdater),
			new(rateLimitingSamplerUpdater),
		}
	}
	return c.c
}

type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (fn optionFunc) apply(c *config) {
	fn(c)
}

// WithInitialSampler creates a SamplerOption that sets the initial sampler
// to use before a remote sampler is created and used.
func WithInitialSampler(sampler trace.Sampler) Option {
	return optionFunc(func(c *config) {
		c.c.sampler = sampler
	})
}

// WithSamplingServerURL creates a SamplerOption that sets the sampling server url
// of the local agent that contains the sampling strategies.
func WithSamplingServerURL(samplingServerURL string) Option {
	return optionFunc(func(c *config) {
		c.c.samplingServerURL = samplingServerURL
	})
}

// WithMaxOperations creates a SamplerOption that sets the maximum number of
// operations the sampler will keep track of.
func WithMaxOperations(maxOperations int) Option {
	return optionFunc(func(c *config) {
		c.c.posParams.MaxOperations = maxOperations
	})
}

// WithOperationNameLateBinding creates a SamplerOption that sets the respective
// field in the perOperationSamplerParams.
func WithOperationNameLateBinding(enable bool) Option {
	return optionFunc(func(c *config) {
		c.c.posParams.OperationNameLateBinding = enable
	})
}

// WithSamplingRefreshInterval creates a SamplerOption that sets how often the
// sampler will poll local agent for the appropriate sampling strategy.
func WithSamplingRefreshInterval(samplingRefreshInterval time.Duration) Option {
	return optionFunc(func(c *config) {
		c.c.samplingRefreshInterval = samplingRefreshInterval
	})
}

// samplingStrategyFetcher creates a SamplerOption that initializes sampling strategy fetcher.
func withSamplingStrategyFetcher(fetcher samplingStrategyFetcher) Option {
	return optionFunc(func(c *config) {
		c.c.samplingFetcher = fetcher
	})
}

// samplingStrategyParser creates a SamplerOption that initializes sampling strategy parser.
func withSamplingStrategyParser(parser samplingStrategyParser) Option {
	return optionFunc(func(c *config) {
		c.c.samplingParser = parser
	})
}

// withUpdaters creates a SamplerOption that initializes sampler updaters.
func withUpdaters(updaters ...samplerUpdater) Option {
	return optionFunc(func(c *config) {
		c.c.updaters = updaters
	})
}
