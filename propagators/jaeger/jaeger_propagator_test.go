package jaeger

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/api/trace"
)

var (
	traceID        = trace.ID{0, 0, 0, 0, 0, 0, 0, 0, 0x7b, 0, 0, 0, 0, 0, 0x1, 0xc8}
	traceID128Str  = "00000000000000007b000000000001c8"
	zeroTraceIDStr = "00000000000000000000000000000000"
	traceID64Str   = "7b000000000001c8"
	spanID         = trace.SpanID{0, 0, 0, 0, 0, 0, 0, 0x7b}
	zeroSpanIDStr  = "0000000000000000"
	spanIDStr      = "000000000000007b"
)

func TestJaeger_Extract(t *testing.T) {
	testData := []struct {
		traceID      string
		spanID       string
		parentSpanID string
		flags        string
		expected     trace.SpanContext
		err          error
	}{
		{
			traceID128Str, spanIDStr, deprecatedParentSpanID, "1",
			trace.SpanContext{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: trace.FlagsSampled,
			},
			nil,
		},
		{
			traceID64Str, spanIDStr, deprecatedParentSpanID, "1",
			trace.SpanContext{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: trace.FlagsSampled,
			},
			nil,
		},
		{
			traceID128Str, spanIDStr, deprecatedParentSpanID, "3",
			trace.SpanContext{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: trace.FlagsSampled | trace.FlagsDebug,
			},
			nil,
		},
		{
			// if we didn't set sampled bit when debug bit is 1, then assuming it's not sampled
			traceID128Str, spanIDStr, deprecatedParentSpanID, "2",
			trace.SpanContext{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: 0x00,
			},
			nil,
		},
		{
			// ignore firehose bit since we don't really have this feature in otel span context
			traceID128Str, spanIDStr, deprecatedParentSpanID, "8",
			trace.SpanContext{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: 0x00,
			},
			nil,
		},
		{
			traceID128Str, spanIDStr, deprecatedParentSpanID, "9",
			trace.SpanContext{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: trace.FlagsSampled,
			},
			nil,
		},
		{
			traceID128Str, spanIDStr, "wired stuff", "1",
			trace.SpanContext{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: trace.FlagsSampled,
			},
			nil,
		},
		{
			fmt.Sprintf("%32s", "This_is_a_string_len_64"), spanIDStr, deprecatedParentSpanID, "1",
			trace.SpanContext{},
			errMalformedTraceID,
		},
		{
			"000000000007b00000000000001c8", spanIDStr, deprecatedParentSpanID, "1",
			trace.SpanContext{},
			errInvalidTraceIDLength,
		},
		{
			traceID128Str, fmt.Sprintf("%16s", "wiredspanid"), deprecatedParentSpanID, "1",
			trace.SpanContext{},
			errMalformedSpanID,
		},
		{
			traceID128Str, "0000000000010", deprecatedParentSpanID, "1",
			trace.SpanContext{},
			errInvalidSpanIDLength,
		},
		{
			// reject invalid traceID(0) and spanID(0)
			zeroTraceIDStr, zeroSpanIDStr, deprecatedParentSpanID, "1",
			trace.SpanContext{},
			errMalformedTraceID,
		},
		{
			// reject invalid traceID(0) and spanID(0)
			traceID128Str, zeroSpanIDStr, deprecatedParentSpanID, "1",
			trace.SpanContext{},
			errMalformedSpanID,
		},
	}

	for _, test := range testData {
		headerVal := strings.Join([]string{test.traceID, test.spanID, test.parentSpanID, test.flags}, separator)
		sc, err := extract(headerVal)

		info := []interface{}{
			"trace ID: %q, span ID: %q, parent span ID: %q, sampled: %q, flags: %q",
			test.traceID,
			test.spanID,
			test.parentSpanID,
			test.flags,
		}

		if !assert.Equal(t, test.err, err, info...) {
			continue
		}

		assert.Equal(t, test.expected, sc, info...)
	}
}
