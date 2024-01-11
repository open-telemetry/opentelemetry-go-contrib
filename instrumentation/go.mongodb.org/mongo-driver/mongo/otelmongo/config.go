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

package otelmongo // import "go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// ScopeName is the instrumentation scope name.
const ScopeName = "go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"

// config is used to configure the mongo tracer.
type config struct {
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider

	Tracer trace.Tracer
	Meter  metric.Meter

	CommandAttributeDisabled bool
}

// newConfig returns a config with all Options set.
func newConfig(opts ...Option) config {
	cfg := config{
		TracerProvider:           otel.GetTracerProvider(),
		MeterProvider:            otel.GetMeterProvider(),
		CommandAttributeDisabled: true,
	}
	for _, opt := range opts {
		opt.apply(&cfg)
	}

	cfg.Tracer = cfg.TracerProvider.Tracer(
		ScopeName,
		trace.WithInstrumentationVersion(Version()),
	)
	cfg.Meter = cfg.MeterProvider.Meter(
		ScopeName,
		metric.WithInstrumentationVersion(Version()),
	)
	return cfg
}

// Option specifies instrumentation configuration options.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (o optionFunc) apply(c *config) {
	o(c)
}

// WithTracerProvider specifies a tracer provider to use for creating a tracer.
// If none is specified, the global provider is used.
func WithTracerProvider(provider trace.TracerProvider) Option {
	return optionFunc(func(cfg *config) {
		if provider != nil {
			cfg.TracerProvider = provider
		}
	})
}

// WithMeterProvider specifies a meter provider to use for recording metrics.
// If none is specified, the global provider is used.
func WithMeterProvider(provider metric.MeterProvider) Option {
	return optionFunc(func(cfg *config) {
		if provider != nil {
			cfg.MeterProvider = provider
		}
	})
}

// WithCommandAttributeDisabled specifies if the MongoDB command is added as an attribute to Spans or not.
// This is disabled by default and the MongoDB command will not be added as an attribute
// to Spans if this option is not provided.
func WithCommandAttributeDisabled(disabled bool) Option {
	return optionFunc(func(cfg *config) {
		cfg.CommandAttributeDisabled = disabled
	})
}
