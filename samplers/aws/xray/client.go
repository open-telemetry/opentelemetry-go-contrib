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
	"encoding/json"
	"log"
	"net/http"
	"net/url"
)

type xrayClient struct {
	// http client for sending unsigned proxied requests to the collector
	httpClient *http.Client

	proxyEndpoint string
}

// newClient returns a http client with proxy endpoint
func newClient(d string) *xrayClient {
	proxyEndpoint := "http://" + d

	proxyURL, err := url.Parse(proxyEndpoint)
	if err != nil {
		log.Println("Bad proxy URL", err)
	}

	httpClient := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
	}

	p := &xrayClient{
		httpClient:    httpClient,
		proxyEndpoint: proxyEndpoint,
	}

	return p
}

// getSamplingRules calls the collector(aws proxy enabled) for sampling rules
func (p *xrayClient) getSamplingRules() ([]ruleProperties, error) {
	req, err := http.NewRequest(http.MethodPost, p.proxyEndpoint+"/GetSamplingRules", nil)
	if err != nil {
		log.Printf("failed to create http request, %v\n", err)
	}

	output, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	err = json.NewDecoder(output.Body).Decode(&data)
	if err != nil {
		return nil, err
	}

	var rules []ruleProperties
	switch x := data["SamplingRuleRecords"].(type) {
	case []interface{}:
		for _, e := range x {
			svcRule := e.(map[string]interface{})["SamplingRule"].(map[string]interface{})

			rules = append(rules, ruleProperties{
				ruleName:      svcRule["RuleName"].(string),
				serviceType:   svcRule["ServiceType"].(string),
				resourceARN:   svcRule["ResourceARN"].(string),
				attributes:    svcRule["Attributes"].(map[string]interface{}),
				serviceName:   svcRule["ServiceName"].(string),
				host:          svcRule["Host"].(string),
				httpMethod:    svcRule["HTTPMethod"].(string),
				urlPath:       svcRule["URLPath"].(string),
				reservoirSize: int64(svcRule["ReservoirSize"].(float64)),
				fixedRate:     svcRule["FixedRate"].(float64),
				priority:      int64(svcRule["Priority"].(float64)),
				version:       int64(svcRule["Version"].(float64)),
			})
		}
	default:
		log.Printf("unhandled type: %T\n", data["SamplingRuleRecords"])
	}

	err = output.Body.Close()
	if err != nil {
		log.Printf("failed to close http response body, %v\n\n", err)
	}

	return rules, nil
}
