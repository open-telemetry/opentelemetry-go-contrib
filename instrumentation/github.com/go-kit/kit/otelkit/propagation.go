package otelkit

import (
	"context"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc/metadata"
)

type MultimapCarrier map[string][]string

// Compile time check that MapCarrier implements the TextMapCarrier.
var _ propagation.TextMapCarrier = MultimapCarrier{}

// Get returns the value associated with the passed key.
func (c MultimapCarrier) Get(key string) string {
	v := c[key]
	if len(v) > 0 {
		return v[0]
	}
	return ""
}

// Set stores the key-value pair.
func (c MultimapCarrier) Set(key, value string) {
	c[key] = []string{value}
}

// Keys lists the keys stored in this carrier.
func (c MultimapCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
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
				ctx = otel.GetTextMapPropagator().Extract(ctx, MultimapCarrier(md))
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
