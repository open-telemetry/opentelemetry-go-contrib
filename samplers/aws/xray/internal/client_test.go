// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestClient(t *testing.T, body []byte) *xrayClient {
	return createTestClientWithStatusCode(t, http.StatusOK, body)
}

func createTestClientWithStatusCode(t *testing.T, status int, body []byte) *xrayClient {
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, _ *http.Request) {
		res.WriteHeader(status)
		_, err := res.Write(body)
		require.NoError(t, err)
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

	assert.Equal(t, samplingRules.SamplingRuleRecords[0].SamplingRule.RuleName, "Default")
	assert.Equal(t, samplingRules.SamplingRuleRecords[0].SamplingRule.ServiceType, "*")
	assert.Equal(t, samplingRules.SamplingRuleRecords[0].SamplingRule.Host, "*")
	assert.Equal(t, samplingRules.SamplingRuleRecords[0].SamplingRule.URLPath, "*")
	assert.Equal(t, samplingRules.SamplingRuleRecords[0].SamplingRule.ReservoirSize, 60.0)
	assert.Equal(t, samplingRules.SamplingRuleRecords[0].SamplingRule.FixedRate, 0.5)

	assert.Equal(t, samplingRules.SamplingRuleRecords[1].SamplingRule.RuleName, "test-rule")
	assert.Equal(t, samplingRules.SamplingRuleRecords[1].SamplingRule.ServiceType, "local")
	assert.Equal(t, samplingRules.SamplingRuleRecords[1].SamplingRule.Host, "*")
	assert.Equal(t, samplingRules.SamplingRuleRecords[1].SamplingRule.URLPath, "/aws-sdk-call")
	assert.Equal(t, samplingRules.SamplingRuleRecords[1].SamplingRule.ReservoirSize, 3.0)
	assert.Equal(t, samplingRules.SamplingRuleRecords[1].SamplingRule.FixedRate, 0.09)

	assert.Equal(t, samplingRules.SamplingRuleRecords[2].SamplingRule.RuleName, "test-rule-1")
	assert.Equal(t, samplingRules.SamplingRuleRecords[2].SamplingRule.ServiceType, "*")
	assert.Equal(t, samplingRules.SamplingRuleRecords[2].SamplingRule.Host, "*")
	assert.Equal(t, samplingRules.SamplingRuleRecords[2].SamplingRule.URLPath, "*")
	assert.Equal(t, samplingRules.SamplingRuleRecords[2].SamplingRule.ReservoirSize, 100.0)
	assert.Equal(t, samplingRules.SamplingRuleRecords[2].SamplingRule.FixedRate, 0.09)
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
	assert.Equal(t, samplingRules.SamplingRuleRecords[0].SamplingRule.Priority, int64(0))
	assert.Equal(t, samplingRules.SamplingRuleRecords[0].SamplingRule.ReservoirSize, 0.0)

	// other values are stored as expected
	assert.Equal(t, samplingRules.SamplingRuleRecords[0].SamplingRule.RuleName, "Default")
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

	assert.Equal(t, *samplingTragets.LastRuleModification, float64(123456))
	assert.Equal(t, *samplingTragets.SamplingTargetDocuments[0].FixedRate, float64(5))
	assert.Equal(t, *samplingTragets.SamplingTargetDocuments[0].Interval, int64(5))
	assert.Equal(t, *samplingTragets.SamplingTargetDocuments[0].ReservoirQuota, 3.0)
	assert.Equal(t, *samplingTragets.SamplingTargetDocuments[0].ReservoirQuotaTTL, float64(456789))
	assert.Equal(t, *samplingTragets.SamplingTargetDocuments[0].RuleName, "r1")
	assert.Equal(t, *samplingTragets.UnprocessedStatistics[0].RuleName, "r1")
	assert.Equal(t, *samplingTragets.UnprocessedStatistics[0].ErrorCode, "200")
	assert.Equal(t, *samplingTragets.UnprocessedStatistics[0].Message, "ok")
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

	samplingTargets, err := client.getSamplingTargets(ctx, nil)
	require.NoError(t, err)

	assert.Nil(t, samplingTargets.SamplingTargetDocuments[0].Interval)
	assert.Nil(t, samplingTargets.SamplingTargetDocuments[0].ReservoirQuota)
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

	actualRules, err := client.getSamplingRules(context.Background())
	assert.Error(t, err)
	assert.ErrorContains(t, err, "xray client: unable to retrieve sampling rules, error on http request: ")
	assert.Nil(t, actualRules)

	actualTargets, err := client.getSamplingTargets(context.Background(), nil)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "xray client: unable to retrieve sampling targets, error on http request: ")
	assert.Nil(t, actualTargets)
}

func TestRespondsWithErrorStatusCode(t *testing.T) {
	client := createTestClientWithStatusCode(t, http.StatusForbidden, []byte("{}"))

	actualRules, err := client.getSamplingRules(context.Background())
	assert.Error(t, err)
	assert.EqualError(t, err, fmt.Sprintf("xray client: unable to retrieve sampling rules, expected response status code 200, got: %d", http.StatusForbidden))
	assert.Nil(t, actualRules)

	actualTargets, err := client.getSamplingTargets(context.Background(), nil)
	assert.Error(t, err)
	assert.EqualError(t, err, fmt.Sprintf("xray client: unable to retrieve sampling targets, expected response status code 200, got: %d", http.StatusForbidden))
	assert.Nil(t, actualTargets)
}

func TestInvalidResponseBody(t *testing.T) {
	type scenarios struct {
		name     string
		response string
	}
	for _, scenario := range []scenarios{
		{
			name:     "empty response",
			response: "",
		},
		{
			name:     "malformed json",
			response: "",
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			client := createTestClient(t, []byte(scenario.response))

			actualRules, err := client.getSamplingRules(context.TODO())

			assert.Error(t, err)
			assert.Nil(t, actualRules)
			assert.ErrorContains(t, err, "xray client: unable to retrieve sampling rules, unable to unmarshal the response body:"+scenario.response)

			actualTargets, err := client.getSamplingTargets(context.TODO(), nil)
			assert.Error(t, err)
			assert.Nil(t, actualTargets)
			assert.ErrorContains(t, err, "xray client: unable to retrieve sampling targets, unable to unmarshal the response body: "+scenario.response)
		})
	}
}
