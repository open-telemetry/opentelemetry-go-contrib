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
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type xrayClient struct {
	// http client for sending unsigned proxied requests to the collector
	httpClient *http.Client

	endpoint *url.URL
}

// newClient returns an HTTP client with proxy endpoint
func newClient(d string) *xrayClient {
	endpoint := "http://" + d

	endpointURL, err := url.Parse(endpoint)
	if err != nil {
		globalLogger.Error(err, "unable to parse endpoint from string")
	}

	return &xrayClient{
		httpClient: &http.Client{},
		endpoint:   endpointURL,
	}
}

// getSamplingRules calls the collector(aws proxy enabled) for sampling rules
func (p *xrayClient) getSamplingRules(ctx context.Context) (*getSamplingRulesOutput, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint.String()+"/GetSamplingRules", nil)
	if err != nil {
		return nil, fmt.Errorf("xray client: failed to create http request: %w", err)
	}

	output, err := p.httpClient.Do(req)
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
