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

package xray

import (
	"time"
)

const (
	defaultProxyEndpoint = "127.0.0.1:2000"
	defaultPollingInterval = 300
)

// SamplerOption is a function that sets config on the sampler
type SamplerOption func(options *samplerOptions)

type samplerOptions struct {
	proxyEndpoint       			string
	samplingRulesPollingInterval 	time.Duration
}

// sets custom proxy endpoint
func WithProxyEndpoint(proxyEndpoint string) SamplerOption {
	return func(o *samplerOptions) {
		o.proxyEndpoint = proxyEndpoint
	}
}

// sets polling interval for sampling rules
func WithSamplingRulesPollingInterval(polingInterval time.Duration) SamplerOption {
	return func(o *samplerOptions) {
		o.samplingRulesPollingInterval = defaultPollingInterval
	}
}

func (o *samplerOptions) applyOptionsAndDefaults(opts ...SamplerOption) *samplerOptions {
	for _, option := range opts {
		option(o)
	}

	if o.proxyEndpoint == "" {
		o.proxyEndpoint = defaultProxyEndpoint
	}
	if o.samplingRulesPollingInterval <= 0 {
		o.samplingRulesPollingInterval = defaultPollingInterval * time.Second
	}

	return o
}
