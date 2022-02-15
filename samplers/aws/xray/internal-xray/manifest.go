package internal_xray

import (
	"context"
	crypto "crypto/rand"
	"errors"
	"fmt"
	"github.com/go-logr/logr"
	"reflect"
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
	logger 		logr.Logger
	clock       clock
	mu          sync.RWMutex
}

// samplingStatisticsDocument is used to store current state of sampling data
type samplingStatisticsDocument struct {
	// The number of requests recorded with borrowed reservoir quota.
	BorrowCount *int64

	// A unique identifier for the service in hexadecimal.
	ClientID *string

	// The number of requests that matched the rule.
	RequestCount *int64

	// The name of the sampling rule.
	RuleName *string

	// The number of requests recorded.
	SampledCount *int64

	// The current time.
	Timestamp *int64
}

// samplingTargetDocument contains updated targeted information retrieved from X-Ray service
type samplingTargetDocument struct {
	// The percentage of matching requests to instrument, after the reservoir is
	// exhausted.
	FixedRate *float64 `json:"FixedRate,omitempty"`

	// The number of seconds for the service to wait before getting sampling targets
	// again.
	Interval *int64 `json:"Interval,omitempty"`

	// The number of requests per second that X-Ray allocated this service.
	ReservoirQuota *int64 `json:"ReservoirQuota,omitempty"`

	// When the reservoir quota expires.
	ReservoirQuotaTTL *float64 `json:"ReservoirQuotaTTL,omitempty"`

	// The name of the sampling rule.
	RuleName *string `json:"RuleName,omitempty"`
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
func (m *Manifest) RefreshManifestRules(ctx context.Context) (err error) {
	// Get sampling rules from X-Ray service backend
	rules, err := m.xrayClient.getSamplingRules(ctx)
	if err != nil {
		return err
	}

	// create temporary manifest with retrieved rules and updates to the original manifest rules
	m.updateRules(rules)

	return
}

func (m *Manifest) updateRules(rules *getSamplingRulesOutput) {
	tempManifest := Manifest{
		Rules: []rule{},
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
func (m *Manifest) RefreshManifestTargets(ctx context.Context) (err error) {
	m.mu.RLock()
	mani := *m // deep copy
	m.mu.RUnlock()

	a := reflect.TypeOf(*m)
	b:= reflect.TypeOf(m)
	c := reflect.TypeOf(mani)
	d := reflect.TypeOf(&mani)
	e := reflect.TypeOf(mani.Rules[0])
	f := reflect.TypeOf(&m.Rules[0])

	fmt.Println(a)
	fmt.Println(b)
	fmt.Println(c)
	fmt.Println(d)
	fmt.Println(e)
	fmt.Println(f)


	// Generate sampling statistics
	statistics, err := m.snapshots(); if err != nil { return err }

	// Do not refresh targets if no statistics to report
	if len(statistics) == 0 {
		m.logger.V(5).Info("No statistics to report and not refreshing sampling targets")
		return nil
	}

	// Get sampling targets
	targets, err := m.xrayClient.getSamplingTargets(ctx, statistics)
	if err != nil {
		return fmt.Errorf("refreshTargets: Error occurred while getting sampling targets: %w", err)
	}

	refresh, err := m.updateTargets(targets); if err != nil {
		return err
	}

	// Perform out-of-band async manifest refresh if flag is set
	if refresh {
		m.logger.V(5).Info("Refreshing sampling rules out-of-band")

		go func() {
			if err := m.RefreshManifestRules(ctx); err != nil {
				m.logger.Error(err, "Error occurred refreshing sampling rules out-of-band")
			}
		}()
	}

	return
}

func (m *Manifest) updateTargets(targets *getSamplingTargetsOutput) (refresh bool, err error) {
	// Update sampling targets
	for _, t := range targets.SamplingTargetDocuments {
		if err := m.updateReservoir(t); err != nil {
			return false, err
		}
	}

	// Consume unprocessed statistics messages
	for _, s := range targets.UnprocessedStatistics {
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
			return false, fmt.Errorf("statistics returned 5xx")
		}

		// Set refresh flag if any sampling statistics return 4xx
		if strings.HasPrefix(*s.ErrorCode, "4") {
			refresh = true
		}
	}

	// Set refresh flag if modifiedAt timestamp from remote is greater than ours.
	if remote := targets.LastRuleModification; remote != nil {
		if int64(*remote) >= m.refreshedAt {
			refresh = true
		}
	}

	return
}

// samplingStatistics takes a snapshot of sampling statistics from all rules, resetting
// statistics counters in the process.
func (m *Manifest) snapshots() ([]*samplingStatisticsDocument, error) {
	statistics := make([]*samplingStatisticsDocument, 0, len(m.Rules)+1)

	// Generate sampling statistics for user-defined rules
	for _, r := range m.Rules {
//		if r.stale(m.clock.now().Unix()) {
			s := r.snapshot()
			clientID, err := generateClientId(); if err != nil {
				return nil, err
			}
			s.ClientID = clientID

			statistics = append(statistics, s)
//		}
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

func (m *Manifest) updateReservoir(t *samplingTargetDocument) (err error) {
	// Pre-emptively dereference xraySvc.SamplingTarget fields and return early on nil values
	// A panic in the middle of an update may leave the rule in an inconsistent state.
	if t.RuleName == nil {
		return errors.New("invalid sampling target. Missing rule name")
	}

	if t.FixedRate == nil {
		return fmt.Errorf("invalid sampling target for rule %s. Missing fixed rate", *t.RuleName)
	}

	//for _, r := range m.Rules {
	//	if r.ruleProperties.RuleName == *t.RuleName {
	//		r.reservoir.refreshedAt = m.clock.now().Unix()
	//
	//		// Update non-optional attributes from response
	//		r.ruleProperties.FixedRate = *t.FixedRate
	//
	//		// Update optional attributes from response
	//		if t.ReservoirQuota != nil {
	//			r.reservoir.quota = *t.ReservoirQuota
	//		}
	//		if t.ReservoirQuotaTTL != nil {
	//			r.reservoir.expiresAt = int64(*t.ReservoirQuotaTTL)
	//		}
	//		if t.Interval != nil {
	//			r.reservoir.interval = *t.Interval
	//		}
	//	}
	//}
	m.Rules[0].reservoir.refreshedAt = 67
	m.Rules[1].reservoir.refreshedAt = 77
	m.Rules[2].reservoir.refreshedAt = 87

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


