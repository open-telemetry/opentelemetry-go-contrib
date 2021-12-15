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
	crypto "crypto/rand"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// RemoteSampler is an implementation of SamplingStrategy.
type RemoteSampler struct {
	// List of known centralized sampling rules
	manifest *centralizedManifest

	// proxy is used for getting quotas and sampling rules
	xrayClient *xrayClient

	// pollerStart, if true represents rule and target pollers are started
	pollerStart bool

	// Unique ID used by XRay service to identify this client
	clientID string

	// samplingRules polling interval, default is 300 seconds
	samplingRulesPollingInterval time.Duration

	// Provides system time
	clock Clock

	fallbackSampler *FallbackSampler

	mu sync.RWMutex
}

// Compile time assertion that remoteSampler implements the Sampler interface.
var _ sdktrace.Sampler = (*RemoteSampler)(nil)

// NewRemoteSampler returns a centralizedSampler which decides to sample a given request or not.
func NewRemoteSampler(opts ...SamplerOption) *RemoteSampler {
	options := new(samplerOptions).applyOptionsAndDefaults(opts...)

	// Generate clientID
	var r [12]byte

	_, err := crypto.Read(r[:])
	if err != nil {
		log.Println("error reading cryptographically secure random number generator")
		return nil
	}

	id := fmt.Sprintf("%02x", r)

	clock := &DefaultClock{}

	m := &centralizedManifest{
		rules: []*centralizedRule{},
		index: map[string]*centralizedRule{},
		clock: clock,
	}

	return &RemoteSampler{
		pollerStart: false,
		clock:       clock,
		manifest:    m,
		clientID:    id,
		xrayClient: newClient(options.proxyEndpoint),
		samplingRulesPollingInterval: options.samplingRulesPollingInterval,
		fallbackSampler: NewFallbackSampler(),
	}
}

func (rs *RemoteSampler) ShouldSample(parameters sdktrace.SamplingParameters) sdktrace.SamplingResult {
	rs.mu.Lock()
	if !rs.pollerStart {
		rs.start()
	}
	rs.mu.Unlock()

	time.Sleep(5*time.Second)

	// Use fallback sampler with sampling config 1 req/sec and 5% of additional requests if manifest is expired
	if rs.manifest.expired() {
		log.Print("Centralized manifest expired. Using fallback sampling strategy")
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

		log.Printf("Applicable rule: %s", *r.ruleProperties.RuleName)

		samplingResult := r.Sample(parameters)

		return samplingResult
	}

	// Match against default rule
	if r := rs.manifest.defaultRule; r != nil {
		log.Printf("Applicable rule: %s", *r.ruleProperties.RuleName)

		samplingResult := r.Sample(parameters)

		return samplingResult
	}

	// Use fallback sampler with sampling config 1 req/sec and 5% of additional requests
	log.Print("Centralized sampling rules are unavailable. Using fallback sampling strategy")
	return rs.fallbackSampler.ShouldSample(parameters)
}

func (rs *RemoteSampler) Description() string {
	return "remote sampling with AWS X-Ray"
}

func (rs *RemoteSampler) start() {
	if !rs.pollerStart {
		rs.startRulePoller()
		rs.startTargetPoller()
	}

	rs.pollerStart = true
}

// startRulePoller starts rule poller.
func (rs *RemoteSampler) startRulePoller() {
	// Initial refresh
	go func() {
		if err := rs.refreshManifest(); err != nil {
			log.Printf("Error occurred while refreshing sampling rules. %v\n", err)
		} else {
			log.Println("Successfully fetched sampling rules")
		}
	}()

	// Periodic manifest refresh
	go func() {
		// Period = 300s, Jitter = 5s
		t := newTicker(rs.samplingRulesPollingInterval, 5*time.Second)

		for range t.C() {
			if err := rs.refreshManifest(); err != nil {
				log.Printf("Error occurred while refreshing sampling rules. %v\n", err)
			} else {
				log.Println("Successfully fetched sampling rules")
			}
		}
	}()
}

// startTargetPoller starts target poller.
func (rs *RemoteSampler) startTargetPoller() {
	// Periodic quota refresh
	go func() {
		// Period = 10.1s, Jitter = 100ms
		t := newTicker(10*time.Second+100*time.Millisecond, 100*time.Millisecond)

		for range t.C() {
			if err := rs.refreshTargets(); err != nil {
				log.Printf("Error occurred while refreshing targets for sampling rules. %v", err)
			}
		}
	}()
}

func (rs *RemoteSampler) refreshManifest() (err error) {
	// Explicitly recover from panics since this is the entry point for a long-running goroutine
	// and we can not allow a panic to propagate to the application code.
	defer func() {
		if r := recover(); r != nil {
			// Resort to bring rules array into consistent state.
			//cs.manifest.sort()

			err = fmt.Errorf("%v", r)
		}
	}()

	// Compute 'now' before calling GetSamplingRules to avoid marking manifest as
	// fresher than it actually is.
	now := rs.clock.Now().Unix()

	// Get sampling rules from proxy
	rules, err := rs.xrayClient.getSamplingRules(context.Background())
	if err != nil {
		return
	}

	// Set of rules to exclude from pruning
	actives := map[centralizedRule]bool{}

	// Create missing rules. Update existing ones.
	failed := false

	for _, records := range rules.SamplingRuleRecords {
		if records.SamplingRule.RuleName == nil {
			log.Println("Sampling rule without rule name is not supported")
			failed = true
			continue
		}

		// Only sampling rule with version 1 is valid
		if records.SamplingRule.Version == nil {
			log.Println("Sampling rule without version number is not supported: ", *records.SamplingRule.RuleName)
			failed = true
			continue
		}

		if *records.SamplingRule.Version != int64(1) {
			log.Println("Sampling rule without version 1 is not supported: ", *records.SamplingRule.RuleName)
			failed = true
			continue
		}

		if len(records.SamplingRule.Attributes) != 0 {
			log.Println("Sampling rule with non nil Attributes is not applicable: ", *records.SamplingRule.RuleName)
			continue
		}

		if records.SamplingRule.ResourceARN == nil {
			log.Println("Sampling rule without ResourceARN is not applicable: ", *records.SamplingRule.RuleName)
			continue
		}

		if *records.SamplingRule.ResourceARN != "*" {
			log.Println("Sampling rule with ResourceARN not equal to * is not applicable: ", *records.SamplingRule.RuleName)
			continue
		}

		// Create/update rule
		r, putErr := rs.manifest.putRule(records.SamplingRule)
		if putErr != nil {
			failed = true
			log.Printf("Error occurred creating/updating rule. %v\n", putErr)
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
func (rs *RemoteSampler) refreshTargets() (err error) {
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
		log.Print("No statistics to report. Not refreshing sampling targets.")
		return nil
	}

	// Get sampling targets
	output, err := rs.xrayClient.getSamplingTargets(context.Background(), statistics)
	if err != nil {
		return
	}

	// Update sampling targets
	for _, t := range output.SamplingTargetDocuments {
		if err = rs.updateTarget(t); err != nil {
			failed = true
			log.Printf("Error occurred updating target for rule. %v", err)
		}
	}

	// Consume unprocessed statistics messages
	for _, s := range output.UnprocessedStatistics {
		log.Printf(
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
		log.Print("Successfully refreshed sampling targets")
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
		log.Print("Refreshing sampling rules out-of-band.")

		go func() {
			if err := rs.refreshManifest(); err != nil {
				log.Printf("Error occurred refreshing sampling rules out-of-band. %v", err)
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

	// Generate sampling statistics for default rule
	if r := rs.manifest.defaultRule; r != nil && r.stale(now) {
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

func main() {
	rs := NewRemoteSampler()

	for i:=0; i< 30;i++ {
		rs.ShouldSample(sdktrace.SamplingParameters{})
		time.Sleep(20*time.Second)
	}
}
