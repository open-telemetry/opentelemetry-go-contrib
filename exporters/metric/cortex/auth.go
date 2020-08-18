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

var (
	// ErrNoBasicAuthUsername occurs when no username was provided for basic
	// authentication.
	ErrNoBasicAuthUsername = fmt.Errorf("No username provided for basic authentication")

	// ErrNoBasicAuthPassword occurs when no password or password file was provided for
	// basic authentication.
	ErrNoBasicAuthPassword = fmt.Errorf("No password or password file provided for basic authentication")

	// ErrFailedToReadFile occurs when a password / bearer token file exists, but could
	// not be read.
	ErrFailedToReadFile = fmt.Errorf("Failed to read password / bearer token file")
)

// addBasicAuth sets the Authorization header for basic authentication using a username
// and a password / password file. To prevent the Exporter from potentially opening a
// password file on every request by calling this method, the Authorization header is also
// added to the Config header map.
func (e *Exporter) addBasicAuth(req *http.Request) error {
	// No need to add basic auth if it isn't provided or if the Authorization header is
	// already set.
	if _, exists := e.config.Headers["Authorization"]; exists {
		return nil
	}
	if e.config.BasicAuth == nil {
		return nil
	}

	// There must be an username for basic authentication.
	username := e.config.BasicAuth["username"]
	if username == "" {
		return ErrNoBasicAuthUsername
	}

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
	if password == "" {
		return ErrNoBasicAuthPassword
	}
	req.SetBasicAuth(username, password)

	return nil
}
