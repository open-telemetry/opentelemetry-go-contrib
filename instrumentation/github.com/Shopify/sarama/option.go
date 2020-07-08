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
	"go.opentelemetry.io/otel/api/kv"
	otelpropagation "go.opentelemetry.io/otel/api/propagation"
	"go.opentelemetry.io/otel/api/trace"
)

const (
	defaultTracerName = "go.opentelemetry.io/contrib/instrumentation/github.com/Shopify/sarama"

	kafkaPartitionKey = kv.Key("messaging.kafka.partition")
)

type config struct {
	ServiceName string
	Tracer      trace.Tracer
	Propagators otelpropagation.Propagators
}

// newConfig returns a config with all Options set.
func newConfig(serviceName string, opts ...Option) config {
	cfg := config{Propagators: global.Propagators(), ServiceName: serviceName}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.Tracer == nil {
		cfg.Tracer = global.Tracer(defaultTracerName)
	}
	return cfg
}

// Option specifies instrumentation configuration options.
type Option func(*config)

// WithTracer specifies a tracer to use for creating spans. If none is
// specified, a tracer named
// "go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin"
// from the global provider is used.
func WithTracer(tracer trace.Tracer) Option {
	return func(cfg *config) {
		cfg.Tracer = tracer
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
