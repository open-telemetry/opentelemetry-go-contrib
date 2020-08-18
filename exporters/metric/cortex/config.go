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
	// ErrTwoPasswords occurs when the YAML file contains both `password` and
	// `password_file`.
	ErrTwoPasswords = fmt.Errorf("Cannot have two passwords in the YAML file")

	// ErrTwoBearerTokens occurs when the YAML file contains both `bearer_token` and
	// `bearer_token_file`.
	ErrTwoBearerTokens = fmt.Errorf("Cannot have two bearer tokens in the YAML file")

	// ErrConflictingAuthorization occurs when the YAML file contains both BasicAuth and
	// bearer token authorization
	ErrConflictingAuthorization = fmt.Errorf("Cannot have both basic auth and bearer token authorization")
)

// Config contains properties the Exporter uses to export metrics data to Cortex.
type Config struct {
	Endpoint            string            `mapstructure:"url"`
	RemoteTimeout       time.Duration     `mapstructure:"remote_timeout"`
	Name                string            `mapstructure:"name"`
	BasicAuth           map[string]string `mapstructure:"basic_auth"`
	BearerToken         string            `mapstructure:"bearer_token"`
	BearerTokenFile     string            `mapstructure:"bearer_token_file"`
	TLSConfig           map[string]string `mapstructure:"tls_config"`
	ProxyURL            string            `mapstructure:"proxy_url"`
	PushInterval        time.Duration     `mapstructure:"push_interval"`
	Quantiles           []float64         `mapstructure:"quantiles"`
	HistogramBoundaries []float64         `mapstructure:"histogram_boundaries"`
	Headers             map[string]string `mapstructure:"headers"`
	Client              *http.Client
}

// Validate checks a Config struct for missing required properties and property conflicts.
// Additionally, it adds default values to missing properties when there is a default.
func (c *Config) Validate() error {
	// Check for mutually exclusive properties.
	if c.BasicAuth != nil {
		if c.BearerToken != "" || c.BearerTokenFile != "" {
			return ErrConflictingAuthorization
		}
		if c.BasicAuth["password"] != "" && c.BasicAuth["password_file"] != "" {
			return ErrTwoPasswords
		}
	}
	if c.BearerToken != "" && c.BearerTokenFile != "" {
		return ErrTwoBearerTokens
	}

	// Add default values for missing properties.
	if c.Endpoint == "" {
		c.Endpoint = "/api/prom/push"
	}
	if c.RemoteTimeout == 0 {
		c.RemoteTimeout = 30 * time.Second
	}
	// Default time interval between pushes for the push controller is 10s.
	if c.PushInterval == 0 {
		c.PushInterval = 10 * time.Second
	}

	return nil
}
