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
	traceID        = trace.TraceID{0, 0, 0, 0, 0, 0, 0, 0, 0x7b, 0, 0, 0, 0, 0, 0x1, 0xc8}
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
		expected     trace.SpanContextConfig
		err          error
		debug        bool
	}{
		{
			traceID128Str, spanIDStr, deprecatedParentSpanID, "1",
			trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: trace.FlagsSampled,
			},
			nil,
			false,
		},
		{
			traceID64Str, spanIDStr, deprecatedParentSpanID, "1",
			trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: trace.FlagsSampled,
			},
			nil,
			false,
		},
		{
			traceID128Str, spanIDStr, deprecatedParentSpanID, "3",
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
			traceID128Str, spanIDStr, deprecatedParentSpanID, "2",
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
			traceID128Str, spanIDStr, deprecatedParentSpanID, "8",
			trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: 0x00,
			},
			nil,
			false,
		},
		{
			traceID128Str, spanIDStr, deprecatedParentSpanID, "9",
			trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: trace.FlagsSampled,
			},
			nil,
			false,
		},
		{
			traceID128Str, spanIDStr, "wired stuff", "1",
			trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: trace.FlagsSampled,
			},
			nil,
			false,
		},
		{
			fmt.Sprintf("%32s", "This_is_a_string_len_64"), spanIDStr, deprecatedParentSpanID, "1",
			trace.SpanContextConfig{},
			errMalformedTraceID,
			false,
		},
		{
			"000000000007b00000000000001c8", spanIDStr, deprecatedParentSpanID, "1",
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
			traceID128Str, "0000000000010", deprecatedParentSpanID, "1",
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
