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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	// generate a test server so we can capture and inspect the request
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		_, err := res.Write([]byte(body))
		require.NoError(t, err)
	}))

	u, err := url.Parse(testServer.URL)
	require.NoError(t, err)

	client, err := newClient(u.Host)
	require.NoError(t, err)

	samplingRules, err := client.getSamplingRules(ctx)
	require.NoError(t, err)

	assert.Equal(t, samplingRules.SamplingRuleRecords[0].SamplingRule.RuleName, "Default")
	assert.Equal(t, samplingRules.SamplingRuleRecords[0].SamplingRule.ServiceType, "*")
	assert.Equal(t, samplingRules.SamplingRuleRecords[0].SamplingRule.Host, "*")
	assert.Equal(t, samplingRules.SamplingRuleRecords[0].SamplingRule.URLPath, "*")
	assert.Equal(t, samplingRules.SamplingRuleRecords[0].SamplingRule.ReservoirSize, int64(60))
	assert.Equal(t, samplingRules.SamplingRuleRecords[0].SamplingRule.FixedRate, 0.5)

	assert.Equal(t, samplingRules.SamplingRuleRecords[1].SamplingRule.RuleName, "test-rule")
	assert.Equal(t, samplingRules.SamplingRuleRecords[1].SamplingRule.ServiceType, "local")
	assert.Equal(t, samplingRules.SamplingRuleRecords[1].SamplingRule.Host, "*")
	assert.Equal(t, samplingRules.SamplingRuleRecords[1].SamplingRule.URLPath, "/aws-sdk-call")
	assert.Equal(t, samplingRules.SamplingRuleRecords[1].SamplingRule.ReservoirSize, int64(3))
	assert.Equal(t, samplingRules.SamplingRuleRecords[1].SamplingRule.FixedRate, 0.09)

	assert.Equal(t, samplingRules.SamplingRuleRecords[2].SamplingRule.RuleName, "test-rule-1")
	assert.Equal(t, samplingRules.SamplingRuleRecords[2].SamplingRule.ServiceType, "*")
	assert.Equal(t, samplingRules.SamplingRuleRecords[2].SamplingRule.Host, "*")
	assert.Equal(t, samplingRules.SamplingRuleRecords[2].SamplingRule.URLPath, "*")
	assert.Equal(t, samplingRules.SamplingRuleRecords[2].SamplingRule.ReservoirSize, int64(100))
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

	// generate a test server so we can capture and inspect the request
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		_, err := res.Write([]byte(body))
		require.NoError(t, err)
	}))
	defer testServer.Close()

	u, err := url.Parse(testServer.URL)
	require.NoError(t, err)

	client, err := newClient(u.Host)
	require.NoError(t, err)

	samplingRules, err := client.getSamplingRules(ctx)
	require.NoError(t, err)

	// Priority and ReservoirSize are missing in API response so they are assigned as nil
	assert.Equal(t, samplingRules.SamplingRuleRecords[0].SamplingRule.Priority, int64(0))
	assert.Equal(t, samplingRules.SamplingRuleRecords[0].SamplingRule.ReservoirSize, int64(0))

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

	// generate a test server so we can capture and inspect the request
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		_, err := res.Write([]byte(body))
		require.NoError(t, err)
	}))
	defer testServer.Close()

	u, err := url.Parse(testServer.URL)
	require.NoError(t, err)

	client, err := newClient(u.Host)
	require.NoError(t, err)

	samplingTragets, err := client.getSamplingTargets(ctx, nil)
	require.NoError(t, err)

	assert.Equal(t, *samplingTragets.LastRuleModification, float64(123456))
	assert.Equal(t, *samplingTragets.SamplingTargetDocuments[0].FixedRate, float64(5))
	assert.Equal(t, *samplingTragets.SamplingTargetDocuments[0].Interval, int64(5))
	assert.Equal(t, *samplingTragets.SamplingTargetDocuments[0].ReservoirQuota, int64(3))
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

	// generate a test server so we can capture and inspect the request
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		_, err := res.Write([]byte(body))
		require.NoError(t, err)
	}))
	defer testServer.Close()

	u, err := url.Parse(testServer.URL)
	require.NoError(t, err)

	client, err := newClient(u.Host)
	require.NoError(t, err)

	samplingTragets, err := client.getSamplingTargets(ctx, nil)
	require.NoError(t, err)

	assert.Nil(t, samplingTragets.SamplingTargetDocuments[0].Interval)
	assert.Nil(t, samplingTragets.SamplingTargetDocuments[0].ReservoirQuota)
}

func TestNewClient(t *testing.T) {
	xrayClient, err := newClient("127.0.0.1:2020")
	require.NoError(t, err)

	assert.Equal(t, xrayClient.endpoint.String(), "http://127.0.0.1:2020")
}

func TestEndpointIsNotReachable(t *testing.T) {
	client, err := newClient("127.0.0.1:2020")
	require.NoError(t, err)

	_, err = client.getSamplingRules(context.Background())
	assert.Error(t, err)
}
