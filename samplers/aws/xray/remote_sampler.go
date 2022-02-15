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

// business logic for remote sampling
func (rs *remoteSampler) ShouldSample(parameters sdktrace.SamplingParameters) sdktrace.SamplingResult {
	return sdktrace.SamplingResult{}
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
	// logic for spinning up rule and target poller
}

func main() {
	ctx := context.Background()
	rs, _ := NewRemoteSampler(ctx, "test", "test-platform")

	for i := 0; i < 1000; i++ {
		rs.ShouldSample(sdktrace.SamplingParameters{})
		time.Sleep(250 * time.Millisecond)
	}
}
