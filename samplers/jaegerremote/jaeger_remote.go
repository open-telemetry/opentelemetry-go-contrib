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

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	jaeger_api_v2 "go.opentelemetry.io/contrib/samplers/jaegerremote/internal/proto-gen/jaeger-idl/proto/api_v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
)

type sampler struct {
	pollingInterval time.Duration

	fetcher samplingStrategyFetcher
	parser  samplingStrategyParser

	sync.RWMutex
	lastStrategyResponse jaeger_api_v2.SamplingStrategyResponse
	sampler              trace.Sampler
}

func (s *sampler) ShouldSample(p trace.SamplingParameters) trace.SamplingResult {
	s.RLock()
	defer s.RUnlock()

	return s.sampler.ShouldSample(p)
}

func (s *sampler) Description() string {
	s.RLock()
	defer s.RUnlock()

	return fmt.Sprintf("JaegerRemoteSampler{%s}", s.sampler.Description())
}

func (s *sampler) pollSamplingStrategies() {
	ticker := time.NewTicker(s.pollingInterval)
	for {
		<-ticker.C
		err := s.updateSamplingStrategies()
		if err != nil {
			otel.Handle(fmt.Errorf("updating jaeger remote sampling strategies failed: %w", err))
		}
	}
}

// updateSamplingStrategies fetches the sampling strategy from backend server
// and updates the sampler if it has changed.
// This function is called automatically on a timer.
func (s *sampler) updateSamplingStrategies() error {
	strategyResponse, err := s.fetcher.Fetch()
	if err != nil {
		return fmt.Errorf("fetching failed: %w", err)
	}

	if !s.hasChanges(strategyResponse) {
		return nil
	}

	sampler, err := s.parser.Parse(strategyResponse)
	if err != nil {
		return fmt.Errorf("parsing failed: %w", err)
	}

	s.Lock()
	defer s.Unlock()

	s.lastStrategyResponse = strategyResponse
	s.sampler = sampler
	return nil
}

func (s *sampler) hasChanges(other jaeger_api_v2.SamplingStrategyResponse) bool {
	s.RLock()
	defer s.RUnlock()

	return s.lastStrategyResponse.StrategyType != other.StrategyType ||
		s.lastStrategyResponse.ProbabilisticSampling != other.ProbabilisticSampling ||
		s.lastStrategyResponse.RateLimitingSampling != other.RateLimitingSampling ||
		s.lastStrategyResponse.OperationSampling != other.OperationSampling
}

// New returns a "go.opentelemetry.io/otel/sdk/trace".Sampler that consults a
// Jaeger remote agent for the sampling strategies for this service.
func New(options ...Option) trace.Sampler {
	cfg := defaultConfig()

	for _, option := range options {
		option.apply(cfg)
	}

	sampler := &sampler{
		fetcher: samplingStrategyFetcherImpl{
			// TODO if no serviceName is set, can we use the one set in the resources?
			serviceName: cfg.service,
			endpoint:    cfg.endpoint,
			httpClient: &http.Client{
				Timeout: 10 * time.Second,
			},
		},
		parser:          samplingStrategyParseImpl{},
		pollingInterval: cfg.pollingInterval,
		sampler:         trace.TraceIDRatioBased(cfg.initialSamplingRate),
	}

	go sampler.pollSamplingStrategies()

	return sampler
}
