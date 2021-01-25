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

package opentracing

// TODO: support baggage

import (
	"context"
	"errors"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/api/trace"
)

const (
	// Default OpenTracing Header names.
	traceIDHeader = "ot-tracer-traceid"
	spanIDHeader  = "ot-tracer-spanid"
	sampledHeader = "ot-tracer-sampled"

	otTraceIDPadding = "0000000000000000"

	traceID64BitsWidth = 64 / 4 // 16 hex character Trace ID.
)

var (
	empty = trace.EmptySpanContext()

	errInvalidSampledHeader = errors.New("invalid OpenTracing Sampled header found")
	errInvalidTraceIDHeader = errors.New("invalid OpenTracing traceID header found")
	errInvalidSpanIDHeader  = errors.New("invalid OpenTracing spanID header found")
	errInvalidScope         = errors.New("require either both traceID and spanID or none")
)

// OpenTracing propagator serializes SpanContext to/from OpenTracing Headers.
type OpenTracing struct {
}

var _ otel.TextMapPropagator = OpenTracing{}

// Inject injects a context into the carrier as OpenTracing headers.
// NOTE: In order to interop with systems that use the OpenTracing header format, trace ids MUST be 64-bits
func (ot OpenTracing) Inject(ctx context.Context, carrier otel.TextMapCarrier) {
	sc := trace.SpanFromContext(ctx).SpanContext()

	if sc.TraceID.IsValid() && sc.SpanID.IsValid() {
		carrier.Set(traceIDHeader, sc.TraceID.String()[len(sc.TraceID.String())-traceID64BitsWidth:])
		carrier.Set(spanIDHeader, sc.SpanID.String())
	}

	if sc.IsSampled() {
		carrier.Set(sampledHeader, "1")
	} else {
		carrier.Set(sampledHeader, "0")
	}

}

// Extract extracts a context from the carrier if it contains OpenTracing headers.
func (ot OpenTracing) Extract(ctx context.Context, carrier otel.TextMapCarrier) context.Context {
	var (
		sc  trace.SpanContext
		err error
	)

	var (
		traceID = carrier.Get(traceIDHeader)
		spanID  = carrier.Get(spanIDHeader)
		sampled = carrier.Get(sampledHeader)
	)
	sc, err = extract(traceID, spanID, sampled)
	if err != nil || !sc.IsValid() {
		return ctx
	}
	return trace.ContextWithRemoteSpanContext(ctx, sc)
}

func (ot OpenTracing) Fields() []string {
	return []string{traceIDHeader, spanIDHeader, sampledHeader}
}

// extract reconstructs a SpanContext from header values based on OpenTracing
// headers.
func extract(traceID, spanID, sampled string) (trace.SpanContext, error) {
	var (
		err           error
		requiredCount int
		sc            = trace.SpanContext{}
	)

	switch strings.ToLower(sampled) {
	case "0", "false":
		// Zero value for TraceFlags sample bit is unset.
	case "1", "true":
		sc.TraceFlags = trace.FlagsSampled
	case "":
		sc.TraceFlags = trace.FlagsDeferred
	default:
		return empty, errInvalidSampledHeader
	}

	if traceID != "" {
		requiredCount++
		id := traceID
		if len(traceID) == 16 {
			// Pad 64-bit trace IDs.
			id = otTraceIDPadding + traceID
		}
		if sc.TraceID, err = trace.IDFromHex(id); err != nil {
			return empty, errInvalidTraceIDHeader
		}
	}

	if spanID != "" {
		requiredCount++
		if sc.SpanID, err = trace.SpanIDFromHex(spanID); err != nil {
			return empty, errInvalidSpanIDHeader
		}
	}

	if requiredCount != 0 && requiredCount != 2 {
		return empty, errInvalidScope
	}

	return sc, nil
}
