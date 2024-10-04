// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package xray provide an OpenTelemetry sampler for the AWS XRAY platform.
//
// Deprecated: xray has no Code Owner.
// After August 21, 2024, it may no longer be supported and may stop
// receiving new releases unless a new Code Owner is found. See
// [this issue] if you would like to become the Code Owner of this module.
//
// [this issue]: https://github.com/open-telemetry/opentelemetry-go-contrib/issues/5554
package xray // import "go.opentelemetry.io/contrib/samplers/aws/xray"

import (
	"context"
	"time"

	"go.opentelemetry.io/contrib/samplers/aws/xray/internal"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/go-logr/logr"
)

// remoteSampler is a sampler for AWS X-Ray which polls sampling rules and sampling targets
// to make a sampling decision based on rules set by users on AWS X-Ray console.
type remoteSampler struct {
	// manifest is the list of known centralized sampling rules.
	manifest *internal.Manifest

	// pollerStarted, if true represents rule and target pollers are started.
	pollerStarted bool

	// samplingRulesPollingInterval, default is 300 seconds.
	samplingRulesPollingInterval time.Duration

	serviceName string

	cloudPlatform string

	fallbackSampler *FallbackSampler

	// logger for logging.
	logger logr.Logger
}

// Compile time assertion that remoteSampler implements the Sampler interface.
var _ sdktrace.Sampler = (*remoteSampler)(nil)

// NewRemoteSampler returns a ParentBased XRay Sampler which decides to sample a given request or not
// based on the sampling rules set by users on AWS X-Ray console. Sampler also periodically polls
// sampling rules and sampling targets.
// NOTE: ctx passed in NewRemoteSampler API is being used in background go routine. Cancellation to this context can kill the background go routine.
// serviceName refers to the name of the service equivalent to the one set in the AWS X-Ray console when adding sampling rules and
// cloudPlatform refers to the cloud platform the service is running on ("ec2", "ecs", "eks", "lambda", etc).
// Guide on AWS X-Ray remote sampling implementation (https://aws-otel.github.io/docs/getting-started/remote-sampling#otel-remote-sampling-implementation-caveats).
func NewRemoteSampler(ctx context.Context, serviceName string, cloudPlatform string, opts ...Option) (sdktrace.Sampler, error) {
	// Create new config based on options or set to default values.
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

	remoteSampler.start(ctx)

	return sdktrace.ParentBased(remoteSampler), nil
}

// ShouldSample matches span attributes with retrieved sampling rules and returns a sampling result.
// If the sampling parameters do not match or the manifest is expired then the fallback sampler is used.
func (rs *remoteSampler) ShouldSample(parameters sdktrace.SamplingParameters) sdktrace.SamplingResult {
	if rs.manifest.Expired() {
		// Use fallback sampler if manifest is expired.
		rs.logger.V(5).Info("manifest is expired so using fallback sampling strategy")

		return rs.fallbackSampler.ShouldSample(parameters)
	}

	r, match, err := rs.manifest.MatchAgainstManifestRules(parameters, rs.serviceName, rs.cloudPlatform)
	if err != nil {
		rs.logger.Error(err, "rule matching error, using fallback sampler")
		return rs.fallbackSampler.ShouldSample(parameters)
	}

	if match {
		// Remote sampling based on rule match.
		return r.Sample(parameters, time.Now())
	}

	// Use fallback sampler if sampling rules does not match against manifest.
	rs.logger.V(5).Info("span does not match rules from manifest(or it is expired), using fallback sampler")
	return rs.fallbackSampler.ShouldSample(parameters)
}

// Description returns description of the sampler being used.
func (rs *remoteSampler) Description() string {
	return "AWSXRayRemoteSampler{remote sampling with AWS X-Ray}"
}

func (rs *remoteSampler) start(ctx context.Context) {
	if !rs.pollerStarted {
		rs.pollerStarted = true
		go rs.startPoller(ctx)
	}
}

// startPoller starts the rule and target poller in a single go routine which runs periodically
// to refresh the manifest and targets.
func (rs *remoteSampler) startPoller(ctx context.Context) {
	// jitter = 5s, default duration 300 seconds.
	rulesTicker := newTicker(rs.samplingRulesPollingInterval, 5*time.Second)
	defer rulesTicker.tick.Stop()

	// jitter = 100ms, default duration 10 seconds.
	targetTicker := newTicker(rs.manifest.SamplingTargetsPollingInterval, 100*time.Millisecond)
	defer targetTicker.tick.Stop()

	// Fetch sampling rules to kick start the remote sampling.
	rs.refreshManifest(ctx)

	for {
		select {
		case _, more := <-rulesTicker.c():
			if !more {
				return
			}

			rs.refreshManifest(ctx)
			continue
		case _, more := <-targetTicker.c():
			if !more {
				return
			}

			refresh := rs.refreshTargets(ctx)

			// If LastRuleModification time is more recent than manifest refresh time,
			// then we explicitly perform refreshing the manifest.
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
		rs.logger.Error(err, "error occurred while refreshing sampling rules")
	} else {
		rs.logger.V(5).Info("successfully fetched sampling rules")
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
