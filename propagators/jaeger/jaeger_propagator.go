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
		debug bool
		err   error
	)

	part0, part1, _, part3, ok := splitHeader(headerVal)
	if !ok {
		return ctx, empty, errMalformedTraceContextVal
	}

	// extract trace ID
	if part0 != "" {
		if len(part0) > traceID128bitsWidth {
			return ctx, empty, errInvalidTraceIDLength
		}
		if len(part0) < traceID128bitsWidth {
			var buf [traceID128bitsWidth]byte
			padCharCount := traceID128bitsWidth - len(part0)
			for j := range padCharCount {
				buf[j] = '0'
			}
			copy(buf[padCharCount:], part0)
			part0 = string(buf[:])
		}

		scc.TraceID, err = trace.TraceIDFromHex(part0)
		if err != nil {
			return ctx, empty, errMalformedTraceID
		}
	}

	// extract span ID
	if part1 != "" {
		if len(part1) > spanIDWidth {
			return ctx, empty, errInvalidSpanIDLength
		}
		if len(part1) < spanIDWidth {
			var buf [spanIDWidth]byte
			padCharCount := spanIDWidth - len(part1)
			for j := range padCharCount {
				buf[j] = '0'
			}
			copy(buf[padCharCount:], part1)
			part1 = string(buf[:])
		}

		scc.SpanID, err = trace.SpanIDFromHex(part1)
		if err != nil {
			return ctx, empty, errMalformedSpanID
		}
	}

	// skip third part as it is deprecated

	// extract flag
	if part3 != "" {
		flag, parseErr := strconv.ParseInt(part3, 16, 64)
		if parseErr != nil {
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

func splitHeader(val string) (p0, p1, p2, p3 string, ok bool) {
	var rest string
	p0, rest, ok = strings.Cut(val, ":")
	if !ok {
		return "", "", "", "", false
	}
	p1, rest, ok = strings.Cut(rest, ":")
	if !ok {
		return "", "", "", "", false
	}
	p2, p3, ok = strings.Cut(rest, ":")
	if !ok || strings.Contains(p3, ":") {
		return "", "", "", "", false
	}
	return p0, p1, p2, p3, true
}

// Fields returns the Jaeger header key whose value is set with Inject.
func (Jaeger) Fields() []string {
	return []string{jaegerHeader}
}
