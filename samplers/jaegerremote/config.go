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

package jaegerremote // import "go.opentelemetry.io/contrib/samplers/jaeger_remote"

import "time"

const (
	defaultPollingInterval = time.Minute
	defaultSamplingRate    = 0.001
)

type config struct {
	service             string
	endpoint            string
	pollingInterval     time.Duration
	initialSamplingRate float64
}

func defaultConfig() *config {
	return &config{
		service:             "",
		endpoint:            "http://localhost:5778",
		pollingInterval:     defaultPollingInterval,
		initialSamplingRate: defaultSamplingRate,
	}
}

type Option interface {
	apply(config *config)
}

type optionFunc func(config *config)

var _ Option = optionFunc(nil)

func (fn optionFunc) apply(config *config) {
	fn(config)
}

func WithService(service string) Option {
	return optionFunc(func(config *config) {
		config.service = service
	})
}

// WithEndpoint sets the endpoint to retrieve the sampling strategy from.
// Defaults to http://localhost:5778
func WithEndpoint(endpoint string) Option {
	return optionFunc(func(config *config) {
		config.endpoint = endpoint
	})
}

// WithPollingInterval sets the interval to poll for the sampling strategy
// file. Defaults to 1 minute.
func WithPollingInterval(pollingInterval time.Duration) Option {
	return optionFunc(func(config *config) {
		config.pollingInterval = pollingInterval
	})
}

// WithInitialSamplingRate sets the sampling rate the sampler starts with,
// before it has fetched the strategy file. Defaults to 0.001.
func WithInitialSamplingRate(initialSamplingRate float64) Option {
	return optionFunc(func(config *config) {
		config.initialSamplingRate = initialSamplingRate
	})
}
