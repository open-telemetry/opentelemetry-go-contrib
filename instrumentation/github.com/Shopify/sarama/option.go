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

package sarama

import (
	"go.opentelemetry.io/otel/api/global"
	otelpropagation "go.opentelemetry.io/otel/api/propagation"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
)

const (
	defaultTracerName = "go.opentelemetry.io/contrib/instrumentation/github.com/Shopify/sarama"

	kafkaPartitionKey = label.Key("messaging.kafka.partition")
)

type config struct {
	TracerProvider trace.Provider
	Propagators    otelpropagation.Propagators

	Tracer trace.Tracer
}

// newConfig returns a config with all Options set.
func newConfig(opts ...Option) config {
	cfg := config{
		Propagators:    global.Propagators(),
		TracerProvider: global.TraceProvider(),
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	cfg.Tracer = cfg.TracerProvider.Tracer(defaultTracerName)

	return cfg
}

// Option specifies instrumentation configuration options.
type Option func(*config)

// WithTracerProvider specifies a tracer provider to use for creating a tracer.
// If none is specified, the global provider is used.
func WithTracerProvider(provider trace.Provider) Option {
	return func(cfg *config) {
		cfg.TracerProvider = provider
	}
}

// WithPropagators specifies propagators to use for extracting
// information from the HTTP requests. If none are specified, global
// ones will be used.
func WithPropagators(propagators otelpropagation.Propagators) Option {
	return func(cfg *config) {
		cfg.Propagators = propagators
	}
}
