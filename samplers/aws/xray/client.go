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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type xrayClient struct {
	// http client for sending sampling requests to the collector
	httpClient *http.Client

	endpoint *url.URL
}

// newClient returns an HTTP client with proxy endpoint
func newClient(addr string) (client *xrayClient, err error) {
	endpoint := "http://" + addr

	endpointURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	return &xrayClient{
		httpClient: &http.Client{},
		endpoint:   endpointURL,
	}, nil
}

// getSamplingRules calls the collector(aws proxy enabled) for sampling rules
func (c *xrayClient) getSamplingRules(ctx context.Context) (*getSamplingRulesOutput, error) {
	samplingRulesInput, err := json.Marshal(getSamplingRulesInput{})
	if err != nil {
		return nil, err
	}
	body := bytes.NewReader(samplingRulesInput)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint.String()+"/GetSamplingRules", body)
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

// getSamplingTargets calls the collector(aws proxy enabled) for sampling targets
func (c *xrayClient) getSamplingTargets(ctx context.Context, s []*samplingStatisticsDocument) (*getSamplingTargetsOutput, error) {
	statistics := getSamplingTargetsInput{
		SamplingStatisticsDocuments: s,
	}

	statisticsByte, err := json.Marshal(statistics)
	if err != nil {
		return nil, err
	}
	body := bytes.NewReader(statisticsByte)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint.String()+"/SamplingTargets", body)
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
