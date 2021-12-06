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
	"testing"

	"github.com/stretchr/testify/assert"
)

// Assert that putRule() creates a new user-defined rule and adds to manifest
func TestCreateUserRule(t *testing.T) {
	resARN := "*"
	r1 := &centralizedRule{
		ruleProperties: &ruleProperties{
			ruleName: "r1",
			priority: 5,
		},
	}

	r3 := &centralizedRule{
		ruleProperties: &ruleProperties{
			ruleName: "r3",
			priority: 7,
		},
	}

	rules := []*centralizedRule{r1, r3}

	index := map[string]*centralizedRule{
		"r1": r1,
		"r3": r3,
	}

	m := &centralizedManifest{
		rules: rules,
		index: index,
	}

	// Output of GetSamplingRules API and Input to putRule().
	serviceName := "www.foo.com"
	httpMethod := "POST"
	urlPath := "/bar/*"
	reservoirSize := int64(10)
	fixedRate := float64(0.05)
	ruleName := "r2"
	host := "local"
	priority := int64(6)
	serviceTye := "*"

	ruleProperties := ruleProperties{
		serviceName:   serviceName,
		httpMethod:    httpMethod,
		urlPath:       urlPath,
		reservoirSize: reservoirSize,
		fixedRate:     fixedRate,
		ruleName:      ruleName,
		priority:      priority,
		host:          host,
		serviceType:   serviceTye,
		resourceARN:   resARN,
	}

	// Expected centralized sampling rule
	clock := &DefaultClock{}
	rand := &DefaultRand{}

	cr := &centralizedReservoir{
		capacity: 10,
		interval: 10,
	}

	exp := &centralizedRule{
		reservoir:      cr,
		ruleProperties: &ruleProperties,
		clock:          clock,
		rand:           rand,
	}

	// Add to manifest, index
	r2, err := m.putRule(&ruleProperties)
	assert.Nil(t, err)
	assert.Equal(t, exp, r2)

	// Assert new rule is present in index
	r2, ok := m.index["r2"]
	assert.True(t, ok)
	assert.Equal(t, exp, r2)

	// Assert new rule present at end of array. putRule() does not preserve order.
	r2 = m.rules[2]
	assert.Equal(t, exp, r2)
}

// Assert that putRule() creates a new default rule and adds to manifest
func TestCreateDefaultRule(t *testing.T) {
	m := &centralizedManifest{
		index: map[string]*centralizedRule{},
	}

	// Output of GetSamplingRules API and Input to putRule().
	reservoirSize := int64(10)
	fixedRate := float64(0.05)
	ruleName := "Default"

	// Expected centralized sampling rule
	clock := &DefaultClock{}
	rand := &DefaultRand{}

	p := &ruleProperties{
		ruleName:      ruleName,
		reservoirSize: reservoirSize,
		fixedRate:     fixedRate,
	}

	cr := &centralizedReservoir{
		capacity: reservoirSize,
		interval: 10,
	}

	exp := &centralizedRule{
		reservoir:      cr,
		ruleProperties: p,
		clock:          clock,
		rand:           rand,
	}

	// Add to manifest
	r, err := m.putRule(p)
	assert.Nil(t, err)
	assert.Equal(t, exp, r)
	assert.Equal(t, exp, m.defaultRule)
}

// Assert that putRule() updates the default rule
func TestUpdateDefaultRule(t *testing.T) {
	clock := &DefaultClock{}
	rand := &DefaultRand{}

	// Original default sampling rule
	r := &centralizedRule{
		ruleProperties: &ruleProperties{
			ruleName:      "Default",
			reservoirSize: 10,
			fixedRate:     0.05,
		},
		reservoir: &centralizedReservoir{
			capacity: 10,
		},
		clock: clock,
		rand:  rand,
	}

	m := &centralizedManifest{
		defaultRule: r,
	}

	// Output of GetSamplingRules API and Input to putRule().
	reservoirSize := int64(20)
	fixedRate := 0.06
	ruleName := "Default"

	// Expected centralized sampling rule
	p := &ruleProperties{
		ruleName:      ruleName,
		reservoirSize: reservoirSize,
		fixedRate:     fixedRate,
	}

	cr := &centralizedReservoir{
		capacity: reservoirSize,
	}

	exp := &centralizedRule{
		reservoir:      cr,
		ruleProperties: p,
		clock:          clock,
		rand:           rand,
	}

	// Update default rule in manifest
	r, err := m.putRule(p)
	assert.Nil(t, err)
	assert.Equal(t, exp, r)
	assert.Equal(t, exp, m.defaultRule)
}

// Assert that creating a user-defined rule which already exists is a no-op
func TestCreateUserRuleNoOp(t *testing.T) {
	resARN := "*"
	serviceTye := ""
	attributes := map[string]interface{}{
		"foo":  "bar",
		"flag": true,
	}

	r1 := &centralizedRule{
		ruleProperties: &ruleProperties{
			ruleName: "r1",
			priority: 5,
		},
	}

	r3 := &centralizedRule{
		ruleProperties: &ruleProperties{
			ruleName: "r3",
			priority: 7,
		},
		reservoir: &centralizedReservoir{
			capacity: 5,
		},
	}

	rules := []*centralizedRule{r1, r3}

	index := map[string]*centralizedRule{
		"r1": r1,
		"r3": r3,
	}

	m := &centralizedManifest{
		rules: rules,
		index: index,
	}

	// Duplicate rule properties. 'r3' already exists. Input to updateRule().
	serviceName := "www.foo.com"
	httpMethod := "POST"
	urlPath := "/bar/*"
	reservoirSize := int64(10)
	fixedRate := float64(0.05)
	ruleName := "r3"
	priority := int64(6)
	host := "h"
	ruleProperties := ruleProperties{
		serviceName:   serviceName,
		httpMethod:    httpMethod,
		urlPath:       urlPath,
		reservoirSize: reservoirSize,
		fixedRate:     fixedRate,
		ruleName:      ruleName,
		priority:      priority,
		host:          host,
		resourceARN:   resARN,
		serviceType:   serviceTye,
		attributes:    attributes,
	}

	// Assert manifest has not changed
	r, err := m.putRule(&ruleProperties)
	assert.Nil(t, err)
	assert.Equal(t, r3, r)
	assert.Equal(t, 2, len(m.rules))
	assert.Equal(t, 2, len(m.index))
	assert.Equal(t, r1, m.rules[0])
	assert.Equal(t, r3, m.rules[1])
}

// Assert that putRule() updates the user-defined rule in the manifest
func TestUpdateUserRule(t *testing.T) {
	resARN := "*"
	serviceType := ""
	attributes := map[string]interface{}{
		"foo":  "bar",
		"flag": true,
	}

	// Original rule
	r1 := &centralizedRule{

		ruleProperties: &ruleProperties{
			ruleName:      "r1",
			priority:      5,
			serviceName:   "*.foo.com",
			httpMethod:    "GET",
			urlPath:       "/resource/*",
			reservoirSize: 15,
			fixedRate:     0.04,
			resourceARN:   resARN,
			serviceType:   serviceType,
			attributes:    attributes,
		},

		reservoir: &centralizedReservoir{
			capacity: 5,
		},
	}

	rules := []*centralizedRule{r1}

	index := map[string]*centralizedRule{
		"r1": r1,
	}

	m := &centralizedManifest{
		rules: rules,
		index: index,
	}

	// Updated rule properties. Input to updateRule().
	serviceName := "www.foo.com"
	httpMethod := "POST"
	urlPath := "/bar/*"
	reservoirSize := int64(10)
	fixedRate := float64(0.05)
	ruleName := "r1"
	priority := int64(6)
	host := "h"

	updated := ruleProperties{
		serviceName:   serviceName,
		httpMethod:    httpMethod,
		urlPath:       urlPath,
		reservoirSize: reservoirSize,
		fixedRate:     fixedRate,
		ruleName:      ruleName,
		priority:      priority,
		host:          host,
		resourceARN:   resARN,
		serviceType:   serviceType,
		attributes:    attributes,
	}

	// Expected updated centralized sampling rule
	cr := &centralizedReservoir{
		capacity: 10,
	}

	exp := &centralizedRule{
		reservoir:      cr,
		ruleProperties: &updated,
	}

	// Assert that rule has been updated
	r, err := m.putRule(&updated)
	assert.Nil(t, err)
	assert.Equal(t, exp, r)
	assert.Equal(t, exp, m.index["r1"])
	assert.Equal(t, exp, m.rules[0])
	assert.Equal(t, 1, len(m.rules))
	assert.Equal(t, 1, len(m.index))
}

// Assert that deleting a rule from the end of the array removes the rule
// and preserves ordering of the sorted array
func TestDeleteLastRule(t *testing.T) {
	r1 := &centralizedRule{
		ruleProperties: &ruleProperties{
			ruleName: "r1",
			priority: 5,
		},
	}

	r2 := &centralizedRule{
		ruleProperties: &ruleProperties{
			ruleName: "r2",
			priority: 6,
		},
	}

	r3 := &centralizedRule{
		ruleProperties: &ruleProperties{
			ruleName: "r3",
			priority: 7,
		},
	}

	rules := []*centralizedRule{r1, r2, r3}

	index := map[string]*centralizedRule{
		"r1": r1,
		"r2": r2,
		"r3": r3,
	}

	m := &centralizedManifest{
		rules: rules,
		index: index,
	}

	// Active rules to exclude from deletion
	a := map[*centralizedRule]bool{
		r1: true,
		r2: true,
	}

	// Delete r3
	m.prune(a)

	// Assert size of manifest
	assert.Equal(t, 2, len(m.rules))
	assert.Equal(t, 2, len(m.index))

	// Assert index consistency
	_, ok := m.index["r3"]
	assert.False(t, ok)
	assert.Equal(t, r1, m.index["r1"])
	assert.Equal(t, r2, m.index["r2"])

	// Assert ordering of array
	assert.Equal(t, r1, m.rules[0])
	assert.Equal(t, r2, m.rules[1])
}

// Assert that deleting a rule from the middle of the array removes the rule
// and preserves ordering of the sorted array
func TestDeleteMiddleRule(t *testing.T) {
	r1 := &centralizedRule{
		ruleProperties: &ruleProperties{
			ruleName: "r1",
			priority: 5,
		},
	}

	r2 := &centralizedRule{
		ruleProperties: &ruleProperties{
			ruleName: "r2",
			priority: 6,
		},
	}

	r3 := &centralizedRule{
		ruleProperties: &ruleProperties{
			ruleName: "r3",
			priority: 7,
		},
	}

	rules := []*centralizedRule{r1, r2, r3}

	index := map[string]*centralizedRule{
		"r1": r1,
		"r2": r2,
		"r3": r3,
	}

	m := &centralizedManifest{
		rules: rules,
		index: index,
	}

	// Active rules to exclude from deletion
	a := map[*centralizedRule]bool{
		r1: true,
		r3: true,
	}

	// Delete r2
	m.prune(a)

	// Assert size of manifest
	assert.Equal(t, 2, len(m.rules))
	assert.Equal(t, 2, len(m.index))

	// Assert index consistency
	_, ok := m.index["r2"]
	assert.False(t, ok)
	assert.Equal(t, r1, m.index["r1"])
	assert.Equal(t, r3, m.index["r3"])

	// Assert ordering of array
	assert.Equal(t, r1, m.rules[0])
	assert.Equal(t, r3, m.rules[1])
}

// Assert that deleting a rule from the beginning of the array removes the rule
// and preserves ordering of the sorted array
func TestDeleteFirstRule(t *testing.T) {
	r1 := &centralizedRule{
		ruleProperties: &ruleProperties{
			ruleName: "r1",
			priority: 5,
		},
	}

	r2 := &centralizedRule{
		ruleProperties: &ruleProperties{
			ruleName: "r2",
			priority: 6,
		},
	}

	r3 := &centralizedRule{
		ruleProperties: &ruleProperties{
			ruleName: "r3",
			priority: 7,
		},
	}

	rules := []*centralizedRule{r1, r2, r3}

	index := map[string]*centralizedRule{
		"r1": r1,
		"r2": r2,
		"r3": r3,
	}

	m := &centralizedManifest{
		rules: rules,
		index: index,
	}

	// Active rules to exclude from deletion
	a := map[*centralizedRule]bool{
		r2: true,
		r3: true,
	}

	// Delete r1
	m.prune(a)

	// Assert size of manifest
	assert.Equal(t, 2, len(m.rules))
	assert.Equal(t, 2, len(m.index))

	// Assert index consistency
	_, ok := m.index["r1"]
	assert.False(t, ok)
	assert.Equal(t, r2, m.index["r2"])
	assert.Equal(t, r3, m.index["r3"])

	// Assert ordering of array
	assert.Equal(t, r2, m.rules[0])
	assert.Equal(t, r3, m.rules[1])
}

// Assert that deleting the only rule from the array removes the rule
func TestDeleteOnlyRule(t *testing.T) {
	r1 := &centralizedRule{
		ruleProperties: &ruleProperties{
			ruleName: "r1",
			priority: 5,
		},
	}

	rules := []*centralizedRule{r1}

	index := map[string]*centralizedRule{
		"r1": r1,
	}

	m := &centralizedManifest{
		rules: rules,
		index: index,
	}

	// Active rules to exclude from deletion
	a := map[*centralizedRule]bool{}

	// Delete r1
	m.prune(a)

	// Assert size of manifest
	assert.Equal(t, 0, len(m.rules))
	assert.Equal(t, 0, len(m.index))

	// Assert index consistency
	_, ok := m.index["r1"]
	assert.False(t, ok)
}

// Assert that deleting rules from an empty array does not panic
func TestDeleteEmptyRulesArray(t *testing.T) {
	var rules []*centralizedRule

	index := map[string]*centralizedRule{}

	m := &centralizedManifest{
		rules: rules,
		index: index,
	}

	// Active rules to exclude from deletion
	a := map[*centralizedRule]bool{}

	// Delete from empty array
	m.prune(a)

	// Assert size of manifest
	assert.Equal(t, 0, len(m.rules))
	assert.Equal(t, 0, len(m.index))
}

// Assert that deleting all rules results in an empty array and does not panic
func TestDeleteAllRules(t *testing.T) {
	r1 := &centralizedRule{
		ruleProperties: &ruleProperties{
			ruleName: "r1",
			priority: 5,
		},
	}

	r2 := &centralizedRule{
		ruleProperties: &ruleProperties{
			ruleName: "r2",
			priority: 6,
		},
	}

	r3 := &centralizedRule{
		ruleProperties: &ruleProperties{
			ruleName: "r3",
			priority: 7,
		},
	}

	rules := []*centralizedRule{r1, r2, r3}

	index := map[string]*centralizedRule{
		"r1": r1,
		"r2": r2,
		"r3": r3,
	}

	m := &centralizedManifest{
		rules: rules,
		index: index,
	}

	// Active rules to exclude from deletion
	a := map[*centralizedRule]bool{}

	// Delete r3
	m.prune(a)

	// Assert size of manifest
	assert.Equal(t, 0, len(m.rules))
	assert.Equal(t, 0, len(m.index))
}

// Assert that sorting an unsorted array results in a sorted array - check priority
func TestSort(t *testing.T) {
	r1 := &centralizedRule{
		ruleProperties: &ruleProperties{
			ruleName: "r1",
			priority: 5,
		},
	}

	r2 := &centralizedRule{
		ruleProperties: &ruleProperties{
			ruleName: "r2",
			priority: 6,
		},
	}

	r3 := &centralizedRule{
		ruleProperties: &ruleProperties{
			ruleName: "r3",
			priority: 7,
		},
	}

	// Unsorted rules array
	rules := []*centralizedRule{r2, r1, r3}

	m := &centralizedManifest{
		rules: rules,
	}

	// Sort array
	m.sort()

	// Assert on order
	assert.Equal(t, r1, m.rules[0])
	assert.Equal(t, r2, m.rules[1])
	assert.Equal(t, r3, m.rules[2])
}
