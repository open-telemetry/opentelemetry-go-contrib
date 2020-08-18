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

package cortex

import (
	"encoding/base64"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAuthentication checks whether http requests are properly authenticated with either
// bearer tokens or basic authentication in the addHeaders method.
func TestAuthentication(t *testing.T) {
	tests := []struct {
		testName                      string
		basicAuth                     map[string]string
		basicAuthPasswordFileContents []byte
		bearerToken                   string
		bearerTokenFile               string
		bearerTokenFileContents       []byte
		expectedAuthHeaderValue       string
		expectedError                 error
	}{
		{
			testName: "Basic Auth with password",
			basicAuth: map[string]string{
				"username": "TestUser",
				"password": "TestPassword",
			},
			expectedAuthHeaderValue: "Basic " + base64.StdEncoding.EncodeToString(
				[]byte("TestUser:TestPassword"),
			),
			expectedError: nil,
		},
	}
	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			// Set up a test server that runs a handler function when it receives a http
			// request. The server writes the request's Authorization header to the
			// response body.
			handler := func(rw http.ResponseWriter, req *http.Request) {
				authHeaderValue := req.Header.Get("Authorization")
				rw.Write([]byte(authHeaderValue))
			}
			server := httptest.NewServer(http.HandlerFunc(handler))
			defer server.Close()

			// Create a HTTP request and add headers to it through an Exporter. Since the
			// Exporter has an empty Headers map, authentication methods will be called.
			exporter := Exporter{
				Config{
					BasicAuth:       test.basicAuth,
					BearerToken:     test.bearerToken,
					BearerTokenFile: test.bearerTokenFile,
				},
			}
			req, err := http.NewRequest(http.MethodPost, server.URL, nil)
			require.Nil(t, err)
			err = exporter.addHeaders(req)

			// Verify the error and if the Authorization header was correctly set.
			if err != nil {
				require.Equal(t, err.Error(), test.expectedError.Error())
			} else {
				require.Nil(t, test.expectedError)
				authHeaderValue := req.Header.Get("Authorization")
				require.Equal(t, authHeaderValue, test.expectedAuthHeaderValue)
			}
		})
	}
}

// createFile writes a file with a slice of bytes at a specified filepath.
func createFile(bytes []byte, filepath string) error {
	err := ioutil.WriteFile(filepath, bytes, 0644)
	if err != nil {
		return err
	}
	return nil
}
