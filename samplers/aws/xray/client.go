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

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
)

type xrayClient struct {
	// http client for sending unsigned proxied requests to the collector
	httpClient *http.Client

	proxyEndpoint string
}

// newClient returns a http client with proxy endpoint
func newClient(d string) *xrayClient {
	proxyEndpoint := "http://" + d

	p := &xrayClient{
		httpClient:    &http.Client{},
		proxyEndpoint: proxyEndpoint,
	}

	return p
}

// getSamplingRules calls the collector(aws proxy enabled) for sampling rules
func (p *xrayClient) getSamplingRules(ctx context.Context) (*getSamplingRulesOutput, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.proxyEndpoint+"/GetSamplingRules", nil)
	if err != nil {
		log.Printf("failed to create http request, %v\n", err)
	}

	output, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(output.Body)
	if err != nil {
		return nil, err
	}

	// Unmarshalling json data to populate getSamplingTargetsOutput struct
	var samplingRulesOutput getSamplingRulesOutput
	err = json.Unmarshal(buf.Bytes(), &samplingRulesOutput)
	if err != nil {
		return nil, err
	}

	err = output.Body.Close()
	if err != nil {
		log.Printf("failed to close http response body, %v\n\n", err)
	}

	return &samplingRulesOutput, nil
}

// getSamplingTargets calls the collector(aws proxy enabled) for sampling targets
func (p *xrayClient) getSamplingTargets(ctx context.Context, s []*samplingStatisticsDocument) (*getSamplingTargetsOutput, error) {
	statistics := getSamplingTargetsInput{
		SamplingStatisticsDocuments: s,
	}

	statisticsByte, _ := json.Marshal(statistics)
	body := bytes.NewReader(statisticsByte)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.proxyEndpoint+"/SamplingTargets", body)
	if err != nil {
		log.Printf("failed to create http request, %v\n", err)
	}

	output, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(output.Body)
	if err != nil {
		return nil, err
	}

	// Unmarshalling json data to populate getSamplingTargetsOutput struct
	var samplingTargetsOutput getSamplingTargetsOutput
	err = json.Unmarshal(buf.Bytes(), &samplingTargetsOutput)
	if err != nil {
		return nil, err
	}

	err = output.Body.Close()
	if err != nil {
		log.Printf("failed to close http response body, %v\n\n", err)
	}

	return &samplingTargetsOutput, nil
}
