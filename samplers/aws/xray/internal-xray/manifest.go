package internal_xray

import (
	"context"
	crypto "crypto/rand"
	"errors"
	"fmt"
	"github.com/go-logr/logr"
	"sort"
	"strings"
	"sync"
)

const defaultInterval = int64(10)

const manifestTTL = 3600

// manifest represents a full sampling ruleset, with a list of
// custom rules and default values for incoming requests that do
// not match any of the provided rules.
type Manifest struct {
	Rules       []rule
	refreshedAt int64
	xrayClient  *xrayClient
	clientID 	string
	logger 		logr.Logger
	clock       clock
	mu          sync.RWMutex
}

func NewManifest(addr string, logger logr.Logger) (*Manifest, error) {
	client, err := newClient(addr); if err != nil {
		return nil, err
	}

	return &Manifest{
		xrayClient: client,
		clock: &defaultClock{},
	}, nil
}

// updates/writes rules to the manifest
func (m *Manifest) RefreshManifest(ctx context.Context) (err error) {
	tempManifest := Manifest{
		Rules: []rule{},
	}

	// Get sampling rules from proxy.
	rules, err := m.xrayClient.getSamplingRules(ctx)
	if err != nil {
		return err
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

		// create rule and store it in temporary manifest to avoid locking issues.
		createErr := tempManifest.createRule(*records.SamplingRule)
		if createErr != nil {
			m.logger.Error(createErr, "Error occurred creating/updating rule")
		}
	}

	// Re-sort to fix matching priorities.
	tempManifest.sort()

	m.mu.Lock()
	m.Rules = tempManifest.Rules
	m.refreshedAt = m.clock.now().Unix()
	m.mu.Unlock()

	return
}

// updates/writes reservoir to the rules which considered as manifest update
func (m Manifest) RefreshTargets(ctx context.Context) (err error) {
	failed := false

	// Flag indicating whether or not manifest should be refreshed
	refresh := false

	tempManifest := Manifest{
		Rules: []rule{},
	}

	m.mu.RLock()
	tempManifest.Rules = m.Rules
	m.mu.RUnlock()

	m.Rules[0].ruleProperties.RuleName = "hero"
	name2 := &tempManifest.Rules[0].ruleProperties.RuleName

	fmt.Println(name2)

	// Generate sampling statistics
	statistics, err := tempManifest.snapshots(); if err != nil {
		return err
	}

	// Do not refresh targets if no statistics to report
	if len(statistics) == 0 {
		return
	}

	// Get sampling targets
	output, err := m.xrayClient.getSamplingTargets(ctx, statistics)
	if err != nil {
		return fmt.Errorf("refreshTargets: Error occurred while getting sampling targets: %w", err)
	}

	// Update sampling targets
	for _, t := range output.SamplingTargetDocuments {
		if err = tempManifest.updateTarget(t); err != nil {
			failed = true
			m.logger.Error(err, "Error occurred updating target for rule")
		}
	}

	m.mu.Lock()
	m.Rules = tempManifest.Rules
	m.mu.Unlock()

	// Consume unprocessed statistics messages
	for _, s := range output.UnprocessedStatistics {
		m.logger.V(5).Info(
			"Error occurred updating sampling target for rule, code and message", "RuleName", *s.RuleName, "ErrorCode",
			*s.ErrorCode,
			"Message", *s.Message,
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
		m.logger.V(5).Info("Successfully refreshed sampling targets")
	}

	// Set refresh flag if modifiedAt timestamp from remote is greater than ours.
	if remote := output.LastRuleModification; remote != nil {
		local := m.refreshedAt

		if int64(*remote) >= local {
			refresh = true
		}
	}

	// Perform out-of-band async manifest refresh if flag is set
	if refresh {
		m.logger.V(5).Info("Refreshing sampling rules out-of-band")

		go func() {
			if err := m.RefreshManifest(ctx); err != nil {
				m.logger.Error(err, "Error occurred refreshing sampling rules out-of-band")
			}
		}()
	}

	return
}

// samplingStatistics takes a snapshot of sampling statistics from all rules, resetting
// statistics counters in the process.
func (m *Manifest) snapshots() ([]*samplingStatisticsDocument, error) {
	statistics := make([]*samplingStatisticsDocument, 0, len(m.Rules)+1)

	// Generate sampling statistics for user-defined rules
	for _, r := range m.Rules {
		//if r.stale(m.clock.now().Unix()) {
			s := r.snapshot()
			clientID, err := generateClientId(); if err != nil {
				return nil, err
			}
			s.ClientID = clientID

			statistics = append(statistics, s)
		//}
	}

	return statistics, nil
}

func (m *Manifest) createRule(ruleProp ruleProperties) (err error) {
	cr := reservoir {
		capacity: ruleProp.ReservoirSize,
		interval: defaultInterval,
	}

	csr := rule {
		reservoir:      cr,
		ruleProperties: ruleProp,
	}

	m.Rules = append(m.Rules, csr)

	return
}

func (m *Manifest) updateTarget(t *samplingTargetDocument) (err error) {
	// Pre-emptively dereference xraySvc.SamplingTarget fields and return early on nil values
	// A panic in the middle of an update may leave the rule in an inconsistent state.
	if t.RuleName == nil {
		return errors.New("invalid sampling target. Missing rule name")
	}

	if t.FixedRate == nil {
		return fmt.Errorf("invalid sampling target for rule %s. Missing fixed rate", *t.RuleName)
	}

	for _, r := range m.Rules {
		if r.ruleProperties.RuleName == *t.RuleName {
			r.reservoir.refreshedAt = m.clock.now().Unix()

			// Update non-optional attributes from response
			r.ruleProperties.FixedRate = *t.FixedRate

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
		}
	}

	return nil
}

// sort sorts the rule array first by priority and then by rule name.
func (m *Manifest) sort() {
	// Comparison function
	less := func(i, j int) bool {
		if m.Rules[i].ruleProperties.Priority == m.Rules[j].ruleProperties.Priority {
			return strings.Compare(m.Rules[i].ruleProperties.RuleName, m.Rules[j].ruleProperties.RuleName) < 0
		}
		return m.Rules[i].ruleProperties.Priority < m.Rules[j].ruleProperties.Priority
	}

	sort.Slice(m.Rules, less)
}

//expired returns true if the manifest has not been successfully refreshed in
//'manifestTTL' seconds.
func (m *Manifest) Expired() bool {
	return m.refreshedAt < m.clock.now().Unix()-manifestTTL
}

// helper functions
func generateClientId() (*string, error) {
	// Generate clientID
	var r [12]byte

	_, err := crypto.Read(r[:])
	if err != nil {
		return nil, fmt.Errorf("unable to generate client ID: %w", err)
	}

	id := fmt.Sprintf("%02x", r)

	return &id, err
}


