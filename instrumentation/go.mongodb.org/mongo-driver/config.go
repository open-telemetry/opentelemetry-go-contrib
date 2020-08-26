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

package mongo

import (
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
)

const (
	defaultTracerName = "go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver"
)

// Config is used to configure the mongo tracer.
type Config struct {
	TraceProvider trace.Provider

	Tracer trace.Tracer
}

// newConfig returns a Config with all Options set.
func newConfig(opts ...Option) Config {
	cfg := Config{
		TraceProvider: global.TraceProvider(),
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	cfg.Tracer = cfg.TraceProvider.Tracer(defaultTracerName)
	return cfg
}

// Option specifies instrumentation configuration options.
type Option func(*Config)

// WithTracerProvider specifies a tracer provider to use for creating a tracer.
// If none is specified, the global provider is used.
func WithTracerProvider(provider trace.Provider) Option {
	return func(cfg *Config) {
		cfg.TraceProvider = provider
	}
}
