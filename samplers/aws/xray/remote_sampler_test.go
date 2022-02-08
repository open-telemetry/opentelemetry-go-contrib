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
  "context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Assert that snapshots returns an array of valid sampling statistics
func TestSnapshots(t *testing.T) {
	clock := &mockClock{
		nowTime: 1500000000,
	}

	id := "c1"
	time := clock.now().Unix()

	name1 := "r1"
	requests1 := int64(1000)
	sampled1 := int64(100)
	borrows1 := int64(5)
	r1 := &reservoir{
		interval: 10,
	}
	csr1 := &rule{
		ruleProperties: &ruleProperties{
			RuleName: getStringPointer(name1),
		},
		matchedRequests:  requests1,
		sampledRequests:  sampled1,
		borrowedRequests: borrows1,
		reservoir:        r1,
		clock:            clock,
	}

	name2 := "r2"
	requests2 := int64(500)
	sampled2 := int64(10)
	borrows2 := int64(0)
	r2 := &reservoir{
		interval: 10,
	}
	csr2 := &rule{
		ruleProperties: &ruleProperties{
			RuleName: getStringPointer(name2),
		},
		matchedRequests:  requests2,
		sampledRequests:  sampled2,
		borrowedRequests: borrows2,
		reservoir:        r2,
		clock:            clock,
	}

	rules := []*rule{csr1, csr2}

	m := &manifest{
		rules: rules,
	}

	sampler := &remoteSampler{
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
	clock := &mockClock{
		nowTime: 1500000000,
	}

	id := "c1"
	time := clock.now().Unix()

	// Stale and active rule
	name1 := "r1"
	requests1 := int64(1000)
	sampled1 := int64(100)
	borrows1 := int64(5)
	r1 := &reservoir{
		interval:    20,
		refreshedAt: 1499999980,
	}
	csr1 := &rule{
		ruleProperties: &ruleProperties{
			RuleName: getStringPointer(name1),
		},
		matchedRequests:  requests1,
		sampledRequests:  sampled1,
		borrowedRequests: borrows1,
		reservoir:        r1,
		clock:            clock,
	}

	// Stale and inactive rule
	name2 := "r2"
	requests2 := int64(0)
	sampled2 := int64(0)
	borrows2 := int64(0)
	r2 := &reservoir{
		interval:    20,
		refreshedAt: 1499999970,
	}
	csr2 := &rule{
		ruleProperties: &ruleProperties{
			RuleName: getStringPointer(name2),
		},
		matchedRequests:  requests2,
		sampledRequests:  sampled2,
		borrowedRequests: borrows2,
		reservoir:        r2,
		clock:            clock,
	}

	// Fresh rule
	name3 := "r3"
	requests3 := int64(1000)
	sampled3 := int64(100)
	borrows3 := int64(5)
	r3 := &reservoir{
		interval:    20,
		refreshedAt: 1499999990,
	}
	csr3 := &rule{
		ruleProperties: &ruleProperties{
			RuleName: getStringPointer(name3),
		},
		matchedRequests:  requests3,
		sampledRequests:  sampled3,
		borrowedRequests: borrows3,
		reservoir:        r3,
		clock:            clock,
	}

	rules := []*rule{csr1, csr2, csr3}

	m := &manifest{
		rules: rules,
	}

	sampler := &remoteSampler{
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
	clock := &mockClock{
		nowTime: 1500000000,
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
	csr := &rule{
		ruleProperties: &ruleProperties{
			RuleName:  getStringPointer("r1"),
			FixedRate: getFloatPointer(0.10),
		},
		reservoir: &reservoir{
			quota:        8,
			refreshedAt:  1499999990,
			expiresAt:    1500000010,
			capacity:     50,
			used:         7,
			currentEpoch: 1500000000,
		},
	}

	rules := []*rule{csr}

	index := map[string]*rule{
		"r1": csr,
	}

	m := &manifest{
		rules: rules,
		index: index,
	}

	s := &remoteSampler{
		manifest: m,
		clock:    clock,
	}

	err := s.updateTarget(st)
	assert.Nil(t, err)

	// Updated sampling rule
	exp := &rule{
		ruleProperties: &ruleProperties{
			RuleName:  getStringPointer("r1"),
			FixedRate: getFloatPointer(0.05),
		},
		reservoir: &reservoir{
			quota:        10,
			refreshedAt:  1500000000,
			expiresAt:    1500000060,
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

	var rules []*rule

	index := map[string]*rule{}

	m := &manifest{
		rules: rules,
		index: index,
	}

	s := &remoteSampler{
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
	csr := &rule{
		ruleProperties: &ruleProperties{
			RuleName:  getStringPointer("r1"),
			FixedRate: getFloatPointer(0.10),
		},
		reservoir: &reservoir{
			quota:        8,
			expiresAt:    1500000010,
			capacity:     50,
			used:         7,
			currentEpoch: 1500000000,
		},
	}

	rules := []*rule{csr}

	index := map[string]*rule{
		"r1": csr,
	}

	m := &manifest{
		rules: rules,
		index: index,
	}

	s := &remoteSampler{
		manifest: m,
	}

	err := s.updateTarget(st)
	assert.NotNil(t, err)

	// Unchanged sampling rule
	exp := &rule{
		ruleProperties: &ruleProperties{
			RuleName:  getStringPointer("r1"),
			FixedRate: getFloatPointer(0.10),
		},
		reservoir: &reservoir{
			quota:        8,
			expiresAt:    1500000010,
			capacity:     50,
			used:         7,
			currentEpoch: 1500000000,
		},
	}

	act := s.manifest.rules[0]

	assert.Equal(t, exp, act)
}

func TestRefreshManifest(t *testing.T) {
	ctx := context.Background()

	body := []byte(`{
  "NextToken": null,
  "SamplingRuleRecords": [
    {
      "CreatedAt": 0,
      "ModifiedAt": 1639517389,
      "SamplingRule": {
        "Attributes": {},
        "FixedRate": 0.5,
        "HTTPMethod": "*",
        "Host": "*",
        "Priority": 10000,
        "ReservoirSize": 60,
        "ResourceARN": "*",
        "RuleARN": "arn:aws:xray:us-west-2:xxxxxxx:sampling-rule/r1",
        "RuleName": "r1",
        "ServiceName": "*",
        "ServiceType": "*",
        "URLPath": "*",
        "Version": 1
      }
    },
    {
      "CreatedAt": 1637691613,
      "ModifiedAt": 1643748669,
      "SamplingRule": {
        "Attributes": {},
        "FixedRate": 0.09,
        "HTTPMethod": "GET",
        "Host": "*",
        "Priority": 1,
        "ReservoirSize": 3,
        "ResourceARN": "*",
        "RuleARN": "arn:aws:xray:us-west-2:xxxxxxx:sampling-rule/r2",
        "RuleName": "r2",
        "ServiceName": "test-rule",
        "ServiceType": "*",
        "URLPath": "/aws-sdk-call",
        "Version": 1
      }
    },
    {
      "CreatedAt": 1639446197,
      "ModifiedAt": 1639446197,
      "SamplingRule": {
        "Attributes": {},
        "FixedRate": 0.09,
        "HTTPMethod": "*",
        "Host": "*",
        "Priority": 100,
        "ReservoirSize": 100,
        "ResourceARN": "*",
        "RuleARN": "arn:aws:xray:us-west-2:xxxxxxx:sampling-rule/r3",
        "RuleName": "r3",
        "ServiceName": "*",
        "ServiceType": "local",
        "URLPath": "*",
        "Version": 1
      }
    }
  ]
}`)

	// generate a test server so we can capture and inspect the request
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		_, err := res.Write([]byte(body))
		require.NoError(t, err)
	}))
	defer testServer.Close()

	u, err := url.Parse(testServer.URL)
	require.NoError(t, err)

	clock := &defaultClock{}

	m := &manifest{
		rules: []*rule{},
		index: map[string]*rule{},
		clock: clock,
	}

	rs := &remoteSampler{
		xrayClient: newClient(u.Host),
		clock:      clock,
		manifest:   m,
	}

	err = rs.refreshManifest(ctx)
	require.NoError(t, err)

	// Rule 'r1'
	r1 := &rule{
		ruleProperties: &ruleProperties{
			RuleName:      getStringPointer("r1"),
			Priority:      getIntPointer(10000),
			Host:          getStringPointer("*"),
			HTTPMethod:    getStringPointer("*"),
			URLPath:       getStringPointer("*"),
			ReservoirSize: getIntPointer(60),
			Version:       getIntPointer(1),
			FixedRate:     getFloatPointer(0.5),
			ServiceName:   getStringPointer("*"),
			ResourceARN:   getStringPointer("*"),
			ServiceType:   getStringPointer("*"),
		},
	}

	// Rule 'r2'
	r2 := &rule{
		ruleProperties: &ruleProperties{
			RuleName:      getStringPointer("r2"),
			Priority:      getIntPointer(1),
			Host:          getStringPointer("*"),
			HTTPMethod:    getStringPointer("GET"),
			URLPath:       getStringPointer("/aws-sdk-call"),
			ReservoirSize: getIntPointer(3),
			FixedRate:     getFloatPointer(0.09),
			Version:       getIntPointer(1),
			ServiceName:   getStringPointer("test-rule"),
			ResourceARN:   getStringPointer("*"),
			ServiceType:   getStringPointer("*"),
		},
	}

	// Rule 'r3'
	r3 := &rule{
		ruleProperties: &ruleProperties{
			RuleName:      getStringPointer("r3"),
			Priority:      getIntPointer(100),
			Host:          getStringPointer("*"),
			HTTPMethod:    getStringPointer("*"),
			URLPath:       getStringPointer("*"),
			ReservoirSize: getIntPointer(100),
			FixedRate:     getFloatPointer(0.09),
			Version:       getIntPointer(1),
			ServiceName:   getStringPointer("*"),
			ResourceARN:   getStringPointer("*"),
			ServiceType:   getStringPointer("local"),
		},
	}
	// Assert on sorting order
	assert.Equal(t, r2.ruleProperties.RuleName, rs.manifest.rules[0].ruleProperties.RuleName)
	assert.Equal(t, r2.ruleProperties.Priority, rs.manifest.rules[0].ruleProperties.Priority)
	assert.Equal(t, r2.ruleProperties.Host, rs.manifest.rules[0].ruleProperties.Host)
	assert.Equal(t, r2.ruleProperties.HTTPMethod, rs.manifest.rules[0].ruleProperties.HTTPMethod)
	assert.Equal(t, r2.ruleProperties.URLPath, rs.manifest.rules[0].ruleProperties.URLPath)
	assert.Equal(t, r2.ruleProperties.ReservoirSize, rs.manifest.rules[0].ruleProperties.ReservoirSize)
	assert.Equal(t, r2.ruleProperties.FixedRate, rs.manifest.rules[0].ruleProperties.FixedRate)
	assert.Equal(t, r2.ruleProperties.Version, rs.manifest.rules[0].ruleProperties.Version)
	assert.Equal(t, r2.ruleProperties.ServiceName, rs.manifest.rules[0].ruleProperties.ServiceName)
	assert.Equal(t, r2.ruleProperties.ResourceARN, rs.manifest.rules[0].ruleProperties.ResourceARN)
	assert.Equal(t, r2.ruleProperties.ServiceType, rs.manifest.rules[0].ruleProperties.ServiceType)

	assert.Equal(t, r3.ruleProperties.RuleName, rs.manifest.rules[1].ruleProperties.RuleName)
	assert.Equal(t, r3.ruleProperties.Priority, rs.manifest.rules[1].ruleProperties.Priority)
	assert.Equal(t, r3.ruleProperties.Host, rs.manifest.rules[1].ruleProperties.Host)
	assert.Equal(t, r3.ruleProperties.HTTPMethod, rs.manifest.rules[1].ruleProperties.HTTPMethod)
	assert.Equal(t, r3.ruleProperties.URLPath, rs.manifest.rules[1].ruleProperties.URLPath)
	assert.Equal(t, r3.ruleProperties.ReservoirSize, rs.manifest.rules[1].ruleProperties.ReservoirSize)
	assert.Equal(t, r3.ruleProperties.FixedRate, rs.manifest.rules[1].ruleProperties.FixedRate)
	assert.Equal(t, r3.ruleProperties.Version, rs.manifest.rules[1].ruleProperties.Version)
	assert.Equal(t, r3.ruleProperties.ServiceName, rs.manifest.rules[1].ruleProperties.ServiceName)
	assert.Equal(t, r3.ruleProperties.ResourceARN, rs.manifest.rules[1].ruleProperties.ResourceARN)
	assert.Equal(t, r3.ruleProperties.ServiceType, rs.manifest.rules[1].ruleProperties.ServiceType)

	assert.Equal(t, r1.ruleProperties.RuleName, rs.manifest.rules[2].ruleProperties.RuleName)
	assert.Equal(t, r1.ruleProperties.Priority, rs.manifest.rules[2].ruleProperties.Priority)
	assert.Equal(t, r1.ruleProperties.Host, rs.manifest.rules[2].ruleProperties.Host)
	assert.Equal(t, r1.ruleProperties.HTTPMethod, rs.manifest.rules[2].ruleProperties.HTTPMethod)
	assert.Equal(t, r1.ruleProperties.URLPath, rs.manifest.rules[2].ruleProperties.URLPath)
	assert.Equal(t, r1.ruleProperties.ReservoirSize, rs.manifest.rules[2].ruleProperties.ReservoirSize)
	assert.Equal(t, r1.ruleProperties.FixedRate, rs.manifest.rules[2].ruleProperties.FixedRate)
	assert.Equal(t, r1.ruleProperties.Version, rs.manifest.rules[2].ruleProperties.Version)
	assert.Equal(t, r1.ruleProperties.ServiceName, rs.manifest.rules[2].ruleProperties.ServiceName)
	assert.Equal(t, r1.ruleProperties.ResourceARN, rs.manifest.rules[2].ruleProperties.ResourceARN)
	assert.Equal(t, r1.ruleProperties.ServiceType, rs.manifest.rules[2].ruleProperties.ServiceType)

	// Assert on size of manifest
	assert.Equal(t, 3, len(rs.manifest.rules))
	assert.Equal(t, 3, len(rs.manifest.index))
}

// assert that invalid rule with ResourceARN other than "*" does not update to the manifest
func TestRefreshManifestAddInvalidRuleType1(t *testing.T) {
	ctx := context.Background()

	// to enable logging
	newConfig()

	// invalid rule due to ResourceARN
	body := []byte(`{
  "NextToken": null,
  "SamplingRuleRecords": [
    {
      "CreatedAt": 0,
      "ModifiedAt": 1639517389,
      "SamplingRule": {
        "Attributes": {},
        "FixedRate": 0.5,
        "HTTPMethod": "*",
        "Host": "*",
        "Priority": 10000,
        "ReservoirSize": 60,
        "ResourceARN": "XYZ",
        "RuleARN": "arn:aws:xray:us-west-2:xxxxxxx:sampling-rule/r1",
        "RuleName": "r1",
        "ServiceName": "*",
        "ServiceType": "*",
        "URLPath": "*",
        "Version": 1
      }
    }
  ]
}`)

	// generate a test server so we can capture and inspect the request
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		_, err := res.Write([]byte(body))
		require.NoError(t, err)

	}))
	defer testServer.Close()

	u, err := url.Parse(testServer.URL)
	require.NoError(t, err)

	clock := &defaultClock{}

	m := &manifest{
		rules: []*rule{},
		index: map[string]*rule{},
		clock: clock,
	}

	rs := &remoteSampler{
		xrayClient: newClient(u.Host),
		clock:      clock,
		manifest:   m,
	}

	err = rs.refreshManifest(ctx)
	require.NoError(t, err)

	// Refresh manifest with updates from mock proxy
	assert.Equal(t, 0, len(rs.manifest.rules)) // Rule not added
}

// assert that invalid rule with attribute does not update to the manifest
func TestRefreshManifestAddInvalidRuleType2(t *testing.T) {
	ctx := context.Background()

	// to enable logging
	newConfig()

	// invalid rule due to attributes
	body := []byte(`{
  "NextToken": null,
  "SamplingRuleRecords": [
    {
      "CreatedAt": 0,
      "ModifiedAt": 1639517389,
      "SamplingRule": {
        "Attributes": {"a":"b"},
        "FixedRate": 0.5,
        "HTTPMethod": "*",
        "Host": "*",
        "Priority": 10000,
        "ReservoirSize": 60,
        "ResourceARN": "*",
        "RuleARN": "arn:aws:xray:us-west-2:xxxxxxx:sampling-rule/r1",
        "RuleName": "r1",
        "ServiceName": "*",
        "ServiceType": "*",
        "URLPath": "*",
        "Version": 1
      }
    }
  ]
}`)

	// generate a test server so we can capture and inspect the request
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		_, err := res.Write([]byte(body))
		require.NoError(t, err)
	}))
	defer testServer.Close()

	u, err := url.Parse(testServer.URL)
	require.NoError(t, err)

	clock := &defaultClock{}

	m := &manifest{
		rules: []*rule{},
		index: map[string]*rule{},
		clock: clock,
	}

	rs := &remoteSampler{
		xrayClient: newClient(u.Host),
		clock:      clock,
		manifest:   m,
	}

	err = rs.refreshManifest(ctx)
	require.NoError(t, err)

	assert.Equal(t, 0, len(rs.manifest.rules)) // rule not added
}

// assert that 1 valid and 1 invalid rule update only valid rule gets stored to the manifest
func TestRefreshManifestAddInvalidRule3(t *testing.T) {
	ctx := context.Background()

	// to enable logging
	newConfig()

	body := []byte(`{
  "NextToken": null,
  "SamplingRuleRecords": [
    {
      "CreatedAt": 0,
      "ModifiedAt": 1639517389,
      "SamplingRule": {
        "Attributes": {},
        "FixedRate": 0.5,
        "HTTPMethod": "*",
        "Host": "*",
        "Priority": 10000,
        "ReservoirSize": 60,
        "ResourceARN": "*",
        "RuleARN": "arn:aws:xray:us-west-2:xxxxxxx:sampling-rule/r1",
        "RuleName": "r1",
        "ServiceName": "*",
        "ServiceType": "*",
        "URLPath": "*",
        "Version": 1
      }
    },
   {
      "CreatedAt": 0,
      "ModifiedAt": 1639517389,
      "SamplingRule": {
        "Attributes": {"a":"b"},
        "FixedRate": 0.5,
        "HTTPMethod": "*",
        "Host": "*",
        "Priority": 10000,
        "ReservoirSize": 60,
        "ResourceARN": "*",
        "RuleARN": "arn:aws:xray:us-west-2:xxxxxxx:sampling-rule/r2",
        "RuleName": "r2",
        "ServiceName": "*",
        "ServiceType": "*",
        "URLPath": "*",
        "Version": 1
      }
    }
  ]
}`)
	// Rule 'r1'
	r1 := &rule{
		ruleProperties: &ruleProperties{
			RuleName:      getStringPointer("r1"),
			Priority:      getIntPointer(10000),
			Host:          getStringPointer("*"),
			HTTPMethod:    getStringPointer("*"),
			URLPath:       getStringPointer("*"),
			ReservoirSize: getIntPointer(60),
			FixedRate:     getFloatPointer(0.5),
			Version:       getIntPointer(1),
			ServiceName:   getStringPointer("*"),
			ResourceARN:   getStringPointer("*"),
			ServiceType:   getStringPointer("*"),
		},
	}

	// generate a test server so we can capture and inspect the request
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		_, err := res.Write([]byte(body))
		require.NoError(t, err)
	}))
	defer testServer.Close()

	u, err := url.Parse(testServer.URL)
	require.NoError(t, err)

	clock := &defaultClock{}

	m := &manifest{
		rules: []*rule{},
		index: map[string]*rule{},
		clock: clock,
	}

	rs := &remoteSampler{
		xrayClient: newClient(u.Host),
		clock:      clock,
		manifest:   m,
	}

	err = rs.refreshManifest(ctx)
	require.NoError(t, err)

	assert.Equal(t, 1, len(rs.manifest.rules))

	assert.Equal(t, r1.ruleProperties.RuleName, rs.manifest.rules[0].ruleProperties.RuleName)
	assert.Equal(t, r1.ruleProperties.Priority, rs.manifest.rules[0].ruleProperties.Priority)
	assert.Equal(t, r1.ruleProperties.Host, rs.manifest.rules[0].ruleProperties.Host)
	assert.Equal(t, r1.ruleProperties.HTTPMethod, rs.manifest.rules[0].ruleProperties.HTTPMethod)
	assert.Equal(t, r1.ruleProperties.URLPath, rs.manifest.rules[0].ruleProperties.URLPath)
	assert.Equal(t, r1.ruleProperties.ReservoirSize, rs.manifest.rules[0].ruleProperties.ReservoirSize)
	assert.Equal(t, r1.ruleProperties.FixedRate, rs.manifest.rules[0].ruleProperties.FixedRate)
	assert.Equal(t, r1.ruleProperties.Version, rs.manifest.rules[0].ruleProperties.Version)
	assert.Equal(t, r1.ruleProperties.ServiceName, rs.manifest.rules[0].ruleProperties.ServiceName)
	assert.Equal(t, r1.ruleProperties.ResourceARN, rs.manifest.rules[0].ruleProperties.ResourceARN)
	assert.Equal(t, r1.ruleProperties.ServiceType, rs.manifest.rules[0].ruleProperties.ServiceType)
}

// assert that manifest rules and index correctly updates from temporary manifest with each update
func TestManifestRulesAndIndexUpdate(t *testing.T) {
	ctx := context.Background()
	count := 0

	// to enable logging
	newConfig()

	// first update
	body1 := []byte(`{
  "NextToken": null,
  "SamplingRuleRecords": [
    {
      "CreatedAt": 0,
      "ModifiedAt": 1639517389,
      "SamplingRule": {
        "Attributes": {},
        "FixedRate": 0.5,
        "HTTPMethod": "*",
        "Host": "*",
        "Priority": 100000,
        "ReservoirSize": 60,
        "ResourceARN": "*",
        "RuleARN": "arn:aws:xray:us-west-2:xxxxxxx:sampling-rule/r1",
        "RuleName": "r1",
        "ServiceName": "*",
        "ServiceType": "*",
        "URLPath": "*",
        "Version": 1
      }
    },
   {
      "CreatedAt": 0,
      "ModifiedAt": 1639517389,
      "SamplingRule": {
        "Attributes": {},
        "FixedRate": 0.5,
        "HTTPMethod": "*",
        "Host": "*",
        "Priority": 10000,
        "ReservoirSize": 60,
        "ResourceARN": "*",
        "RuleARN": "arn:aws:xray:us-west-2:xxxxxxx:sampling-rule/r2",
        "RuleName": "r2",
        "ServiceName": "*",
        "ServiceType": "*",
        "URLPath": "*",
        "Version": 1
      }
    }
  ]
}`)
	// second update
	body2 := []byte(`{
  "NextToken": null,
  "SamplingRuleRecords": [
    {
      "CreatedAt": 0,
      "ModifiedAt": 1639517389,
      "SamplingRule": {
        "Attributes": {},
        "FixedRate": 0.5,
        "HTTPMethod": "*",
        "Host": "*",
        "Priority": 100000,
        "ReservoirSize": 60,
        "ResourceARN": "*",
        "RuleARN": "arn:aws:xray:us-west-2:xxxxxxx:sampling-rule/r1",
        "RuleName": "r1",
        "ServiceName": "*",
        "ServiceType": "*",
        "URLPath": "*",
        "Version": 1
      }
    }
  ]
}`)

	// generate a test server so we can capture and inspect the request
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if count == 0 {
			// first update
			_, err := res.Write([]byte(body1))
			require.NoError(t, err)
		} else {
			// second update
			_, err := res.Write([]byte(body2))
			require.NoError(t, err)
		}
	}))
	defer testServer.Close()

	u, err := url.Parse(testServer.URL)
	require.NoError(t, err)

	clock := &defaultClock{}

	m := &manifest{
		rules: []*rule{},
		index: map[string]*rule{},
		clock: clock,
	}

	rs := &remoteSampler{
		xrayClient: newClient(u.Host),
		clock:      clock,
		manifest:   m,
	}

	err = rs.refreshManifest(ctx)
	require.NoError(t, err)

	// assert that manifest has 2 rules and indexes currently
	assert.Equal(t, 2, len(rs.manifest.rules))
	assert.Equal(t, 2, len(rs.manifest.index))

	assert.Equal(t, rs.manifest.rules[0].ruleProperties.RuleName, getStringPointer("r2"))
	assert.Equal(t, rs.manifest.rules[1].ruleProperties.RuleName, getStringPointer("r1"))

	// assert that both the rules are available in manifest index
	_, okRule1 := rs.manifest.index[*rs.manifest.rules[0].ruleProperties.RuleName]
	_, okRule2 := rs.manifest.index[*rs.manifest.rules[1].ruleProperties.RuleName]

	assert.True(t, okRule1)
	assert.True(t, okRule2)

	// second update
	count++
	err = rs.refreshManifest(ctx)
	require.NoError(t, err)

	// assert that manifest has 1 "r1" rule and index currently
	assert.Equal(t, 1, len(rs.manifest.rules))
	assert.Equal(t, 1, len(rs.manifest.index))

	assert.Equal(t, rs.manifest.rules[0].ruleProperties.RuleName, getStringPointer("r1"))

	// assert that "r1" rule available in index
	_, okRule := rs.manifest.index[*rs.manifest.rules[0].ruleProperties.RuleName]
	assert.True(t, okRule)
}

// assert that NewRemoteSampler returns a sampler with *xray.remoteSampler type
func TestNewRemoteSampler(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rs, err := NewRemoteSampler(ctx, "test", "local")
	require.NoError(t, err)

	s := &remoteSampler{}
	assert.Equal(t, reflect.TypeOf(rs), reflect.TypeOf(s))
}
