// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestClient(t *testing.T, body []byte) *xrayClient {
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, _ *http.Request) {
		_, err := res.Write(body)
		assert.NoError(t, err)
	}))
	t.Cleanup(testServer.Close)

	u, err := url.Parse(testServer.URL)
	require.NoError(t, err)

	client, err := newClient(*u)
	require.NoError(t, err)

	return client
}

func TestGetSamplingRules(t *testing.T) {
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
        "RuleARN": "arn:aws:xray:us-west-2:xxxxxxx:sampling-rule/Default",
        "RuleName": "Default",
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
        "RuleARN": "arn:aws:xray:us-west-2:xxxxxxx:sampling-rule/test-rule",
        "RuleName": "test-rule",
        "ServiceName": "test-rule",
        "ServiceType": "local",
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
        "RuleARN": "arn:aws:xray:us-west-2:xxxxxxx:sampling-rule/test-rule-1",
        "RuleName": "test-rule-1",
        "ServiceName": "*",
        "ServiceType": "*",
        "URLPath": "*",
        "Version": 1
      }
    }
  ]
}`)
	ctx := context.Background()

	client := createTestClient(t, body)

	samplingRules, err := client.getSamplingRules(ctx)
	require.NoError(t, err)

	assert.Equal(t, "Default", samplingRules.SamplingRuleRecords[0].SamplingRule.RuleName)
	assert.Equal(t, "*", samplingRules.SamplingRuleRecords[0].SamplingRule.ServiceType)
	assert.Equal(t, "*", samplingRules.SamplingRuleRecords[0].SamplingRule.Host)
	assert.Equal(t, "*", samplingRules.SamplingRuleRecords[0].SamplingRule.URLPath)
	assert.Equal(t, 60.0, samplingRules.SamplingRuleRecords[0].SamplingRule.ReservoirSize)
	assert.Equal(t, 0.5, samplingRules.SamplingRuleRecords[0].SamplingRule.FixedRate)

	assert.Equal(t, "test-rule", samplingRules.SamplingRuleRecords[1].SamplingRule.RuleName)
	assert.Equal(t, "local", samplingRules.SamplingRuleRecords[1].SamplingRule.ServiceType)
	assert.Equal(t, "*", samplingRules.SamplingRuleRecords[1].SamplingRule.Host)
	assert.Equal(t, "/aws-sdk-call", samplingRules.SamplingRuleRecords[1].SamplingRule.URLPath)
	assert.Equal(t, 3.0, samplingRules.SamplingRuleRecords[1].SamplingRule.ReservoirSize)
	assert.Equal(t, 0.09, samplingRules.SamplingRuleRecords[1].SamplingRule.FixedRate)

	assert.Equal(t, "test-rule-1", samplingRules.SamplingRuleRecords[2].SamplingRule.RuleName)
	assert.Equal(t, "*", samplingRules.SamplingRuleRecords[2].SamplingRule.ServiceType)
	assert.Equal(t, "*", samplingRules.SamplingRuleRecords[2].SamplingRule.Host)
	assert.Equal(t, "*", samplingRules.SamplingRuleRecords[2].SamplingRule.URLPath)
	assert.Equal(t, 100.0, samplingRules.SamplingRuleRecords[2].SamplingRule.ReservoirSize)
	assert.Equal(t, 0.09, samplingRules.SamplingRuleRecords[2].SamplingRule.FixedRate)
}

func TestGetSamplingRulesWithMissingValues(t *testing.T) {
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
        "ResourceARN": "*",
        "RuleARN": "arn:aws:xray:us-west-2:xxxxxxx:sampling-rule/Default",
        "RuleName": "Default",
        "ServiceName": "*",
        "ServiceType": "*",
        "URLPath": "*",
        "Version": 1
      }
    }
  ]
}`)
	ctx := context.Background()

	client := createTestClient(t, body)

	samplingRules, err := client.getSamplingRules(ctx)
	require.NoError(t, err)

	// Priority and ReservoirSize are missing in API response so they are assigned as nil
	assert.Equal(t, int64(0), samplingRules.SamplingRuleRecords[0].SamplingRule.Priority)
	assert.Equal(t, 0.0, samplingRules.SamplingRuleRecords[0].SamplingRule.ReservoirSize)

	// other values are stored as expected
	assert.Equal(t, "Default", samplingRules.SamplingRuleRecords[0].SamplingRule.RuleName)
}

func TestGetSamplingTargets(t *testing.T) {
	body := []byte(`{
   "LastRuleModification": 123456,
   "SamplingTargetDocuments": [ 
      { 
         "FixedRate": 5,
         "Interval": 5,
         "ReservoirQuota": 3,
         "ReservoirQuotaTTL": 456789,
         "RuleName": "r1"
      }
   ],
   "UnprocessedStatistics": [ 
      { 
         "ErrorCode": "200",
         "Message": "ok",
         "RuleName": "r1"
      }
   ]
}`)

	ctx := context.Background()

	client := createTestClient(t, body)

	samplingTragets, err := client.getSamplingTargets(ctx, nil)
	require.NoError(t, err)

	assert.Equal(t, float64(123456), *samplingTragets.LastRuleModification)
	assert.Equal(t, float64(5), *samplingTragets.SamplingTargetDocuments[0].FixedRate)
	assert.Equal(t, int64(5), *samplingTragets.SamplingTargetDocuments[0].Interval)
	assert.Equal(t, 3.0, *samplingTragets.SamplingTargetDocuments[0].ReservoirQuota)
	assert.Equal(t, float64(456789), *samplingTragets.SamplingTargetDocuments[0].ReservoirQuotaTTL)
	assert.Equal(t, "r1", *samplingTragets.SamplingTargetDocuments[0].RuleName)
	assert.Equal(t, "r1", *samplingTragets.UnprocessedStatistics[0].RuleName)
	assert.Equal(t, "200", *samplingTragets.UnprocessedStatistics[0].ErrorCode)
	assert.Equal(t, "ok", *samplingTragets.UnprocessedStatistics[0].Message)
}

func TestGetSamplingTargetsMissingValues(t *testing.T) {
	body := []byte(`{
   "LastRuleModification": 123456,
   "SamplingTargetDocuments": [ 
      { 
         "FixedRate": 5,
         "ReservoirQuotaTTL": 456789,
         "RuleName": "r1"
      }
   ],
   "UnprocessedStatistics": [ 
      { 
         "ErrorCode": "200",
         "Message": "ok",
         "RuleName": "r1"
      }
   ]
}`)

	ctx := context.Background()

	client := createTestClient(t, body)

	samplingTragets, err := client.getSamplingTargets(ctx, nil)
	require.NoError(t, err)

	assert.Nil(t, samplingTragets.SamplingTargetDocuments[0].Interval)
	assert.Nil(t, samplingTragets.SamplingTargetDocuments[0].ReservoirQuota)
}

func TestNilContext(t *testing.T) {
	client := createTestClient(t, []byte(``))
	samplingRulesOutput, err := client.getSamplingRules(context.TODO())
	require.Error(t, err)
	require.Nil(t, samplingRulesOutput)

	samplingTargetsOutput, err := client.getSamplingTargets(context.TODO(), nil)
	require.Error(t, err)
	require.Nil(t, samplingTargetsOutput)
}

func TestNewClient(t *testing.T) {
	endpoint, err := url.Parse("http://127.0.0.1:2020")
	require.NoError(t, err)

	xrayClient, err := newClient(*endpoint)
	require.NoError(t, err)

	assert.Equal(t, "http://127.0.0.1:2020/GetSamplingRules", xrayClient.samplingRulesURL)
	assert.Equal(t, "http://127.0.0.1:2020/SamplingTargets", xrayClient.samplingTargetsURL)
}

func TestEndpointIsNotReachable(t *testing.T) {
	endpoint, err := url.Parse("http://127.0.0.1:2020")
	require.NoError(t, err)

	client, err := newClient(*endpoint)
	require.NoError(t, err)

	_, err = client.getSamplingRules(context.Background())
	assert.Error(t, err)
}
