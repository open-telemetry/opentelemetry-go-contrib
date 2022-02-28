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

package cortex_test

import (
	"time"

	"go.opentelemetry.io/contrib/exporters/metric/cortex" // nolint:staticcheck // allow import of deprecated pkg.
)

// Config struct with default values. This is used to verify the output of Validate().
var validatedStandardConfig = cortex.Config{
	Endpoint:      "/api/prom/push",
	Name:          "Config",
	RemoteTimeout: 30 * time.Second,
	PushInterval:  10 * time.Second,
	Quantiles:     []float64{0.5, 0.9, 0.95, 0.99},
}

// Config struct with default values other than the remote timeout. This is used to verify
// the output of Validate().
var validatedCustomTimeoutConfig = cortex.Config{
	Endpoint:      "/api/prom/push",
	Name:          "Config",
	RemoteTimeout: 10 * time.Second,
	PushInterval:  10 * time.Second,
	Quantiles:     []float64{0.5, 0.9, 0.95, 0.99},
}

// Config struct with default values other than the quantiles. This is used to verify
// the output of Validate().
var validatedQuantilesConfig = cortex.Config{
	Endpoint:      "/api/prom/push",
	Name:          "Config",
	RemoteTimeout: 30 * time.Second,
	PushInterval:  10 * time.Second,
	Quantiles:     []float64{0, 0.5, 1},
}

// Example Config struct with a custom remote timeout.
var exampleRemoteTimeoutConfig = cortex.Config{
	Endpoint:      "/api/prom/push",
	Name:          "Config",
	PushInterval:  10 * time.Second,
	RemoteTimeout: 10 * time.Second,
}

// Example Config struct without a remote timeout.
var exampleNoRemoteTimeoutConfig = cortex.Config{
	Endpoint:     "/api/prom/push",
	Name:         "Config",
	PushInterval: 10 * time.Second,
}

// Example Config struct without a push interval.
var exampleNoPushIntervalConfig = cortex.Config{
	Endpoint:      "/api/prom/push",
	Name:          "Config",
	RemoteTimeout: 30 * time.Second,
}

// Example Config struct without a http client.
var exampleNoClientConfig = cortex.Config{
	Endpoint:      "/api/prom/push",
	Name:          "Config",
	RemoteTimeout: 30 * time.Second,
	PushInterval:  10 * time.Second,
}

// Example Config struct without an endpoint.
var exampleNoEndpointConfig = cortex.Config{
	Name:          "Config",
	RemoteTimeout: 30 * time.Second,
	PushInterval:  10 * time.Second,
}

// Example Config struct with two bearer tokens.
var exampleTwoBearerTokenConfig = cortex.Config{
	Endpoint:        "/api/prom/push",
	Name:            "Config",
	RemoteTimeout:   30 * time.Second,
	PushInterval:    10 * time.Second,
	BearerToken:     "bearer_token",
	BearerTokenFile: "bearer_token_file",
}

// Example Config struct with two passwords.
var exampleTwoPasswordConfig = cortex.Config{
	Endpoint:      "/api/prom/push",
	Name:          "Config",
	RemoteTimeout: 30 * time.Second,
	PushInterval:  10 * time.Second,
	BasicAuth: map[string]string{
		"username":      "user",
		"password":      "password",
		"password_file": "passwordFile",
	},
}

// Example Config struct with both basic auth and bearer token authentication.
var exampleTwoAuthConfig = cortex.Config{
	Endpoint:      "/api/prom/push",
	Name:          "Config",
	RemoteTimeout: 30 * time.Second,
	PushInterval:  10 * time.Second,
	BasicAuth: map[string]string{
		"username": "user",
		"password": "password",
	},
	BearerToken: "bearer_token",
}

// Example Config struct with no password for basic authentication.
var exampleNoPasswordConfig = cortex.Config{
	Endpoint:      "/api/prom/push",
	Name:          "Config",
	RemoteTimeout: 30 * time.Second,
	PushInterval:  10 * time.Second,
	BasicAuth: map[string]string{
		"username": "user",
	},
}

// Example Config struct with no password for basic authentication.
var exampleNoUsernameConfig = cortex.Config{
	Endpoint:      "/api/prom/push",
	Name:          "Config",
	RemoteTimeout: 30 * time.Second,
	PushInterval:  10 * time.Second,
	BasicAuth: map[string]string{
		"password": "password",
	},
}

// Example Config struct with invalid quantiles.
var exampleInvalidQuantilesConfig = cortex.Config{
	Endpoint:      "/api/prom/push",
	Name:          "Config",
	RemoteTimeout: 30 * time.Second,
	PushInterval:  10 * time.Second,
	Quantiles:     []float64{0, 1, 2, 3},
}

// Example Config struct with valid quantiles.
var exampleValidQuantilesConfig = cortex.Config{
	Endpoint:      "/api/prom/push",
	Name:          "Config",
	RemoteTimeout: 30 * time.Second,
	PushInterval:  10 * time.Second,
	Quantiles:     []float64{0, 0.5, 1},
}
