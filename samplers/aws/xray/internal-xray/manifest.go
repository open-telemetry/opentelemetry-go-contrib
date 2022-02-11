package internal_xray

import (
	"context"
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
	Rules       []Rule
	refreshedAt int64
	XrayClient  *xrayClient
	Logger 		logr.Logger
	ClientID 	string
	Clock       Clock
	mu          sync.RWMutex
}

// centralizedRule represents a centralized sampling rule
type Rule struct {
	// Centralized reservoir for keeping track of reservoir usage
	Reservoir Reservoir

	// sampling rule properties
	RuleProperties RuleProperties

	// Number of requests matched against this rule
	MatchedRequests int64

	// Number of requests sampled using this rule
	SampledRequests int64

	// Number of requests borrowed
	BorrowedRequests int64

	mu sync.RWMutex
}

type Reservoir struct {
	// Quota assigned to client
	Quota int64

	// Quota refresh timestamp
	RefreshedAt int64

	// Quota expiration timestamp
	ExpiresAt int64

	// Polling interval for quota
	interval int64

	// Total size of reservoir
	capacity int64

	// Reservoir consumption for current epoch
	Used int64

	// Unix epoch. Reservoir usage is reset every second.
	CurrentEpoch int64
}

// properties is the base set of properties that define a sampling rule.
type RuleProperties struct {
	RuleName      string            `json:"RuleName"`
	ServiceType   string            `json:"ServiceType"`
	ResourceARN   string            `json:"ResourceARN"`
	Attributes    map[string]string `json:"Attributes"`
	ServiceName   string            `json:"ServiceName"`
	Host          string            `json:"Host"`
	HTTPMethod    string            `json:"HTTPMethod"`
	URLPath       string            `json:"URLPath"`
	ReservoirSize int64             `json:"ReservoirSize"`
	FixedRate     float64           `json:"FixedRate"`
	Priority      int64             `json:"Priority"`
	Version       int64             `json:"Version"`
}

// updates/writes rules to the manifest
func (m *Manifest) RefreshManifest(ctx context.Context) (err error) {
	tempManifest := &Manifest{
		Rules: []Rule{},
		XrayClient: m.XrayClient,
		Logger: m.Logger,
		ClientID: m.ClientID,
		Clock: m.Clock,
	}

	// Get sampling rules from proxy.
	rules, err := m.XrayClient.getSamplingRules(ctx)
	if err != nil {
		return err
	}

	for _, records := range rules.SamplingRuleRecords {
		if records.SamplingRule.RuleName == "" {
			tempManifest.Logger.V(5).Info("Sampling rule without rule name is not supported")
			continue
		}

		if records.SamplingRule.Version != int64(1) {
			tempManifest.Logger.V(5).Info("Sampling rule without Version 1 is not supported", "RuleName", records.SamplingRule.RuleName)
			continue
		}

		// create rule and store it in temporary manifest to avoid locking issues.
		createErr := tempManifest.createRule(records.SamplingRule)
		if createErr != nil {
			tempManifest.Logger.Error(createErr, "Error occurred creating/updating rule")
		}
	}

	// Re-sort to fix matching priorities.
	tempManifest.sort()
	// Update refreshedAt timestamp
	tempManifest.refreshedAt = tempManifest.Clock.now().Unix()

	// assign temp manifest to original copy/one sync refresh.

	m.mu.Lock()
	m = tempManifest
	m.mu.Unlock()

	return
}

// updates/writes reservoir to the rules which considered as manifest update
func (m *Manifest) RefreshTarget(ctx context.Context) error {


}

// samplingStatistics takes a snapshot of sampling statistics from all rules, resetting
// statistics counters in the process.
func (m *Manifest) snapshots() []*samplingStatisticsDocument {
	clock := &DefaultClock{}
	now := clock.now().Unix()

	statistics := make([]*samplingStatisticsDocument, 0, len(m.Rules)+1)

	// Generate sampling statistics for user-defined rules
	for _, r := range m.Rules {
		if r.stale(now) {
			s := r.snapshot()
			s.ClientID = &rs.clientID

			statistics = append(statistics, s)
		}
	}

	return statistics
}

func (m *Manifest) createRule(ruleProp *RuleProperties) (err error) {
	cr := Reservoir {
		capacity: ruleProp.ReservoirSize,
		interval: defaultInterval,
	}

	csr := Rule {
		Reservoir:      cr,
		RuleProperties: *ruleProp,
	}

	m.Rules = append(m.Rules, csr)

	return
}

// sort sorts the rule array first by priority and then by rule name.
func (m *Manifest) sort() {
	// Comparison function
	less := func(i, j int) bool {
		if m.Rules[i].RuleProperties.Priority == m.Rules[j].RuleProperties.Priority {
			return strings.Compare(m.Rules[i].RuleProperties.RuleName, m.Rules[j].RuleProperties.RuleName) < 0
		}
		return m.Rules[i].RuleProperties.Priority < m.Rules[j].RuleProperties.Priority
	}

	sort.Slice(m.Rules, less)
}

//expired returns true if the manifest has not been successfully refreshed in
//'manifestTTL' seconds.
func (m *Manifest) expired() bool {
	clock := &DefaultClock{}
	now := clock.now().Unix()

	return m.refreshedAt < now-manifestTTL
}


