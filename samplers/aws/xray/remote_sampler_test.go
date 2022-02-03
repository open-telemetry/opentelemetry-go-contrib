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

func TestRefreshManifest(t *testing.T) {
	ctx := context.Background()

	body := "{\"NextToken\":null,\"SamplingRuleRecords\":[{\"CreatedAt\":0.0,\"ModifiedAt\":1.639517389E9,\"SamplingRule\":{\"Attributes\":{},\"FixedRate\":0.5,\"HTTPMethod\":\"*\",\"Host\":\"*\",\"Priority\":10000,\"ReservoirSize\":60,\"ResourceARN\":\"*\",\"RuleARN\":\"arn:aws:xray:us-west-2:836082170990:sampling-rule/Default\",\"RuleName\":\"Default\",\"ServiceName\":\"*\",\"ServiceType\":\"*\",\"URLPath\":\"*\",\"Version\":1}},{\"CreatedAt\":1.637691613E9,\"ModifiedAt\":1.643748669E9,\"SamplingRule\":{\"Attributes\":{},\"FixedRate\":0.09,\"HTTPMethod\":\"GET\",\"Host\":\"*\",\"Priority\":1,\"ReservoirSize\":3,\"ResourceARN\":\"*\",\"RuleARN\":\"arn:aws:xray:us-west-2:836082170990:sampling-rule/test-rule\",\"RuleName\":\"test-rule\",\"ServiceName\":\"test-rule\",\"ServiceType\":\"local\",\"URLPath\":\"/aws-sdk-call\",\"Version\":1}},{\"CreatedAt\":1.639446197E9,\"ModifiedAt\":1.639446197E9,\"SamplingRule\":{\"Attributes\":{},\"FixedRate\":0.09,\"HTTPMethod\":\"*\",\"Host\":\"*\",\"Priority\":100,\"ReservoirSize\":100,\"ResourceARN\":\"*\",\"RuleARN\":\"arn:aws:xray:us-west-2:836082170990:sampling-rule/test-rule-1\",\"RuleName\":\"test-rule-1\",\"ServiceName\":\"*\",\"ServiceType\":\"*\",\"URLPath\":\"*\",\"Version\":1}}]}"

	// generate a test server so we can capture and inspect the request
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		_, _ = res.Write([]byte(body))
	}))
	defer testServer.Close()

	u, _ := url.Parse(testServer.URL)

	clock := &DefaultClock{}

	m := &centralizedManifest{
		rules: []*centralizedRule{},
		index: map[string]*centralizedRule{},
		clock: clock,
	}

	rs := &RemoteSampler{
		xrayClient: newClient(u.Host),
		clock:      clock,
		manifest:   m,
	}

	_ = rs.refreshManifest(ctx)

	// Rule 'r1'
	r1 := &centralizedRule{
		ruleProperties: &ruleProperties{
			RuleName:      getStringPointer("Default"),
			Priority:      getIntPointer(10000),
			Host:          getStringPointer("*"),
			HTTPMethod:    getStringPointer("*"),
			URLPath:       getStringPointer("*"),
			ReservoirSize: getIntPointer(40),
			Version:       getIntPointer(1),
			FixedRate:     getFloatPointer(0.5),
			ServiceName:   getStringPointer("*"),
			ResourceARN:   getStringPointer("*"),
			ServiceType:   getStringPointer(""),
		},
	}

	// Rule 'r2'
	r2 := &centralizedRule{
		ruleProperties: &ruleProperties{
			RuleName:      getStringPointer("test-rule"),
			Priority:      getIntPointer(1),
			Host:          getStringPointer("*"),
			HTTPMethod:    getStringPointer("GET"),
			URLPath:       getStringPointer("/aws-sdk-call"),
			ReservoirSize: getIntPointer(3),
			FixedRate:     getFloatPointer(0.09),
			Version:       getIntPointer(1),
			ServiceName:   getStringPointer("test-rule"),
			ResourceARN:   getStringPointer("*"),
			ServiceType:   getStringPointer("local"),
		},
	}

	// Rule 'r3'
	r3 := &centralizedRule{
		ruleProperties: &ruleProperties{
			RuleName:      getStringPointer("test-rule-1"),
			Priority:      getIntPointer(100),
			Host:          getStringPointer("*"),
			HTTPMethod:    getStringPointer("*"),
			URLPath:       getStringPointer("*"),
			ReservoirSize: getIntPointer(100),
			FixedRate:     getFloatPointer(0.09),
			Version:       getIntPointer(1),
			ServiceName:   getStringPointer("*"),
			ResourceARN:   getStringPointer("*"),
			ServiceType:   getStringPointer(""),
		},
	}
	// Assert on sorting order
	assert.Equal(t, r2.ruleProperties.RuleName, rs.manifest.rules[0].ruleProperties.RuleName)
	assert.Equal(t, r3.ruleProperties.RuleName, rs.manifest.rules[1].ruleProperties.RuleName)
	assert.Equal(t, r1.ruleProperties.RuleName, rs.manifest.rules[2].ruleProperties.RuleName)

	// Assert on size of manifest
	assert.Equal(t, 3, len(rs.manifest.rules))
	assert.Equal(t, 3, len(rs.manifest.index))
}

func TestRefreshManifestRuleAdditionInvalidRule1(t *testing.T) {
	ctx := context.Background()
	newConfig()

	// Rule 'r1'
	r1 := &centralizedRule{
		ruleProperties: &ruleProperties{
			RuleName:    getStringPointer("r1"),
			Priority:    getIntPointer(4),
			ResourceARN: getStringPointer("XYZ"), // invalid
		},
		reservoir: &centralizedReservoir{
			quota:    10,
			capacity: 50,
		},
	}

	// Sorted array
	rules := []*centralizedRule{r1}

	index := map[string]*centralizedRule{
		"r1": r1,
	}

	manifest := &centralizedManifest{
		rules:       rules,
		index:       index,
		refreshedAt: 1500000000,
	}

	body := "{\"NextToken\":null,\"SamplingRuleRecords\":[{\"CreatedAt\":1.639446197E9,\"ModifiedAt\":1.639446197E9,\"SamplingRule\":{\"Attributes\":{},\"FixedRate\":0.05,\"HTTPMethod\":\"POST\",\"Host\":\"*\",\"Priority\":4,\"ReservoirSize\":50,\"ResourceARN\":\"XYZ\",\"RuleARN\":\"arn:aws:xray:us-west-2:836082170990:sampling-rule/r1\",\"RuleName\":\"r1\",\"ServiceName\":\"www.foo.com\",\"ServiceType\":\"\",\"URLPath\":\"/resource/bar\",\"Version\":1}}]}"

	// generate a test server so we can capture and inspect the request
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		_, _ = res.Write([]byte(body))
	}))
	defer testServer.Close()

	u, _ := url.Parse(testServer.URL)

	clock := &DefaultClock{}

	rs := &RemoteSampler{
		xrayClient: newClient(u.Host),
		clock:      clock,
		manifest:   manifest,
	}

	rs.manifest = manifest
	err := rs.refreshManifest(ctx)

	assert.Nil(t, err)
	// Refresh manifest with updates from mock proxy
	assert.Equal(t, 0, len(rs.manifest.rules)) // Rule not added
}

func TestRefreshManifestRuleAdditionInvalidRule2(t *testing.T) { // non nil Attributes
	ctx := context.Background()

	attributes := make(map[string]*string)
	attributes["a"] = getStringPointer("*")

	// Rule 'r1'
	r1 := &centralizedRule{
		ruleProperties: &ruleProperties{
			RuleName:    getStringPointer("r1"),
			Priority:    getIntPointer(4),
			ResourceARN: getStringPointer("*"),
		},
		reservoir: &centralizedReservoir{
			quota:    10,
			capacity: 50,
		},
	}

	// Sorted array
	rules := []*centralizedRule{r1}

	index := map[string]*centralizedRule{
		"r1": r1,
	}

	manifest := &centralizedManifest{
		rules:       rules,
		index:       index,
		refreshedAt: 1500000000,
	}

	body := "{\"NextToken\":null,\"SamplingRuleRecords\":[{\"CreatedAt\":1.639446197E9,\"ModifiedAt\":1.639446197E9,\"SamplingRule\":{\"Attributes\":{\"a\":\"b\"},\"FixedRate\":0.05,\"HTTPMethod\":\"POST\",\"Host\":\"*\",\"Priority\":4,\"ReservoirSize\":50,\"ResourceARN\":\"XYZ\",\"RuleARN\":\"arn:aws:xray:us-west-2:836082170990:sampling-rule/r1\",\"RuleName\":\"r1\",\"ServiceName\":\"www.foo.com\",\"ServiceType\":\"\",\"URLPath\":\"/resource/bar\",\"Version\":1}}]}"

	// generate a test server so we can capture and inspect the request
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		_, _ = res.Write([]byte(body))
	}))
	defer testServer.Close()

	u, _ := url.Parse(testServer.URL)

	clock := &DefaultClock{}

	rs := &RemoteSampler{
		xrayClient: newClient(u.Host),
		clock:      clock,
		manifest:   manifest,
	}

	err := rs.refreshManifest(ctx)

	assert.Nil(t, err)
	assert.Equal(t, 0, len(rs.manifest.rules)) // rule not added
}

func TestRefreshManifestRuleAdditionInvalidRule3(t *testing.T) { // 1 valid and 1 invalid rule
	ctx := context.Background()

	// Rule 'r1'
	r1 := &centralizedRule{
		ruleProperties: &ruleProperties{
			RuleName:    getStringPointer("r1"),
			Priority:    getIntPointer(4),
			ResourceARN: getStringPointer("*"),
		},
		reservoir: &centralizedReservoir{
			quota:    10,
			capacity: 50,
		},
	}

	// Rule 'r2'
	r2 := &centralizedRule{
		ruleProperties: &ruleProperties{
			RuleName:    getStringPointer("r2"),
			Priority:    getIntPointer(4),
			ResourceARN: getStringPointer("*"),
		},
		reservoir: &centralizedReservoir{
			quota:    10,
			capacity: 50,
		},
	}

	// Sorted array
	rules := []*centralizedRule{r1}

	index := map[string]*centralizedRule{
		"r1": r1,
	}

	manifest := &centralizedManifest{
		rules:       rules,
		index:       index,
		refreshedAt: 1500000000,
	}

	body := "{\"NextToken\":null,\"SamplingRuleRecords\":[{\"CreatedAt\":0.0,\"ModifiedAt\":1.639517389E9,\"SamplingRule\":{\"Attributes\":{\"a\":\"b\"},\"FixedRate\":0.5,\"HTTPMethod\":\"*\",\"Host\":\"*\",\"Priority\":10000,\"ReservoirSize\":60,\"ResourceARN\":\"*\",\"RuleARN\":\"arn:aws:xray:us-west-2:836082170990:sampling-rule/Default\",\"RuleName\":\"r1\",\"ServiceName\":\"*\",\"ServiceType\":\"*\",\"URLPath\":\"*\",\"Version\":1}},{\"CreatedAt\":1.637691613E9,\"ModifiedAt\":1.643748669E9,\"SamplingRule\":{\"Attributes\":{},\"FixedRate\":0.09,\"HTTPMethod\":\"GET\",\"Host\":\"*\",\"Priority\":1,\"ReservoirSize\":3,\"ResourceARN\":\"*\",\"RuleARN\":\"arn:aws:xray:us-west-2:836082170990:sampling-rule/test-rule\",\"RuleName\":\"r2\",\"ServiceName\":\"test-rule\",\"ServiceType\":\"local\",\"URLPath\":\"/aws-sdk-call\",\"Version\":1}}]}"

	// generate a test server so we can capture and inspect the request
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		_, _ = res.Write([]byte(body))
	}))
	defer testServer.Close()

	u, _ := url.Parse(testServer.URL)

	clock := &DefaultClock{}

	rs := &RemoteSampler{
		xrayClient: newClient(u.Host),
		clock:      clock,
		manifest:   manifest,
	}

	err := rs.refreshManifest(ctx)

	assert.Nil(t, err)
	assert.Equal(t, 1, len(rs.manifest.rules)) // u1 not added
	assert.Equal(t, r2.ruleProperties.RuleName, rs.manifest.rules[0].ruleProperties.RuleName)
}

// Assert that an invalid rule update does not update the rule
func TestRefreshManifestInvalidRuleUpdate(t *testing.T) {
	ctx := context.Background()

	// Rule 'r1'
	r1 := &centralizedRule{
		ruleProperties: &ruleProperties{
			RuleName:      getStringPointer("r1"),
			Priority:      getIntPointer(4),
			ResourceARN:   getStringPointer("*"),
			ServiceName:   getStringPointer("www.foo.com"),
			HTTPMethod:    getStringPointer("POST"),
			URLPath:       getStringPointer("/resource/bar"),
			ReservoirSize: getIntPointer(50),
			FixedRate:     getFloatPointer(0.05),
		},
		reservoir: &centralizedReservoir{
			quota:    10,
			capacity: 50,
		},
	}

	// Rule 'r3'
	r3 := &centralizedRule{
		ruleProperties: &ruleProperties{
			RuleName:      getStringPointer("r1"),
			Priority:      getIntPointer(8),
			ResourceARN:   getStringPointer("*"),
			ServiceName:   getStringPointer("www.bar.com"),
			HTTPMethod:    getStringPointer("POST"),
			URLPath:       getStringPointer("/resource/foo"),
			ReservoirSize: getIntPointer(40),
			FixedRate:     getFloatPointer(0.10),
			Host:          getStringPointer("h3"),
		},
		reservoir: &centralizedReservoir{
			quota:    10,
			capacity: 50,
		},
	}

	// Sorted array
	rules := []*centralizedRule{r1, r3}

	index := map[string]*centralizedRule{
		"r1": r1,
		"r3": r3,
	}

	manifest := &centralizedManifest{
		rules:       rules,
		index:       index,
		refreshedAt: 1500000000,
	}

	body := "{\"NextToken\":null,\"SamplingRuleRecords\":[{\"CreatedAt\":0.0,\"ModifiedAt\":1.639517389E9,\"SamplingRule\":{\"Attributes\":{},\"FixedRate\":0.5,\"HTTPMethod\":\"*\",\"Host\":\"*\",\"Priority\":10000,\"ReservoirSize\":60,\"ResourceARN\":\"*\",\"RuleARN\":\"arn:aws:xray:us-west-2:836082170990:sampling-rule/Default\",\"FixedRate\":0.09,\"RuleName\":\"r1\",\"ServiceName\":\"*\",\"ServiceType\":\"*\",\"URLPath\":\"*\",\"Version\":1}},{\"CreatedAt\":1.637691613E9,\"ModifiedAt\":1.643748669E9,\"SamplingRule\":{\"Attributes\":{},\"FixedRate\":0.09,\"HTTPMethod\":\"GET\",\"Host\":\"*\",\"Priority\":1,\"ReservoirSize\":3,\"ResourceARN\":\"*\",\"RuleARN\":\"arn:aws:xray:us-west-2:836082170990:sampling-rule/test-rule\",\"RuleName\":\"r3\",\"ServiceName\":\"test-rule\",\"ServiceType\":\"local\",\"URLPath\":\"/aws-sdk-call\",\"Version\":2}}]}"

	// generate a test server so we can capture and inspect the request
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		_, _ = res.Write([]byte(body))
	}))
	defer testServer.Close()

	u, _ := url.Parse(testServer.URL)

	clock := &DefaultClock{}

	rs := &RemoteSampler{
		xrayClient: newClient(u.Host),
		clock:      clock,
		manifest:   manifest,
	}

	err := rs.refreshManifest(ctx)
	assert.NotNil(t, err)

	// Assert on size of manifest
	assert.Equal(t, 1, len(rs.manifest.rules))
	assert.Equal(t, 1, len(rs.manifest.index))

	// Assert on sorting order
	assert.Equal(t, r1.ruleProperties.RuleName, rs.manifest.rules[0].ruleProperties.RuleName)
}
