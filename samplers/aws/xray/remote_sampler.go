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
	"errors"
	"fmt"
	"strings"
	"time"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// global variable for logging
var globalLogger Logger

// RemoteSampler is an implementation of SamplingStrategy.
type RemoteSampler struct {
	// List of known centralized sampling rules
	manifest *centralizedManifest

	// proxy is used for getting quotas and sampling rules
	xrayClient *xrayClient

	// pollerStart, if true represents rule and target pollers are started
	pollerStart bool

	// samplingRules polling interval, default is 300 seconds
	samplingRulesPollingInterval time.Duration

	// Unique ID used by XRay service to identify this client
	clientID string

	// Provides system time
	clock Clock

	fallbackSampler *FallbackSampler
}

// Compile time assertion that remoteSampler implements the Sampler interface.
var _ sdktrace.Sampler = (*RemoteSampler)(nil)

// NewRemoteSampler returns a centralizedSampler which decides to sample a given request or not.
func NewRemoteSampler(ctx context.Context, opts ...Option) (*RemoteSampler, error) {
	cfg := newConfig(opts...)

	// Generate clientID
	var r [12]byte

	_, err := crypto.Read(r[:])
	if err != nil {
		return nil, fmt.Errorf("unable to generate client ID: %w", err)
	}

	id := fmt.Sprintf("%02x", r)

	clock := &DefaultClock{}

	m := &centralizedManifest{
		rules: []*centralizedRule{},
		index: map[string]*centralizedRule{},
		clock: clock,
	}

	remoteSampler := &RemoteSampler{
		pollerStart:                  false,
		clock:                        clock,
		manifest:                     m,
		clientID:                     id,
		xrayClient:                   newClient(cfg.proxyEndpoint),
		samplingRulesPollingInterval: cfg.samplingRulesPollingInterval,
		fallbackSampler:              NewFallbackSampler(),
	}

	// starts the rule and target poller
	remoteSampler.start(ctx)

	return remoteSampler, nil
}

func (rs *RemoteSampler) ShouldSample(parameters sdktrace.SamplingParameters) sdktrace.SamplingResult {
	// Use fallback sampler with sampling config 1 req/sec and 5% of additional requests if manifest is expired
	if rs.manifest.expired() {
		globalLogger.Printf("Centralized manifest expired. Using fallback sampling strategy")
		return rs.fallbackSampler.ShouldSample(parameters)
	}

	rs.manifest.mu.RLock()
	defer rs.manifest.mu.RUnlock()

	// Match against known rules
	for _, r := range rs.manifest.rules {

		r.mu.RLock()
		applicable := r.AppliesTo()
		r.mu.RUnlock()

		if !applicable {
			continue
		}

		globalLogger.Printf("Applicable rule: %s", *r.ruleProperties.RuleName)

		samplingResult := r.Sample(parameters)

		return samplingResult
	}

	// Use fallback sampler with sampling config 1 req/sec and 5% of additional requests
	globalLogger.Printf("Centralized sampling rules are unavailable. Using fallback sampling strategy")
	return rs.fallbackSampler.ShouldSample(parameters)
}

func (rs *RemoteSampler) Description() string {
	return "remote sampling with AWS X-Ray"
}

func (rs *RemoteSampler) start(ctx context.Context) {
	if !rs.pollerStart {
		rs.pollerStart = true
		rs.startRulePoller(ctx)
		rs.startTargetPoller(ctx)
	}
}

// startRulePoller starts rule poller.
func (rs *RemoteSampler) startRulePoller(ctx context.Context) {
	go func() {
		// Period = 300s, Jitter = 5s
		t := newTicker(rs.samplingRulesPollingInterval, 5*time.Second)

		// Periodic manifest refresh
		for {
			if err := rs.refreshManifest(ctx); err != nil {
				globalLogger.Printf("Error occurred while refreshing sampling rules. %v\n", err)
			} else {
				globalLogger.Println("Successfully fetched sampling rules")
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

// startTargetPoller starts target poller.
func (rs *RemoteSampler) startTargetPoller(ctx context.Context) {
	go func() {
		// Period = 10.1s, Jitter = 100ms
		t := newTicker(10*time.Second+100*time.Millisecond, 100*time.Millisecond)

		// Periodic quota refresh
		for {
			if err := rs.refreshTargets(ctx); err != nil {
				globalLogger.Printf("Error occurred while refreshing targets for sampling rules. %v", err)
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

func (rs *RemoteSampler) refreshManifest(ctx context.Context) (err error) {
	// Explicitly recover from panics since this is the entry point for a long-running goroutine
	// and we can not allow a panic to propagate to the application code.
	defer func() {
		if r := recover(); r != nil {
			// Resort to bring rules array into consistent state.
			rs.manifest.sort()

			err = fmt.Errorf("%v", r)
		}
	}()

	// Compute 'now' before calling GetSamplingRules to avoid marking manifest as
	// fresher than it actually is.
	now := rs.clock.Now().Unix()

	// Get sampling rules from proxy
	rules, err := rs.xrayClient.getSamplingRules(ctx)
	if err != nil {
		return
	}

	// Set of rules to exclude from pruning
	actives := map[centralizedRule]bool{}

	// Create missing rules. Update existing ones.
	failed := false

	for _, records := range rules.SamplingRuleRecords {
		if records.SamplingRule.RuleName == nil {
			globalLogger.Println("Sampling rule without rule name is not supported")
			failed = true
			continue
		}

		// Only sampling rule with version 1 is valid
		if records.SamplingRule.Version == nil {
			globalLogger.Println("Sampling rule without version number is not supported: ", *records.SamplingRule.RuleName)
			failed = true
			continue
		}

		if *records.SamplingRule.Version != int64(1) {
			globalLogger.Println("Sampling rule without version 1 is not supported: ", *records.SamplingRule.RuleName)
			failed = true
			continue
		}

		if len(records.SamplingRule.Attributes) != 0 {
			globalLogger.Println("Sampling rule with non nil Attributes is not applicable: ", *records.SamplingRule.RuleName)
			continue
		}

		if records.SamplingRule.ResourceARN == nil {
			globalLogger.Println("Sampling rule without ResourceARN is not applicable: ", *records.SamplingRule.RuleName)
			continue
		}

		if *records.SamplingRule.ResourceARN != "*" {
			globalLogger.Println("Sampling rule with ResourceARN not equal to * is not applicable: ", *records.SamplingRule.RuleName)
			continue
		}

		// Create/update rule
		r, putErr := rs.manifest.putRule(records.SamplingRule)
		if putErr != nil {
			failed = true
			globalLogger.Printf("Error occurred creating/updating rule. %v\n", putErr)
		} else if r != nil {
			actives[*r] = true
		}
	}

	// Set err if updates failed
	if failed {
		err = errors.New("error occurred creating/updating rules")
	}

	// Prune inactive rules
	rs.manifest.prune(actives)

	// Re-sort to fix matching priorities
	rs.manifest.sort()

	// Update refreshedAt timestamp
	rs.manifest.mu.Lock()
	rs.manifest.refreshedAt = now
	rs.manifest.mu.Unlock()

	return
}

// refreshTargets refreshes targets for sampling rules. It calls the XRay service proxy with sampling
// statistics for the previous interval and receives targets for the next interval.
func (rs *RemoteSampler) refreshTargets(ctx context.Context) (err error) {
	// Explicitly recover from panics since this is the entry point for a long-running goroutine
	// and we can not allow a panic to propagate to customer code.
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()

	// Flag indicating batch failure
	failed := false

	// Flag indicating whether or not manifest should be refreshed
	refresh := false

	// Generate sampling statistics
	statistics := rs.snapshots()

	// Do not refresh targets if no statistics to report
	if len(statistics) == 0 {
		globalLogger.Printf("No statistics to report. Not refreshing sampling targets.")
		return nil
	}

	// Get sampling targets
	output, err := rs.xrayClient.getSamplingTargets(ctx, statistics)
	if err != nil {
		return fmt.Errorf("refreshTargets: Error occurred while getting sampling targets: %w", err)
	}

	// Update sampling targets
	for _, t := range output.SamplingTargetDocuments {
		if err = rs.updateTarget(t); err != nil {
			failed = true
			globalLogger.Printf("Error occurred updating target for rule. %v", err)
		}
	}

	// Consume unprocessed statistics messages
	for _, s := range output.UnprocessedStatistics {
		globalLogger.Printf(
			"Error occurred updating sampling target for rule: %s, code: %s, message: %s",
			*s.RuleName,
			*s.ErrorCode,
			*s.Message,
		)

		// Do not set any flags if error is unknown
		if s.ErrorCode == nil || s.RuleName == nil {
			continue
		}

		// Set batch failure if any sampling statistics return 5xx
		if strings.HasPrefix(*s.ErrorCode, "5") {
			failed = true
		}

		// Set refresh flag if any sampling statistics return 4xx
		if strings.HasPrefix(*s.ErrorCode, "4") {
			refresh = true
		}
	}

	// Set err if updates failed
	if failed {
		err = errors.New("error occurred updating sampling targets")
	} else {
		globalLogger.Printf("Successfully refreshed sampling targets")
	}

	// Set refresh flag if modifiedAt timestamp from remote is greater than ours.
	if remote := output.LastRuleModification; remote != nil {
		rs.manifest.mu.RLock()
		local := rs.manifest.refreshedAt
		rs.manifest.mu.RUnlock()

		if int64(*remote) >= local {
			refresh = true
		}
	}

	// Perform out-of-band async manifest refresh if flag is set
	if refresh {
		globalLogger.Printf("Refreshing sampling rules out-of-band.")

		go func() {
			if err := rs.refreshManifest(ctx); err != nil {
				globalLogger.Printf("Error occurred refreshing sampling rules out-of-band. %v", err)
			}
		}()
	}

	return
}

// samplingStatistics takes a snapshot of sampling statistics from all rules, resetting
// statistics counters in the process.
func (rs *RemoteSampler) snapshots() []*samplingStatisticsDocument {
	now := rs.clock.Now().Unix()

	rs.manifest.mu.RLock()
	defer rs.manifest.mu.RUnlock()

	statistics := make([]*samplingStatisticsDocument, 0, len(rs.manifest.rules)+1)

	// Generate sampling statistics for user-defined rules
	for _, r := range rs.manifest.rules {
		if !r.stale(now) {
			continue
		}

		s := r.snapshot()
		s.ClientID = &rs.clientID

		statistics = append(statistics, s)
	}

	return statistics
}

// updateTarget updates sampling targets for the rule specified in the target struct.
func (rs *RemoteSampler) updateTarget(t *samplingTargetDocument) (err error) {
	// Pre-emptively dereference xraySvc.SamplingTarget fields and return early on nil values
	// A panic in the middle of an update may leave the rule in an inconsistent state.
	if t.RuleName == nil {
		return errors.New("invalid sampling target. Missing rule name")
	}

	if t.FixedRate == nil {
		return fmt.Errorf("invalid sampling target for rule %s. Missing fixed rate", *t.RuleName)
	}

	// Rule for given target
	rs.manifest.mu.RLock()
	r, ok := rs.manifest.index[*t.RuleName]
	rs.manifest.mu.RUnlock()

	if !ok {
		return fmt.Errorf("rule %s not found", *t.RuleName)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.reservoir.refreshedAt = rs.clock.Now().Unix()

	// Update non-optional attributes from response
	*r.ruleProperties.FixedRate = *t.FixedRate

	// Update optional attributes from response
	if t.ReservoirQuota != nil {
		r.reservoir.quota = *t.ReservoirQuota
	}
	if t.ReservoirQuotaTTL != nil {
		r.reservoir.expiresAt = int64(*t.ReservoirQuotaTTL)
	}
	if t.Interval != nil {
		r.reservoir.interval = *t.Interval
	}

	return nil
}
