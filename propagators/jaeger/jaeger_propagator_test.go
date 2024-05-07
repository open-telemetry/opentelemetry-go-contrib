// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package jaeger

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/trace"
)

var (
	traceID        = trace.TraceID{0, 0, 0, 0, 0, 0, 0, 0, 0xb, 0, 0, 0, 0, 0, 0x1, 0xc8}
	traceID128Str  = "00000000000000000b000000000001c8"
	zeroTraceIDStr = "00000000000000000000000000000000"
	traceID64Str   = "0b000000000001c8"
	traceID60Str   = "b000000000001c8"
	spanID         = trace.SpanID{0, 0, 0, 0, 0, 0, 0, 0x7b}
	zeroSpanIDStr  = "0000000000000000"
	spanID64Str    = "000000000000007b"
	spanID60Str    = "00000000000007b"
)

func TestJaeger_Extract(t *testing.T) {
	testData := []struct {
		traceID      string
		spanID       string
		parentSpanID string
		flags        string
		expected     trace.SpanContextConfig
		err          error
		debug        bool
	}{
		{
			traceID128Str, spanID64Str, deprecatedParentSpanID, "1",
			trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: trace.FlagsSampled,
			},
			nil,
			false,
		},
		{
			traceID64Str, spanID64Str, deprecatedParentSpanID, "1",
			trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: trace.FlagsSampled,
			},
			nil,
			false,
		},
		{
			traceID60Str, spanID60Str, deprecatedParentSpanID, "1",
			trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: trace.FlagsSampled,
			},
			nil,
			false,
		},
		{
			traceID128Str, spanID64Str, deprecatedParentSpanID, "3",
			trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: trace.FlagsSampled,
			},
			nil,
			true,
		},
		{
			// if we didn't set sampled bit when debug bit is 1, then assuming it's not sampled
			traceID128Str, spanID64Str, deprecatedParentSpanID, "2",
			trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: 0x00,
			},
			nil,
			false,
		},
		{
			// ignore firehose bit since we don't really have this feature in otel span context
			traceID128Str, spanID64Str, deprecatedParentSpanID, "8",
			trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: 0x00,
			},
			nil,
			false,
		},
		{
			traceID128Str, spanID64Str, deprecatedParentSpanID, "9",
			trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: trace.FlagsSampled,
			},
			nil,
			false,
		},
		{
			traceID128Str, spanID64Str, "wired stuff", "1",
			trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: trace.FlagsSampled,
			},
			nil,
			false,
		},
		{
			fmt.Sprintf("%32s", "This_is_a_string_len_64"), spanID64Str, deprecatedParentSpanID, "1",
			trace.SpanContextConfig{},
			errMalformedTraceID,
			false,
		},
		{
			"0000000000000007b00000000000001c8", spanID64Str, deprecatedParentSpanID, "1",
			trace.SpanContextConfig{},
			errInvalidTraceIDLength,
			false,
		},
		{
			traceID128Str, fmt.Sprintf("%16s", "wiredspanid"), deprecatedParentSpanID, "1",
			trace.SpanContextConfig{},
			errMalformedSpanID,
			false,
		},
		{
			traceID128Str, "00000000000000010", deprecatedParentSpanID, "1",
			trace.SpanContextConfig{},
			errInvalidSpanIDLength,
			false,
		},
		{
			// reject invalid traceID(0) and spanID(0)
			zeroTraceIDStr, zeroSpanIDStr, deprecatedParentSpanID, "1",
			trace.SpanContextConfig{},
			errMalformedTraceID,
			false,
		},
		{
			// reject invalid traceID(0) and spanID(0)
			traceID128Str, zeroSpanIDStr, deprecatedParentSpanID, "1",
			trace.SpanContextConfig{},
			errMalformedSpanID,
			false,
		},
	}

	for _, test := range testData {
		headerVal := strings.Join([]string{test.traceID, test.spanID, test.parentSpanID, test.flags}, separator)
		ctx, sc, err := extract(context.Background(), headerVal)

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

		assert.Equal(t, trace.NewSpanContext(test.expected), sc, info...)
		assert.Equal(t, test.debug, debugFromContext(ctx))
	}
}
