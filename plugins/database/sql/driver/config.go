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

package driver

import (
	oteltrace "go.opentelemetry.io/otel/api/trace"
)

// Config is used to configure the tracing database driver.
type Config struct {
	Tracer oteltrace.Tracer
}

// Option specifies instrumentation configuration options.
type Option func(*Config)

// WithTracer specifies a tracer to use for creating spans. If none is
// specified, a tracer named
// "go.opentelemetry.io/contrib/plugins/database/sql/driver" from the
// global provider is used.
func WithTracer(tracer oteltrace.Tracer) Option {
	return func(cfg *Config) {
		cfg.Tracer = tracer
	}
}
