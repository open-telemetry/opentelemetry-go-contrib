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

package internal // import "go.opentelemetry.io/contrib/samplers/aws/xray/internal"

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// getSamplingRulesOutput is used to store parsed json sampling rules.
type getSamplingRulesOutput struct {
	SamplingRuleRecords []*samplingRuleRecords `json:"SamplingRuleRecords"`
}

type samplingRuleRecords struct {
	SamplingRule *ruleProperties `json:"SamplingRule"`
}

// ruleProperties is the base set of properties that define a sampling rule.
type ruleProperties struct {
	RuleName      string            `json:"RuleName"`
	ServiceType   string            `json:"ServiceType"`
	ResourceARN   string            `json:"ResourceARN"`
	Attributes    map[string]string `json:"Attributes"`
	ServiceName   string            `json:"ServiceName"`
	Host          string            `json:"Host"`
	HTTPMethod    string            `json:"HTTPMethod"`
	URLPath       string            `json:"URLPath"`
	ReservoirSize float64           `json:"ReservoirSize"`
	FixedRate     float64           `json:"FixedRate"`
	Priority      int64             `json:"Priority"`
	Version       int64             `json:"Version"`
}

type getSamplingTargetsInput struct {
	SamplingStatisticsDocuments []*samplingStatisticsDocument
}

// samplingStatisticsDocument is used to store current state of sampling data.
type samplingStatisticsDocument struct {
	// A unique identifier for the service in hexadecimal.
	ClientID *string

	// The name of the sampling rule.
	RuleName *string

	// The number of requests that matched the rule.
	RequestCount *int64

	// The number of requests borrowed.
	BorrowCount *int64

	// The number of requests sampled using the rule.
	SampledCount *int64

	// The current time.
	Timestamp *int64
}

// getSamplingTargetsOutput is used to store parsed json sampling targets.
type getSamplingTargetsOutput struct {
	LastRuleModification    *float64                  `json:"LastRuleModification,omitempty"`
	SamplingTargetDocuments []*samplingTargetDocument `json:"SamplingTargetDocuments,omitempty"`
	UnprocessedStatistics   []*unprocessedStatistic   `json:"UnprocessedStatistics,omitempty"`
}

// samplingTargetDocument contains updated targeted information retrieved from X-Ray service.
type samplingTargetDocument struct {
	// The percentage of matching requests to instrument, after the reservoir is
	// exhausted.
	FixedRate *float64 `json:"FixedRate,omitempty"`

	// The number of seconds for the service to wait before getting sampling targets
	// again.
	Interval *int64 `json:"Interval,omitempty"`

	// The number of requests per second that X-Ray allocated this service.
	ReservoirQuota *float64 `json:"ReservoirQuota,omitempty"`

	// The reservoir quota expires.
	ReservoirQuotaTTL *float64 `json:"ReservoirQuotaTTL,omitempty"`

	// The name of the sampling rule.
	RuleName *string `json:"RuleName,omitempty"`
}

type unprocessedStatistic struct {
	ErrorCode *string `json:"ErrorCode,omitempty"`
	Message   *string `json:"Message,omitempty"`
	RuleName  *string `json:"RuleName,omitempty"`
}

type xrayClient struct {
	// HTTP client for sending sampling requests to the collector.
	httpClient *http.Client

	// Resolved URL to call getSamplingRules API.
	samplingRulesURL string

	// Resolved URL to call getSamplingTargets API.
	samplingTargetsURL string
}

// newClient returns an HTTP client with proxy endpoint.
func newClient(endpoint url.URL) (client *xrayClient, err error) {
	// Construct resolved URLs for getSamplingRules and getSamplingTargets API calls.
	endpoint.Path = "/GetSamplingRules"
	samplingRulesURL := endpoint

	endpoint.Path = "/SamplingTargets"
	samplingTargetsURL := endpoint

	return &xrayClient{
		httpClient:         &http.Client{},
		samplingRulesURL:   samplingRulesURL.String(),
		samplingTargetsURL: samplingTargetsURL.String(),
	}, nil
}

// getSamplingRules calls the collector(aws proxy enabled) for sampling rules.
func (c *xrayClient) getSamplingRules(ctx context.Context) (*getSamplingRulesOutput, error) {
	emptySamplingRulesInputJSON := []byte(`{"NextToken": null}`)

	body := bytes.NewReader(emptySamplingRulesInputJSON)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.samplingRulesURL, body)
	if err != nil {
		return nil, fmt.Errorf("xray client: failed to create http request: %w", err)
	}

	output, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("xray client: unable to retrieve sampling settings: %w", err)
	}
	defer output.Body.Close()

	var samplingRulesOutput *getSamplingRulesOutput
	if err := json.NewDecoder(output.Body).Decode(&samplingRulesOutput); err != nil {
		return nil, fmt.Errorf("xray client: unable to unmarshal the response body: %w", err)
	}

	return samplingRulesOutput, nil
}

// getSamplingTargets calls the collector(aws proxy enabled) for sampling targets.
func (c *xrayClient) getSamplingTargets(ctx context.Context, s []*samplingStatisticsDocument) (*getSamplingTargetsOutput, error) {
	statistics := getSamplingTargetsInput{
		SamplingStatisticsDocuments: s,
	}

	statisticsByte, err := json.Marshal(statistics)
	if err != nil {
		return nil, err
	}
	body := bytes.NewReader(statisticsByte)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.samplingTargetsURL, body)
	if err != nil {
		return nil, fmt.Errorf("xray client: failed to create http request: %w", err)
	}

	output, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("xray client: unable to retrieve sampling settings: %w", err)
	}
	defer output.Body.Close()

	var samplingTargetsOutput *getSamplingTargetsOutput
	if err := json.NewDecoder(output.Body).Decode(&samplingTargetsOutput); err != nil {
		return nil, fmt.Errorf("xray client: unable to unmarshal the response body: %w", err)
	}

	return samplingTargetsOutput, nil
}
