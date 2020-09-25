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

package otelgrpc

import (
	"context"

	"google.golang.org/grpc/metadata"

	"go.opentelemetry.io/otel/api/baggage"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/propagation"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
)

// Option is a function that allows configuration of the grpc Extract()
// and Inject() functions
type Option func(*config)

type config struct {
	propagators propagation.Propagators
}

func newConfig(opts []Option) *config {
	c := &config{propagators: global.Propagators()}
	for _, o := range opts {
		o(c)
	}
	return c
}

// WithPropagators sets the propagators to use for Extraction and Injection
func WithPropagators(props propagation.Propagators) Option {
	return func(c *config) {
		c.propagators = props
	}
}

type metadataSupplier struct {
	metadata *metadata.MD
}

func (s *metadataSupplier) Get(key string) string {
	values := s.metadata.Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func (s *metadataSupplier) Set(key string, value string) {
	s.metadata.Set(key, value)
}

// Inject injects correlation context and span context into the gRPC
// metadata object. This function is meant to be used on outgoing
// requests.
func Inject(ctx context.Context, metadata *metadata.MD, opts ...Option) {
	c := newConfig(opts)
	propagation.InjectHTTP(ctx, c.propagators, &metadataSupplier{
		metadata: metadata,
	})
}

// Extract returns the correlation context and span context that
// another service encoded in the gRPC metadata object with Inject.
// This function is meant to be used on incoming requests.
func Extract(ctx context.Context, metadata *metadata.MD, opts ...Option) ([]label.KeyValue, trace.SpanContext) {
	c := newConfig(opts)
	ctx = propagation.ExtractHTTP(ctx, c.propagators, &metadataSupplier{
		metadata: metadata,
	})

	spanContext := trace.RemoteSpanContextFromContext(ctx)
	var correlationCtxLabels []label.KeyValue
	baggage.MapFromContext(ctx).Foreach(func(l label.KeyValue) bool {
		correlationCtxLabels = append(correlationCtxLabels, l)
		return true
	})

	return correlationCtxLabels, spanContext
}
