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
	"fmt"
	"io/ioutil"
	"net/http"
)

// ErrFailedToReadFile occurs when a password / bearer token file exists, but could
// not be read.
var ErrFailedToReadFile = fmt.Errorf("failed to read password / bearer token file")

// addBasicAuth sets the Authorization header for basic authentication using a username
// and a password / password file. The header value is not changed if an Authorization
// header already exists and no action is taken if the Exporter is not configured with
// basic authorization credentials.
func (e *Exporter) addBasicAuth(req *http.Request) error {
	// No need to add basic auth if it isn't provided or if the Authorization header is
	// already set.
	if _, exists := e.config.Headers["Authorization"]; exists {
		return nil
	}
	if e.config.BasicAuth == nil {
		return nil
	}

	username := e.config.BasicAuth["username"]

	// Use password from password file if it exists.
	passwordFile := e.config.BasicAuth["password_file"]
	if passwordFile != "" {
		file, err := ioutil.ReadFile(passwordFile)
		if err != nil {
			return ErrFailedToReadFile
		}
		password := string(file)
		req.SetBasicAuth(username, password)
		return nil
	}

	// Use provided password.
	password := e.config.BasicAuth["password"]
	req.SetBasicAuth(username, password)

	return nil
}

// addBearerTokenAuth sets the Authorization header for bearer tokens using a bearer token
// string or a bearer token file. The header value is not changed if an Authorization
// header already exists and no action is taken if the Exporter is not configured with
// bearer token credentials.
func (e *Exporter) addBearerTokenAuth(req *http.Request) error {
	// No need to add bearer token auth if the Authorization header is already set.
	if _, exists := e.config.Headers["Authorization"]; exists {
		return nil
	}

	// Use bearer token from bearer token file if it exists.
	if e.config.BearerTokenFile != "" {
		file, err := ioutil.ReadFile(e.config.BearerTokenFile)
		if err != nil {
			return ErrFailedToReadFile
		}
		bearerTokenString := "Bearer " + string(file)
		req.Header.Set("Authorization", bearerTokenString)
		return nil
	}

	// Otherwise, use bearer token field.
	if e.config.BearerToken != "" {
		bearerTokenString := "Bearer " + e.config.BearerToken
		req.Header.Set("Authorization", bearerTokenString)
	}

	return nil
}

// buildClient returns a http client that uses TLS and has the user-specified proxy and
// timeout.
func (e *Exporter) buildClient() (*http.Client, error) {
	client := http.Client{
		Timeout: e.config.RemoteTimeout,
	}
	return &client, nil
}
