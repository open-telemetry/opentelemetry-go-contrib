// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package b3

import (
	"context"
	"encoding/hex"
	"errors"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	// Default B3 Header names.
	b3ContextHeader      = "b3"
	b3DebugFlagHeader    = "x-b3-flags"
	b3TraceIDHeader      = "x-b3-traceid"
	b3SpanIDHeader       = "x-b3-spanid"
	b3SampledHeader      = "x-b3-sampled"
	b3ParentSpanIDHeader = "x-b3-parentspanid"

	// B3 Single Header encoding widths.
	separatorWidth      = 1       // Single "-" character.
	samplingWidth       = 1       // Single hex character.
	traceID64BitsWidth  = 64 / 4  // 16 hex character Trace ID.
	traceID128BitsWidth = 128 / 4 // 32 hex character Trace ID.
	spanIDWidth         = 16      // 16 hex character ID.
	parentSpanIDWidth   = 16      // 16 hex character ID.
)

var (
	empty = trace.SpanContext{}

	errInvalidSampledByte        = errors.New("invalid B3 Sampled found")
	errInvalidSampledHeader      = errors.New("invalid B3 Sampled header found")
	errInvalidTraceIDHeader      = errors.New("invalid B3 traceID header found")
	errInvalidSpanIDHeader       = errors.New("invalid B3 spanID header found")
	errInvalidParentSpanIDHeader = errors.New("invalid B3 ParentSpanID header found")
	errInvalidScope              = errors.New("require either both traceID and spanID or none")
	errInvalidScopeParent        = errors.New("traceID and spanID required for ParentSpanID")
	errInvalidScopeParentSingle  = errors.New("traceID, spanID and Sampled required for ParentSpanID")
	errEmptyContext              = errors.New("empty request context")
	errInvalidTraceIDValue       = errors.New("invalid B3 traceID value found")
	errInvalidSpanIDValue        = errors.New("invalid B3 spanID value found")
	errInvalidParentSpanIDValue  = errors.New("invalid B3 ParentSpanID value found")
)

type propagator struct {
	cfg config
}

var _ propagation.TextMapPropagator = propagator{}

// New creates a B3 implementation of propagation.TextMapPropagator.
// B3 propagator serializes SpanContext to/from B3 Headers.
// This propagator supports both versions of B3 headers,
//  1. Single Header:
//     b3: {TraceId}-{SpanId}-{SamplingState}-{ParentSpanId}
//  2. Multiple Headers:
//     x-b3-traceid: {TraceId}
//     x-b3-parentspanid: {ParentSpanId}
//     x-b3-spanid: {SpanId}
//     x-b3-sampled: {SamplingState}
//     x-b3-flags: {DebugFlag}
//
// The Single Header propagator is used by default.
func New(opts ...Option) propagation.TextMapPropagator {
	cfg := newConfig(opts...)
	return propagator{
		cfg: *cfg,
	}
}

// Inject injects a context into the carrier as B3 headers.
// The parent span ID is omitted because it is not tracked in the
// SpanContext.
func (b3 propagator) Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	sc := trace.SpanFromContext(ctx).SpanContext()

	if b3.cfg.InjectEncoding.supports(B3SingleHeader) || b3.cfg.InjectEncoding == B3Unspecified {
		var buf [64]byte
		pos := 0

		if sc.TraceID().IsValid() && sc.SpanID().IsValid() {
			tid := sc.TraceID()
			sid := sc.SpanID()
			hex.Encode(buf[pos:pos+32], tid[:])
			pos += 32
			buf[pos] = '-'
			pos++
			hex.Encode(buf[pos:pos+16], sid[:])
			pos += 16
		}

		if debugFromContext(ctx) {
			if pos > 0 {
				buf[pos] = '-'
				pos++
			}
			buf[pos] = 'd'
			pos++
		} else if !deferredFromContext(ctx) {
			if pos > 0 {
				buf[pos] = '-'
				pos++
			}
			if sc.IsSampled() {
				buf[pos] = '1'
			} else {
				buf[pos] = '0'
			}
			pos++
		}

		if pos > 0 {
			carrier.Set(b3ContextHeader, string(buf[:pos]))
		} else {
			carrier.Set(b3ContextHeader, "")
		}
	}

	if b3.cfg.InjectEncoding.supports(B3MultipleHeader) {
		if sc.TraceID().IsValid() && sc.SpanID().IsValid() {
			var tidBuf [32]byte
			var sidBuf [16]byte
			tid := sc.TraceID()
			sid := sc.SpanID()
			hex.Encode(tidBuf[:], tid[:])
			hex.Encode(sidBuf[:], sid[:])
			carrier.Set(b3TraceIDHeader, string(tidBuf[:]))
			carrier.Set(b3SpanIDHeader, string(sidBuf[:]))
		}

		if debugFromContext(ctx) {
			// Since Debug implies deferred, don't also send "X-B3-Sampled".
			carrier.Set(b3DebugFlagHeader, "1")
		} else if !deferredFromContext(ctx) {
			if sc.IsSampled() {
				carrier.Set(b3SampledHeader, "1")
			} else {
				carrier.Set(b3SampledHeader, "0")
			}
		}
	}
}

// Extract extracts a context from the carrier if it contains B3 headers.
func (propagator) Extract(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	var (
		sc  trace.SpanContext
		err error
	)

	// Default to Single Header if a valid value exists.
	if h := carrier.Get(b3ContextHeader); h != "" {
		ctx, sc, err = extractSingle(ctx, h)
		if err == nil && sc.IsValid() {
			return trace.ContextWithRemoteSpanContext(ctx, sc)
		}
		// The Single Header value was invalid, fallback to Multiple Header.
	}

	var (
		traceID      = carrier.Get(b3TraceIDHeader)
		spanID       = carrier.Get(b3SpanIDHeader)
		parentSpanID = carrier.Get(b3ParentSpanIDHeader)
		sampled      = carrier.Get(b3SampledHeader)
		debugFlag    = carrier.Get(b3DebugFlagHeader)
	)
	ctx, sc, err = extractMultiple(ctx, traceID, spanID, parentSpanID, sampled, debugFlag)
	if err != nil || !sc.IsValid() {
		// clear the deferred flag if we don't have a valid SpanContext
		return withDeferred(ctx, false)
	}
	return trace.ContextWithRemoteSpanContext(ctx, sc)
}

func (b3 propagator) Fields() []string {
	// Mirror the encoding conditions used by Inject so Fields reports exactly
	// the headers that may be written. In particular, B3Unspecified defaults to
	// single-header injection (the `b3` header), not multiple headers.
	header := []string{}
	if b3.cfg.InjectEncoding.supports(B3SingleHeader) || b3.cfg.InjectEncoding == B3Unspecified {
		header = append(header, b3ContextHeader)
	}
	if b3.cfg.InjectEncoding.supports(B3MultipleHeader) {
		header = append(header, b3TraceIDHeader, b3SpanIDHeader, b3SampledHeader, b3DebugFlagHeader)
	}
	return header
}

func decodeHexNibble(c byte) (byte, bool) {
	if c >= '0' && c <= '9' {
		return c - '0', true
	}
	if c >= 'a' && c <= 'f' {
		return c - 'a' + 10, true
	}
	return 0, false
}

func parseHexBytes(dst []byte, s string) bool {
	if len(s) != len(dst)*2 {
		return false
	}
	for i := range dst {
		hi, ok1 := decodeHexNibble(s[i*2])
		lo, ok2 := decodeHexNibble(s[i*2+1])
		if !ok1 || !ok2 {
			return false
		}
		dst[i] = (hi << 4) | lo
	}
	return true
}

func parseTraceID(s string) (trace.TraceID, error) {
	var tid trace.TraceID
	if len(s) == 32 {
		if !parseHexBytes(tid[:], s) || !tid.IsValid() {
			return tid, errInvalidTraceIDValue
		}
		return tid, nil
	}
	if len(s) == 16 {
		if !parseHexBytes(tid[8:], s) || !tid.IsValid() {
			return tid, errInvalidTraceIDValue
		}
		return tid, nil
	}
	return tid, errInvalidTraceIDValue
}

func parseSpanID(s string) (trace.SpanID, error) {
	var sid trace.SpanID
	if len(s) != 16 || !parseHexBytes(sid[:], s) || !sid.IsValid() {
		return sid, errInvalidSpanIDValue
	}
	return sid, nil
}

// extractMultiple reconstructs a SpanContext from header values based on B3
// Multiple header. It is based on the implementation found here:
// https://github.com/openzipkin/zipkin-go/blob/v0.2.2/propagation/b3/spancontext.go
// and adapted to support a SpanContext.
func extractMultiple(ctx context.Context, traceID, spanID, parentSpanID, sampled, flags string) (context.Context, trace.SpanContext, error) {
	var (
		err           error
		requiredCount int
		scc           = trace.SpanContextConfig{}
	)

	// correct values for an existing sampled header are "0" and "1".
	// For legacy support and being lenient to other tracing implementations we
	// allow "true" and "false" as inputs for interop purposes.
	switch sampled {
	case "0", "false", "False", "FALSE":
		// Zero value for TraceFlags sample bit is unset.
	case "1", "true", "True", "TRUE":
		scc.TraceFlags = trace.FlagsSampled
	case "":
		ctx = withDeferred(ctx, true)
	default:
		return ctx, empty, errInvalidSampledHeader
	}

	// The only accepted value for Flags is "1". This will set Debug bitmask and
	// sampled bitmask to 1 since debug implicitly means sampled. All other
	// values and omission of header will be ignored. According to the spec. User
	// shouldn't send X-B3-Sampled header along with X-B3-Flags header. Thus we will
	// ignore X-B3-Sampled header when X-B3-Flags header is sent and valid.
	if flags == "1" {
		ctx = withDeferred(ctx, false)
		ctx = withDebug(ctx, true)
		scc.TraceFlags |= trace.FlagsSampled
	}

	if traceID != "" {
		requiredCount++
		scc.TraceID, err = parseTraceID(traceID)
		if err != nil {
			return ctx, empty, errInvalidTraceIDHeader
		}
	}

	if spanID != "" {
		requiredCount++
		scc.SpanID, err = parseSpanID(spanID)
		if err != nil {
			return ctx, empty, errInvalidSpanIDHeader
		}
	}

	if requiredCount != 0 && requiredCount != 2 {
		return ctx, empty, errInvalidScope
	}

	if parentSpanID != "" {
		if requiredCount == 0 {
			return ctx, empty, errInvalidScopeParent
		}
		// Validate parent span ID but we do not use it so do not save it.
		if _, err = parseSpanID(parentSpanID); err != nil {
			return ctx, empty, errInvalidParentSpanIDHeader
		}
	}

	return ctx, trace.NewSpanContext(scc), nil
}

// extractSingle reconstructs a SpanContext from contextHeader based on a B3
// Single header. It is based on the implementation found here:
// https://github.com/openzipkin/zipkin-go/blob/v0.2.2/propagation/b3/spancontext.go
// and adapted to support a SpanContext.
func extractSingle(ctx context.Context, contextHeader string) (context.Context, trace.SpanContext, error) {
	if contextHeader == "" {
		return ctx, empty, errEmptyContext
	}

	var (
		scc           = trace.SpanContextConfig{}
		samplingState byte
		hasSampling   bool
	)

	headerLen := len(contextHeader)

	switch {
	case headerLen == samplingWidth:
		samplingState = contextHeader[0]
		hasSampling = true
	case headerLen == traceID64BitsWidth || headerLen == traceID128BitsWidth:
		// Trace ID by itself is invalid.
		return ctx, empty, errInvalidScope
	case headerLen >= traceID64BitsWidth+spanIDWidth+separatorWidth:
		pos := 0
		var tid trace.TraceID
		var err error

		switch {
		case contextHeader[traceID64BitsWidth] == '-':
			// traceID must be 64 bits
			tid, err = parseTraceID(contextHeader[:traceID64BitsWidth])
			if err != nil {
				return ctx, empty, errInvalidTraceIDValue
			}
			pos = traceID64BitsWidth
		case headerLen > 32 && contextHeader[32] == '-':
			// traceID must be 128 bits
			tid, err = parseTraceID(contextHeader[:traceID128BitsWidth])
			if err != nil {
				return ctx, empty, errInvalidTraceIDValue
			}
			pos = traceID128BitsWidth
		default:
			return ctx, empty, errInvalidTraceIDValue
		}
		scc.TraceID = tid

		pos += separatorWidth // {traceID}-

		if headerLen < pos+spanIDWidth {
			return ctx, empty, errInvalidSpanIDValue
		}
		scc.SpanID, err = parseSpanID(contextHeader[pos : pos+spanIDWidth])
		if err != nil {
			return ctx, empty, errInvalidSpanIDValue
		}
		pos += spanIDWidth // {traceID}-{spanID}

		if headerLen > pos {
			if headerLen == pos+separatorWidth {
				// {traceID}-{spanID}- is invalid.
				return ctx, empty, errInvalidSampledByte
			}
			pos += separatorWidth // {traceID}-{spanID}-

			switch headerLen {
			case pos + samplingWidth:
				samplingState = contextHeader[pos]
				hasSampling = true
			case pos + parentSpanIDWidth:
				// {traceID}-{spanID}-{parentSpanID} is invalid.
				return ctx, empty, errInvalidScopeParentSingle
			case pos + samplingWidth + separatorWidth + parentSpanIDWidth:
				samplingState = contextHeader[pos]
				hasSampling = true
				pos += samplingWidth + separatorWidth // {traceID}-{spanID}-{sampling}-

				// Validate parent span ID but we do not use it so do not save it.
				_, err = parseSpanID(contextHeader[pos:])
				if err != nil {
					return ctx, empty, errInvalidParentSpanIDValue
				}
			default:
				return ctx, empty, errInvalidParentSpanIDValue
			}
		}
	default:
		return ctx, empty, errInvalidTraceIDValue
	}

	if !hasSampling {
		ctx = withDeferred(ctx, true)
	} else {
		switch samplingState {
		case 'd':
			ctx = withDebug(ctx, true)
			scc.TraceFlags = trace.FlagsSampled
		case '1':
			scc.TraceFlags = trace.FlagsSampled
		case '0':
			// Zero value for TraceFlags sample bit is unset.
		default:
			return ctx, empty, errInvalidSampledByte
		}
	}

	return ctx, trace.NewSpanContext(scc), nil
}
