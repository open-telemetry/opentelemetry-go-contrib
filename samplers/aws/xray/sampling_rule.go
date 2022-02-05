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

// ToDo: other fields will be used in business logic for remote sampling
// rule represents a centralized sampling rule
type rule struct {
	// Centralized reservoir for keeping track of reservoir usage
	reservoir *reservoir

	// sampling rule properties
	ruleProperties *ruleProperties

	// Number of requests matched against this rule
	//matchedRequests int64
	//
	// Number of requests sampled using this rule
	//sampledRequests int64
	//
	// Number of requests burrowed
	//borrowedRequests int64

	// Provides system time
	clock clock

	// Provides random numbers
	rand Rand

	//mu sync.RWMutex
}

// properties is the base set of properties that define a sampling rule.
type ruleProperties struct {
	RuleName      *string            `json:"RuleName"`
	ServiceType   *string            `json:"ServiceType"`
	ResourceARN   *string            `json:"ResourceARN"`
	Attributes    map[string]*string `json:"Attributes"`
	ServiceName   *string            `json:"ServiceName"`
	Host          *string            `json:"Host"`
	HTTPMethod    *string            `json:"HTTPMethod"`
	URLPath       *string            `json:"URLPath"`
	ReservoirSize *int64             `json:"ReservoirSize"`
	FixedRate     *float64           `json:"FixedRate"`
	Priority      *int64             `json:"Priority"`
	Version       *int64             `json:"Version"`
}

// getSamplingRulesInput is used to store
type getSamplingRulesInput struct {
	NextToken *string `json:"NextToken"`
}

type samplingRuleRecords struct {
	SamplingRule *ruleProperties `json:"SamplingRule"`
}

// getSamplingRulesOutput is used to store parsed json sampling rules
type getSamplingRulesOutput struct {
	SamplingRuleRecords []*samplingRuleRecords `json:"SamplingRuleRecords"`
}
