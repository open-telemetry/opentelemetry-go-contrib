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
	defaultTracerName = "go.opentelemetry.io/contrib/instrumentations/go.mongodb.org/mongo-driver"
)

// Config is used to configure the mongo tracer.
type Config struct {
	Tracer trace.Tracer
}

// newConfig returns a Config with all Options set.
func newConfig(opts ...Option) Config {
	cfg := Config{}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.Tracer == nil {
		cfg.Tracer = global.Tracer(defaultTracerName)
	}
	return cfg
}

// Option specifies instrumentation configuration options.
type Option func(*Config)

// WithTracer specifies a tracer to use for creating spans. If none is
// specified, a tracer named
// "go.opentelemetry.io/contrib/instrumentations/go.mongodb.org/mongo-driver"
// from the global provider is used.
func WithTracer(tracer trace.Tracer) Option {
	return func(cfg *Config) {
		cfg.Tracer = tracer
	}
}
