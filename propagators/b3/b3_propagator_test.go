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

package b3

import (
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
	}{
		{
			"", "", "", "0", "",
			trace.SpanContextConfig{},
			nil,
		},
		{
			"", "", "", "", "",
			trace.SpanContextConfig{TraceFlags: trace.FlagsDeferred},
			nil,
		},
		{
			"", "", "", "1", "",
			trace.SpanContextConfig{TraceFlags: trace.FlagsSampled},
			nil,
		},
		{
			"", "", "", "", "1",
			trace.SpanContextConfig{TraceFlags: trace.FlagsSampled | trace.FlagsDebug},
			nil,
		},
		{
			"", "", "", "0", "1",
			trace.SpanContextConfig{TraceFlags: trace.FlagsDebug | trace.FlagsSampled},
			nil,
		},
		{
			"", "", "", "1", "1",
			trace.SpanContextConfig{TraceFlags: trace.FlagsSampled | trace.FlagsDebug},
			nil,
		},
		{
			traceIDStr, spanIDStr, "", "", "",
			trace.SpanContextConfig{TraceID: traceID, SpanID: spanID, TraceFlags: trace.FlagsDeferred},
			nil,
		},
		{
			traceIDStr, spanIDStr, "", "0", "",
			trace.SpanContextConfig{TraceID: traceID, SpanID: spanID},
			nil,
		},
		// Ensure backwards compatibility.
		{
			traceIDStr, spanIDStr, "", "false", "",
			trace.SpanContextConfig{TraceID: traceID, SpanID: spanID},
			nil,
		},
		{
			traceIDStr, spanIDStr, "", "1", "",
			trace.SpanContextConfig{TraceID: traceID, SpanID: spanID, TraceFlags: trace.FlagsSampled},
			nil,
		},
		// Ensure backwards compatibility.
		{
			traceIDStr, spanIDStr, "", "true", "",
			trace.SpanContextConfig{TraceID: traceID, SpanID: spanID, TraceFlags: trace.FlagsSampled},
			nil,
		},
		{
			traceIDStr, spanIDStr, "", "a", "",
			trace.SpanContextConfig{},
			errInvalidSampledHeader,
		},
		{
			traceIDStr, spanIDStr, "", "1", "1",
			trace.SpanContextConfig{TraceID: traceID, SpanID: spanID, TraceFlags: trace.FlagsSampled | trace.FlagsDebug},
			nil,
		},
		// Invalid flags are discarded.
		{
			traceIDStr, spanIDStr, "", "1", "invalid",
			trace.SpanContextConfig{TraceID: traceID, SpanID: spanID, TraceFlags: trace.FlagsSampled},
			nil,
		},
		// Support short trace IDs.
		{
			"00000000000001c8", spanIDStr, "", "0", "",
			trace.SpanContextConfig{
				TraceID: trace.TraceID{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x1, 0xc8},
				SpanID:  spanID,
			},
			nil,
		},
		{
			"00000000000001c", spanIDStr, "", "0", "",
			trace.SpanContextConfig{},
			errInvalidTraceIDHeader,
		},
		{
			"00000000000001c80", spanIDStr, "", "0", "",
			trace.SpanContextConfig{},
			errInvalidTraceIDHeader,
		},
		{
			traceIDStr[:len(traceIDStr)-2], spanIDStr, "", "0", "",
			trace.SpanContextConfig{},
			errInvalidTraceIDHeader,
		},
		{
			traceIDStr + "0", spanIDStr, "", "0", "",
			trace.SpanContextConfig{},
			errInvalidTraceIDHeader,
		},
		{
			traceIDStr, "00000000000001c", "", "0", "",
			trace.SpanContextConfig{},
			errInvalidSpanIDHeader,
		},
		{
			traceIDStr, "00000000000001c80", "", "0", "",
			trace.SpanContextConfig{},
			errInvalidSpanIDHeader,
		},
		{
			traceIDStr, "", "", "0", "",
			trace.SpanContextConfig{},
			errInvalidScope,
		},
		{
			"", spanIDStr, "", "0", "",
			trace.SpanContextConfig{},
			errInvalidScope,
		},
		{
			"", "", spanIDStr, "0", "",
			trace.SpanContextConfig{},
			errInvalidScopeParent,
		},
		{
			traceIDStr, spanIDStr, "00000000000001c8", "0", "",
			trace.SpanContextConfig{TraceID: traceID, SpanID: spanID},
			nil,
		},
		{
			traceIDStr, spanIDStr, "00000000000001c", "0", "",
			trace.SpanContextConfig{},
			errInvalidParentSpanIDHeader,
		},
		{
			traceIDStr, spanIDStr, "00000000000001c80", "0", "",
			trace.SpanContextConfig{},
			errInvalidParentSpanIDHeader,
		},
	}

	for _, test := range tests {
		actual, err := extractMultiple(
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
	}
}

func TestExtractSingle(t *testing.T) {
	tests := []struct {
		header   string
		expected trace.SpanContextConfig
		err      error
	}{
		{"0", trace.SpanContextConfig{}, nil},
		{"1", trace.SpanContextConfig{TraceFlags: trace.FlagsSampled}, nil},
		{"d", trace.SpanContextConfig{TraceFlags: trace.FlagsDebug | trace.FlagsSampled}, nil},
		{"a", trace.SpanContextConfig{}, errInvalidSampledByte},
		{"3", trace.SpanContextConfig{}, errInvalidSampledByte},
		{"000000000000007b", trace.SpanContextConfig{}, errInvalidScope},
		{"000000000000007b00000000000001c8", trace.SpanContextConfig{}, errInvalidScope},
		// Support short trace IDs.
		{
			"00000000000001c8-000000000000007b",
			trace.SpanContextConfig{
				TraceID:    trace.TraceID{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x1, 0xc8},
				SpanID:     spanID,
				TraceFlags: trace.FlagsDeferred,
			},
			nil,
		},
		{
			"000000000000007b00000000000001c8-000000000000007b",
			trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: trace.FlagsDeferred,
			},
			nil,
		},
		{
			"000000000000007b00000000000001c8-000000000000007b-",
			trace.SpanContextConfig{},
			errInvalidSampledByte,
		},
		{
			"000000000000007b00000000000001c8-000000000000007b-3",
			trace.SpanContextConfig{},
			errInvalidSampledByte,
		},
		{
			"000000000000007b00000000000001c8-000000000000007b-00000000000001c8",
			trace.SpanContextConfig{},
			errInvalidScopeParentSingle,
		},
		{
			"000000000000007b00000000000001c8-000000000000007b-1",
			trace.SpanContextConfig{TraceID: traceID, SpanID: spanID, TraceFlags: trace.FlagsSampled},
			nil,
		},
		// ParentSpanID is discarded, but should still result in a parsable header.
		{
			"000000000000007b00000000000001c8-000000000000007b-1-00000000000001c8",
			trace.SpanContextConfig{TraceID: traceID, SpanID: spanID, TraceFlags: trace.FlagsSampled},
			nil,
		},
		{
			"000000000000007b00000000000001c8-000000000000007b-1-00000000000001c",
			trace.SpanContextConfig{},
			errInvalidParentSpanIDValue,
		},
		{"", trace.SpanContextConfig{}, errEmptyContext},
	}

	for _, test := range tests {
		actual, err := extractSingle(test.header)
		if !assert.Equal(t, test.err, err, "header: %s", test.header) {
			continue
		}
		assert.Equal(t, trace.NewSpanContext(test.expected), actual, "header: %s", test.header)
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
			assert.False(t, e == o, "%v == %v", e, o)
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
