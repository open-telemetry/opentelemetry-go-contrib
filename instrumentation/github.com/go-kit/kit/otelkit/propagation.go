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

package otelkit

import (
	"context"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"google.golang.org/grpc/metadata"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type MultimapCarrier map[string][]string

// Compile time check that MultimapCarrier implements the TextMapCarrier.
var _ propagation.TextMapCarrier = &MultimapCarrier{}

// Get returns the value associated with the passed key.
func (c *MultimapCarrier) Get(key string) string {
	v := (*c)[key]
	if len(v) > 0 {
		return v[0]
	}
	return ""
}

// Set stores the key-value pair.
func (c *MultimapCarrier) Set(key, value string) {
	(*c)[key] = []string{value}
}

// Keys lists the keys stored in this carrier.
func (c *MultimapCarrier) Keys() []string {
	keys := make([]string, 0, len(*c))
	for k := range *c {
		keys = append(keys, k)
	}
	return keys
}

// GrpcPropagationMiddleware uses gRPC metadata from the incoming context,
// if it exists, and extracts the traceparent to propagate context information
// that enables distributed tracing.
func GrpcPropagationMiddleware() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			if md, ok := metadata.FromIncomingContext(ctx); ok {
				ctx = otel.GetTextMapPropagator().Extract(ctx, (*MultimapCarrier)(&md))
			}
			return next(ctx, request)
		}
	}
}

// HttpPropagationMiddleware uses HTTP header from the incoming request,
// if it exists, and extracts the traceparent to propagate context information
// that enables distributed tracing.
func HttpPropagationMiddleware() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			if r, ok := request.(*http.Request); ok {
				ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(r.Header))
			}
			return next(ctx, request)
		}
	}
}
