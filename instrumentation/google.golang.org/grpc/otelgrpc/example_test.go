// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgrpc_test

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

func ExampleNewClientHandler() {
	_, _ = grpc.NewClient("localhost", grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
}

func ExampleNewServerHandler() {
	_ = grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()))
}

// ExampleNewClientHandler_withMetricAttributesFn demonstrates how to add a dynamic
// client-side attribute to the auto-instrumented metrics.
func ExampleNewClientHandler_withMetricAttributesFn() {
	// should be centralized, example only
	type retryAttemptKey struct{}

	// a middleware must populate that key with the actual retry attempt, e.g.,
	// ...
	// ctx := context.WithValue(context.Background(), retryAttemptKey{}, attempt)
	// ...

	_, _ = grpc.NewClient("localhost", grpc.WithStatsHandler(otelgrpc.NewClientHandler(
		otelgrpc.WithMetricAttributesFn(func(ctx context.Context) []attribute.KeyValue {
			if attempt, ok := ctx.Value(retryAttemptKey{}).(int); ok {
				return []attribute.KeyValue{
					// Caution: example only.
					// This must be a controlled, bounded and very limited set of numbers
					// so that you don't end up with very high cardinality.
					attribute.Int("retry.attempt", attempt),
				}
			}

			return nil
		}),
	)))
}

// ExampleNewClientHandler_withMetricAttributesFn demonstrates how to add a dynamic
// server-side attribute to the auto-instrumented metrics.
func ExampleNewServerHandler_withMetricAttributesFn() {
	_ = grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler(
		otelgrpc.WithMetricAttributesFn(func(ctx context.Context) []attribute.KeyValue {
			md, ok := metadata.FromIncomingContext(ctx)
			if ok {
				if origins := md.Get("origin"); len(origins) > 0 && origins[0] != "" {
					return []attribute.KeyValue{attribute.String("origin", origins[0])}
				}
			}

			// Some use-cases might require to fallback to a default.
			return []attribute.KeyValue{attribute.String("origin", "unknown")}
		}),
	)))
}
