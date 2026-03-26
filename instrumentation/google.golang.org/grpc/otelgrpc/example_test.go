// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgrpc_test

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
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

// ExampleNewClientHandler_withMetricAttributesFn_interceptor demonstrates how to add a dynamic
// client-side attribute using gRPC interceptors to the auto-instrumented metrics.
func ExampleNewClientHandler_withMetricAttributesFn_interceptor() {
	// should be centralized, example only
	type retryAttemptKey struct{}

	// a gRPC client interceptor must populate that key with the actual retry attempt, e.g.,
	// ...
	// interceptor := func(ctx context.Context, method string, req, reply any,
	//     cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	//     ...
	//     ctx = context.WithValue(ctx, retryAttemptKey{}, attempt)
	//     return invoker(ctx, method, req, reply, cc, opts...)
	// }
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

// ExampleNewClientHandler_withMetricAttributesFn_baggage demonstrates how to add a dynamic
// client-side attribute using W3C baggage to the auto-instrumented metrics.
func ExampleNewClientHandler_withMetricAttributesFn_baggage() {
	// Baggage must be set in the context before making the call, e.g.,
	// ...
	// member, err := baggage.NewMember("traffic.type", "internal")
	// ...
	// bag, err := baggage.New(member)
	// ...
	// ctx := baggage.ContextWithBaggage(ctx, bag)
	// ...

	_, _ = grpc.NewClient("localhost", grpc.WithStatsHandler(otelgrpc.NewClientHandler(
		otelgrpc.WithMetricAttributesFn(func(ctx context.Context) []attribute.KeyValue {
			bag := baggage.FromContext(ctx)
			if trafficType := bag.Member("traffic.type"); trafficType.Value() != "" {
				return []attribute.KeyValue{
					attribute.String("traffic.type", trafficType.Value()),
				}
			}

			return nil
		}),
	)))
}

// ExampleNewServerHandler_withMetricAttributesFn_metadata demonstrates how to add a dynamic
// server-side attribute using gRPC metadata to the auto-instrumented metrics.
func ExampleNewServerHandler_withMetricAttributesFn_metadata() {
	// The client must set metadata in the outgoing context beforehand, e.g.,
	// ...
	// ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("origin", "some-origin"))
	// ...
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

// ExampleNewServerHandler_withMetricAttributesFn_baggage demonstrates how to add a dynamic
// server-side attribute using W3C baggage to the auto-instrumented metrics.
func ExampleNewServerHandler_withMetricAttributesFn_baggage() {
	// The client must set baggage in context beforehand and have baggage propagators configured to
	// inject it into the headers (see https://pkg.go.dev/go.opentelemetry.io/otel/propagation#section-documentation), e.g.,
	//
	// conn, err := grpc.NewClient(
	// ...
	// grpc.WithStatsHandler(otelgrpc.NewClientHandler(
	//	    otelgrpc.WithPropagators(propagation.NewCompositeTextMapPropagator(
	//			  propagation.Baggage{},
	//		)),
	//	)),
	//)
	// ...
	// member, err := baggage.NewMember("tenant.tier", "premium")
	// ...
	// bag, err := baggage.New(member)
	// ...
	// ctx := baggage.ContextWithBaggage(ctx, bag)
	// ...

	_ = grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler(
		// Propagators are required to extract baggage from incoming request headers.
		otelgrpc.WithPropagators(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		)),
		otelgrpc.WithMetricAttributesFn(func(ctx context.Context) []attribute.KeyValue {
			bag := baggage.FromContext(ctx)
			if tier := bag.Member("tenant.tier"); tier.Value() != "" {
				return []attribute.KeyValue{
					attribute.String("tenant.tier", tier.Value()),
				}
			}

			return nil
		}),
	)))
}
