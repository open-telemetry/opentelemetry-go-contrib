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

func TestGetSamplingRulesStatusCodeCheck(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		body          []byte
		expectedError bool
	}{
		{
			name:          "Success",
			statusCode:    http.StatusOK,
			body:          []byte(`{"SamplingRuleRecords":[{"SamplingRule":{"RuleName":"test-rule"}}]}`),
			expectedError: false,
		},
		{
			name:          "Client Error",
			statusCode:    http.StatusBadRequest,
			body:          []byte(`Bad Request`),
			expectedError: true,
		},
		{
			name:          "Server Error",
			statusCode:    http.StatusInternalServerError,
			body:          []byte(`Internal Server Error`),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, _ *http.Request) {
				res.WriteHeader(tt.statusCode)
				_, err := res.Write(tt.body)
				require.NoError(t, err)
			}))
			t.Cleanup(testServer.Close)

			u, err := url.Parse(testServer.URL)
			require.NoError(t, err)

			client, err := newClient(*u)
			require.NoError(t, err)

			ctx := context.Background()
			_, err = client.getSamplingRules(ctx)
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
