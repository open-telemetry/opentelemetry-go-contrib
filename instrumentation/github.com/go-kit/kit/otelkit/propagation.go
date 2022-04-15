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

package otelkit // import "go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit"

import (
	"context"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"google.golang.org/grpc/metadata"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// GrpcPropagationMiddleware uses gRPC metadata from the incoming context,
// if it exists, and extracts the traceparent to propagate context information
// that enables distributed tracing.
func GRPCPropagationMiddleware() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			if md, ok := metadata.FromIncomingContext(ctx); ok {
				ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(md))
			}
			return next(ctx, request)
		}
	}
}

// HTTPPropagationMiddleware uses HTTP header from the incoming request,
// if it exists, and extracts the traceparent to propagate context information
// that enables distributed tracing.
func HTTPPropagationMiddleware() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			if r, ok := request.(*http.Request); ok {
				ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(r.Header))
			}
			return next(ctx, request)
		}
	}
}
