// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
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

func createSamplingTargetDocument(name string, interval int64, rate, quota, ttl float64) *samplingTargetDocument { //nolint:unparam
	return &samplingTargetDocument{
		FixedRate:         &rate,
		Interval:          &interval,
		ReservoirQuota:    &quota,
		ReservoirQuotaTTL: &ttl,
		RuleName:          &name,
	}
}

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

	m := &Manifest{
		clock:       clock,
		refreshedAt: time.Unix(3700, 0),
	}

	assert.True(t, m.Expired())
}

// assert that if collector is not enabled at specified endpoint, returns an error.
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
		reservoir: &reservoir{
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
		reservoir: &reservoir{
			expiresAt: time.Unix(14050, 0),
		},
	}

	m := &Manifest{
		Rules: []Rule{r1, r2},
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
		reservoir: &reservoir{
			expiresAt: time.Unix(14050, 0),
		},
	}

	m := &Manifest{
		Rules: []Rule{r1},
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
		reservoir: &reservoir{
			expiresAt: time.Unix(14050, 0),
		},
	}

	m := &Manifest{
		Rules: []Rule{r1},
	}

	exp, match, err := m.MatchAgainstManifestRules(sdktrace.SamplingParameters{Attributes: commonLabels}, "test", "local")
	require.True(t, match)
	require.NoError(t, err)

	// assert that manifest rule r1 is a match
	assert.Nil(t, err)
	assert.Equal(t, *exp, r1)
}

// assert that when no known rule is match then returned rule is nil,
// matched flag is false.
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
		reservoir: &reservoir{
			expiresAt: time.Unix(14050, 0),
		},
	}

	m := &Manifest{
		Rules: []Rule{r1},
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

	m := &Manifest{
		Rules:      []Rule{},
		xrayClient: createTestClient(t, body),
		clock:      &defaultClock{},
	}

	err := m.RefreshManifestRules(ctx)
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
		reservoir: &reservoir{
			capacity: 60,
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
		reservoir: &reservoir{
			capacity: 3,
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
		reservoir: &reservoir{
			capacity: 100,
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

	m := &Manifest{
		Rules:      []Rule{},
		xrayClient: createTestClient(t, body),
		clock:      &defaultClock{},
	}

	err := m.RefreshManifestRules(ctx)
	require.NoError(t, err)

	// assert on rule gets added
	require.Len(t, m.Rules, 1)
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

	m := &Manifest{
		Rules:      []Rule{},
		xrayClient: createTestClient(t, body),
		clock:      &defaultClock{},
		logger:     testr.New(t),
	}

	err := m.RefreshManifestRules(ctx)
	require.NoError(t, err)

	// assert on rule not added
	require.Len(t, m.Rules, 0)
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

	m := &Manifest{
		Rules:      []Rule{},
		xrayClient: createTestClient(t, body),
		clock:      &defaultClock{},
		logger:     testr.New(t),
	}

	err := m.RefreshManifestRules(ctx)
	require.NoError(t, err)

	// assert on rule not added
	require.Len(t, m.Rules, 0)
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
		reservoir: &reservoir{
			capacity: 60,
		},
		samplingStatistics: &samplingStatistics{},
	}

	m := &Manifest{
		Rules:      []Rule{},
		xrayClient: createTestClient(t, body),
		clock:      &defaultClock{},
		logger:     testr.New(t),
	}

	err := m.RefreshManifestRules(ctx)
	require.NoError(t, err)

	require.Len(t, m.Rules, 1)

	// assert on r1
	assert.Equal(t, r1, m.Rules[0])
}

// assert that inactive rule so return early without doing getSamplingTargets call.
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
		reservoir: &reservoir{
			capacity: 100,
		},
		samplingStatistics: &samplingStatistics{
			matchedRequests: int64(0),
		},
	}

	m := &Manifest{
		Rules:  []Rule{r1},
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
		reservoir: &reservoir{
			capacity: 100,
		},
		samplingStatistics: &samplingStatistics{
			matchedRequests: int64(5),
		},
	}

	m := &Manifest{
		Rules:       []Rule{r1},
		clock:       clock,
		logger:      testr.New(t),
		xrayClient:  createTestClient(t, body),
		refreshedAt: time.Unix(18000000, 0),
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
		samplingStatistics: &samplingStatistics{
			matchedRequests: int64(5),
		},
		reservoir: &reservoir{},
	}

	r2 := Rule{
		ruleProperties: ruleProperties{
			RuleName: "r2",
		},
		samplingStatistics: &samplingStatistics{},
		reservoir:          &reservoir{},
	}

	r3 := Rule{
		ruleProperties: ruleProperties{
			RuleName: "r3",
		},
		samplingStatistics: &samplingStatistics{},
		reservoir:          &reservoir{},
	}

	m := &Manifest{
		Rules:       []Rule{r1, r2, r3},
		clock:       clock,
		logger:      testr.New(t),
		xrayClient:  createTestClient(t, body),
		refreshedAt: time.Unix(18000000, 0),
	}

	_, err := m.RefreshManifestTargets(context.Background())
	require.NoError(t, err)

	// assert that sampling rules polling interval is minimum of all target intervals min(15, 5, 25)
	assert.Equal(t, 5*time.Second, m.SamplingTargetsPollingInterval)
}

// assert that a valid sampling target updates its rule.
func TestUpdateTargets(t *testing.T) {
	targets := &getSamplingTargetsOutput{
		SamplingTargetDocuments: []*samplingTargetDocument{createSamplingTargetDocument("r1", 0, 0.05, 10, 1500000060)},
	}

	// sampling rule about to be updated with new target
	r1 := Rule{
		ruleProperties: ruleProperties{
			RuleName:  "r1",
			FixedRate: 0.10,
		},
		reservoir: &reservoir{
			quota:       8,
			refreshedAt: time.Unix(1499999990, 0),
			expiresAt:   time.Unix(1500000010, 0),
			capacity:    50,
		},
	}

	clock := &mockClock{
		nowTime: 1500000000,
	}

	m := &Manifest{
		Rules: []Rule{r1},
		clock: clock,
	}

	refresh, err := m.updateTargets(targets)
	require.NoError(t, err)

	// assert refresh is false
	assert.False(t, refresh)

	// Updated sampling rule
	exp := Rule{
		ruleProperties: ruleProperties{
			RuleName:  "r1",
			FixedRate: 0.05,
		},
		reservoir: &reservoir{
			quota:       10,
			refreshedAt: time.Unix(1500000000, 0),
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
	targetLastRuleModifiedTime := float64(1500000020)
	targets := &getSamplingTargetsOutput{
		SamplingTargetDocuments: []*samplingTargetDocument{createSamplingTargetDocument("r1", 0, 0.05, 10, 1500000060)},
		LastRuleModification:    &targetLastRuleModifiedTime,
	}

	// sampling rule about to be updated with new target
	r1 := Rule{
		ruleProperties: ruleProperties{
			RuleName:  "r1",
			FixedRate: 0.10,
		},
		reservoir: &reservoir{
			quota:       8,
			refreshedAt: time.Unix(1499999990, 0),
			expiresAt:   time.Unix(1500000010, 0),
			capacity:    50,
		},
	}

	clock := &mockClock{
		nowTime: 1500000000,
	}

	m := &Manifest{
		Rules:       []Rule{r1},
		refreshedAt: clock.now(),
		clock:       clock,
	}

	refresh, err := m.updateTargets(targets)
	require.NoError(t, err)

	// assert refresh is false
	assert.True(t, refresh)

	// Updated sampling rule
	exp := Rule{
		ruleProperties: ruleProperties{
			RuleName:  "r1",
			FixedRate: 0.05,
		},
		reservoir: &reservoir{
			quota:       10,
			refreshedAt: time.Unix(1500000000, 0),
			expiresAt:   time.Unix(1500000060, 0),
			capacity:    50,
		},
	}

	// assert that updated the rule targets of rule r1
	assert.Equal(t, exp, m.Rules[0])
}

// unprocessed statistics error code is 5xx then updateTargets returns an error, if 4xx refresh flag set to true.
func TestUpdateTargetsUnprocessedStatistics(t *testing.T) {
	// case for 5xx
	ruleName := "r1"
	errorCode500 := "500"
	unprocessedStats5xx := unprocessedStatistic{
		ErrorCode: &errorCode500,
		RuleName:  &ruleName,
	}

	targets5xx := &getSamplingTargetsOutput{
		SamplingTargetDocuments: []*samplingTargetDocument{createSamplingTargetDocument(ruleName, 0, 0.05, 10, 1500000060)},
		UnprocessedStatistics:   []*unprocessedStatistic{&unprocessedStats5xx},
	}

	clock := &mockClock{
		nowTime: 1500000000,
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
		RuleName:  &ruleName,
	}

	targets4xx := &getSamplingTargetsOutput{
		SamplingTargetDocuments: []*samplingTargetDocument{createSamplingTargetDocument(ruleName, 0, 0.05, 10, 1500000060)},
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
		SamplingTargetDocuments: []*samplingTargetDocument{createSamplingTargetDocument(ruleName, 0, 0.05, 10, 1500000060)},
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
	// manifest only has rule r2 but not rule with r1 which targets just received
	r1 := Rule{
		ruleProperties: ruleProperties{
			RuleName:  "r2",
			FixedRate: 0.10,
		},
		reservoir: &reservoir{
			quota:       8,
			refreshedAt: time.Unix(1499999990, 0),
			expiresAt:   time.Unix(1500000010, 0),
			capacity:    50,
		},
	}

	m := &Manifest{
		Rules: []Rule{r1},
	}

	err := m.updateReservoir(createSamplingTargetDocument("r1", 0, 0.05, 10, 1500000060))
	require.NoError(t, err)

	// assert that rule reservoir value does not get updated and still same as r1
	assert.Equal(t, m.Rules[0], r1)
}

// assert that a sampling target with missing Fixed Rate returns an error.
func TestUpdateReservoirMissingFixedRate(t *testing.T) {
	// manifest rule which we're trying to update with above target st
	r1 := Rule{
		ruleProperties: ruleProperties{
			RuleName:  "r2",
			FixedRate: 0.10,
		},
		reservoir: &reservoir{
			quota:       8,
			refreshedAt: time.Unix(1499999990, 0),
			expiresAt:   time.Unix(1500000010, 0),
			capacity:    50,
		},
	}

	m := &Manifest{
		Rules: []Rule{r1},
	}

	st := createSamplingTargetDocument("r1", 0, 0, 10, 1500000060)
	st.FixedRate = nil
	err := m.updateReservoir(st)
	require.Error(t, err)
}

// assert that a sampling target with missing Rule Name returns an error.
func TestUpdateReservoirMissingRuleName(t *testing.T) {
	// manifest rule which we're trying to update with above target st
	r1 := Rule{
		ruleProperties: ruleProperties{
			RuleName:  "r2",
			FixedRate: 0.10,
		},
		reservoir: &reservoir{
			quota:       8,
			refreshedAt: time.Unix(1499999990, 0),
			expiresAt:   time.Unix(1500000010, 0),
			capacity:    50,
		},
	}

	m := &Manifest{
		Rules: []Rule{r1},
	}

	st := createSamplingTargetDocument("r1", 0, 0, 10, 1500000060)
	st.RuleName = nil
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
		reservoir: &reservoir{
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
		reservoir: &reservoir{
			interval: 10,
		},
		samplingStatistics: &samplingStatistics{
			matchedRequests:  requests2,
			sampledRequests:  sampled2,
			borrowedRequests: borrowed2,
		},
	}

	id := "c1"
	m := &Manifest{
		Rules:    []Rule{r1, r2},
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

	r1 := Rule{
		ruleProperties: ruleProperties{
			RuleName: name1,
		},
		reservoir: &reservoir{
			interval:    20,
			refreshedAt: time.Unix(1499999970, 0),
		},
		samplingStatistics: &samplingStatistics{
			matchedRequests:  requests1,
			sampledRequests:  sampled1,
			borrowedRequests: borrowed1,
		},
	}

	// fresh and inactive rule
	name2 := "r2"
	requests2 := int64(0)
	sampled2 := int64(0)
	borrowed2 := int64(0)

	r2 := Rule{
		ruleProperties: ruleProperties{
			RuleName: name2,
		},
		reservoir: &reservoir{
			interval:    20,
			refreshedAt: time.Unix(1499999990, 0),
		},
		samplingStatistics: &samplingStatistics{
			matchedRequests:  requests2,
			sampledRequests:  sampled2,
			borrowedRequests: borrowed2,
		},
	}

	// fresh rule
	name3 := "r3"
	requests3 := int64(1000)
	sampled3 := int64(100)
	borrowed3 := int64(5)

	r3 := Rule{
		ruleProperties: ruleProperties{
			RuleName: name3,
		},
		reservoir: &reservoir{
			interval:    20,
			refreshedAt: time.Unix(1499999990, 0),
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

// assert that deep copy creates a new manifest object with new address space.
func TestDeepCopy(t *testing.T) {
	r1 := Rule{
		ruleProperties: ruleProperties{
			RuleName:      "r1",
			Priority:      100,
			Host:          "http://127.0.0.0.1:2020",
			HTTPMethod:    "POST",
			URLPath:       "/test",
			ReservoirSize: 100,
			FixedRate:     0.09,
			Version:       1,
			ServiceName:   "openTelemetry",
			ResourceARN:   "*",
			ServiceType:   "local",
			Attributes:    map[string]string{},
		},
		reservoir: &reservoir{
			capacity: 100,
		},
		samplingStatistics: &samplingStatistics{
			matchedRequests:  int64(5),
			borrowedRequests: int64(1),
			sampledRequests:  int64(3),
		},
	}

	r2 := Rule{
		ruleProperties: ruleProperties{
			RuleName:      "r2",
			Priority:      10,
			Host:          "http://127.0.0.0.1:2020",
			HTTPMethod:    "GET",
			URLPath:       "/test/path",
			ReservoirSize: 100,
			FixedRate:     0.09,
			Version:       1,
			ServiceName:   "x-ray",
			ResourceARN:   "*",
			ServiceType:   "local",
			Attributes:    map[string]string{},
		},
		reservoir: &reservoir{
			capacity: 100,
		},
		samplingStatistics: &samplingStatistics{
			matchedRequests:  int64(5),
			borrowedRequests: int64(1),
			sampledRequests:  int64(3),
		},
	}

	clock := &mockClock{
		nowTime: 1500000000,
	}

	m := &Manifest{
		Rules:                          []Rule{r1, r2},
		SamplingTargetsPollingInterval: 10 * time.Second,
		refreshedAt:                    time.Unix(1500000, 0),
		xrayClient:                     createTestClient(t, []byte(`hello world!`)),
		logger:                         testr.New(t),
		clock:                          clock,
	}

	manifest := m.deepCopy()

	require.Len(t, m.Rules, 2)
	require.Len(t, manifest.Rules, 2)

	assert.Equal(t, &m.xrayClient, &manifest.xrayClient)

	assert.NotSame(t, &m.clock, &manifest.clock)
	assert.NotSame(t, &m.refreshedAt, &manifest.refreshedAt)
	assert.NotSame(t, &m.SamplingTargetsPollingInterval, &manifest.SamplingTargetsPollingInterval)
	assert.NotSame(t, &m.logger, &manifest.logger)
	assert.NotSame(t, &m.mu, &manifest.mu)

	// rule properties has different address space in m and manifest
	assert.NotSame(t, &m.Rules[0].ruleProperties.RuleName, &manifest.Rules[0].ruleProperties.RuleName)
	assert.NotSame(t, &m.Rules[0].ruleProperties.ServiceName, &manifest.Rules[0].ruleProperties.RuleName)
	assert.NotSame(t, &m.Rules[0].ruleProperties.ServiceType, &manifest.Rules[0].ruleProperties.RuleName)
	assert.NotSame(t, &m.Rules[0].ruleProperties.Host, &manifest.Rules[0].ruleProperties.RuleName)
	assert.NotSame(t, &m.Rules[0].ruleProperties.HTTPMethod, &manifest.Rules[0].ruleProperties.RuleName)
	assert.NotSame(t, &m.Rules[0].ruleProperties.URLPath, &manifest.Rules[0].ruleProperties.RuleName)
	assert.NotSame(t, &m.Rules[0].ruleProperties.FixedRate, &manifest.Rules[0].ruleProperties.RuleName)
	assert.NotSame(t, &m.Rules[0].ruleProperties.ReservoirSize, &manifest.Rules[0].ruleProperties.RuleName)
	assert.NotSame(t, &m.Rules[0].ruleProperties.ResourceARN, &manifest.Rules[0].ruleProperties.RuleName)
	assert.NotSame(t, &m.Rules[0].ruleProperties.Priority, &manifest.Rules[0].ruleProperties.Priority)
	assert.NotSame(t, &m.Rules[0].ruleProperties.Version, &manifest.Rules[0].ruleProperties.Version)
	assert.NotSame(t, &m.Rules[0].ruleProperties.Attributes, &manifest.Rules[0].ruleProperties.Attributes)

	// reservoir has different address space in m and manifest
	assert.NotSame(t, &m.Rules[0].reservoir.refreshedAt, &manifest.Rules[0].reservoir.refreshedAt)
	assert.NotSame(t, &m.Rules[0].reservoir.expiresAt, &manifest.Rules[0].reservoir.expiresAt)
	assert.NotSame(t, &m.Rules[0].reservoir.lastTick, &manifest.Rules[0].reservoir.lastTick)
	assert.NotSame(t, &m.Rules[0].reservoir.interval, &manifest.Rules[0].reservoir.interval)
	assert.NotSame(t, &m.Rules[0].reservoir.capacity, &manifest.Rules[0].reservoir.capacity)
	assert.NotSame(t, &m.Rules[0].reservoir.quota, &manifest.Rules[0].reservoir.quota)
	assert.NotSame(t, &m.Rules[0].reservoir.quotaBalance, &manifest.Rules[0].reservoir.quotaBalance)

	// samplings statistics has same address space since it is a pointer
	assert.Equal(t, &m.Rules[0].samplingStatistics, &manifest.Rules[0].samplingStatistics)
}

// assert that sorting an unsorted array results in a sorted array - check priority.
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

// assert that sorting an unsorted array results in a sorted array - check priority and rule name.
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
	r1 := Rule{reservoir: &reservoir{interval: time.Duration(10)}}
	r2 := Rule{reservoir: &reservoir{interval: time.Duration(5)}}
	r3 := Rule{reservoir: &reservoir{interval: time.Duration(25)}}

	rules := []Rule{r1, r2, r3}
	m := &Manifest{Rules: rules}

	minPoll := m.minimumPollingInterval()

	assert.Equal(t, 5*time.Second, minPoll)
}

// asserts the minimum value of all the targets when some targets has 0 interval.
func TestMinPollIntervalZeroCase(t *testing.T) {
	r1 := Rule{reservoir: &reservoir{interval: time.Duration(0)}}
	r2 := Rule{reservoir: &reservoir{interval: time.Duration(0)}}
	r3 := Rule{reservoir: &reservoir{interval: time.Duration(5)}}

	rules := []Rule{r1, r2, r3}
	m := &Manifest{Rules: rules}

	minPoll := m.minimumPollingInterval()

	assert.Equal(t, 0*time.Second, minPoll)
}

// asserts the minimum value of all the targets when some targets has negative interval.
func TestMinPollIntervalNegativeCase(t *testing.T) {
	r1 := Rule{reservoir: &reservoir{interval: time.Duration(-5)}}
	r2 := Rule{reservoir: &reservoir{interval: time.Duration(0)}}
	r3 := Rule{reservoir: &reservoir{interval: time.Duration(0)}}

	rules := []Rule{r1, r2, r3}
	m := &Manifest{Rules: rules}

	minPoll := m.minimumPollingInterval()

	assert.Equal(t, -5*time.Second, minPoll)
}

// asserts that manifest with empty rules return 0.
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

// validate no data race is happening when updating rule properties in manifest while matching.
func TestUpdatingRulesWhileMatchingConcurrentSafe(t *testing.T) {
	// getSamplingRules response
	ruleRecords := samplingRuleRecords{
		SamplingRule: &ruleProperties{
			RuleName:      "r1",
			Priority:      10000,
			Host:          "localhost",
			HTTPMethod:    "*",
			URLPath:       "/test/path",
			ReservoirSize: 40,
			FixedRate:     0.9,
			Version:       1,
			ServiceName:   "helios",
			ResourceARN:   "*",
			ServiceType:   "*",
		},
	}

	s := &getSamplingRulesOutput{
		SamplingRuleRecords: []*samplingRuleRecords{&ruleRecords},
	}

	// existing rule in manifest
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
			ServiceName:   "test",
			ResourceARN:   "*",
			ServiceType:   "*",
		},
		reservoir: &reservoir{
			expiresAt: time.Unix(14050, 0),
		},
	}

	rules := []Rule{r1}

	clock := &mockClock{
		nowTime: 1500000000,
	}

	m := &Manifest{
		Rules: rules,
		clock: clock,
	}

	// async rule updates
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 100; i++ {
			m.updateRules(s)
			time.Sleep(time.Millisecond)
		}
	}()

	// matching logic
	for i := 0; i < 100; i++ {
		_, match, err := m.MatchAgainstManifestRules(sdktrace.SamplingParameters{}, "helios", "macos")
		require.NoError(t, err)
		require.False(t, match)
	}
	<-done
}

// validate no data race is happening when updating rule properties and rule targets in manifest while matching.
func TestUpdatingRulesAndTargetsWhileMatchingConcurrentSafe(t *testing.T) {
	// getSamplingRules response to update existing manifest rule
	ruleRecords := samplingRuleRecords{
		SamplingRule: &ruleProperties{
			RuleName:      "r1",
			Priority:      10000,
			Host:          "localhost",
			HTTPMethod:    "*",
			URLPath:       "/test/path",
			ReservoirSize: 40,
			FixedRate:     0.9,
			Version:       1,
			ServiceName:   "helios",
			ResourceARN:   "*",
			ServiceType:   "*",
		},
	}

	// existing rule already present in manifest
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
			ServiceName:   "test",
			ResourceARN:   "*",
			ServiceType:   "*",
		},
		reservoir: &reservoir{
			refreshedAt: time.Unix(13000000, 0),
		},
	}
	clock := &mockClock{
		nowTime: 1500000000,
	}

	rules := []Rule{r1}

	m := &Manifest{
		Rules: rules,
		clock: clock,
	}

	var wg sync.WaitGroup

	// async rule updates
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			m.updateRules(&getSamplingRulesOutput{
				SamplingRuleRecords: []*samplingRuleRecords{&ruleRecords},
			})
			time.Sleep(time.Millisecond)
		}
	}()

	// async target updates
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			manifest := m.deepCopy()

			err := manifest.updateReservoir(createSamplingTargetDocument("r1", 0, 0.05, 10, 13000000))
			require.NoError(t, err)
			time.Sleep(time.Millisecond)

			m.mu.Lock()
			m.Rules = manifest.Rules
			m.mu.Unlock()
		}
	}()

	// matching logic
	for i := 0; i < 100; i++ {
		_, match, err := m.MatchAgainstManifestRules(sdktrace.SamplingParameters{}, "helios", "macos")
		require.NoError(t, err)
		require.False(t, match)
		time.Sleep(time.Millisecond)
	}

	wg.Wait()
}

// Validate Rules are preserved when a rule is updated with the same ruleProperties.
func TestPreserveRulesWithSameRuleProperties(t *testing.T) {
	// getSamplingRules response to update existing manifest rule, with matching ruleProperties
	ruleRecords := samplingRuleRecords{
		SamplingRule: &ruleProperties{
			RuleName:      "r1",
			Priority:      10000,
			Host:          "localhost",
			HTTPMethod:    "*",
			URLPath:       "/test/path",
			ReservoirSize: 40,
			FixedRate:     0.9,
			Version:       1,
			ServiceName:   "helios",
			ResourceARN:   "*",
			ServiceType:   "*",
		},
	}

	// existing rule already present in manifest
	r1 := Rule{
		ruleProperties: ruleProperties{
			RuleName:      "r1",
			Priority:      10000,
			Host:          "localhost",
			HTTPMethod:    "*",
			URLPath:       "/test/path",
			ReservoirSize: 40,
			FixedRate:     0.9,
			Version:       1,
			ServiceName:   "helios",
			ResourceARN:   "*",
			ServiceType:   "*",
		},
		reservoir: &reservoir{
			capacity:     100,
			quota:        100,
			quotaBalance: 80,
			refreshedAt:  time.Unix(13000000, 0),
		},
		samplingStatistics: &samplingStatistics{
			matchedRequests:  500,
			sampledRequests:  10,
			borrowedRequests: 0,
		},
	}
	clock := &mockClock{
		nowTime: 1500000000,
	}

	rules := []Rule{r1}

	m := &Manifest{
		Rules: rules,
		clock: clock,
	}

	// Update rules
	m.updateRules(&getSamplingRulesOutput{
		SamplingRuleRecords: []*samplingRuleRecords{&ruleRecords},
	})

	require.Equal(t, r1.reservoir, m.Rules[0].reservoir)
	require.Equal(t, r1.samplingStatistics, m.Rules[0].samplingStatistics)
}

// Validate Rules are NOT preserved when a rule is updated with a different ruleProperties with the same RuleName.
func TestDoNotPreserveRulesWithDifferentRuleProperties(t *testing.T) {
	// getSamplingRules response to update existing manifest rule, with different ruleProperties
	ruleRecords := samplingRuleRecords{
		SamplingRule: &ruleProperties{
			RuleName:      "r1",
			Priority:      10000,
			Host:          "localhost",
			HTTPMethod:    "*",
			URLPath:       "/test/path",
			ReservoirSize: 40,
			FixedRate:     0.9,
			Version:       1,
			ServiceName:   "helios",
			ResourceARN:   "*",
			ServiceType:   "*",
		},
	}

	// existing rule already present in manifest
	r1 := Rule{
		ruleProperties: ruleProperties{
			RuleName:      "r1",
			Priority:      10001,
			Host:          "localhost",
			HTTPMethod:    "*",
			URLPath:       "/test/path",
			ReservoirSize: 40,
			FixedRate:     0.9,
			Version:       1,
			ServiceName:   "helios",
			ResourceARN:   "*",
			ServiceType:   "*",
		},
		reservoir: &reservoir{
			capacity:     100,
			quota:        100,
			quotaBalance: 80,
			refreshedAt:  time.Unix(13000000, 0),
		},
		samplingStatistics: &samplingStatistics{
			matchedRequests:  500,
			sampledRequests:  10,
			borrowedRequests: 0,
		},
	}
	clock := &mockClock{
		nowTime: 1500000000,
	}

	rules := []Rule{r1}

	m := &Manifest{
		Rules: rules,
		clock: clock,
	}

	// Update rules
	m.updateRules(&getSamplingRulesOutput{
		SamplingRuleRecords: []*samplingRuleRecords{&ruleRecords},
	})

	require.Equal(t, m.Rules[0].reservoir.quota, 0.0)
	require.Equal(t, m.Rules[0].reservoir.quotaBalance, 0.0)
	require.Equal(t, *m.Rules[0].samplingStatistics, samplingStatistics{
		matchedRequests:  0,
		sampledRequests:  0,
		borrowedRequests: 0,
	})
}

// validate no data race is when capturing sampling statistics in manifest while sampling.
func TestUpdatingSamplingStatisticsWhenSamplingConcurrentSafe(t *testing.T) {
	// existing rule already present in manifest
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
			ServiceName:   "test",
			ResourceARN:   "*",
			ServiceType:   "*",
		},
		reservoir: &reservoir{
			refreshedAt: time.Unix(15000000, 0),
		},
		samplingStatistics: &samplingStatistics{
			matchedRequests:  5,
			borrowedRequests: 0,
			sampledRequests:  0,
		},
	}
	clock := &mockClock{
		nowTime: 18000000,
	}

	rules := []Rule{r1}

	m := &Manifest{
		Rules: rules,
		clock: clock,
	}

	// async snapshot updates
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 100; i++ {
			manifest := m.deepCopy()

			_, err := manifest.snapshots()
			require.NoError(t, err)

			m.mu.Lock()
			m.Rules = manifest.Rules
			m.mu.Unlock()
			time.Sleep(time.Millisecond)
		}
	}()

	// sampling logic
	for i := 0; i < 100; i++ {
		_ = r1.Sample(sdktrace.SamplingParameters{}, time.Unix(clock.nowTime+int64(i), 0))
		time.Sleep(time.Millisecond)
	}
	<-done
}
