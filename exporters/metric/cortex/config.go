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
	"net/http"
	"time"
)

var (
	// ErrTwoPasswords occurs when the YAML file contains both `password` and `password_file`.
	ErrTwoPasswords = fmt.Errorf("Cannot have two passwords in the YAML file")

	// ErrTwoBearerTokens occurs when the YAML file contains `bearer_token` and `bearer_token_file`.
	ErrTwoBearerTokens = fmt.Errorf("Cannot have two bearer tokens in the YAML file")
)

// Config contains properties the Exporter uses to export metrics data to Cortex.
type Config struct {
	Endpoint        string
	RemoteTimeout   time.Duration
	Name            string
	BasicAuth       map[string]string
	BearerToken     string
	BearerTokenFile string
	TLSConfig       map[string]string
	ProxyURL        string
	PushInterval    time.Duration
	Headers         map[string]string
	Client          *http.Client
}

// Validate checks a Config struct for missing required properties and property conflicts.
// Additionally, it adds default values to missing properties when there is a default.
func (c *Config) Validate() error {
	return nil
}
