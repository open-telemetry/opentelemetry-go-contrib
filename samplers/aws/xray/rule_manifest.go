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
	Default     *centralizedRule
	Rules       []*centralizedRule
	Index       map[string]*centralizedRule
	refreshedAt int64
	clock       Clock
	mu          sync.RWMutex
}

// putRule updates the named rule if it already exists or creates it if it does not.
// May break ordering of the sorted rules array if it creates a new rule.
func (m *centralizedManifest) putRule(ruleProperties *properties) (r *centralizedRule, err error) {
	defer func() {
		if x := recover(); x != nil {
			err = fmt.Errorf("%v", x)
		}
	}()

	name := ruleProperties.ruleName

	// Default rule
	if name == defaultRule {
		m.mu.RLock()
		r = m.Default
		m.mu.RUnlock()

		// Update rule if already exists
		if r != nil {
			m.updateDefaultRule(ruleProperties)

			return
		}

		// Create Default rule
		r = m.createDefaultRule(ruleProperties)

		return
	}

	// User-defined rule
	m.mu.RLock()
	r, ok := m.Index[name]
	m.mu.RUnlock()

	// Create rule if it does not exist
	if !ok {
		r = m.createUserRule(ruleProperties)

		return
	}

	// Update existing rule
	m.updateUserRule(r, ruleProperties)

	return
}

// createUserRule creates a user-defined centralizedRule, appends it to the sorted array,
// adds it to the index, and returns the newly created rule.
func (m *centralizedManifest) createUserRule(ruleProperties *properties) *centralizedRule {
	// Create CentralizedRule from xraySvc.SamplingRule
	clock := &DefaultClock{}
	rand := &DefaultRand{}

	cr := &centralizedReservoir{
		capacity: ruleProperties.reservoirSize,
		interval: defaultInterval,
	}

	csr := &centralizedRule{
		reservoir:  cr,
		properties: ruleProperties,
		clock:      clock,
		rand:       rand,
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Return early if rule already exists
	if r, ok := m.Index[ruleProperties.ruleName]; ok {
		return r
	}

	// Update sorted array
	m.Rules = append(m.Rules, csr)

	// Update index
	m.Index[ruleProperties.ruleName] = csr

	return csr
}

// updateUserRule updates the properties of the user-defined centralizedRule using the given
// *properties.
func (m *centralizedManifest) updateUserRule(r *centralizedRule, ruleProperties *properties) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.properties = ruleProperties
	r.reservoir.capacity = ruleProperties.reservoirSize
}

// createDefaultRule creates a default centralizedRule and adds it to the manifest.
func (m *centralizedManifest) createDefaultRule(ruleProperties *properties) *centralizedRule {
	// Create CentralizedRule from xraySvc.SamplingRule
	clock := &DefaultClock{}
	rand := &DefaultRand{}

	cr := &centralizedReservoir{
		capacity: ruleProperties.reservoirSize,
		interval: defaultInterval,
	}

	csr := &centralizedRule{
		reservoir:  cr,
		properties: ruleProperties,
		clock:      clock,
		rand:       rand,
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Return early if rule already exists
	if d := m.Default; d != nil {
		return d
	}

	// Update manifest if rule does not exist
	m.Default = csr

	// Update index
	m.Index[ruleProperties.ruleName] = csr

	return csr
}

// updateDefaultRule updates the properties of the default CentralizedRule using the given
// *properties.
func (m *centralizedManifest) updateDefaultRule(ruleProperties *properties) {
	r := m.Default

	r.mu.Lock()
	defer r.mu.Unlock()

	r.properties = ruleProperties
	r.reservoir.capacity = ruleProperties.reservoirSize
}

// prune removes all rules in the manifest not present in the given list of active rules.
// Preserves ordering of sorted array.
func (m *centralizedManifest) prune(actives map[*centralizedRule]bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Iterate in reverse order to avoid adjusting index for each deleted rule
	for i := len(m.Rules) - 1; i >= 0; i-- {
		r := m.Rules[i]

		if _, ok := actives[r]; !ok {
			m.deleteRule(i)
		}
	}
}

// deleteRule deletes the rule from the array, and the index.
// Assumes write lock is already held.
// Preserves ordering of sorted array.
func (m *centralizedManifest) deleteRule(idx int) {
	// Remove from index
	delete(m.Index, m.Rules[idx].ruleName)

	// Delete by reslicing without index
	a := append(m.Rules[:idx], m.Rules[idx+1:]...)

	// Set pointer to nil to free capacity from underlying array
	m.Rules[len(m.Rules)-1] = nil

	// Assign resliced rules
	m.Rules = a
}

// sort sorts the rule array first by priority and then by rule name.
func (m *centralizedManifest) sort() {
	// Comparison function
	less := func(i, j int) bool {
		if m.Rules[i].priority == m.Rules[j].priority {
			return strings.Compare(m.Rules[i].ruleName, m.Rules[j].ruleName) < 0
		}
		return m.Rules[i].priority < m.Rules[j].priority
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	sort.Slice(m.Rules, less)
}

// expired returns true if the manifest has not been successfully refreshed in
// 'manifestTTL' seconds.
//func (m *centralizedManifest) expired() bool {
//	m.mu.RLock()
//	defer m.mu.RUnlock()
//
//	return m.refreshedAt < m.clock.Now().Unix()-manifestTTL
//}
