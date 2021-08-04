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

package jaeger_remote // import "go.opentelemetry.io/contrib/samplers/jaeger_remote"

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	jaeger_api_v2 "github.com/jaegertracing/jaeger/proto-gen/api_v2"
)

type samplingStrategyFetcher interface {
	Fetch() (jaeger_api_v2.SamplingStrategyResponse, error)
}

type samplingStrategyFetcherImpl struct {
	serviceName string
	endpoint    string
	httpClient  *http.Client
}

var _ samplingStrategyFetcher = samplingStrategyFetcherImpl{}

func (f samplingStrategyFetcherImpl) Fetch() (s jaeger_api_v2.SamplingStrategyResponse, err error) {
	uri := f.endpoint + "?service=" + url.QueryEscape(f.serviceName)

	resp, err := f.httpClient.Get(uri)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	if resp.StatusCode != http.StatusOK {
		return s, fmt.Errorf("request failed (%d): %s", resp.StatusCode, body)
	}

	err = json.Unmarshal(body, &s)
	return
}
