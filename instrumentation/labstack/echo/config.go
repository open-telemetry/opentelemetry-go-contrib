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

package echo

import (
	otelpropagation "go.opentelemetry.io/otel/api/propagation"
	oteltrace "go.opentelemetry.io/otel/api/trace"
)

// Config is used to configure the mux middleware.
type Config struct {
	Tracer      oteltrace.Tracer
	Propagators otelpropagation.Propagators
}

// Option specifies instrumentation configuration options.
type Option func(*Config)

// WithTracer specifies a tracer to use for creating spans. If none is
// specified, a tracer named
// "go.opentelemetry.io/contrib/instrumentation/labstack/echo" from the global
// provider is used.
func WithTracer(tracer oteltrace.Tracer) Option {
	return func(cfg *Config) {
		cfg.Tracer = tracer
	}
}

// WithPropagators specifies propagators to use for extracting
// information from the HTTP requests. If none are specified, global
// ones will be used.
func WithPropagators(propagators otelpropagation.Propagators) Option {
	return func(cfg *Config) {
		cfg.Propagators = propagators
	}
}
