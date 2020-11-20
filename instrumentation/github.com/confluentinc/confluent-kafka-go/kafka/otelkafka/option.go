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

package otelkafka

import (
	"context"

	otelcontrib "go.opentelemetry.io/contrib"
	"go.opentelemetry.io/otel"
	otelglobal "go.opentelemetry.io/otel/api/global"
	oteltrace "go.opentelemetry.io/otel/api/trace"
)

const (
	tracerName = "go.opentelemetry.io/contrib/instrumentation/github.com/confluentinc/confluent-kafka-go/kafka/otelkafka"
)

type config struct {
	Tracer         oteltrace.Tracer
	TracerProvider oteltrace.TracerProvider
	Propagators    otel.TextMapPropagator
	ctx            context.Context
}

func newConfig(opts ...Option) *config {
	cfg := &config{
		ctx: context.Background(),
	}

	if cfg.TracerProvider == nil {
		cfg.TracerProvider = otelglobal.TracerProvider()
	}

	if cfg.Propagators == nil {
		cfg.Propagators = otelglobal.TextMapPropagator()
	}

	for _, opt := range opts {
		opt(cfg)
	}

	cfg.Tracer = cfg.TracerProvider.Tracer(
		tracerName,
		oteltrace.WithInstrumentationVersion(otelcontrib.SemVersion()),
	)
	return cfg
}

// Option specifies instrumentation configuration options.
type Option func(*config)

// WithPropagators specifies propagators to use for extracting If none are specified, global
// ones will be used.
func WithPropagators(propagators otel.TextMapPropagator) Option {
	return func(cfg *config) {
		cfg.Propagators = propagators
	}
}

// WithTracerProvider specifies a tracer provider to use for creating a tracer.
// If none is specified, the global provider is used.
func WithTracerProvider(provider oteltrace.TracerProvider) Option {
	return func(cfg *config) {
		cfg.TracerProvider = provider
	}
}

// WithContext sets the config context to ctx.
func WithContext(ctx context.Context) Option {
	return func(cfg *config) {
		cfg.ctx = ctx
	}
}
