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

package main

import (
	"context"
	"go.opentelemetry.io/contrib/samplers/aws/xray/internal_xray"
	"sync"
	"time"

	"github.com/go-logr/logr"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// remoteSampler is a sampler for AWS X-Ray which polls sampling rules and sampling targets
// to make a sampling decision based on rules set by users on AWS X-Ray console
type remoteSampler struct {
	// manifest is the list of known centralized sampling rules.
	manifest *internal_xray.Manifest

	// pollerStarted, if true represents rule and target pollers are started.
	pollerStarted bool

	// samplingRulesPollingInterval, default is 300 seconds.
	samplingRulesPollingInterval time.Duration

	// matching attribute
	serviceName string

	// matching attribute
	cloudPlatform string

	// fallback sampler
	fallbackSampler *FallbackSampler

	// logger for logging
	logger logr.Logger

	mu sync.RWMutex
}

// Compile time assertion that remoteSampler implements the Sampler interface.
var _ sdktrace.Sampler = (*remoteSampler)(nil)

// NewRemoteSampler returns a sampler which decides to sample a given request or not
// based on the sampling rules set by users on AWS X-Ray console. Sampler also periodically polls
// sampling rules and sampling targets.
func NewRemoteSampler(ctx context.Context, serviceName string, cloudPlatform string, opts ...Option) (sdktrace.Sampler, error) {
	cfg := newConfig(opts...)

	// validate config
	err := validateConfig(cfg)
	if err != nil {
		return nil, err
	}

	// create manifest with config
	m, err := internal_xray.NewManifest(cfg.endpoint, cfg.logger); if err != nil {
		return nil, err
	}

	remoteSampler := &remoteSampler{
		manifest:                     m,
		samplingRulesPollingInterval: cfg.samplingRulesPollingInterval,
		fallbackSampler:              NewFallbackSampler(),
		serviceName:                  serviceName,
		cloudPlatform:                cloudPlatform,
		logger:                       cfg.logger,
	}

	// starts the rule and target poller
	remoteSampler.start(ctx)

	return remoteSampler, nil
}

func (rs *remoteSampler) ShouldSample(parameters sdktrace.SamplingParameters) sdktrace.SamplingResult {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	// Use fallback sampler when manifest is expired
	if rs.manifest.Expired() {
		rs.logger.V(5).Info("Centralized manifest expired. Using fallback sampling strategy")
		return rs.fallbackSampler.ShouldSample(parameters)
	}

	// Match against known rules
	for _, r := range rs.manifest.Rules {
		applicable := r.AppliesTo(parameters, rs.serviceName, rs.cloudPlatform)

		if applicable {
			return r.Sample(parameters)
		}
	}

	// Use fallback sampler when request does not match against known rules
	rs.logger.V(5).Info("No match against centralized rules using fallback sampling strategy")
	return rs.fallbackSampler.ShouldSample(parameters)
}

func (rs *remoteSampler) Description() string {
	return "AwsXrayRemoteSampler{" + rs.getDescription() + "}"
}

func (rs *remoteSampler) getDescription() string {
	return "remote sampling with AWS X-Ray"
}

func (rs *remoteSampler) start(ctx context.Context) {
	if !rs.pollerStarted {
		rs.pollerStarted = true
		rs.startPoller(ctx)
	}
}

// startPoller starts the rule and target poller in a separate go routine which runs periodically to refresh manifest and
// targets
func (rs *remoteSampler) startPoller(ctx context.Context) {
//	go func() {
		// Period = 300s, Jitter = 5s
		rulesTicker := newTicker(rs.samplingRulesPollingInterval, 5*time.Second)

		// Period = 10.1s, Jitter = 100ms
		targetTicker := newTicker(5*time.Second+100*time.Millisecond, 100*time.Millisecond)

		// fetch sampling rules
		if err := rs.manifest.RefreshManifestRules(ctx); err != nil {
			rs.logger.Error(err, "Error occurred while refreshing sampling rules")
		} else {
			rs.logger.V(5).Info("Successfully fetched sampling rules")
		}

		for {
			select {
			case _, more := <-rulesTicker.C():
				if !more {
					return
				}

				// fetch sampling rules
				if err := rs.manifest.RefreshManifestRules(ctx); err != nil {
					rs.logger.Error(err, "Error occurred while refreshing sampling rules")
				} else {
					rs.logger.V(5).Info("Successfully fetched sampling rules")
				}
				continue
			case _, more := <-targetTicker.C():
				if !more {
					return
				}

				// fetch sampling targets
				if err := rs.manifest.RefreshManifestTargets(ctx); err != nil {
					rs.logger.Error(err, "Error occurred while refreshing targets for sampling rules")
				}
				continue
			case <-ctx.Done():
				return
			}
		}
//	}()
}

func main() {
	ctx := context.Background()
	rs, _ := NewRemoteSampler(ctx, "test", "test-platform")

	for i := 0; i < 1000; i++ {
		rs.ShouldSample(sdktrace.SamplingParameters{})
		time.Sleep(250 * time.Millisecond)
	}
}
