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

package internal_xray

// samplingStatisticsDocument is used to store current state of sampling data
type samplingStatisticsDocument struct {
	// The number of requests recorded with borrowed reservoir quota.
	BorrowCount *int64

	// A unique identifier for the service in hexadecimal.
	ClientID *string

	// The number of requests that matched the rule.
	RequestCount *int64

	// The name of the sampling rule.
	RuleName *string

	// The number of requests recorded.
	SampledCount *int64

	// The current time.
	Timestamp *int64
}

// properties is the base set of properties that define a sampling rule.
type ruleProperties struct {
	RuleName      string            `json:"RuleName"`
	ServiceType   string            `json:"ServiceType"`
	ResourceARN   string            `json:"ResourceARN"`
	Attributes    map[string]string `json:"Attributes"`
	ServiceName   string            `json:"ServiceName"`
	Host          string            `json:"Host"`
	HTTPMethod    string            `json:"HTTPMethod"`
	URLPath       string            `json:"URLPath"`
	ReservoirSize int64             `json:"ReservoirSize"`
	FixedRate     float64           `json:"FixedRate"`
	Priority      int64             `json:"Priority"`
	Version       int64             `json:"Version"`
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

// samplingTargetDocument contains updated targeted information retrieved from X-Ray service
type samplingTargetDocument struct {
	// The percentage of matching requests to instrument, after the reservoir is
	// exhausted.
	FixedRate *float64 `json:"FixedRate,omitempty"`

	// The number of seconds for the service to wait before getting sampling targets
	// again.
	Interval *int64 `json:"Interval,omitempty"`

	// The number of requests per second that X-Ray allocated this service.
	ReservoirQuota *int64 `json:"ReservoirQuota,omitempty"`

	// When the reservoir quota expires.
	ReservoirQuotaTTL *float64 `json:"ReservoirQuotaTTL,omitempty"`

	// The name of the sampling rule.
	RuleName *string `json:"RuleName,omitempty"`
}

type unprocessedStatistic struct {
	ErrorCode *string `json:"ErrorCode,omitempty"`
	Message   *string `json:"Message,omitempty"`
	RuleName  *string `json:"RuleName,omitempty"`
}

type getSamplingTargetsInput struct {
	SamplingStatisticsDocuments []*samplingStatisticsDocument
}

// getSamplingTargetsOutput is used to store parsed json sampling targets
type getSamplingTargetsOutput struct {
	LastRuleModification    *float64                  `json:"LastRuleModification,omitempty"`
	SamplingTargetDocuments []*samplingTargetDocument `json:"SamplingTargetDocuments,omitempty"`
	UnprocessedStatistics   []*unprocessedStatistic   `json:"UnprocessedStatistics,omitempty"`
}
