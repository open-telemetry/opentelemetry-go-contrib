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
	"github.com/stretchr/testify/assert"
	"testing"
)

// Assert that snapshots returns an array of valid sampling statistics
func TestSnapshots(t *testing.T) {
	clock := &MockClock{
		NowTime: 1500000000,
	}

	id := "c1"
	time := clock.Now().Unix()

	name1 := "r1"
	requests1 := int64(1000)
	sampled1 := int64(100)
	borrows1 := int64(5)
	r1 := &centralizedReservoir{
		interval: 10,
	}
	csr1 := &centralizedRule{
		ruleProperties: &ruleProperties{
			RuleName: getStringPointer(name1),
		},
		matchedRequests:  requests1,
		sampledRequests:   sampled1,
		borrowedRequests:   borrows1,
		reservoir: r1,
		clock:     clock,
	}

	name2 := "r2"
	requests2 := int64(500)
	sampled2 := int64(10)
	borrows2 := int64(0)
	r2 := &centralizedReservoir{
		interval: 10,
	}
	csr2 := &centralizedRule{
		ruleProperties: &ruleProperties{
			RuleName: getStringPointer(name2),
		},
		matchedRequests:  requests2,
		sampledRequests:   sampled2,
		borrowedRequests:   borrows2,
		reservoir: r2,
		clock:     clock,
	}

	rules := []*centralizedRule{csr1, csr2}

	m := &centralizedManifest{
		rules: rules,
	}

	sampler := &RemoteSampler{
		manifest: m,
		clientID: id,
		clock:    clock,
	}

	// Expected SamplingStatistics structs
	ss1 := samplingStatisticsDocument{
		ClientID:     &id,
		RequestCount: &requests1,
		RuleName:     &name1,
		SampledCount: &sampled1,
		BorrowCount:  &borrows1,
		Timestamp:    &time,
	}

	ss2 := samplingStatisticsDocument{
		ClientID:     &id,
		RequestCount: &requests2,
		RuleName:     &name2,
		SampledCount: &sampled2,
		BorrowCount:  &borrows2,
		Timestamp:    &time,
	}

	statistics := sampler.snapshots()

	assert.Equal(t, ss1, *statistics[0])
	assert.Equal(t, ss2, *statistics[1])
}

// Assert that fresh and inactive rules are not included in a snapshot
func TestMixedSnapshots(t *testing.T) {
	clock := &MockClock{
		NowTime: 1500000000,
	}

	id := "c1"
	time := clock.Now().Unix()

	// Stale and active rule
	name1 := "r1"
	requests1 := int64(1000)
	sampled1 := int64(100)
	borrows1 := int64(5)
	r1 := &centralizedReservoir{
		interval:    20,
		refreshedAt: 1499999980,
	}
	csr1 := &centralizedRule{
		ruleProperties: &ruleProperties{
			RuleName: getStringPointer(name1),
		},
		matchedRequests:  requests1,
		sampledRequests:   sampled1,
		borrowedRequests:   borrows1,
		reservoir: r1,
		clock:     clock,
	}

	// Stale and inactive rule
	name2 := "r2"
	requests2 := int64(0)
	sampled2 := int64(0)
	borrows2 := int64(0)
	r2 := &centralizedReservoir{
		interval:    20,
		refreshedAt: 1499999970,
	}
	csr2 := &centralizedRule{
		ruleProperties: &ruleProperties{
			RuleName: getStringPointer(name2),
		},
		matchedRequests:  requests2,
		sampledRequests:   sampled2,
		borrowedRequests:   borrows2,
		reservoir: r2,
		clock:     clock,
	}

	// Fresh rule
	name3 := "r3"
	requests3 := int64(1000)
	sampled3 := int64(100)
	borrows3 := int64(5)
	r3 := &centralizedReservoir{
		interval:    20,
		refreshedAt: 1499999990,
	}
	csr3 := &centralizedRule{
		ruleProperties: &ruleProperties{
			RuleName: getStringPointer(name3),
		},
		matchedRequests:  requests3,
		sampledRequests:   sampled3,
		borrowedRequests:   borrows3,
		reservoir: r3,
		clock:     clock,
	}

	rules := []*centralizedRule{csr1, csr2, csr3}

	m := &centralizedManifest{
		rules: rules,
	}

	sampler := &RemoteSampler{
		manifest: m,
		clientID: id,
		clock:    clock,
	}

	// Expected SamplingStatistics structs
	ss1 := samplingStatisticsDocument{
		ClientID:     &id,
		RequestCount: &requests1,
		RuleName:     &name1,
		SampledCount: &sampled1,
		BorrowCount:  &borrows1,
		Timestamp:    &time,
	}

	statistics := sampler.snapshots()

	assert.Equal(t, 1, len(statistics))
	assert.Equal(t, ss1, *statistics[0])
}

// Assert that a valid sampling target updates its rule
func TestUpdateTarget(t *testing.T) {
	clock := &MockClock{
		NowTime: 1500000000,
	}

	// Sampling target received from centralized sampling backend
	rate := 0.05
	quota := int64(10)
	ttl := float64(1500000060)
	name := "r1"
	st := &samplingTargetDocument{
		FixedRate:         &rate,
		ReservoirQuota:    &quota,
		ReservoirQuotaTTL: &ttl,
		RuleName:          &name,
	}

	// Sampling rule about to be updated with new target
	csr := &centralizedRule{
		ruleProperties: &ruleProperties{
			RuleName: getStringPointer("r1"),
			FixedRate: getFloatPointer(0.10),
		},
		reservoir: &centralizedReservoir{
			quota:       8,
			refreshedAt: 1499999990,
			expiresAt:   1500000010,
			capacity:     50,
			used:         7,
			currentEpoch: 1500000000,
		},
	}

	rules := []*centralizedRule{csr}

	index := map[string]*centralizedRule{
		"r1": csr,
	}

	m := &centralizedManifest{
		rules: rules,
		index: index,
	}

	s := &RemoteSampler{
		manifest: m,
		clock:    clock,
	}

	err := s.updateTarget(st)
	assert.Nil(t, err)

	// Updated sampling rule
	exp := &centralizedRule{
		ruleProperties: &ruleProperties{
			RuleName: getStringPointer("r1"),
			FixedRate: getFloatPointer(0.05),
		},
		reservoir: &centralizedReservoir{
			quota:       10,
			refreshedAt: 1500000000,
			expiresAt:   1500000060,
			capacity:     50,
			used:         7,
			currentEpoch: 1500000000,
		},
	}

	assert.Equal(t, exp, s.manifest.rules[0])
}

// Assert that a missing sampling rule returns an error
func TestUpdateTargetMissingRule(t *testing.T) {
	// Sampling target received from centralized sampling backend
	rate := 0.05
	quota := int64(10)
	ttl := float64(1500000060)
	name := "r1"
	st := &samplingTargetDocument{
		FixedRate:         &rate,
		ReservoirQuota:    &quota,
		ReservoirQuotaTTL: &ttl,
		RuleName:          &name,
	}

	var rules []*centralizedRule

	index := map[string]*centralizedRule{}

	m := &centralizedManifest{
		rules: rules,
		index: index,
	}

	s := &RemoteSampler{
		manifest: m,
	}

	err := s.updateTarget(st)
	assert.NotNil(t, err)
}

// Assert that an invalid sampling target returns an error and does not panic
func TestUpdateTargetPanicRecovery(t *testing.T) {
	// Invalid sampling target missing FixedRate.
	quota := int64(10)
	ttl := float64(1500000060)
	name := "r1"
	st := &samplingTargetDocument{
		ReservoirQuota:    &quota,
		ReservoirQuotaTTL: &ttl,
		RuleName:          &name,
	}

	// Sampling rule about to be updated with new target
	csr := &centralizedRule{
		ruleProperties: &ruleProperties{
			RuleName: getStringPointer("r1"),
			FixedRate: getFloatPointer(0.10),
		},
		reservoir: &centralizedReservoir{
			quota:     8,
			expiresAt: 1500000010,
			capacity:     50,
			used:         7,
			currentEpoch: 1500000000,
		},
	}

	rules := []*centralizedRule{csr}

	index := map[string]*centralizedRule{
		"r1": csr,
	}

	m := &centralizedManifest{
		rules: rules,
		index: index,
	}

	s := &RemoteSampler{
		manifest: m,
	}

	err := s.updateTarget(st)
	assert.NotNil(t, err)

	// Unchanged sampling rule
	exp := &centralizedRule{
		ruleProperties: &ruleProperties{
			RuleName: getStringPointer("r1"),
			FixedRate: getFloatPointer(0.10),
		},
		reservoir: &centralizedReservoir{
			quota:     8,
			expiresAt: 1500000010,
			capacity:     50,
			used:         7,
			currentEpoch: 1500000000,
		},
	}

	act := s.manifest.rules[0]

	assert.Equal(t, exp, act)
}
