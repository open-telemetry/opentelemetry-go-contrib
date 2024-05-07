// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package opencensus // import "go.opentelemetry.io/contrib/propagators/opencensus"

import (
	"context"

	ocpropagation "go.opencensus.io/trace/propagation"

	"go.opentelemetry.io/otel/bridge/opencensus"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type key uint

const binaryKey key = 0

// binaryHeader is the same as traceContextKey is in opencensus:
// https://github.com/census-instrumentation/opencensus-go/blob/3fb168f674736c026e623310bfccb0691e6dec8a/plugin/ocgrpc/trace_common.go#L30
const binaryHeader = "grpc-trace-bin"

// Binary is an OpenTelemetry implementation of the OpenCensus grpc binary format.
// Binary propagation was temporarily removed from opentelemetry.  See
// https://github.com/open-telemetry/opentelemetry-specification/issues/437
type Binary struct{}

var _ propagation.TextMapPropagator = Binary{}

// Inject injects context into the TextMapCarrier.
func (b Binary) Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	binaryContext := ctx.Value(binaryKey)
	if state, ok := binaryContext.(string); binaryContext != nil && ok {
		carrier.Set(binaryHeader, state)
	}

	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return
	}
	h := ocpropagation.Binary(opencensus.OTelSpanContextToOC(sc))
	carrier.Set(binaryHeader, string(h))
}

// Extract extracts the SpanContext from the TextMapCarrier.
func (b Binary) Extract(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	state := carrier.Get(binaryHeader)
	if state != "" {
		ctx = context.WithValue(ctx, binaryKey, state)
	}

	sc := b.extract(carrier)
	if !sc.IsValid() {
		return ctx
	}
	return trace.ContextWithRemoteSpanContext(ctx, sc)
}

func (b Binary) extract(carrier propagation.TextMapCarrier) trace.SpanContext {
	h := carrier.Get(binaryHeader)
	if h == "" {
		return trace.SpanContext{}
	}
	ocContext, ok := ocpropagation.FromBinary([]byte(h))
	if !ok {
		return trace.SpanContext{}
	}
	return opencensus.OCSpanContextToOTel(ocContext)
}

// Fields returns the fields that this propagator modifies.
func (b Binary) Fields() []string {
	return []string{binaryHeader}
}
