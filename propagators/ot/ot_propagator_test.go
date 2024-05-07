// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ot

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/trace"
)

var (
	traceID        = trace.TraceID{0, 0, 0, 0, 0, 0, 0, 0, 0x7b, 0, 0, 0, 0, 0, 0x1, 0xc8}
	traceID128Str  = "00000000000000007b000000000001c8"
	zeroTraceIDStr = "00000000000000000000000000000000"
	traceID64Str   = "7b000000000001c8"
	spanID         = trace.SpanID{0, 0, 0, 0, 0, 0, 0, 0x7b}
	zeroSpanIDStr  = "0000000000000000"
	spanIDStr      = "000000000000007b"
)

func TestOT_Extract(t *testing.T) {
	testData := []struct {
		traceID  string
		spanID   string
		sampled  string
		expected trace.SpanContextConfig
		err      error
	}{
		{
			traceID128Str, spanIDStr, "1",
			trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: trace.FlagsSampled,
			},
			nil,
		},
		{
			traceID64Str, spanIDStr, "1",
			trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: trace.FlagsSampled,
			},
			nil,
		},
		{
			traceID128Str, spanIDStr, "",
			trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: 0x00,
			},
			nil,
		},
		{
			// if we didn't set sampled bit when debug bit is 1, then assuming it's not sampled
			traceID128Str, spanIDStr, "0",
			trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: 0x00,
			},
			nil,
		},
		{
			traceID128Str, spanIDStr, "1",
			trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: trace.FlagsSampled,
			},
			nil,
		},
		{
			fmt.Sprintf("%32s", "This_is_a_string_len_64"), spanIDStr, "1",
			trace.SpanContextConfig{},
			errInvalidTraceIDHeader,
		},
		{
			"000000000007b00000000000001c8", spanIDStr, "1",
			trace.SpanContextConfig{},
			errInvalidTraceIDHeader,
		},
		{
			traceID128Str, fmt.Sprintf("%16s", "wiredspanid"), "1",
			trace.SpanContextConfig{},
			errInvalidSpanIDHeader,
		},
		{
			traceID128Str, "0000000000010", "1",
			trace.SpanContextConfig{},
			errInvalidSpanIDHeader,
		},
		{
			// reject invalid traceID(0) and spanID(0)
			zeroTraceIDStr, zeroSpanIDStr, "1",
			trace.SpanContextConfig{},
			errInvalidTraceIDHeader,
		},
		{
			// reject invalid spanID(0)
			traceID128Str, zeroSpanIDStr, "1",
			trace.SpanContextConfig{},
			errInvalidSpanIDHeader,
		},
		{
			// reject invalid spanID(0)
			traceID128Str, spanIDStr, "invalid",
			trace.SpanContextConfig{},
			errInvalidSampledHeader,
		},
	}

	for _, test := range testData {
		sc, err := extract(test.traceID, test.spanID, test.sampled)

		info := []interface{}{
			"trace ID: %q, span ID: %q, sampled: %q",
			test.traceID,
			test.spanID,
			test.sampled,
		}

		if !assert.Equal(t, test.err, err, info...) {
			continue
		}

		assert.Equal(t, trace.NewSpanContext(test.expected), sc, info...)
	}
}
