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
		expected trace.SpanContext
		err      error
	}{
		{
			traceID128Str, spanIDStr, "1",
			trace.SpanContext{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: trace.FlagsSampled,
			},
			nil,
		},
		{
			traceID64Str, spanIDStr, "1",
			trace.SpanContext{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: trace.FlagsSampled,
			},
			nil,
		},
		{
			traceID128Str, spanIDStr, "",
			trace.SpanContext{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: trace.FlagsDeferred,
			},
			nil,
		},
		{
			// if we didn't set sampled bit when debug bit is 1, then assuming it's not sampled
			traceID128Str, spanIDStr, "0",
			trace.SpanContext{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: 0x00,
			},
			nil,
		},
		{
			traceID128Str, spanIDStr, "1",
			trace.SpanContext{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: trace.FlagsSampled,
			},
			nil,
		},
		{
			fmt.Sprintf("%32s", "This_is_a_string_len_64"), spanIDStr, "1",
			trace.SpanContext{},
			errInvalidTraceIDHeader,
		},
		{
			"000000000007b00000000000001c8", spanIDStr, "1",
			trace.SpanContext{},
			errInvalidTraceIDHeader,
		},
		{
			traceID128Str, fmt.Sprintf("%16s", "wiredspanid"), "1",
			trace.SpanContext{},
			errInvalidSpanIDHeader,
		},
		{
			traceID128Str, "0000000000010", "1",
			trace.SpanContext{},
			errInvalidSpanIDHeader,
		},
		{
			// reject invalid traceID(0) and spanID(0)
			zeroTraceIDStr, zeroSpanIDStr, "1",
			trace.SpanContext{},
			errInvalidTraceIDHeader,
		},
		{
			// reject invalid spanID(0)
			traceID128Str, zeroSpanIDStr, "1",
			trace.SpanContext{},
			errInvalidSpanIDHeader,
		},
		{
			// reject invalid spanID(0)
			traceID128Str, spanIDStr, "invalid",
			trace.SpanContext{},
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

		assert.Equal(t, test.expected, sc, info...)
	}
}
