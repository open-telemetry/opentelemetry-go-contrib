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

package grpc

import (
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/propagation"
	"go.opentelemetry.io/otel/api/trace"
)

const (
	defaultTracerName = "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc"
)

// Option specifies instrumentation configuration options.
type Option func(*config)

type config struct {
	tracer      trace.Tracer
	propagators propagation.Propagators
}

func newConfig(opts []Option) *config {
	c := &config{propagators: global.Propagators()}
	for _, o := range opts {
		o(c)
	}
	if c.tracer == nil {
		c.tracer = global.Tracer(defaultTracerName)
	}
	return c
}

// WithPropagators sets the propagators to use for Extraction and Injection
func WithPropagators(props propagation.Propagators) Option {
	return func(c *config) {
		c.propagators = props
	}
}

// WithTracer specifies a tracer to use for creating spans. If none is
// specified, a tracer named
// "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc"
// from the global provider is used.
func WithTracer(tracer trace.Tracer) Option {
	return func(cfg *config) {
		cfg.tracer = tracer
	}
}
