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

package otelgrpc // import "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// metricsInfo contains metrics information for an RPC.
type metricsInfo struct {
	msgReceived int64
	msgSent     int64
}

// traceInfo contains tracing information for an RPC.
type traceInfo struct {
	name string
	kind trace.SpanKind
}

// gRPCContext contains all the information needed to record metrics and traces.
type gRPCContext struct {
	metricsInfo *metricsInfo
	traceInfo   *traceInfo
	attrs       []attribute.KeyValue
}

// AddAttrs adds attributes to the given context.
func AddAttrs(ctx context.Context, attrs ...attribute.KeyValue) context.Context {
	gctx, _ := gRPCContextFromContext(ctx)
	gctx.addAttrs(attrs...)
	return contextWithGRPCContext(ctx, gctx)
}

// add attributes to a gRPCContext.
func (g *gRPCContext) addAttrs(attrs ...attribute.KeyValue) {
	g.attrs = append(g.attrs, attrs...)
}

type gRPCContextKey struct{}

// contextWithGRPCContext returns a new context with the provided gRPCContext attached.
func contextWithGRPCContext(ctx context.Context, gctx *gRPCContext) context.Context {
	return context.WithValue(ctx, gRPCContextKey{}, gctx)
}

// gRPCContextFromContext retrieves a GRPCContext instance from the provided context if
// one is available.  If no GRPCContext was found in the provided context a new, empty
// GRPCContext is returned and the second return value is false.  In this case it is
// safe to use the GRPCContext but any attributes added to it will not be used.
func gRPCContextFromContext(ctx context.Context) (*gRPCContext, bool) { // nolint: revive
	l, ok := ctx.Value(gRPCContextKey{}).(*gRPCContext)
	if !ok {
		l = &gRPCContext{
			metricsInfo: &metricsInfo{},
			traceInfo:   &traceInfo{},
		}
	}
	return l, ok
}
