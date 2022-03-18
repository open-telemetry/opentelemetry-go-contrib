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

package internal // import "go.opentelemetry.io/contrib/samplers/aws/xray/internal"

import (
	"context"
	crypto "crypto/rand"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/jinzhu/copier"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const manifestTTL = 3600

// Manifest represents a full sampling ruleset and provides
// option for configuring Logger, Clock and xrayClient.
type Manifest struct {
	Rules                          []Rule
	SamplingTargetsPollingInterval time.Duration
	refreshedAt                    time.Time
	xrayClient                     *xrayClient
	clientID                       *string
	logger                         logr.Logger
	clock                          Clock
	mu                             sync.RWMutex
}

// NewManifest return manifest object configured with logging, clock and xrayClient.
func NewManifest(addr url.URL, logger logr.Logger) (*Manifest, error) {
	// generate client for getSamplingRules and getSamplingTargets API call
	client, err := newClient(addr)
	if err != nil {
		return nil, err
	}

	// generate clientID for sampling statistics
	clientID, err := generateClientID()
	if err != nil {
		return nil, err
	}

	return &Manifest{
		xrayClient:                     client,
		clock:                          &DefaultClock{},
		logger:                         logger,
		SamplingTargetsPollingInterval: 10 * time.Second,
		clientID:                       clientID,
	}, nil
}

// Expired returns true if the manifest has not been successfully refreshed in
// manifestTTL seconds.
func (m *Manifest) Expired() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	manifestLiveTime := m.refreshedAt.Add(time.Second * manifestTTL)
	return m.clock.Now().After(manifestLiveTime)
}

// MatchAgainstManifestRules returns a Rule and boolean flag set as true
// if rule has been match against span attributes, otherwise nil and false
func (m *Manifest) MatchAgainstManifestRules(parameters sdktrace.SamplingParameters, serviceName string, cloudPlatform string) (*Rule, bool, error) {
	m.mu.RLock()
	rules := m.Rules
	m.mu.RUnlock()

	matched := false

	for index := range rules {
		isRuleMatch, err := rules[index].appliesTo(parameters, serviceName, cloudPlatform)
		if err != nil {
			return nil, isRuleMatch, err
		}

		if isRuleMatch {
			matched = true
			return &rules[index], matched, nil
		}
	}

	return nil, matched, nil
}

// RefreshManifestRules writes sampling rule properties to the manifest object.
func (m *Manifest) RefreshManifestRules(ctx context.Context) (err error) {
	// get sampling rules from AWS X-Ray console
	rules, err := m.xrayClient.getSamplingRules(ctx)
	if err != nil {
		return err
	}

	// update the retrieved sampling rules to manifest object
	m.updateRules(rules)

	return
}

// RefreshManifestTargets updates sampling targets (statistics) for each rule
func (m *Manifest) RefreshManifestTargets(ctx context.Context) (refresh bool, err error) {
	var manifest Manifest

	// deep copy centralized manifest object to temporary manifest to avoid thread safety issue
	m.mu.RLock()
	err = copier.CopyWithOption(&manifest, m, copier.Option{IgnoreEmpty: false, DeepCopy: true})
	if err != nil {
		return false, err
	}
	m.mu.RUnlock()

	// generate sampling statistics based on the data in temporary manifest
	statistics, err := manifest.snapshots()
	if err != nil {
		return false, err
	}

	// return if no statistics to report
	if len(statistics) == 0 {
		m.logger.V(5).Info("no statistics to report and not refreshing sampling targets")
		return false, nil
	}

	// get sampling targets (statistics) for every expired rule from AWS X-Ray
	targets, err := m.xrayClient.getSamplingTargets(ctx, statistics)
	if err != nil {
		return false, fmt.Errorf("refreshTargets: error occurred while getting sampling targets: %w", err)
	}

	m.logger.V(5).Info("successfully fetched sampling targets")

	// update temporary manifest with retrieved targets (statistics) for each rule
	refresh, err = manifest.updateTargets(targets)
	if err != nil {
		return refresh, err
	}

	// find next polling interval for targets
	minPoll := manifest.minimumPollingInterval()
	if minPoll > 0 {
		m.SamplingTargetsPollingInterval = minPoll
	}

	// update centralized manifest object
	m.mu.Lock()
	m.Rules = manifest.Rules
	m.mu.Unlock()

	return
}

func (m *Manifest) updateRules(rules *getSamplingRulesOutput) {
	tempManifest := Manifest{
		Rules: []Rule{},
	}

	for _, records := range rules.SamplingRuleRecords {
		if records.SamplingRule.RuleName == "" {
			m.logger.V(5).Info("sampling rule without rule name is not supported")
			continue
		}

		if records.SamplingRule.Version != int64(1) {
			m.logger.V(5).Info("sampling rule without Version 1 is not supported", "RuleName", records.SamplingRule.RuleName)
			continue
		}

		// create rule and store it in temporary manifest to avoid thread safety issues
		tempManifest.createRule(*records.SamplingRule)
	}

	// Re-sort to fix matching priorities.
	tempManifest.sort()

	m.mu.Lock()
	m.Rules = tempManifest.Rules
	m.refreshedAt = m.clock.Now()
	m.mu.Unlock()
}

func (m *Manifest) createRule(ruleProp ruleProperties) {
	cr := reservoir{
		capacity: ruleProp.ReservoirSize,
		mu:       &sync.RWMutex{},
	}

	csr := Rule{
		reservoir:          cr,
		ruleProperties:     ruleProp,
		samplingStatistics: &samplingStatistics{},
	}

	m.Rules = append(m.Rules, csr)
}

func (m *Manifest) updateTargets(targets *getSamplingTargetsOutput) (refresh bool, err error) {
	// update sampling targets for each rule
	for _, t := range targets.SamplingTargetDocuments {
		if err := m.updateReservoir(t); err != nil {
			return false, err
		}
	}

	// consume unprocessed statistics messages
	for _, s := range targets.UnprocessedStatistics {
		m.logger.V(5).Info(
			"error occurred updating sampling target for rule, code and message", "RuleName", s.RuleName, "ErrorCode",
			s.ErrorCode,
			"Message", s.Message,
		)

		// do not set any flags if error is unknown
		if s.ErrorCode == nil || s.RuleName == nil {
			continue
		}

		// set batch failure if any sampling statistics returned 5xx
		if strings.HasPrefix(*s.ErrorCode, "5") {
			return false, fmt.Errorf("sampling statistics returned 5xx")
		}

		// set refresh flag if any sampling statistics returned 4xx
		if strings.HasPrefix(*s.ErrorCode, "4") {
			refresh = true
		}
	}

	// set refresh flag if modifiedAt timestamp from remote is greater than ours
	if remote := targets.LastRuleModification; remote != nil {
		// convert unix timestamp to time.Time
		lastRuleModification := time.Unix(int64(*targets.LastRuleModification), 0)

		if lastRuleModification.After(m.refreshedAt) {
			refresh = true
		}
	}

	return
}

func (m *Manifest) updateReservoir(t *samplingTargetDocument) (err error) {
	if t.RuleName == nil {
		return fmt.Errorf("invalid sampling target. Missing rule name")
	}

	if t.FixedRate == nil {
		return fmt.Errorf("invalid sampling target for rule %s. Missing fixed rate", *t.RuleName)
	}

	for index := range m.Rules {
		if m.Rules[index].ruleProperties.RuleName == *t.RuleName {
			m.Rules[index].reservoir.refreshedAt = m.clock.Now()

			// Update non-optional attributes from response
			m.Rules[index].ruleProperties.FixedRate = *t.FixedRate

			// Update optional attributes from response
			if t.ReservoirQuota != nil {
				m.Rules[index].reservoir.quota = *t.ReservoirQuota
			}
			if t.ReservoirQuotaTTL != nil {
				m.Rules[index].reservoir.expiresAt = time.Unix(int64(*t.ReservoirQuotaTTL), 0)
			}
			if t.Interval != nil {
				m.Rules[index].reservoir.interval = time.Duration(*t.Interval)
			}
		}
	}

	return
}

// snapshots takes a snapshot of sampling statistics from all rules, resetting
// statistics counters in the process.
func (m *Manifest) snapshots() ([]*samplingStatisticsDocument, error) {
	statistics := make([]*samplingStatisticsDocument, 0, len(m.Rules)+1)

	// Generate sampling statistics for user-defined rules
	for index := range m.Rules {
		if m.Rules[index].stale(m.clock.Now()) {
			s := m.Rules[index].snapshot(m.clock.Now())
			s.ClientID = m.clientID

			statistics = append(statistics, s)
		}
	}

	return statistics, nil
}

// sort sorts the rule array first by priority and then by rule name.
func (m *Manifest) sort() {
	// comparison function
	less := func(i, j int) bool {
		if m.Rules[i].ruleProperties.Priority == m.Rules[j].ruleProperties.Priority {
			return strings.Compare(m.Rules[i].ruleProperties.RuleName, m.Rules[j].ruleProperties.RuleName) < 0
		}
		return m.Rules[i].ruleProperties.Priority < m.Rules[j].ruleProperties.Priority
	}

	sort.Slice(m.Rules, less)
}

// minimumPollingInterval finds the minimum interval amongst all the targets
func (m *Manifest) minimumPollingInterval() time.Duration {
	if len(m.Rules) == 0 {
		return time.Duration(0)
	}

	minPoll := time.Duration(math.MaxInt64)
	for _, rules := range m.Rules {
		if minPoll >= rules.reservoir.interval {
			minPoll = rules.reservoir.interval
		}
	}

	return minPoll * time.Second
}

// generateClientID generates random client ID
func generateClientID() (*string, error) {
	var r [12]byte

	_, err := crypto.Read(r[:])
	if err != nil {
		return nil, fmt.Errorf("unable to generate client ID: %w", err)
	}

	id := fmt.Sprintf("%02x", r)

	return &id, err
}
