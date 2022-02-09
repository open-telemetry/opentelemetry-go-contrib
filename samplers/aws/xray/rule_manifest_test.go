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
	"testing"

	"github.com/stretchr/testify/assert"
)

// utility functions to get pointers value
func getIntPointer(val int64) *int64 {
	return &val
}

func getStringPointer(val string) *string {
	return &val
}

func getFloatPointer(val float64) *float64 {
	return &val
}

// Assert that createRule() creates a new rule and adds to manifest
func TestCreateRule(t *testing.T) {
	resARN := "*"
	r1 := &rule{
		ruleProperties: &ruleProperties{
			RuleName: getStringPointer("r1"),
			Priority: getIntPointer(5),
		},
	}

	r3 := &rule{
		ruleProperties: &ruleProperties{
			RuleName: getStringPointer("r3"),
			Priority: getIntPointer(7),
		},
	}

	rules := []*rule{r1, r3}

	index := map[string]*rule{
		"r1": r1,
		"r3": r3,
	}

	m := &manifest{
		rules: rules,
		index: index,
	}

	// Output of GetSamplingRules API and Input to putRule().
	serviceName := "www.foo.com"
	httpMethod := "POST"
	urlPath := "/bar/*"
	reservoirSize := int64(10)
	fixedRate := 0.05
	ruleName := "r2"
	host := "local"
	priority := int64(6)
	serviceType := "*"

	ruleProperties := &ruleProperties{
		ServiceName:   &serviceName,
		HTTPMethod:    &httpMethod,
		URLPath:       &urlPath,
		ReservoirSize: &reservoirSize,
		FixedRate:     &fixedRate,
		RuleName:      &ruleName,
		Priority:      &priority,
		Host:          &host,
		ServiceType:   &serviceType,
		ResourceARN:   &resARN,
	}

	// Expected centralized sampling rule
	clock := &defaultClock{}
	rand := &defaultRand{}

	cr := &reservoir{
		capacity: 10,
		interval: 10,
	}

	exp := &rule{
		reservoir:      cr,
		ruleProperties: ruleProperties,
		clock:          clock,
		rand:           rand,
	}

	// Add to manifest, index
	err := m.createRule(ruleProperties)
	assert.Nil(t, err)

	// Assert new rule is present in index
	r2, ok := m.index["r2"]
	assert.True(t, ok)
	assert.Equal(t, exp, r2)

	// Assert new rule present at end of array. putRule() does not preserve order.
	r2 = m.rules[2]
	assert.Equal(t, exp, r2)
}

// Assert that sorting an unsorted array results in a sorted array - check priority
func TestSort1(t *testing.T) {
	r1 := &rule{
		ruleProperties: &ruleProperties{
			RuleName: getStringPointer("r1"),
			Priority: getIntPointer(5),
		},
	}

	r2 := &rule{
		ruleProperties: &ruleProperties{
			RuleName: getStringPointer("r2"),
			Priority: getIntPointer(6),
		},
	}

	r3 := &rule{
		ruleProperties: &ruleProperties{
			RuleName: getStringPointer("r3"),
			Priority: getIntPointer(7),
		},
	}

	// Unsorted rules array
	rules := []*rule{r2, r1, r3}

	m := &manifest{
		rules: rules,
	}

	// Sort array
	m.sort()

	// Assert on order
	assert.Equal(t, r1, m.rules[0])
	assert.Equal(t, r2, m.rules[1])
	assert.Equal(t, r3, m.rules[2])
}

// Assert that sorting an unsorted array results in a sorted array - check priority and rule name
func TestSort2(t *testing.T) {
	r1 := &rule{
		ruleProperties: &ruleProperties{
			RuleName: getStringPointer("r1"),
			Priority: getIntPointer(5),
		},
	}

	r2 := &rule{
		ruleProperties: &ruleProperties{
			RuleName: getStringPointer("r2"),
			Priority: getIntPointer(6),
		},
	}

	r3 := &rule{
		ruleProperties: &ruleProperties{
			RuleName: getStringPointer("r3"),
			Priority: getIntPointer(7),
		},
	}

	// Unsorted rules array
	rules := []*rule{r2, r1, r3}

	m := &manifest{
		rules: rules,
	}

	// Sort array
	m.sort() // r1 should precede r2

	// Assert on order
	assert.Equal(t, r1, m.rules[0])
	assert.Equal(t, r2, m.rules[1])
	assert.Equal(t, r3, m.rules[2])
}
