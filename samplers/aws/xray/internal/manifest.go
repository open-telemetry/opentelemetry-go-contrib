package internal

import (
	"context"
	crypto "crypto/rand"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/jinzhu/copier"
	"go.opentelemetry.io/contrib/samplers/aws/xray/internal/util"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"sort"
	"strings"
	"sync"
	"time"
)

const defaultInterval = int64(10)

const manifestTTL = 3600

// Manifest represents a full sampling ruleset and provides
// option for configuring Logger, Clock and xrayClient.
type Manifest struct {
	Rules       					[]Rule
	SamplingTargetsPollingInterval  time.Duration
	refreshedAt 					int64
	xrayClient  					*xrayClient
	clientID						*string
	logger      					logr.Logger
	clock       					util.Clock
	mu          					sync.RWMutex
}

// NewManifest return manifest object configured with logging, clock and xrayClient.
func NewManifest(addr string, logger logr.Logger) (*Manifest, error) {
	// generate client for getSamplingRules and getSamplingTargets API call
	client, err := newClient(addr); if err != nil {
		return nil, err
	}

	// generate clientID for sampling statistics
	clientID, err := generateClientId(); if err != nil {
		return nil, err
	}

	return &Manifest{
		xrayClient: client,
		clock: &util.DefaultClock{},
		logger: logger,
		SamplingTargetsPollingInterval: 10 * time.Second,
		clientID: clientID,
	}, nil
}

// Expired returns true if the manifest has not been successfully refreshed in
// manifestTTL seconds.
func (m *Manifest) Expired() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.refreshedAt < m.clock.Now().Unix()-manifestTTL
}

// MatchAgainstManifestRules returns a Rule and boolean flag set as true if rule has been match against span attributes, otherwise nil and false
func (m *Manifest) MatchAgainstManifestRules(parameters sdktrace.SamplingParameters, serviceName string, cloudPlatform string) (*Rule, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	matched := false

	for index, r := range m.Rules {
		isRuleMatch := r.appliesTo(parameters, serviceName, cloudPlatform)

		if isRuleMatch {
			matched = true
			return &m.Rules[index], matched
		}
	}

	return nil, matched
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
func (m *Manifest) RefreshManifestTargets(ctx context.Context) (err error) {
	var manifest Manifest

	// deep copy centralized manifest object to temporary manifest to avoid thread safety issue
	m.mu.RLock()
	err = copier.CopyWithOption(&manifest, m, copier.Option{IgnoreEmpty: false, DeepCopy: true}); if err != nil {
		return err
	}
	m.mu.RUnlock()

	// generate sampling statistics based on the data in temporary manifest
	statistics, err := manifest.snapshots(); if err != nil { return err }

	// return if no statistics to report
	if len(statistics) == 0 {
		m.logger.V(5).Info("no statistics to report and not refreshing sampling targets")
		return nil
	}

	// get sampling targets (statistics) for every expired rule from AWS X-Ray
	targets, err := m.xrayClient.getSamplingTargets(ctx, statistics)
	if err != nil {
		return fmt.Errorf("refreshTargets: error occurred while getting sampling targets: %w", err)
	} else {
		m.logger.V(5).Info("successfully fetched sampling targets")
	}

	// update temporary manifest with retrieved targets (statistics) for each rule
	refresh, err := manifest.updateTargets(targets); if err != nil {
		return err
	}

	// find next polling interval for targets
	minPoll := manifest.minimumPollingInterval(targets)
	if minPoll > 0 {
		m.SamplingTargetsPollingInterval = time.Duration(minPoll) * time.Second
	}

	// update centralized manifest object
	m.mu.Lock()
	m.Rules = manifest.Rules
	m.mu.Unlock()

	// perform out-of-band async manifest refresh if refresh is set to true
	if refresh {
		m.logger.V(5).Info("refreshing sampling rules out-of-band")

		go func() {
			if err := m.RefreshManifestRules(ctx); err != nil {
				m.logger.Error(err, "error occurred refreshing sampling rules out-of-band")
			}
		}()
	}

	return
}

func (m *Manifest) updateRules(rules *getSamplingRulesOutput) {
	tempManifest := Manifest{
		Rules: []Rule{},
	}

	for _, records := range rules.SamplingRuleRecords {
		if records.SamplingRule.RuleName == "" {
			m.logger.V(5).Info("Sampling rule without rule name is not supported")
			continue
		}

		if records.SamplingRule.Version != int64(1) {
			m.logger.V(5).Info("Sampling rule without Version 1 is not supported", "RuleName", records.SamplingRule.RuleName)
			continue
		}

		// create rule and store it in temporary manifest to avoid thread safety issues
		tempManifest.createRule(*records.SamplingRule)
	}

	// Re-sort to fix matching priorities.
	tempManifest.sort()

	m.mu.Lock()
	m.Rules = tempManifest.Rules
	m.refreshedAt = m.clock.Now().Unix()
	m.mu.Unlock()

	return
}

func (m *Manifest) createRule(ruleProp ruleProperties) {
	cr := reservoir {
		capacity: ruleProp.ReservoirSize,
		interval: defaultInterval,
	}

	csr := Rule {
		reservoir:      cr,
		ruleProperties: ruleProp,
	}

	m.Rules = append(m.Rules, csr)

	return
}

func (m *Manifest) updateTargets(targets *getSamplingTargetsOutput) (refresh bool, err error) {
	// update sampling targets for each rule
	for _, t := range targets.SamplingTargetDocuments {
		if t.RuleName != nil && t.ReservoirQuota != nil {
			fmt.Println("rule name")
			fmt.Println(*t.RuleName)
			fmt.Println("assigned quota")
			fmt.Println(*t.ReservoirQuota)
		}

		if err := m.updateReservoir(t); err != nil {
			return false, err
		}
	}

	// consume unprocessed statistics messages
	for _, s := range targets.UnprocessedStatistics {
		m.logger.V(5).Info(
			"error occurred updating sampling target for rule, code and message", "RuleName", *s.RuleName, "ErrorCode",
			*s.ErrorCode,
			"Message", *s.Message,
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
		if int64(*remote) >= m.refreshedAt {
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

	for index, rule := range m.Rules {
		if rule.ruleProperties.RuleName == *t.RuleName {
			m.Rules[index].reservoir.refreshedAt = m.clock.Now().Unix()

			// Update non-optional attributes from response
			m.Rules[index].ruleProperties.FixedRate = *t.FixedRate

			// Update optional attributes from response
			if t.ReservoirQuota != nil {
				m.Rules[index].reservoir.quota = *t.ReservoirQuota
			}
			if t.ReservoirQuotaTTL != nil {
				m.Rules[index].reservoir.expiresAt = int64(*t.ReservoirQuotaTTL)
			}
			if t.Interval != nil {
				m.Rules[index].reservoir.interval = *t.Interval
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
	for _, r := range m.Rules {
		if r.stale(m.clock.Now().Unix()) {
			s := r.snapshot()
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
func (m *Manifest) minimumPollingInterval(targets *getSamplingTargetsOutput) (minPoll int64) {
	minPoll = 0
	for _, t := range targets.SamplingTargetDocuments {
		if t.Interval != nil {
			if minPoll == 0 {
				minPoll = *t.Interval
			} else {
				if minPoll > *t.Interval {
					minPoll = *t.Interval
				}
			}
		}
	}

	return minPoll
}

// generateClientId generates random client ID
func generateClientId() (*string, error) {
	var r [12]byte

	_, err := crypto.Read(r[:])
	if err != nil {
		return nil, fmt.Errorf("unable to generate client ID: %w", err)
	}

	id := fmt.Sprintf("%02x", r)

	return &id, err
}


