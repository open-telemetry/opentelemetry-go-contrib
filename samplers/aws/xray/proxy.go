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
	"io"
	"log"
	"net/http"
	"net/url"
)

type proxy struct {
	// http client for sending unsigned proxied requests to the daemon
	httpClient *http.Client

	proxyEndpoint string
}

// newProxy returns a http client with proxy endpoint
func newProxy(d string) (*proxy, error) {
	log.Printf("X-Ray proxy using address : %v\n", d)
	proxyEndpoint := "http://" + d

	proxyURL, err := url.Parse(proxyEndpoint)
	if err != nil {
		log.Println("Bad proxy URL", err)
		return nil, err
	}

	httpClient := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
	}

	p := &proxy{
		httpClient:    httpClient,
		proxyEndpoint: proxyEndpoint,
	}

	return p, nil
}

// getSamplingRules calls the collector(aws proxy enabled) for sampling rules
func (p *proxy) getSamplingRules() (interface{}, error) {
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
		log.Printf("failed to read http response body, %v\n", err)
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("failed to close http response body, %v\n\n", err)
		}
	}(output.Body)

	return data["SamplingRuleRecords"], nil
}
