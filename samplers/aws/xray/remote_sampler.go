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
	"context"
	"time"

	"go.opentelemetry.io/contrib/samplers/aws/xray/internal/util"

	"go.opentelemetry.io/contrib/samplers/aws/xray/internal"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/go-logr/logr"
)

// remoteSampler is a sampler for AWS X-Ray which polls sampling rules and sampling targets
// to make a sampling decision based on rules set by users on AWS X-Ray console
type remoteSampler struct {
	// manifest is the list of known centralized sampling rules.
	manifest *internal.Manifest

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
}

// Compile time assertion that remoteSampler implements the Sampler interface.
var _ sdktrace.Sampler = (*remoteSampler)(nil)

// NewRemoteSampler returns a sampler which decides to sample a given request or not
// based on the sampling rules set by users on AWS X-Ray console. Sampler also periodically polls
// sampling rules and sampling targets.
func NewRemoteSampler(ctx context.Context, serviceName string, cloudPlatform string, opts ...Option) (sdktrace.Sampler, error) {
	// create new config based on options or set to default values
	cfg, err := newConfig(opts...)
	if err != nil {
		return nil, err
	}

	// create manifest with config
	m, err := internal.NewManifest(cfg.endpoint, cfg.logger)
	if err != nil {
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

// ShouldSample matches span attributes with retrieved sampling rules and perform sampling,
// if rules does not match or manifest is expired then use fallback sampling.
func (rs *remoteSampler) ShouldSample(parameters sdktrace.SamplingParameters) sdktrace.SamplingResult {
	if rs.manifest.Expired() {
		// Use fallback sampler if manifest is expired
		rs.logger.V(5).Info("manifest is expired so using fallback sampling strategy")

		return rs.fallbackSampler.ShouldSample(parameters)
	}

	// match against known rules
	r, match, err := rs.manifest.MatchAgainstManifestRules(parameters, rs.serviceName, rs.cloudPlatform)
	if err != nil {
		rs.logger.Error(err, "regexp matching error while matching span and resource attributes")
		return sdktrace.SamplingResult{}
	}

	if match {
		// remote sampling based on rule match
		return r.Sample(parameters, time.Now())
	}

	// Use fallback sampler if
	// sampling rules does not match against manifest
	rs.logger.V(5).Info("span attributes does not match to the sampling rules or manifest is expired so using fallback sampling strategy")
	return rs.fallbackSampler.ShouldSample(parameters)
}

// Description returns description of the sampler being used.
func (rs *remoteSampler) Description() string {
	return "AwsXrayRemoteSampler{remote sampling with AWS X-Ray}"
}

func (rs *remoteSampler) start(ctx context.Context) {
	if !rs.pollerStarted {
		rs.pollerStarted = true
		go rs.startPoller(ctx)
	}
}

// startPoller starts the rule and target poller in a single go routine which runs periodically
// to refresh manifest and targets.
func (rs *remoteSampler) startPoller(ctx context.Context) {
	// jitter = 5s, default 300 seconds
	rulesTicker := util.NewTicker(rs.samplingRulesPollingInterval, 5*time.Second)

	// jitter = 100ms, default 10 seconds
	targetTicker := util.NewTicker(rs.manifest.SamplingTargetsPollingInterval, 100*time.Millisecond)

	// fetch sampling rules to kick start the remote sampling
	rs.refreshManifest(ctx)

	for {
		select {
		case _, more := <-rulesTicker.C():
			if !more {
				return
			}

			// fetch sampling rules and updates manifest
			rs.refreshManifest(ctx)
			continue
		case _, more := <-targetTicker.C():
			if !more {
				return
			}

			// fetch sampling targets and updates manifest
			refresh := rs.refreshTargets(ctx)

			// out of band manifest refresh if it manifest is not updated
			if refresh {
				rs.refreshManifest(ctx)
			}
			continue
		case <-ctx.Done():
			return
		}
	}
}

// refreshManifest refreshes the manifest retrieved via getSamplingRules API.
func (rs *remoteSampler) refreshManifest(ctx context.Context) {
	if err := rs.manifest.RefreshManifestRules(ctx); err != nil {
		rs.logger.Error(err, "Error occurred while refreshing sampling rules")
	} else {
		rs.logger.V(5).Info("Successfully fetched sampling rules")
	}
}

// refreshTarget refreshes the sampling targets in manifest retrieved via getSamplingTargets API.
func (rs *remoteSampler) refreshTargets(ctx context.Context) bool {
	refresh := false
	var err error
	if refresh, err = rs.manifest.RefreshManifestTargets(ctx); err != nil {
		rs.logger.Error(err, "error occurred while refreshing sampling rule targets")
	}
	return refresh
}
