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
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestGetSamplingRules(t *testing.T) {
	body := "{\"NextToken\":null,\"SamplingRuleRecords\":[{\"CreatedAt\":0.0,\"ModifiedAt\":1.639517389E9,\"SamplingRule\":{\"Attributes\":{},\"FixedRate\":0.5,\"HTTPMethod\":\"*\",\"Host\":\"*\",\"Priority\":10000,\"ReservoirSize\":60,\"ResourceARN\":\"*\",\"RuleARN\":\"arn:aws:xray:us-west-2:836082170990:sampling-rule/Default\",\"RuleName\":\"Default\",\"ServiceName\":\"*\",\"ServiceType\":\"*\",\"URLPath\":\"*\",\"Version\":1}},{\"CreatedAt\":1.637691613E9,\"ModifiedAt\":1.643748669E9,\"SamplingRule\":{\"Attributes\":{},\"FixedRate\":0.09,\"HTTPMethod\":\"GET\",\"Host\":\"*\",\"Priority\":1,\"ReservoirSize\":3,\"ResourceARN\":\"*\",\"RuleARN\":\"arn:aws:xray:us-west-2:836082170990:sampling-rule/test-rule\",\"RuleName\":\"test-rule\",\"ServiceName\":\"test-rule\",\"ServiceType\":\"local\",\"URLPath\":\"/aws-sdk-call\",\"Version\":1}},{\"CreatedAt\":1.639446197E9,\"ModifiedAt\":1.639446197E9,\"SamplingRule\":{\"Attributes\":{},\"FixedRate\":0.09,\"HTTPMethod\":\"*\",\"Host\":\"*\",\"Priority\":100,\"ReservoirSize\":100,\"ResourceARN\":\"*\",\"RuleARN\":\"arn:aws:xray:us-west-2:836082170990:sampling-rule/test-rule-1\",\"RuleName\":\"test-rule-1\",\"ServiceName\":\"*\",\"ServiceType\":\"*\",\"URLPath\":\"*\",\"Version\":1}}]}"
	ctx := context.Background()

	// generate a test server so we can capture and inspect the request
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		_, _ = res.Write([]byte(body))
	}))

	u, _ := url.Parse(testServer.URL)
	client := newClient(u.Host)

	samplingRules, _ := client.getSamplingRules(ctx)
	assert.Equal(t, *samplingRules.SamplingRuleRecords[0].SamplingRule.RuleName, "Default")
	assert.Equal(t, *samplingRules.SamplingRuleRecords[0].SamplingRule.ServiceType, "*")
	assert.Equal(t, *samplingRules.SamplingRuleRecords[0].SamplingRule.Host, "*")
	assert.Equal(t, *samplingRules.SamplingRuleRecords[0].SamplingRule.URLPath, "*")
	assert.Equal(t, *samplingRules.SamplingRuleRecords[0].SamplingRule.ReservoirSize, int64(60))
	assert.Equal(t, *samplingRules.SamplingRuleRecords[0].SamplingRule.FixedRate, 0.5)

	assert.Equal(t, *samplingRules.SamplingRuleRecords[1].SamplingRule.RuleName, "test-rule")
	assert.Equal(t, *samplingRules.SamplingRuleRecords[1].SamplingRule.ServiceType, "local")
	assert.Equal(t, *samplingRules.SamplingRuleRecords[1].SamplingRule.Host, "*")
	assert.Equal(t, *samplingRules.SamplingRuleRecords[1].SamplingRule.URLPath, "/aws-sdk-call")
	assert.Equal(t, *samplingRules.SamplingRuleRecords[1].SamplingRule.ReservoirSize, int64(3))
	assert.Equal(t, *samplingRules.SamplingRuleRecords[1].SamplingRule.FixedRate, 0.09)

	assert.Equal(t, *samplingRules.SamplingRuleRecords[2].SamplingRule.RuleName, "test-rule-1")
	assert.Equal(t, *samplingRules.SamplingRuleRecords[2].SamplingRule.ServiceType, "*")
	assert.Equal(t, *samplingRules.SamplingRuleRecords[2].SamplingRule.Host, "*")
	assert.Equal(t, *samplingRules.SamplingRuleRecords[2].SamplingRule.URLPath, "*")
	assert.Equal(t, *samplingRules.SamplingRuleRecords[2].SamplingRule.ReservoirSize, int64(100))
	assert.Equal(t, *samplingRules.SamplingRuleRecords[2].SamplingRule.FixedRate, 0.09)
}
