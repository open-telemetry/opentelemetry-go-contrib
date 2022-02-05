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
	"sort"
	"strings"
	"sync"
)

const defaultInterval = int64(10)

// manifest represents a full sampling ruleset, with a list of
// custom rules and default values for incoming requests that do
// not match any of the provided rules.
type manifest struct {
	rules       []*rule
	index       map[string]*rule
	refreshedAt int64
	clock       clock
	mu          sync.RWMutex
}

// createRule creates a user-defined rule, appends it to the sorted array,
// adds it to the index, and returns the newly created rule.
func (m *manifest) createRule(ruleProp *ruleProperties) (err error) {
	clock := &defaultClock{}
	rand := &defaultRand{}

	cr := &reservoir{
		capacity: *ruleProp.ReservoirSize,
		interval: defaultInterval,
	}

	csr := &rule{
		reservoir:      cr,
		ruleProperties: ruleProp,
		clock:          clock,
		rand:           rand,
	}

	// Update sorted array
	m.rules = append(m.rules, csr)

	// Update index
	m.index[*ruleProp.RuleName] = csr

	return
}

// sort sorts the rule array first by priority and then by rule name.
func (m *manifest) sort() {
	// Comparison function
	less := func(i, j int) bool {
		if *m.rules[i].ruleProperties.Priority == *m.rules[j].ruleProperties.Priority {
			return strings.Compare(*m.rules[i].ruleProperties.RuleName, *m.rules[j].ruleProperties.RuleName) < 0
		}
		return *m.rules[i].ruleProperties.Priority < *m.rules[j].ruleProperties.Priority
	}

	sort.Slice(m.rules, less)
}
