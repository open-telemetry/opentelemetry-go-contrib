// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package jaeger

import (
	"context"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	jaegerHeader        = "uber-trace-id"
	separator           = ":"
	traceID128bitsWidth = 128 / 4
	spanIDWidth         = 64 / 4

	idPaddingChar = "0"

	flagsDebug   = 0x02
	flagsSampled = 0x01
)

var (
	empty = trace.SpanContext{}

	errMalformedTraceContextVal = errors.New("header value of uber-trace-id should contain four different part separated by : ")
	errInvalidTraceIDLength     = errors.New("invalid trace id length, must be either 16 or 32")
	errMalformedTraceID         = errors.New("cannot decode trace id from header, should be a string of hex, lowercase trace id can't be all zero")
	errInvalidSpanIDLength      = errors.New("invalid span id length, must be 16")
	errMalformedSpanID          = errors.New("cannot decode span id from header, should be a string of hex, lowercase span id can't be all zero")
	errMalformedFlag            = errors.New("cannot decode flag")
)

// Jaeger propagator serializes SpanContext to/from Jaeger Headers
//
// Jaeger format:
//
// uber-trace-id: {trace-id}:{span-id}:{parent-span-id}:{flags}.
type Jaeger struct{}

var _ propagation.TextMapPropagator = &Jaeger{}

// Inject injects a context to the carrier following jaeger format.
// The parent span ID is set to an dummy parent span id as the most implementations do.
func (Jaeger) Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	sc := trace.SpanFromContext(ctx).SpanContext()
	if !sc.TraceID().IsValid() || !sc.SpanID().IsValid() {
		return
	}

	carrier.Set(jaegerHeader, formatHeader(ctx, sc))
}

func formatHeader(ctx context.Context, sc trace.SpanContext) string {
	var h [53]byte
	traceID := sc.TraceID()
	spanID := sc.SpanID()
	hex.Encode(h[:32], traceID[:])
	h[32] = ':'
	hex.Encode(h[33:49], spanID[:])
	h[49] = ':'
	h[50] = '0'
	h[51] = ':'

	switch {
	case debugFromContext(ctx):
		h[52] = '3'
	case sc.IsSampled():
		h[52] = '1'
	default:
		h[52] = '0'
	}

	return string(h[:])
}

// Extract extracts a context from the carrier if it contains Jaeger headers.
func (Jaeger) Extract(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	// extract tracing information
	if h := carrier.Get(jaegerHeader); h != "" {
		ctx, sc, err := extract(ctx, h)
		if err == nil && sc.IsValid() {
			return trace.ContextWithRemoteSpanContext(ctx, sc)
		}
	}

	return ctx
}

func extract(ctx context.Context, headerVal string) (context.Context, trace.SpanContext, error) {
	var (
		scc   = trace.SpanContextConfig{}
		err   error
		debug bool
	)

	parts := strings.Split(headerVal, separator)
	if len(parts) != 4 {
		return ctx, empty, errMalformedTraceContextVal
	}

	// extract trace ID
	if parts[0] != "" {
		id := parts[0]
		if len(id) > traceID128bitsWidth {
			return ctx, empty, errInvalidTraceIDLength
		}
		// padding when length is less than 32
		if len(id) < traceID128bitsWidth {
			padCharCount := traceID128bitsWidth - len(id)
			id = strings.Repeat(idPaddingChar, padCharCount) + id
		}
		scc.TraceID, err = trace.TraceIDFromHex(id)
		if err != nil {
			return ctx, empty, errMalformedTraceID
		}
	}

	// extract span ID
	if parts[1] != "" {
		id := parts[1]
		if len(id) > spanIDWidth {
			return ctx, empty, errInvalidSpanIDLength
		}
		// padding when length is less than 16
		if len(id) < spanIDWidth {
			padCharCount := spanIDWidth - len(id)
			id = strings.Repeat(idPaddingChar, padCharCount) + id
		}
		scc.SpanID, err = trace.SpanIDFromHex(id)
		if err != nil {
			return ctx, empty, errMalformedSpanID
		}
	}

	// skip third part as it is deprecated

	// extract flag
	if parts[3] != "" {
		flagStr := parts[3]
		flag, err := strconv.ParseInt(flagStr, 16, 64)
		if err != nil {
			return ctx, empty, errMalformedFlag
		}
		if flag&flagsSampled == flagsSampled {
			// if sample bit is set, we check if debug bit is also set
			if flag&flagsDebug == flagsDebug {
				scc.TraceFlags |= trace.FlagsSampled
				debug = true
			} else {
				scc.TraceFlags |= trace.FlagsSampled
			}
		}
		// ignore other bit, including firehose since we don't have corresponding flag in trace context.
	}
	if debug != debugFromContext(ctx) {
		ctx = withDebug(ctx, debug)
	}
	return ctx, trace.NewSpanContext(scc), nil
}

// Fields returns the Jaeger header key whose value is set with Inject.
func (Jaeger) Fields() []string {
	return []string{jaegerHeader}
}
