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
	"net"
	"net/http"
	"net/url"
	"time"
)

var (
	// ErrTwoPasswords occurs when the YAML file contains both `password` and
	// `password_file`.
	ErrTwoPasswords = fmt.Errorf("Cannot have two passwords in the YAML file")

	// ErrTwoBearerTokens occurs when the YAML file contains both `bearer_token` and
	// `bearer_token_file`.
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
	// Check for mutually exclusive properties.
	if c.BearerToken != "" && c.BearerTokenFile != "" {
		return ErrTwoBearerTokens
	}
	if c.BasicAuth["password"] != "" && c.BasicAuth["password_file"] != "" {
		return ErrTwoPasswords
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
	if c.Client == nil && c.ProxyURL != "" {
		parsedProxyURL, err := url.Parse(c.ProxyURL)
		if err != nil {
			return err
		}

		// This is the same as http.DefaultClient and http.DefaultTransport other than the
		// timeout and proxy.
		c.Client = &http.Client{
			Timeout: c.RemoteTimeout,
			Transport: &http.Transport{
				Proxy: http.ProxyURL(parsedProxyURL),
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
					DualStack: true,
				}).DialContext,
				ForceAttemptHTTP2:     true,
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
		}
	}
	if c.Client == nil {
		c.Client = &http.Client{
			Timeout: c.RemoteTimeout,
		}
	}

	return nil
}
