// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgrpc_test

import (
	"context"

	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

type fullMethodKey struct{}

// strictNotFoundHandler wraps an otelgrpc client stats.Handler and marks
// NotFound responses as errors for the configured methods, even though
// NotFound is configured as a non-error code for the wrapped handler.
type strictNotFoundHandler struct {
	stats.Handler

	// strict contains full method names (e.g. "/pkg.Svc/Method") for which
	// NotFound is unexpected and must be reported as an error.
	strict map[string]struct{}
}

func (h *strictNotFoundHandler) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	// Let otelgrpc start the span first, then retain the full method name so
	// HandleRPC can use it.
	ctx = h.Handler.TagRPC(ctx, info)
	return context.WithValue(ctx, fullMethodKey{}, info.FullMethodName)
}

func (h *strictNotFoundHandler) HandleRPC(ctx context.Context, rs stats.RPCStats) {
	if end, ok := rs.(*stats.End); ok && end.Error != nil {
		method, _ := ctx.Value(fullMethodKey{}).(string)
		if _, strict := h.strict[method]; strict {
			if s, _ := status.FromError(end.Error); s.Code() == codes.NotFound {
				// The span status must be set before forwarding stats.End:
				// otelgrpc ends the span while handling it.
				trace.SpanFromContext(ctx).SetStatus(otelcodes.Error, s.Message())
			}
		}
	}
	h.Handler.HandleRPC(ctx, rs)
}

// ExampleWithNonErrorCodes_perMethod demonstrates how to treat NotFound as a
// non-error for all client RPCs except specific methods, by wrapping the
// client stats.Handler.
//
// The span status has to be set inside the stats.Handler, before stats.End is
// forwarded to otelgrpc. A client interceptor cannot be used for this: client
// interceptors run outside the per-attempt stats.Handler lifecycle, so the span
// has already ended by the time the invoker returns and modifying it then has
// no effect.
func ExampleWithNonErrorCodes_perMethod() {
	handler := &strictNotFoundHandler{
		Handler: otelgrpc.NewClientHandler(
			otelgrpc.WithNonErrorCodes(codes.NotFound),
		),
		strict: map[string]struct{}{
			"/pkg.Svc/MustExist": {},
		},
	}
	_, _ = grpc.NewClient("localhost", grpc.WithStatsHandler(handler))
}
