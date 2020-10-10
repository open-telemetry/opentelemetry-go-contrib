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

package otelhttptrace

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/semconv"
)

// Option is a function that allows configuration of the httptrace Extract()
// and Inject() functions
type Option func(*config)

type config struct {
	propagators otel.TextMapPropagator
}

func newConfig(opts []Option) *config {
	c := &config{propagators: global.TextMapPropagator()}
	for _, o := range opts {
		o(c)
	}
	return c
}

// WithPropagators sets the propagators to use for Extraction and Injection
func WithPropagators(props otel.TextMapPropagator) Option {
	return func(c *config) {
		c.propagators = props
	}
}

// Returns the Attributes, Context Entries, and SpanContext that were encoded by Inject.
func Extract(ctx context.Context, req *http.Request, opts ...Option) ([]label.KeyValue, []label.KeyValue, trace.SpanContext) {
	c := newConfig(opts)
	ctx = c.propagators.Extract(ctx, req.Header)

	attrs := append(
		semconv.HTTPServerAttributesFromHTTPRequest("", "", req),
		semconv.NetAttributesFromHTTPRequest("tcp", req)...,
	)

	labels := otel.Baggage(ctx)
	return attrs, (&labels).ToSlice(), trace.RemoteSpanContextFromContext(ctx)
}

func Inject(ctx context.Context, req *http.Request, opts ...Option) {
	c := newConfig(opts)
	c.propagators.Inject(ctx, req.Header)
}
