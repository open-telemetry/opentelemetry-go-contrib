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

// WithMaxOperations creates a SamplerOption that sets the maximum number of
// operations the sampler will keep track of.
func WithMaxOperations(maxOperations int) SamplerOption {
	return func(o *samplerConfig) {
		o.posParams.MaxOperations = maxOperations
	}
}

// WithOperationNameLateBinding creates a SamplerOption that sets the respective
// field in the perOperationSamplerParams.
func WithOperationNameLateBinding(enable bool) SamplerOption {
	return func(o *samplerConfig) {
		o.posParams.OperationNameLateBinding = enable
	}
}

// WithInitialSampler creates a SamplerOption that sets the initial sampler
// to use before a remote sampler is created and used.
func WithInitialSampler(sampler trace.Sampler) SamplerOption {
	return func(o *samplerConfig) {
		o.sampler = sampler
	}
}

// WithSamplingServerURL creates a SamplerOption that sets the sampling server url
// of the local agent that contains the sampling strategies.
func WithSamplingServerURL(samplingServerURL string) SamplerOption {
	return func(o *samplerConfig) {
		o.samplingServerURL = samplingServerURL
	}
}

// WithSamplingRefreshInterval creates a SamplerOption that sets how often the
// sampler will poll local agent for the appropriate sampling strategy.
func WithSamplingRefreshInterval(samplingRefreshInterval time.Duration) SamplerOption {
	return func(o *samplerConfig) {
		o.samplingRefreshInterval = samplingRefreshInterval
	}
}

// samplingStrategyFetcher creates a SamplerOption that initializes sampling strategy fetcher.
func withSamplingStrategyFetcher(fetcher samplingStrategyFetcher) SamplerOption {
	return func(o *samplerConfig) {
		o.samplingFetcher = fetcher
	}
}

// samplingStrategyParser creates a SamplerOption that initializes sampling strategy parser.
func withSamplingStrategyParser(parser samplingStrategyParser) SamplerOption {
	return func(o *samplerConfig) {
		o.samplingParser = parser
	}
}

// withUpdaters creates a SamplerOption that initializes sampler updaters.
func withUpdaters(updaters ...samplerUpdater) SamplerOption {
	return func(o *samplerConfig) {
		o.updaters = updaters
	}
}

func (o *samplerConfig) applyOptionsAndDefaults(opts ...SamplerOption) *samplerConfig {
	for _, option := range opts {
		option(o)
	}
	if o.sampler == nil {
		o.sampler = newProbabilisticSampler(0.001)
	}
	if o.samplingServerURL == "" {
		o.samplingServerURL = defaultSamplingServerURL
	}
	if o.samplingRefreshInterval <= 0 {
		o.samplingRefreshInterval = defaultSamplingRefreshInterval
	}
	if o.samplingFetcher == nil {
		o.samplingFetcher = newHTTPSamplingStrategyFetcher(o.samplingServerURL)
	}
	if o.samplingParser == nil {
		o.samplingParser = new(samplingStrategyParserImpl)
	}
	if o.updaters == nil {
		o.updaters = []samplerUpdater{
			&perOperationSamplerUpdater{
				MaxOperations:            o.posParams.MaxOperations,
				OperationNameLateBinding: o.posParams.OperationNameLateBinding,
			},
			new(probabilisticSamplerUpdater),
			new(rateLimitingSamplerUpdater),
		}
	}
	return o
}
