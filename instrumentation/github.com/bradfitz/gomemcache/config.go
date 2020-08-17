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

package gomemcache

import (
	oteltrace "go.opentelemetry.io/otel/api/trace"
)

type config struct {
	serviceName   string
	traceProvider oteltrace.Provider
}

// Option is used to configure the client.
type Option func(*config)

// WithTracer configures the client with the provided trace provider.
func WithTraceProvider(traceProvider oteltrace.Provider) Option {
	return func(cfg *config) {
		cfg.traceProvider = traceProvider
	}
}

// WithServiceName sets the service name.
func WithServiceName(serviceName string) Option {
	return func(cfg *config) {
		cfg.serviceName = serviceName
	}
}
