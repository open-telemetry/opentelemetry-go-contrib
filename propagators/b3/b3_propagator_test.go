// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package b3

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/trace"
)

var (
	traceID    = trace.TraceID{0, 0, 0, 0, 0, 0, 0, 0x7b, 0, 0, 0, 0, 0, 0, 0x1, 0xc8}
	traceIDStr = "000000000000007b00000000000001c8"
	spanID     = trace.SpanID{0, 0, 0, 0, 0, 0, 0, 0x7b}
	spanIDStr  = "000000000000007b"
)

func TestExtractMultiple(t *testing.T) {
	tests := []struct {
		traceID      string
		spanID       string
		parentSpanID string
		sampled      string
		flags        string
		expected     trace.SpanContextConfig
		err          error
		debug        bool
		deferred     bool
	}{
		{
			"", "", "", "0", "",
			trace.SpanContextConfig{},
			nil, false, false,
		},
		{
			"", "", "", "", "",
			trace.SpanContextConfig{},
			nil, false, true,
		},
		{
			"", "", "", "1", "",
			trace.SpanContextConfig{TraceFlags: trace.FlagsSampled},
			nil, false, false,
		},
		{
			"", "", "", "", "1",
			trace.SpanContextConfig{TraceFlags: trace.FlagsSampled},
			nil, true, false,
		},
		{
			"", "", "", "0", "1",
			trace.SpanContextConfig{TraceFlags: trace.FlagsSampled},
			nil, true, false,
		},
		{
			"", "", "", "1", "1",
			trace.SpanContextConfig{TraceFlags: trace.FlagsSampled},
			nil, true, false,
		},
		{
			traceIDStr, spanIDStr, "", "", "",
			trace.SpanContextConfig{TraceID: traceID, SpanID: spanID},
			nil, false, true,
		},
		{
			traceIDStr, spanIDStr, "", "0", "",
			trace.SpanContextConfig{TraceID: traceID, SpanID: spanID},
			nil, false, false,
		},
		// Ensure backwards compatibility.
		{
			traceIDStr, spanIDStr, "", "false", "",
			trace.SpanContextConfig{TraceID: traceID, SpanID: spanID},
			nil, false, false,
		},
		{
			traceIDStr, spanIDStr, "", "1", "",
			trace.SpanContextConfig{TraceID: traceID, SpanID: spanID, TraceFlags: trace.FlagsSampled},
			nil, false, false,
		},
		// Ensure backwards compatibility.
		{
			traceIDStr, spanIDStr, "", "true", "",
			trace.SpanContextConfig{TraceID: traceID, SpanID: spanID, TraceFlags: trace.FlagsSampled},
			nil, false, false,
		},
		{
			traceIDStr, spanIDStr, "", "a", "",
			trace.SpanContextConfig{},
			errInvalidSampledHeader, false, false,
		},
		{
			traceIDStr, spanIDStr, "", "1", "1",
			trace.SpanContextConfig{TraceID: traceID, SpanID: spanID, TraceFlags: trace.FlagsSampled},
			nil, true, false,
		},
		// Invalid flags are discarded.
		{
			traceIDStr, spanIDStr, "", "1", "invalid",
			trace.SpanContextConfig{TraceID: traceID, SpanID: spanID, TraceFlags: trace.FlagsSampled},
			nil, false, false,
		},
		// Support short trace IDs.
		{
			"00000000000001c8", spanIDStr, "", "0", "",
			trace.SpanContextConfig{
				TraceID: trace.TraceID{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x1, 0xc8},
				SpanID:  spanID,
			},
			nil, false, false,
		},
		{
			"00000000000001c", spanIDStr, "", "0", "",
			trace.SpanContextConfig{},
			errInvalidTraceIDHeader, false, false,
		},
		{
			"00000000000001c80", spanIDStr, "", "0", "",
			trace.SpanContextConfig{},
			errInvalidTraceIDHeader, false, false,
		},
		{
			traceIDStr[:len(traceIDStr)-2], spanIDStr, "", "0", "",
			trace.SpanContextConfig{},
			errInvalidTraceIDHeader, false, false,
		},
		{
			traceIDStr + "0", spanIDStr, "", "0", "",
			trace.SpanContextConfig{},
			errInvalidTraceIDHeader, false, false,
		},
		{
			traceIDStr, "00000000000001c", "", "0", "",
			trace.SpanContextConfig{},
			errInvalidSpanIDHeader, false, false,
		},
		{
			traceIDStr, "00000000000001c80", "", "0", "",
			trace.SpanContextConfig{},
			errInvalidSpanIDHeader, false, false,
		},
		{
			traceIDStr, "", "", "0", "",
			trace.SpanContextConfig{},
			errInvalidScope, false, false,
		},
		{
			"", spanIDStr, "", "0", "",
			trace.SpanContextConfig{},
			errInvalidScope, false, false,
		},
		{
			"", "", spanIDStr, "0", "",
			trace.SpanContextConfig{},
			errInvalidScopeParent, false, false,
		},
		{
			traceIDStr, spanIDStr, "00000000000001c8", "0", "",
			trace.SpanContextConfig{TraceID: traceID, SpanID: spanID},
			nil, false, false,
		},
		{
			traceIDStr, spanIDStr, "00000000000001c", "0", "",
			trace.SpanContextConfig{},
			errInvalidParentSpanIDHeader, false, false,
		},
		{
			traceIDStr, spanIDStr, "00000000000001c80", "0", "",
			trace.SpanContextConfig{},
			errInvalidParentSpanIDHeader, false, false,
		},
	}

	for _, test := range tests {
		ctx, actual, err := extractMultiple(
			context.Background(),
			test.traceID,
			test.spanID,
			test.parentSpanID,
			test.sampled,
			test.flags,
		)
		info := []interface{}{
			"trace ID: %q, span ID: %q, parent span ID: %q, sampled: %q, flags: %q",
			test.traceID,
			test.spanID,
			test.parentSpanID,
			test.sampled,
			test.flags,
		}
		if !assert.Equal(t, test.err, err, info...) {
			continue
		}
		assert.Equal(t, trace.NewSpanContext(test.expected), actual, info...)
		assert.Equal(t, debugFromContext(ctx), test.debug, info...)
		assert.Equal(t, deferredFromContext(ctx), test.deferred, info...)
	}
}

func TestExtractSingle(t *testing.T) {
	tests := []struct {
		header   string
		expected trace.SpanContextConfig
		err      error
		debug    bool
		deferred bool
	}{
		{"0", trace.SpanContextConfig{}, nil, false, false},
		{"1", trace.SpanContextConfig{TraceFlags: trace.FlagsSampled}, nil, false, false},
		{"d", trace.SpanContextConfig{TraceFlags: trace.FlagsSampled}, nil, true, false},
		{"a", trace.SpanContextConfig{}, errInvalidSampledByte, false, false},
		{"3", trace.SpanContextConfig{}, errInvalidSampledByte, false, false},
		{"000000000000007b", trace.SpanContextConfig{}, errInvalidScope, false, false},
		{"000000000000007b00000000000001c8", trace.SpanContextConfig{}, errInvalidScope, false, false},
		// TraceID with illegal length
		{
			"000001c8-000000000000007b",
			trace.SpanContextConfig{},
			errInvalidTraceIDValue, false, false,
		},
		// SpanID with illegal length
		{
			"000000000000007b00000000000001c8-0000007b",
			trace.SpanContextConfig{},
			errInvalidSpanIDValue, false, false,
		},
		// Support short trace IDs.
		{
			"00000000000001c8-000000000000007b",
			trace.SpanContextConfig{
				TraceID: trace.TraceID{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x1, 0xc8},
				SpanID:  spanID,
			},
			nil, false, true,
		},
		{
			"000000000000007b00000000000001c8-000000000000007b",
			trace.SpanContextConfig{
				TraceID: traceID,
				SpanID:  spanID,
			},
			nil, false, true,
		},
		{
			"000000000000007b00000000000001c8-000000000000007b-",
			trace.SpanContextConfig{},
			errInvalidSampledByte, false, false,
		},
		{
			"000000000000007b00000000000001c8-000000000000007b-3",
			trace.SpanContextConfig{},
			errInvalidSampledByte, false, false,
		},
		{
			"000000000000007b00000000000001c8-000000000000007b-00000000000001c8",
			trace.SpanContextConfig{},
			errInvalidScopeParentSingle, false, false,
		},
		{
			"000000000000007b00000000000001c8-000000000000007b-1",
			trace.SpanContextConfig{TraceID: traceID, SpanID: spanID, TraceFlags: trace.FlagsSampled},
			nil, false, false,
		},
		// ParentSpanID is discarded, but should still result in a parsable header.
		{
			"000000000000007b00000000000001c8-000000000000007b-1-00000000000001c8",
			trace.SpanContextConfig{TraceID: traceID, SpanID: spanID, TraceFlags: trace.FlagsSampled},
			nil, false, false,
		},
		{
			"000000000000007b00000000000001c8-000000000000007b-1-00000000000001c",
			trace.SpanContextConfig{},
			errInvalidParentSpanIDValue, false, false,
		},
		{"", trace.SpanContextConfig{}, errEmptyContext, false, false},
	}

	for _, test := range tests {
		ctx, actual, err := extractSingle(context.Background(), test.header)
		if !assert.Equal(t, test.err, err, "header: %s", test.header) {
			continue
		}
		assert.Equal(t, trace.NewSpanContext(test.expected), actual, "header: %s", test.header)
		assert.Equal(t, debugFromContext(ctx), test.debug)
		assert.Equal(t, deferredFromContext(ctx), test.deferred)
	}
}

func TestB3EncodingOperations(t *testing.T) {
	encodings := []Encoding{
		B3MultipleHeader,
		B3SingleHeader,
		B3Unspecified,
	}

	// Test for overflow (or something really unexpected).
	for i, e := range encodings {
		for j := i + 1; j < i+len(encodings); j++ {
			o := encodings[j%len(encodings)]
			assert.NotEqual(t, e, o, "%v == %v", e, o)
		}
	}

	// B3Unspecified is a special case, it supports only itself, but is
	// supported by everything.
	assert.True(t, B3Unspecified.supports(B3Unspecified))
	for _, e := range encodings[:len(encodings)-1] {
		assert.False(t, B3Unspecified.supports(e), e)
		assert.True(t, e.supports(B3Unspecified), e)
	}

	// Skip the special case for B3Unspecified.
	for i, e := range encodings[:len(encodings)-1] {
		// Everything should support itself.
		assert.True(t, e.supports(e))
		for j := i + 1; j < i+len(encodings); j++ {
			o := encodings[j%len(encodings)]
			// Any "or" combination should be supportive of an operand.
			assert.True(t, (e | o).supports(e), "(%[0]v|%[1]v).supports(%[0]v)", e, o)
			// Bitmasks should be unique.
			assert.False(t, o.supports(e), "%v.supports(%v)", o, e)
		}
	}

	// Encoding.supports should be more inclusive than equality.
	all := ^B3Unspecified
	for _, e := range encodings {
		assert.True(t, all.supports(e))
	}
}
