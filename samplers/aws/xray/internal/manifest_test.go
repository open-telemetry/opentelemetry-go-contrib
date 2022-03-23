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

package internal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"

	"github.com/go-logr/logr/testr"
	"github.com/stretchr/testify/require"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/stretchr/testify/assert"
)

// assert that new manifest has certain non-nil attributes.
func TestNewManifest(t *testing.T) {
	logger := testr.New(t)

	endpoint, err := url.Parse("http://127.0.0.1:2020")
	require.NoError(t, err)

	m, err := NewManifest(*endpoint, logger)
	require.NoError(t, err)

	assert.NotEmpty(t, m.logger)
	assert.NotEmpty(t, m.clientID)
	assert.NotEmpty(t, m.SamplingTargetsPollingInterval)

	assert.NotNil(t, m.xrayClient)
}

// assert that manifest is expired.
func TestExpiredManifest(t *testing.T) {
	clock := &mockClock{
		nowTime: 10000,
	}

	refreshedAt := time.Unix(3700, 0)
	m := &Manifest{
		clock:       clock,
		refreshedAt: refreshedAt,
	}

	assert.True(t, m.Expired())
}

// assert that if collector is not enabled at specified endpoint, returns an error
func TestRefreshManifestError(t *testing.T) {
	// collector is not running at port 2020 so expect error
	endpoint, err := url.Parse("http://127.0.0.1:2020")
	require.NoError(t, err)

	client, err := newClient(*endpoint)
	require.NoError(t, err)

	m := &Manifest{
		xrayClient: client,
	}

	err = m.RefreshManifestRules(context.Background())
	assert.Error(t, err)
}

// assert that manifest rule r2 is a match for sampling.
func TestMatchAgainstManifestRules(t *testing.T) {
	r1 := Rule{
		ruleProperties: ruleProperties{
			RuleName:      "r1",
			Priority:      10000,
			Host:          "*",
			HTTPMethod:    "*",
			URLPath:       "*",
			ReservoirSize: 60,
			FixedRate:     0.5,
			Version:       1,
			ServiceName:   "helios",
			ResourceARN:   "*",
			ServiceType:   "*",
		},
		reservoir: reservoir{
			expiresAt: time.Unix(14050, 0),
		},
	}

	r2 := Rule{
		ruleProperties: ruleProperties{
			RuleName:      "r2",
			Priority:      100,
			Host:          "*",
			HTTPMethod:    "*",
			URLPath:       "*",
			ReservoirSize: 6,
			FixedRate:     0.5,
			Version:       1,
			ServiceName:   "test",
			ResourceARN:   "*",
			ServiceType:   "local",
		},
		reservoir: reservoir{
			expiresAt: time.Unix(14050, 0),
		},
	}

	rules := []Rule{r1, r2}

	m := &Manifest{
		Rules: rules,
	}

	exp, match, err := m.MatchAgainstManifestRules(sdktrace.SamplingParameters{}, "test", "local")
	require.True(t, match)
	require.NoError(t, err)

	// assert that manifest rule r2 is a match
	assert.Equal(t, *exp, r2)
}

// assert that if rules has attribute and span has those attribute with same value then matching will happen.
func TestMatchAgainstManifestRulesAttributeMatch(t *testing.T) {
	commonLabels := []attribute.KeyValue{
		attribute.String("labelA", "chocolate"),
		attribute.String("labelB", "raspberry"),
	}

	r1 := Rule{
		ruleProperties: ruleProperties{
			RuleName:      "r1",
			Priority:      10000,
			Host:          "*",
			HTTPMethod:    "*",
			URLPath:       "*",
			ReservoirSize: 60,
			FixedRate:     0.5,
			Version:       1,
			ServiceName:   "*",
			ResourceARN:   "*",
			ServiceType:   "*",
			Attributes: map[string]string{
				"labelA": "chocolate",
				"labelB": "raspberry",
			},
		},
		reservoir: reservoir{
			expiresAt: time.Unix(14050, 0),
		},
	}

	rules := []Rule{r1}

	m := &Manifest{
		Rules: rules,
	}

	exp, match, err := m.MatchAgainstManifestRules(sdktrace.SamplingParameters{Attributes: commonLabels}, "test", "local")
	require.True(t, match)
	require.NoError(t, err)

	// assert that manifest rule r1 is a match
	assert.Equal(t, *exp, r1)
}

// assert that wildcard attributes will match.
func TestMatchAgainstManifestRulesAttributeWildCardMatch(t *testing.T) {
	commonLabels := []attribute.KeyValue{
		attribute.String("labelA", "chocolate"),
		attribute.String("labelB", "raspberry"),
	}

	r1 := Rule{
		ruleProperties: ruleProperties{
			RuleName:      "r1",
			Priority:      10000,
			Host:          "*",
			HTTPMethod:    "*",
			URLPath:       "*",
			ReservoirSize: 60,
			FixedRate:     0.5,
			Version:       1,
			ServiceName:   "*",
			ResourceARN:   "*",
			ServiceType:   "*",
			Attributes: map[string]string{
				"labelA": "choco*",
				"labelB": "rasp*",
			},
		},
		reservoir: reservoir{
			expiresAt: time.Unix(14050, 0),
		},
	}

	rules := []Rule{r1}

	m := &Manifest{
		Rules: rules,
	}

	exp, match, err := m.MatchAgainstManifestRules(sdktrace.SamplingParameters{Attributes: commonLabels}, "test", "local")
	require.True(t, match)
	require.NoError(t, err)

	// assert that manifest rule r1 is a match
	assert.Nil(t, err)
	assert.Equal(t, *exp, r1)
}

// assert that when no known rule is match then returned rule is nil,
// matched flag is false
func TestMatchAgainstManifestRulesNoMatch(t *testing.T) {
	r1 := Rule{
		ruleProperties: ruleProperties{
			RuleName:      "r1",
			Priority:      10000,
			Host:          "*",
			HTTPMethod:    "*",
			URLPath:       "*",
			ReservoirSize: 60,
			FixedRate:     0.5,
			Version:       1,
			ServiceName:   "test-no-match",
			ResourceARN:   "*",
			ServiceType:   "local",
		},
		reservoir: reservoir{
			expiresAt: time.Unix(14050, 0),
		},
	}

	rules := []Rule{r1}

	m := &Manifest{
		Rules: rules,
	}

	rule, isMatch, err := m.MatchAgainstManifestRules(sdktrace.SamplingParameters{}, "test", "local")

	// assert that when no known rule is match then returned rule is nil
	require.NoError(t, err)
	assert.False(t, isMatch)
	assert.Nil(t, rule)
}

func TestRefreshManifestRules(t *testing.T) {
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
		_, err := res.Write(body)
		require.NoError(t, err)
	}))
	t.Cleanup(testServer.Close)

	client, err := createTestClient(testServer.URL)
	require.NoError(t, err)

	m := &Manifest{
		Rules:      []Rule{},
		xrayClient: client,
		clock:      &defaultClock{},
	}

	err = m.RefreshManifestRules(ctx)
	require.NoError(t, err)

	r1 := Rule{
		ruleProperties: ruleProperties{
			RuleName:      "r1",
			Priority:      10000,
			Host:          "*",
			HTTPMethod:    "*",
			URLPath:       "*",
			ReservoirSize: 60,
			Version:       1,
			FixedRate:     0.5,
			ServiceName:   "*",
			ResourceARN:   "*",
			ServiceType:   "*",
			Attributes:    map[string]string{},
		},
		reservoir: reservoir{
			capacity: 60,
			mu:       &sync.RWMutex{},
		},
		samplingStatistics: &samplingStatistics{},
	}

	r2 := Rule{
		ruleProperties: ruleProperties{
			RuleName:      "r2",
			Priority:      1,
			Host:          "*",
			HTTPMethod:    "GET",
			URLPath:       "/aws-sdk-call",
			ReservoirSize: 3,
			FixedRate:     0.09,
			Version:       1,
			ServiceName:   "test-rule",
			ResourceARN:   "*",
			ServiceType:   "*",
			Attributes:    map[string]string{},
		},
		reservoir: reservoir{
			capacity: 3,
			mu:       &sync.RWMutex{},
		},
		samplingStatistics: &samplingStatistics{},
	}

	r3 := Rule{
		ruleProperties: ruleProperties{
			RuleName:      "r3",
			Priority:      100,
			Host:          "*",
			HTTPMethod:    "*",
			URLPath:       "*",
			ReservoirSize: 100,
			FixedRate:     0.09,
			Version:       1,
			ServiceName:   "*",
			ResourceARN:   "*",
			ServiceType:   "local",
			Attributes:    map[string]string{},
		},
		reservoir: reservoir{
			capacity: 100,
			mu:       &sync.RWMutex{},
		},
		samplingStatistics: &samplingStatistics{},
	}

	require.Len(t, m.Rules, 3)

	// Assert on sorting order
	assert.Equal(t, r2, m.Rules[0])
	assert.Equal(t, r3, m.Rules[1])
	assert.Equal(t, r1, m.Rules[2])
}

// assert that rule with no ServiceName updates manifest successfully with empty values.
func TestRefreshManifestMissingServiceName(t *testing.T) {
	ctx := context.Background()

	// rule with no ServiceName
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
        "ServiceType": "*",
        "URLPath": "*",
        "Version": 1
      }
    }
  ]
}`)

	// generate a test server so we can capture and inspect the request
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		_, err := res.Write(body)
		require.NoError(t, err)

	}))
	t.Cleanup(testServer.Close)

	client, err := createTestClient(testServer.URL)
	require.NoError(t, err)

	m := &Manifest{
		Rules:      []Rule{},
		xrayClient: client,
		clock:      &defaultClock{},
	}

	err = m.RefreshManifestRules(ctx)
	require.NoError(t, err)

	// assert on rule gets added
	assert.Len(t, m.Rules, 1)
}

// assert that rule with no RuleName does not update to the manifest.
func TestRefreshManifestMissingRuleName(t *testing.T) {
	ctx := context.Background()

	// rule with no RuleName
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
        "ServiceName": "test",
        "URLPath": "*",
        "Version": 1
      }
    }
  ]
}`)

	// generate a test server so we can capture and inspect the request
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		_, err := res.Write(body)
		require.NoError(t, err)

	}))
	t.Cleanup(testServer.Close)

	client, err := createTestClient(testServer.URL)
	require.NoError(t, err)

	m := &Manifest{
		Rules:      []Rule{},
		xrayClient: client,
		clock:      &defaultClock{},
		logger:     testr.New(t),
	}

	err = m.RefreshManifestRules(ctx)
	require.NoError(t, err)

	// assert on rule not added
	assert.Len(t, m.Rules, 0)
}

// assert that rule with version greater than one does not update to the manifest.
func TestRefreshManifestIncorrectVersion(t *testing.T) {
	ctx := context.Background()

	// rule with Version 5
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
		"ServiceName": "test",
        "ServiceType": "*",
        "URLPath": "*",
        "Version": 5
      }
    }
  ]
}`)

	// generate a test server so we can capture and inspect the request
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		_, err := res.Write(body)
		require.NoError(t, err)

	}))
	t.Cleanup(testServer.Close)

	client, err := createTestClient(testServer.URL)
	require.NoError(t, err)

	m := &Manifest{
		Rules:      []Rule{},
		xrayClient: client,
		clock:      &defaultClock{},
		logger:     testr.New(t),
	}

	err = m.RefreshManifestRules(ctx)
	require.NoError(t, err)

	// assert on rule not added
	assert.Len(t, m.Rules, 0)
}

// assert that 1 valid and 1 invalid rule update only valid rule gets stored to the manifest.
func TestRefreshManifestAddOneInvalidRule(t *testing.T) {
	ctx := context.Background()

	// RuleName is missing from r2
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
        "ServiceName": "*",
        "ServiceType": "*",
        "URLPath": "*",
        "Version": 1
      }
    }
  ]
}`)

	r1 := Rule{
		ruleProperties: ruleProperties{
			RuleName:      "r1",
			Priority:      10000,
			Host:          "*",
			HTTPMethod:    "*",
			URLPath:       "*",
			ReservoirSize: 60,
			FixedRate:     0.5,
			Version:       1,
			ServiceName:   "*",
			ResourceARN:   "*",
			ServiceType:   "*",
			Attributes:    map[string]string{},
		},
		reservoir: reservoir{
			capacity: 60,
			mu:       &sync.RWMutex{},
		},
		samplingStatistics: &samplingStatistics{},
	}

	// generate a test server so we can capture and inspect the request
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		_, err := res.Write(body)
		require.NoError(t, err)
	}))
	defer testServer.Close()

	u, err := url.Parse(testServer.URL)
	require.NoError(t, err)

	client, err := newClient(*u)
	require.NoError(t, err)

	m := &Manifest{
		Rules:      []Rule{},
		xrayClient: client,
		clock:      &defaultClock{},
		logger:     testr.New(t),
	}

	err = m.RefreshManifestRules(ctx)
	require.NoError(t, err)

	assert.Len(t, m.Rules, 1)

	// assert on r1
	assert.Equal(t, r1, m.Rules[0])
}

// assert that inactive rule so return early without doing getSamplingTargets call
func TestRefreshManifestTargetNoSnapShot(t *testing.T) {
	clock := &mockClock{
		nowTime: 15000000,
	}

	r1 := Rule{
		ruleProperties: ruleProperties{
			RuleName:      "r3",
			Priority:      100,
			Host:          "*",
			HTTPMethod:    "*",
			URLPath:       "*",
			ReservoirSize: 100,
			FixedRate:     0.09,
			Version:       1,
			ServiceName:   "*",
			ResourceARN:   "*",
			ServiceType:   "local",
			Attributes:    map[string]string{},
		},
		reservoir: reservoir{
			capacity: 100,
		},
		samplingStatistics: &samplingStatistics{
			matchedRequests: int64(0),
		},
	}

	rules := []Rule{r1}

	m := &Manifest{
		Rules:  rules,
		clock:  clock,
		logger: testr.New(t),
	}

	refresh, err := m.RefreshManifestTargets(context.Background())
	assert.False(t, refresh)
	assert.NoError(t, err)
}

// assert that refresh manifest targets successfully updates reservoir value for a rule.
func TestRefreshManifestTargets(t *testing.T) {
	// RuleName is missing from r2
	body := []byte(`{
   "LastRuleModification": 17000000,
   "SamplingTargetDocuments": [ 
      { 
         "FixedRate": 0.06,
         "Interval": 25,
         "ReservoirQuota": 23,
         "ReservoirQuotaTTL": 15000000,
         "RuleName": "r1"
      }
   ],
   "UnprocessedStatistics": [ 
      { 
         "ErrorCode": "200",
         "Message": "Ok",
         "RuleName": "r1"
      }
   ]
}`)

	clock := &mockClock{
		nowTime: 150,
	}

	r1 := Rule{
		ruleProperties: ruleProperties{
			RuleName:      "r1",
			Priority:      100,
			Host:          "*",
			HTTPMethod:    "*",
			URLPath:       "*",
			ReservoirSize: 100,
			FixedRate:     0.09,
			Version:       1,
			ServiceName:   "*",
			ResourceARN:   "*",
			ServiceType:   "local",
			Attributes:    map[string]string{},
		},
		reservoir: reservoir{
			capacity: 100,
			mu:       &sync.RWMutex{},
		},
		samplingStatistics: &samplingStatistics{
			matchedRequests: int64(5),
		},
	}

	rules := []Rule{r1}

	// generate a test server so we can capture and inspect the request
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		_, err := res.Write(body)
		require.NoError(t, err)
	}))
	t.Cleanup(testServer.Close)

	client, err := createTestClient(testServer.URL)
	require.NoError(t, err)

	refreshedAt := time.Unix(18000000, 0)
	m := &Manifest{
		Rules:       rules,
		clock:       clock,
		logger:      testr.New(t),
		xrayClient:  client,
		refreshedAt: refreshedAt,
	}

	refresh, err := m.RefreshManifestTargets(context.Background())
	assert.False(t, refresh)
	require.NoError(t, err)

	// assert target updates
	require.Len(t, m.Rules, 1)
	assert.Equal(t, m.Rules[0].ruleProperties.FixedRate, 0.06)
	assert.Equal(t, m.Rules[0].reservoir.quota, 23.0)
	assert.Equal(t, m.Rules[0].reservoir.expiresAt, time.Unix(15000000, 0))
	assert.Equal(t, m.Rules[0].reservoir.interval, time.Duration(25))
}

// assert that refresh manifest targets successfully updates samplingTargetsPollingInterval.
func TestRefreshManifestTargetsPollIntervalUpdateTest(t *testing.T) {
	body := []byte(`{
   "LastRuleModification": 17000000,
   "SamplingTargetDocuments": [ 
      { 
         "FixedRate": 0.06,
         "Interval": 15,
         "ReservoirQuota": 23,
         "ReservoirQuotaTTL": 15000000,
         "RuleName": "r1"
      },
	  { 
         "FixedRate": 0.06,
         "Interval": 5,
         "ReservoirQuota": 23,
         "ReservoirQuotaTTL": 15000000,
         "RuleName": "r2"
      },
      { 
         "FixedRate": 0.06,
         "Interval": 25,
         "ReservoirQuota": 23,
         "ReservoirQuotaTTL": 15000000,
         "RuleName": "r3"
      }
   ],
   "UnprocessedStatistics": [ 
      { 
         "ErrorCode": "200",
         "Message": "Ok",
         "RuleName": "r3"
      }
   ]
}`)

	clock := &mockClock{
		nowTime: 150,
	}

	r1 := Rule{
		ruleProperties: ruleProperties{
			RuleName: "r1",
		},
		reservoir: reservoir{
			mu: &sync.RWMutex{},
		},
		samplingStatistics: &samplingStatistics{
			matchedRequests: int64(5),
		},
	}

	r2 := Rule{
		ruleProperties: ruleProperties{
			RuleName: "r2",
		},
		reservoir: reservoir{
			mu: &sync.RWMutex{},
		},
		samplingStatistics: &samplingStatistics{},
	}

	r3 := Rule{
		ruleProperties: ruleProperties{
			RuleName: "r3",
		},
		reservoir: reservoir{
			mu: &sync.RWMutex{},
		},
		samplingStatistics: &samplingStatistics{},
	}

	rules := []Rule{r1, r2, r3}

	// generate a test server so we can capture and inspect the request
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		_, err := res.Write(body)
		require.NoError(t, err)
	}))
	t.Cleanup(testServer.Close)

	client, err := createTestClient(testServer.URL)
	require.NoError(t, err)

	refreshedAt := time.Unix(18000000, 0)

	m := &Manifest{
		Rules:       rules,
		clock:       clock,
		logger:      testr.New(t),
		xrayClient:  client,
		refreshedAt: refreshedAt,
	}

	_, err = m.RefreshManifestTargets(context.Background())
	require.NoError(t, err)

	// assert that sampling rules polling interval is minimum of all target intervals min(15, 5, 25)
	assert.Equal(t, 5*time.Second, m.SamplingTargetsPollingInterval)
}

// assert that a valid sampling target updates its rule.
func TestUpdateTargets(t *testing.T) {
	clock := &mockClock{
		nowTime: 1500000000,
	}

	// sampling target received from centralized sampling backend
	rate := 0.05
	quota := float64(10)
	ttl := float64(1500000060)
	name := "r1"

	st := samplingTargetDocument{
		FixedRate:         &rate,
		ReservoirQuota:    &quota,
		ReservoirQuotaTTL: &ttl,
		RuleName:          &name,
	}

	targets := &getSamplingTargetsOutput{
		SamplingTargetDocuments: []*samplingTargetDocument{&st},
	}

	refreshedAt1 := time.Unix(1499999990, 0)
	// sampling rule about to be updated with new target
	r1 := Rule{
		ruleProperties: ruleProperties{
			RuleName:  "r1",
			FixedRate: 0.10,
		},
		reservoir: reservoir{
			quota:       8,
			refreshedAt: refreshedAt1,
			expiresAt:   time.Unix(1500000010, 0),
			capacity:    50,
		},
	}

	rules := []Rule{r1}

	m := &Manifest{
		Rules: rules,
		clock: clock,
	}

	refresh, err := m.updateTargets(targets)
	require.NoError(t, err)

	// assert refresh is false
	assert.False(t, refresh)

	refreshedAt2 := time.Unix(1500000000, 0)
	// Updated sampling rule
	exp := Rule{
		ruleProperties: ruleProperties{
			RuleName:  "r1",
			FixedRate: 0.05,
		},
		reservoir: reservoir{
			quota:       10,
			refreshedAt: refreshedAt2,
			expiresAt:   time.Unix(1500000060, 0),
			capacity:    50,
		},
	}

	// assert that updated the rule targets of rule r1
	assert.Equal(t, exp, m.Rules[0])
}

// assert that when last rule modification time is greater than manifest refresh time we need to update manifest
// out of band (async).
func TestUpdateTargetsRefreshFlagTest(t *testing.T) {
	clock := &mockClock{
		nowTime: 1500000000,
	}

	// sampling target received from centralized sampling backend
	rate := 0.05
	quota := float64(10)
	ttl := float64(1500000060)
	name := "r1"

	st := samplingTargetDocument{
		FixedRate:         &rate,
		ReservoirQuota:    &quota,
		ReservoirQuotaTTL: &ttl,
		RuleName:          &name,
	}

	targetLastRuleModifiedTime := float64(1500000020)
	targets := &getSamplingTargetsOutput{
		SamplingTargetDocuments: []*samplingTargetDocument{&st},
		LastRuleModification:    &targetLastRuleModifiedTime,
	}

	refreshedAt1 := time.Unix(1499999990, 0)
	// sampling rule about to be updated with new target
	r1 := Rule{
		ruleProperties: ruleProperties{
			RuleName:  "r1",
			FixedRate: 0.10,
		},
		reservoir: reservoir{
			quota:       8,
			refreshedAt: refreshedAt1,
			expiresAt:   time.Unix(1500000010, 0),
			capacity:    50,
		},
	}

	rules := []Rule{r1}

	m := &Manifest{
		Rules:       rules,
		refreshedAt: clock.now(),
		clock:       clock,
	}

	refresh, err := m.updateTargets(targets)
	require.NoError(t, err)

	// assert refresh is false
	assert.True(t, refresh)

	refreshedAt2 := time.Unix(1500000000, 0)
	// Updated sampling rule
	exp := Rule{
		ruleProperties: ruleProperties{
			RuleName:  "r1",
			FixedRate: 0.05,
		},
		reservoir: reservoir{
			quota:       10,
			refreshedAt: refreshedAt2,
			expiresAt:   time.Unix(1500000060, 0),
			capacity:    50,
		},
	}

	// assert that updated the rule targets of rule r1
	assert.Equal(t, exp, m.Rules[0])
}

// unprocessed statistics error code is 5xx then updateTargets returns an error, if 4xx refresh flag set to true.
func TestUpdateTargetsUnprocessedStatistics(t *testing.T) {
	clock := &mockClock{
		nowTime: 1500000000,
	}

	// sampling target received from centralized sampling backend
	rate := 0.05
	quota := float64(10)
	ttl := float64(1500000060)
	name := "r1"

	st := samplingTargetDocument{
		FixedRate:         &rate,
		ReservoirQuota:    &quota,
		ReservoirQuotaTTL: &ttl,
		RuleName:          &name,
	}

	// case for 5xx
	errorCode500 := "500"
	unprocessedStats5xx := unprocessedStatistic{
		ErrorCode: &errorCode500,
		RuleName:  &name,
	}

	targets5xx := &getSamplingTargetsOutput{
		SamplingTargetDocuments: []*samplingTargetDocument{&st},
		UnprocessedStatistics:   []*unprocessedStatistic{&unprocessedStats5xx},
	}

	m := &Manifest{
		clock:  clock,
		logger: testr.New(t),
	}

	refresh, err := m.updateTargets(targets5xx)
	// assert error happened since unprocessed stats has returned 5xx error code
	require.Error(t, err)

	// assert refresh is false
	assert.False(t, refresh)

	// case for 4xx
	errorCode400 := "400"
	unprocessedStats4xx := unprocessedStatistic{
		ErrorCode: &errorCode400,
		RuleName:  &name,
	}

	targets4xx := &getSamplingTargetsOutput{
		SamplingTargetDocuments: []*samplingTargetDocument{&st},
		UnprocessedStatistics:   []*unprocessedStatistic{&unprocessedStats4xx},
	}

	refresh, err = m.updateTargets(targets4xx)
	// assert that no error happened since unprocessed stats has returned 4xx error code
	require.NoError(t, err)

	// assert refresh is true
	assert.True(t, refresh)

	// case when rule error code is unknown do not set any flag
	unprocessedStats := unprocessedStatistic{
		ErrorCode: nil,
		RuleName:  nil,
	}

	targets := &getSamplingTargetsOutput{
		SamplingTargetDocuments: []*samplingTargetDocument{&st},
		UnprocessedStatistics:   []*unprocessedStatistic{&unprocessedStats},
	}

	m = &Manifest{
		clock:  clock,
		logger: testr.New(t),
	}

	refresh, err = m.updateTargets(targets)
	require.NoError(t, err)

	// assert refresh is false
	assert.False(t, refresh)

}

// assert that a missing sampling rule in manifest does not update it's reservoir values.
func TestUpdateReservoir(t *testing.T) {
	// Sampling target received from centralized sampling backend
	rate := 0.05
	quota := float64(10)
	ttl := float64(1500000060)
	name := "r1"
	st := &samplingTargetDocument{
		FixedRate:         &rate,
		ReservoirQuota:    &quota,
		ReservoirQuotaTTL: &ttl,
		RuleName:          &name,
	}

	refreshedAt1 := time.Unix(1499999990, 0)
	// manifest only has rule r2 but not rule with r1 which targets just received
	r1 := Rule{
		ruleProperties: ruleProperties{
			RuleName:  "r2",
			FixedRate: 0.10,
		},
		reservoir: reservoir{
			quota:       8,
			refreshedAt: refreshedAt1,
			expiresAt:   time.Unix(1500000010, 0),
			capacity:    50,
		},
	}

	rules := []Rule{r1}

	m := &Manifest{
		Rules: rules,
	}

	err := m.updateReservoir(st)
	require.NoError(t, err)

	// assert that rule reservoir value does not get updated and still same as r1
	assert.Equal(t, m.Rules[0], r1)
}

// assert that a sampling target with missing Fixed Rate returns an error.
func TestUpdateReservoirMissingFixedRate(t *testing.T) {
	// Sampling target received from centralized sampling backend
	quota := float64(10)
	ttl := float64(1500000060)
	name := "r1"
	st := &samplingTargetDocument{
		ReservoirQuota:    &quota,
		ReservoirQuotaTTL: &ttl,
		RuleName:          &name,
	}

	refreshedAt1 := time.Unix(1499999990, 0)
	// manifest rule which we're trying to update with above target st
	r1 := Rule{
		ruleProperties: ruleProperties{
			RuleName:  "r2",
			FixedRate: 0.10,
		},
		reservoir: reservoir{
			quota:       8,
			refreshedAt: refreshedAt1,
			expiresAt:   time.Unix(1500000010, 0),
			capacity:    50,
		},
	}

	rules := []Rule{r1}

	m := &Manifest{
		Rules: rules,
	}

	err := m.updateReservoir(st)
	require.Error(t, err)
}

// assert that a sampling target with missing Rule Name returns an error.
func TestUpdateReservoirMissingRuleName(t *testing.T) {
	// Sampling target received from centralized sampling backend
	rate := 0.05
	quota := float64(10)
	ttl := float64(1500000060)
	st := &samplingTargetDocument{
		ReservoirQuota:    &quota,
		ReservoirQuotaTTL: &ttl,
		FixedRate:         &rate,
	}

	refreshedAt1 := time.Unix(1499999990, 0)
	// manifest rule which we're trying to update with above target st
	r1 := Rule{
		ruleProperties: ruleProperties{
			RuleName:  "r2",
			FixedRate: 0.10,
		},
		reservoir: reservoir{
			quota:       8,
			refreshedAt: refreshedAt1,
			expiresAt:   time.Unix(1500000010, 0),
			capacity:    50,
		},
	}

	rules := []Rule{r1}

	m := &Manifest{
		Rules: rules,
	}

	err := m.updateReservoir(st)
	require.Error(t, err)
}

// assert that snapshots returns an array of valid sampling statistics.
func TestSnapshots(t *testing.T) {
	clock := &mockClock{
		nowTime: 1500000000,
	}

	time1 := clock.now().Unix()

	name1 := "r1"
	requests1 := int64(1000)
	sampled1 := int64(100)
	borrowed1 := int64(5)
	r1 := Rule{
		ruleProperties: ruleProperties{
			RuleName: name1,
		},
		reservoir: reservoir{
			interval: 10,
		},
		samplingStatistics: &samplingStatistics{
			matchedRequests:  requests1,
			sampledRequests:  sampled1,
			borrowedRequests: borrowed1,
		},
	}

	name2 := "r2"
	requests2 := int64(500)
	sampled2 := int64(10)
	borrowed2 := int64(0)
	r2 := Rule{
		ruleProperties: ruleProperties{
			RuleName: name2,
		},
		reservoir: reservoir{
			interval: 10,
		},
		samplingStatistics: &samplingStatistics{
			matchedRequests:  requests2,
			sampledRequests:  sampled2,
			borrowedRequests: borrowed2,
		},
	}

	rules := []Rule{r1, r2}

	id := "c1"
	m := &Manifest{
		Rules:    rules,
		clientID: &id,
		clock:    clock,
	}

	// Expected SamplingStatistics structs
	ss1 := samplingStatisticsDocument{
		ClientID:     &id,
		RequestCount: &requests1,
		RuleName:     &name1,
		SampledCount: &sampled1,
		BorrowCount:  &borrowed1,
		Timestamp:    &time1,
	}

	ss2 := samplingStatisticsDocument{
		ClientID:     &id,
		RequestCount: &requests2,
		RuleName:     &name2,
		SampledCount: &sampled2,
		BorrowCount:  &borrowed2,
		Timestamp:    &time1,
	}

	statistics, err := m.snapshots()
	require.NoError(t, err)

	// match time
	*statistics[0].Timestamp = 1500000000
	*statistics[1].Timestamp = 1500000000

	assert.Equal(t, ss1, *statistics[0])
	assert.Equal(t, ss2, *statistics[1])
}

// assert that fresh and inactive rules are not included in a snapshot.
func TestMixedSnapshots(t *testing.T) {
	clock := &mockClock{
		nowTime: 1500000000,
	}

	id := "c1"
	time1 := clock.now().Unix()

	// stale and active rule
	name1 := "r1"
	requests1 := int64(1000)
	sampled1 := int64(100)
	borrowed1 := int64(5)

	refreshedAt1 := time.Unix(1499999970, 0)
	r1 := Rule{
		ruleProperties: ruleProperties{
			RuleName: name1,
		},
		reservoir: reservoir{
			interval:    20,
			refreshedAt: refreshedAt1,
		},
		samplingStatistics: &samplingStatistics{
			matchedRequests:  requests1,
			sampledRequests:  sampled1,
			borrowedRequests: borrowed1,
		},
	}

	refreshedAt2 := time.Unix(1499999990, 0)
	// fresh and inactive rule
	name2 := "r2"
	requests2 := int64(0)
	sampled2 := int64(0)
	borrowed2 := int64(0)

	r2 := Rule{
		ruleProperties: ruleProperties{
			RuleName: name2,
		},
		reservoir: reservoir{
			interval:    20,
			refreshedAt: refreshedAt2,
		},
		samplingStatistics: &samplingStatistics{
			matchedRequests:  requests2,
			sampledRequests:  sampled2,
			borrowedRequests: borrowed2,
		},
	}

	refreshedAt3 := time.Unix(1499999990, 0)
	// fresh rule
	name3 := "r3"
	requests3 := int64(1000)
	sampled3 := int64(100)
	borrowed3 := int64(5)

	r3 := Rule{
		ruleProperties: ruleProperties{
			RuleName: name3,
		},
		reservoir: reservoir{
			interval:    20,
			refreshedAt: refreshedAt3,
		},
		samplingStatistics: &samplingStatistics{
			matchedRequests:  requests3,
			sampledRequests:  sampled3,
			borrowedRequests: borrowed3,
		},
	}

	rules := []Rule{r1, r2, r3}

	m := &Manifest{
		clientID: &id,
		clock:    clock,
		Rules:    rules,
	}

	ss1 := samplingStatisticsDocument{
		ClientID:     &id,
		RequestCount: &requests1,
		RuleName:     &name1,
		SampledCount: &sampled1,
		BorrowCount:  &borrowed1,
		Timestamp:    &time1,
	}

	statistics, err := m.snapshots()
	require.NoError(t, err)

	// assert that only inactive rules are added to the statistics
	require.Len(t, statistics, 1)
	assert.Equal(t, ss1, *statistics[0])
}

// Assert that sorting an unsorted array results in a sorted array - check priority.
func TestSortBasedOnPriority(t *testing.T) {
	r1 := Rule{
		ruleProperties: ruleProperties{
			RuleName: "r1",
			Priority: 5,
		},
	}

	r2 := Rule{
		ruleProperties: ruleProperties{
			RuleName: "r2",
			Priority: 6,
		},
	}

	r3 := Rule{
		ruleProperties: ruleProperties{
			RuleName: "r3",
			Priority: 7,
		},
	}

	// Unsorted rules array
	rules := []Rule{r2, r1, r3}

	m := &Manifest{
		Rules: rules,
	}

	// Sort array
	m.sort()

	// Assert on order
	assert.Equal(t, r1, m.Rules[0])
	assert.Equal(t, r2, m.Rules[1])
	assert.Equal(t, r3, m.Rules[2])
}

// Assert that sorting an unsorted array results in a sorted array - check priority and rule name.
func TestSortBasedOnRuleName(t *testing.T) {
	r1 := Rule{
		ruleProperties: ruleProperties{
			RuleName: "r1",
			Priority: 5,
		},
	}

	r2 := Rule{
		ruleProperties: ruleProperties{
			RuleName: "r2",
			Priority: 5,
		},
	}

	r3 := Rule{
		ruleProperties: ruleProperties{
			RuleName: "r3",
			Priority: 7,
		},
	}

	// Unsorted rules array
	rules := []Rule{r2, r1, r3}

	m := &Manifest{
		Rules: rules,
	}

	// Sort array
	m.sort()

	// Assert on order
	assert.Equal(t, r1, m.Rules[0])
	assert.Equal(t, r2, m.Rules[1])
	assert.Equal(t, r3, m.Rules[2])
}

// asserts the minimum value of all the targets.
func TestMinPollInterval(t *testing.T) {
	r1 := Rule{reservoir: reservoir{interval: time.Duration(10)}}
	r2 := Rule{reservoir: reservoir{interval: time.Duration(5)}}
	r3 := Rule{reservoir: reservoir{interval: time.Duration(25)}}

	rules := []Rule{r1, r2, r3}
	m := &Manifest{Rules: rules}

	minPoll := m.minimumPollingInterval()

	assert.Equal(t, 5*time.Second, minPoll)
}

// asserts the minimum value of all the targets when some targets has 0 interval.
func TestMinPollIntervalZeroCase(t *testing.T) {
	r1 := Rule{reservoir: reservoir{interval: time.Duration(0)}}
	r2 := Rule{reservoir: reservoir{interval: time.Duration(0)}}
	r3 := Rule{reservoir: reservoir{interval: time.Duration(5)}}

	rules := []Rule{r1, r2, r3}
	m := &Manifest{Rules: rules}

	minPoll := m.minimumPollingInterval()

	assert.Equal(t, 0*time.Second, minPoll)
}

// asserts the minimum value of all the targets when some targets has negative interval.
func TestMinPollIntervalNegativeCase(t *testing.T) {
	r1 := Rule{reservoir: reservoir{interval: time.Duration(-5)}}
	r2 := Rule{reservoir: reservoir{interval: time.Duration(0)}}
	r3 := Rule{reservoir: reservoir{interval: time.Duration(0)}}

	rules := []Rule{r1, r2, r3}
	m := &Manifest{Rules: rules}

	minPoll := m.minimumPollingInterval()

	assert.Equal(t, -5*time.Second, minPoll)
}

// asserts that manifest with empty rules return 0
func TestMinPollIntervalNoRules(t *testing.T) {
	var rules []Rule
	m := &Manifest{Rules: rules}

	minPoll := m.minimumPollingInterval()

	assert.Equal(t, 0*time.Second, minPoll)
}

// assert that able to successfully generate the client ID.
func TestGenerateClientID(t *testing.T) {
	clientID, err := generateClientID()
	require.NoError(t, err)
	assert.NotEmpty(t, clientID)
}
