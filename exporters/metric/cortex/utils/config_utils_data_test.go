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

package utils_test

import (
	"time"

	"go.opentelemetry.io/contrib/exporters/metric/cortex" // nolint:staticcheck // allow import of deprecated pkg.
)

// This is an example YAML file that produces a Config struct without errors.
var validYAML = []byte(`url: /api/prom/push
remote_timeout: 30s
push_interval: 5s
name: Valid Config Example
basic_auth:
  username: user
  password: password
tls_config:
  ca_file: cafile
  cert_file: certfile
  key_file: keyfile
  server_name: server
  insecure_skip_verify: true
headers:
  test: header 
`)

// YAML file with no remote_timout property. It should produce a Config struct without
// errors since a default remote_timeout will be added.
var noTimeoutYAML = []byte(`url: /api/prom/push
push_interval: 5s
name: Valid Config Example
basic_auth:
  username: user
  password: password
tls_config:
  ca_file: cafile
  cert_file: certfile
  key_file: keyfile
  server_name: server
  insecure_skip_verify: true
headers:
  test: header
`)

// YAML file with no Cortex endpoint. It should produce a Config struct without errors
// since the endpoint will be set to "/api/prom/push" by default.
var noEndpointYAML = []byte(`remote_timeout: 30s
push_interval: 5s
name: Valid Config Example
basic_auth:
  username: user
  password: password
tls_config:
  ca_file: cafile
  cert_file: certfile
  key_file: keyfile
  server_name: server
  insecure_skip_verify: true
headers:
  test: header
`)

// YAML file with both password and password_file properties. It should fail to produce a
// Config struct since password and password_file are mutually exclusive.
var twoPasswordsYAML = []byte(`url: /api/prom/push
remote_timeout: 30s
name: Valid Config Example
basic_auth:
  username: user
  password: password
  password_file: passwordfile
tls_config:
  ca_file: cafile
  cert_file: certfile
  key_file: keyfile
  server_name: server
  insecure_skip_verify: true
headers:
  test: header
`)

// YAML file with both bearer_token and bearer_token_file properties. It should fail to
// produce a Config struct since bearer_token and bearer_token_file are mutually
// exclusive.
var twoBearerTokensYAML = []byte(`url: /api/prom/push
remote_timeout: 30s
name: Valid Config Example
bearer_token: qwerty12345
bearer_token_file: bearertokenfile
tls_config:
  ca_file: cafile
  cert_file: certfile
  key_file: keyfile
  server_name: server
  insecure_skip_verify: true
headers:
  test: header
`)

// YAML file that sets custom quantiles and produces a Config struct without errors.
var quantilesYAML = []byte(`url: /api/prom/push
remote_timeout: 30s
push_interval: 5s
name: Valid Config Example
basic_auth:
  username: user
  password: password
tls_config:
  ca_file: cafile
  cert_file: certfile
  key_file: keyfile
  server_name: server
  insecure_skip_verify: true
headers:
  test: header 
quantiles:
  - 0
  - 0.5
  - 1
`)

// YAML file that sets custom histogram bucket boundaries and produces a Config struct
// without errors.
var bucketBoundariesYAML = []byte(`url: /api/prom/push
remote_timeout: 30s
push_interval: 5s
name: Valid Config Example
basic_auth:
  username: user
  password: password
tls_config:
  ca_file: cafile
  cert_file: certfile
  key_file: keyfile
  server_name: server
  insecure_skip_verify: true
headers:
  test: header 
histogram_boundaries:
  - 100
  - 300
  - 500
`)

// ValidConfig is the resulting Config struct from reading validYAML.
var validConfig = cortex.Config{
	Endpoint:      "/api/prom/push",
	RemoteTimeout: 30 * time.Second,
	Name:          "Valid Config Example",
	BasicAuth: map[string]string{
		"username": "user",
		"password": "password",
	},
	BearerToken:     "",
	BearerTokenFile: "",
	TLSConfig: map[string]string{
		"ca_file":              "cafile",
		"cert_file":            "certfile",
		"key_file":             "keyfile",
		"server_name":          "server",
		"insecure_skip_verify": "1",
	},
	ProxyURL:     nil,
	PushInterval: 5 * time.Second,
	Headers: map[string]string{
		"test": "header",
	},
	Quantiles: []float64{0.5, 0.9, 0.95, 0.99},
}

// customQuantilesConfig is the resulting Config struct from reading quantilesYAML.
var customQuantilesConfig = cortex.Config{
	Endpoint:      "/api/prom/push",
	RemoteTimeout: 30 * time.Second,
	Name:          "Valid Config Example",
	BasicAuth: map[string]string{
		"username": "user",
		"password": "password",
	},
	BearerToken:     "",
	BearerTokenFile: "",
	TLSConfig: map[string]string{
		"ca_file":              "cafile",
		"cert_file":            "certfile",
		"key_file":             "keyfile",
		"server_name":          "server",
		"insecure_skip_verify": "1",
	},
	ProxyURL:     nil,
	PushInterval: 5 * time.Second,
	Headers: map[string]string{
		"test": "header",
	},
	Quantiles: []float64{0, 0.5, 1},
}

// customBucketBoundariesConfig is the resulting Config struct from reading
// bucketBoundariesYAML.
var customBucketBoundariesConfig = cortex.Config{
	Endpoint:      "/api/prom/push",
	RemoteTimeout: 30 * time.Second,
	Name:          "Valid Config Example",
	BasicAuth: map[string]string{
		"username": "user",
		"password": "password",
	},
	BearerToken:     "",
	BearerTokenFile: "",
	TLSConfig: map[string]string{
		"ca_file":              "cafile",
		"cert_file":            "certfile",
		"key_file":             "keyfile",
		"server_name":          "server",
		"insecure_skip_verify": "1",
	},
	ProxyURL:     nil,
	PushInterval: 5 * time.Second,
	Headers: map[string]string{
		"test": "header",
	},
	Quantiles:           []float64{0.5, 0.9, 0.95, 0.99},
	HistogramBoundaries: []float64{100, 300, 500},
}
