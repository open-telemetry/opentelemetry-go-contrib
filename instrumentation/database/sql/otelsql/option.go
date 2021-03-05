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

package otelsql

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Option is the interface that applies a configuration option.
type Option interface {
	// Apply sets the Option value of a config.
	Apply(*config)
}

var _ Option = OptionFunc(nil)

// OptionFunc implements the Option interface.
type OptionFunc func(*config)

func (f OptionFunc) Apply(c *config) {
	f(c)
}

// WithTracerProvider specifies a tracer provider to use for creating a tracer.
// If none is specified, the global provider is used.
func WithTracerProvider(provider trace.TracerProvider) Option {
	return OptionFunc(func(cfg *config) {
		cfg.TracerProvider = provider
	})
}

// WithAttributes specifies attributes that will be set to each span.
func WithAttributes(attributes ...attribute.KeyValue) Option {
	return OptionFunc(func(cfg *config) {
		cfg.Attributes = attributes
	})
}

// WithSpanNameFormatter takes an interface that will be called on every
// operation and the returned string will become the span name.
func WithSpanNameFormatter(spanNameFormatter SpanNameFormatter) Option {
	return OptionFunc(func(cfg *config) {
		cfg.SpanNameFormatter = spanNameFormatter
	})
}

// WithSpanOptions specifies configuration for span to decide whether to enable some features.
func WithSpanOptions(opts SpanOptions) Option {
	return OptionFunc(func(cfg *config) {
		cfg.SpanOptions = opts
	})
}
