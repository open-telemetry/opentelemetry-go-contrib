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

package jaegerremote

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	jaeger_api_v2 "go.opentelemetry.io/contrib/samplers/jaegerremote/internal/proto-gen/jaeger-idl/proto/api_v2"
)

func Test_samplingStrategyFetcherImpl_Fetch(t *testing.T) {
	tests := []struct {
		name               string
		responseStatusCode int
		responseBody       string
		expectedErr        string
		expectedStrategy   jaeger_api_v2.SamplingStrategyResponse
	}{
		{
			name:               "RequestOK",
			responseStatusCode: http.StatusOK,
			responseBody: `{
  "strategyType": 0,
  "probabilisticSampling": {
    "samplingRate": 0.5
  }
}`,
			expectedStrategy: jaeger_api_v2.SamplingStrategyResponse{
				StrategyType: jaeger_api_v2.SamplingStrategyType_PROBABILISTIC,
				ProbabilisticSampling: &jaeger_api_v2.ProbabilisticSamplingStrategy{
					SamplingRate: 0.5,
				},
			},
		},
		{
			name:               "RequestError",
			responseStatusCode: http.StatusTooManyRequests,
			responseBody:       "you are sending too many requests",
			expectedErr:        "request failed (429): you are sending too many requests",
		},
		{
			name:               "InvalidResponseData",
			responseStatusCode: http.StatusOK,
			responseBody:       `{"strategy`,
			expectedErr:        "unexpected end of JSON input",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/?service=foo", r.URL.RequestURI())

				w.WriteHeader(tt.responseStatusCode)
				_, err := w.Write([]byte(tt.responseBody))
				assert.NoError(t, err)
			}))
			defer server.Close()

			fetcher := samplingStrategyFetcherImpl{
				serviceName: "foo",
				endpoint:    server.URL,
				httpClient:  http.DefaultClient,
			}

			strategyResponse, err := fetcher.Fetch()
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStrategy, strategyResponse)
			}
		})
	}
}
