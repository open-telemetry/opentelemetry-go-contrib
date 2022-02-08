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

package xray

import (
	"context"
	crypto "crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// global variable for logging
var globalLogger logr.Logger

// remoteSampler is a sampler for AWS X-Ray which polls sampling rules and sampling targets
// to make a sampling decision based on rules set by users on AWS X-Ray console
type remoteSampler struct {
	// manifest is the list of known centralized sampling rules.
	manifest *manifest

	// xrayClient is used for getting quotas and sampling rules.
	xrayClient *xrayClient

	// pollerStarted, if true represents rule and target pollers are started.
	pollerStarted bool

	// samplingRulesPollingInterval, default is 300 seconds.
	samplingRulesPollingInterval time.Duration

	// Unique ID used by XRay service to identify this client.
	clientID string

	// Provides time.
	clock clock

	mu sync.RWMutex
}

// Compile time assertion that remoteSampler implements the Sampler interface.
var _ sdktrace.Sampler = (*remoteSampler)(nil)

// NewRemoteSampler returns a sampler which decides to sample a given request or not
// based on the sampling rules set by users on AWS X-Ray console. Sampler also periodically polls
// sampling rules and sampling targets.
func NewRemoteSampler(ctx context.Context, opts ...Option) (sdktrace.Sampler, error) {
	cfg := newConfig(opts...)

	// validate config
	err := validateConfig(cfg)
	if err != nil {
		return nil, err
	}

	// Generate clientID
	var r [12]byte

	_, err = crypto.Read(r[:])
	if err != nil {
		return nil, fmt.Errorf("unable to generate client ID: %w", err)
	}

	id := fmt.Sprintf("%02x", r)

	clock := &defaultClock{}

	m := &manifest{
		rules: []*rule{},
		index: map[string]*rule{},
		clock: clock,
	}

	client, err := newClient(cfg.endpoint)
	if err != nil {
		return nil, err
	}

	remoteSampler := &remoteSampler{
		clock:                        clock,
		manifest:                     m,
		clientID:                     id,
		xrayClient:                   client,
		samplingRulesPollingInterval: cfg.samplingRulesPollingInterval,
	}

	// starts the rule poller
	remoteSampler.start(ctx)

	return remoteSampler, nil
}

func (rs *remoteSampler) ShouldSample(parameters sdktrace.SamplingParameters) sdktrace.SamplingResult {
	// ToDo: add business logic for remote sampling

	return sdktrace.TraceIDRatioBased(0.05).ShouldSample(parameters)
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

func (rs *remoteSampler) startPoller(ctx context.Context) {
	// ToDo: add logic for getting sampling targets
	go func() {
		// Period = 300s, Jitter = 5s
		t := newTicker(rs.samplingRulesPollingInterval, 5*time.Second)

		// Periodic manifest refresh
		for {
			if err := rs.refreshManifest(ctx); err != nil {
				globalLogger.Error(err, "Error occurred while refreshing sampling rules")
			} else {
				globalLogger.V(1).Info("Successfully fetched sampling rules")
			}
			select {
			case _, more := <-t.C():
				if !more {
					return
				}
				continue
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (rs *remoteSampler) refreshManifest(ctx context.Context) (err error) {
	// Compute 'now' before calling GetSamplingRules to avoid marking manifest as
	// fresher than it actually is.
	now := rs.clock.now()

	// Get sampling rules from proxy.
	rules, err := rs.xrayClient.getSamplingRules(ctx)
	if err != nil {
		return
	}

	// temporary manifest declaration.
	tempManifest := &manifest{
		rules: []*rule{},
		index: map[string]*rule{},
		clock: &defaultClock{},
	}

	for _, records := range rules.SamplingRuleRecords {
		if records.SamplingRule.RuleName == nil {
			globalLogger.V(1).Info("Sampling rule without rule name is not supported")
			continue
		}

		// Only sampling rule with version 1 is valid
		if records.SamplingRule.Version == nil {
			globalLogger.V(1).Info("Sampling rule without Version is not supported", "RuleName", *records.SamplingRule.RuleName)
			continue
		}

		if *records.SamplingRule.Version != int64(1) {
			globalLogger.V(1).Info("Sampling rule without Version 1 is not supported", "RuleName", *records.SamplingRule.RuleName)
			continue
		}

		if records.SamplingRule.ServiceName == nil {
			globalLogger.V(1).Info("Sampling rule without ServiceName is not supported", "RuleName", *records.SamplingRule.RuleName)
			continue
		}

		if records.SamplingRule.ServiceType == nil {
			globalLogger.V(1).Info("Sampling rule without ServiceType is not supported", "RuleName", *records.SamplingRule.RuleName)
			continue
		}

		if records.SamplingRule.Host == nil {
			globalLogger.V(1).Info("Sampling rule without Host is not supported", "RuleName", *records.SamplingRule.RuleName)
			continue
		}

		if records.SamplingRule.HTTPMethod == nil {
			globalLogger.V(1).Info("Sampling rule without HTTPMethod is not supported", "RuleName", *records.SamplingRule.RuleName)
			continue
		}

		if records.SamplingRule.URLPath == nil {
			globalLogger.V(1).Info("Sampling rule without URLPath is not supported", "RuleName", *records.SamplingRule.RuleName)
			continue
		}

		if records.SamplingRule.ReservoirSize == nil {
			globalLogger.V(1).Info("Sampling rule without ReservoirSize  is not supported", "RuleName", *records.SamplingRule.RuleName)
			continue
		}

		if records.SamplingRule.FixedRate == nil {
			globalLogger.V(1).Info("Sampling rule without FixedRate is not supported", "RuleName", *records.SamplingRule.RuleName)
			continue
		}

		if records.SamplingRule.ResourceARN == nil {
			globalLogger.V(1).Info("Sampling rule without ResourceARN is not applicable", "RuleName", *records.SamplingRule.RuleName)
			continue
		}

		if records.SamplingRule.Priority == nil {
			globalLogger.V(1).Info("Sampling rule without version number is not supported", "RuleName", *records.SamplingRule.RuleName)
			continue
		}

		// create rule and store it in temporary manifest to avoid locking issues.
		createErr := tempManifest.createRule(records.SamplingRule)
		if createErr != nil {
			globalLogger.Error(createErr, "Error occurred creating/updating rule")
		}
	}

	// Re-sort to fix matching priorities.
	tempManifest.sort()
	// Update refreshedAt timestamp
	tempManifest.refreshedAt = now

	// assign temp manifest to original copy/one sync refresh.
	rs.mu.Lock()
	rs.manifest = tempManifest
	rs.mu.Unlock()

	return
}
