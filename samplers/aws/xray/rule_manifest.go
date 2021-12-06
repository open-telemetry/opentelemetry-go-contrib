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
	"fmt"
	"sort"
	"strings"
	"sync"
)

const defaultRule = "Default"
const defaultInterval = int64(10)

//const manifestTTL = 3600 // Seconds

//// centralizedManifest represents a full sampling ruleset, with a list of
//// custom rules and default values for incoming requests that do
//// not match any of the provided rules.
type centralizedManifest struct {
	defaultRule *centralizedRule
	rules       []*centralizedRule
	index       map[string]*centralizedRule
	refreshedAt int64
	clock       Clock
	mu          sync.RWMutex
}

// putRule updates the named rule if it already exists or creates it if it does not.
// May break ordering of the sorted rules array if it creates a new rule.
func (m *centralizedManifest) putRule(rule *ruleProperties) (r *centralizedRule, err error) {
	defer func() {
		if x := recover(); x != nil {
			err = fmt.Errorf("%v", x)
		}
	}()

	name := rule.ruleName

	// Default rule
	if name == defaultRule {
		m.mu.RLock()
		r = m.defaultRule
		m.mu.RUnlock()

		// Update rule if already exists
		if r != nil {
			m.updateDefaultRule(rule)

			return
		}

		// Create Default rule
		r = m.createDefaultRule(rule)

		return
	}

	// User-defined rule
	m.mu.RLock()
	r, ok := m.index[name]
	m.mu.RUnlock()

	// Create rule if it does not exist
	if !ok {
		r = m.createUserRule(rule)

		return
	}

	// Update existing rule
	m.updateUserRule(r, rule)

	return
}

// createUserRule creates a user-defined centralizedRule, appends it to the sorted array,
// adds it to the index, and returns the newly created rule.
func (m *centralizedManifest) createUserRule(rule *ruleProperties) *centralizedRule {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Return early if rule already exists
	if r, ok := m.index[rule.ruleName]; ok {
		return r
	}

	// Create CentralizedRule from xraySvc.SamplingRule
	clock := &DefaultClock{}
	rand := &DefaultRand{}

	cr := &centralizedReservoir{
		capacity: rule.reservoirSize,
		interval: defaultInterval,
	}

	csr := &centralizedRule{
		reservoir:      cr,
		ruleProperties: rule,
		clock:          clock,
		rand:           rand,
	}

	// Update sorted array
	m.rules = append(m.rules, csr)

	// Update index
	m.index[rule.ruleName] = csr

	return csr
}

// updateUserRule updates the properties of the user-defined centralizedRule using the given
// *properties.
func (m *centralizedManifest) updateUserRule(r *centralizedRule, rule *ruleProperties) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.ruleProperties = rule
	r.reservoir.capacity = rule.reservoirSize
}

// createDefaultRule creates a default centralizedRule and adds it to the manifest.
func (m *centralizedManifest) createDefaultRule(rule *ruleProperties) *centralizedRule {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Return early if rule already exists
	if d := m.defaultRule; d != nil {
		return d
	}

	// Create CentralizedRule from xraySvc.SamplingRule
	clock := &DefaultClock{}
	rand := &DefaultRand{}

	cr := &centralizedReservoir{
		capacity: rule.reservoirSize,
		interval: defaultInterval,
	}

	csr := &centralizedRule{
		reservoir:      cr,
		ruleProperties: rule,
		clock:          clock,
		rand:           rand,
	}

	// Update manifest if rule does not exist
	m.defaultRule = csr

	// Update index
	m.index[rule.ruleName] = csr

	return csr
}

// updateDefaultRule updates the properties of the default CentralizedRule using the given
// *properties.
func (m *centralizedManifest) updateDefaultRule(rule *ruleProperties) {
	r := m.defaultRule

	r.mu.Lock()
	defer r.mu.Unlock()

	r.ruleProperties = rule
	r.reservoir.capacity = rule.reservoirSize
}

// prune removes all rules in the manifest not present in the given list of active rules.
// Preserves ordering of sorted array.
func (m *centralizedManifest) prune(actives map[*centralizedRule]bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Iterate in reverse order to avoid adjusting index for each deleted rule
	for i := len(m.rules) - 1; i >= 0; i-- {
		r := m.rules[i]

		if _, ok := actives[r]; !ok {
			// Remove from index
			delete(m.index, m.rules[i].ruleProperties.ruleName)

			// Delete by reslicing without index
			a := append(m.rules[:i], m.rules[i+1:]...)

			// Set pointer to nil to free capacity from underlying array
			m.rules[len(m.rules)-1] = nil

			// Assign resliced rules
			m.rules = a
		}
	}
}

// sort sorts the rule array first by priority and then by rule name.
func (m *centralizedManifest) sort() {
	// Comparison function
	less := func(i, j int) bool {
		if m.rules[i].ruleProperties.priority == m.rules[j].ruleProperties.priority {
			return strings.Compare(m.rules[i].ruleProperties.ruleName, m.rules[j].ruleProperties.ruleName) < 0
		}
		return m.rules[i].ruleProperties.priority < m.rules[j].ruleProperties.priority
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	sort.Slice(m.rules, less)
}

// expired returns true if the manifest has not been successfully refreshed in
// 'manifestTTL' seconds.
//func (m *centralizedManifest) expired() bool {
//	m.mu.RLock()
//	defer m.mu.RUnlock()
//
//	return m.refreshedAt < m.clock.Now().Unix()-manifestTTL
//}
