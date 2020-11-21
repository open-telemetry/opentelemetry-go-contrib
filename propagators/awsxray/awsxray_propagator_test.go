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

package aws

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/api/trace"
)

var (
	traceID                   = trace.ID{0x8a, 0x3c, 0x60, 0xf7, 0xd1, 0x88, 0xf8, 0xfa, 0x79, 0xd4, 0x8a, 0x39, 0x1a, 0x77, 0x8f, 0xa6}
	xrayTraceID               = "1-8a3c60f7-d188f8fa79d48a391a778fa6"
	parentID64Str             = "53995c3f42cd8ad8"
	parentSpanID              = trace.SpanID{0x53, 0x99, 0x5c, 0x3f, 0x42, 0xcd, 0x8a, 0xd8}
	zeroSpanIDStr             = "0000000000000000"
	zeroTraceIDStr            = "1-00000000-000000000000000000000000"
	invalidTraceHeaderID      = "1b00000000b000000000000000000000000"
	wrongVersionTraceHeaderID = "5b00000000b000000000000000000000000"
)

func TestAwsXrayExtract(t *testing.T) {
	testData := []struct {
		traceID      string
		parentSpanID string
		samplingFlag string
		expected     trace.SpanContext
		err          error
	}{
		{
			xrayTraceID, parentID64Str, notSampled,
			trace.SpanContext{
				TraceID:    traceID,
				SpanID:     parentSpanID,
				TraceFlags: traceFlagNone,
			},
			nil,
		},
		{
			xrayTraceID, parentID64Str, isSampled,
			trace.SpanContext{
				TraceID:    traceID,
				SpanID:     parentSpanID,
				TraceFlags: traceFlagSampled,
			},
			nil,
		},
		{
			zeroTraceIDStr, parentID64Str, isSampled,
			trace.SpanContext{},
			errMalformedTraceID,
		},
		{
			xrayTraceID, zeroSpanIDStr, isSampled,
			trace.SpanContext{},
			errInvalidSpanIDLength,
		},
		{
			invalidTraceHeaderID, parentID64Str, isSampled,
			trace.SpanContext{},
			errMalformedTraceID,
		},
		{
			wrongVersionTraceHeaderID, parentID64Str, isSampled,
			trace.SpanContext{},
			errMalformedTraceID,
		},
	}

	for _, test := range testData {
		headerVal := strings.Join([]string{traceIDKey, kvDelimiter, test.traceID, traceHeaderDelimiter, parentIDKey, kvDelimiter,
			test.parentSpanID, traceHeaderDelimiter, sampleFlagKey, kvDelimiter, test.samplingFlag}, "")

		sc, err := extract(headerVal)

		info := []interface{}{
			"trace ID: %q, parent span ID: %q, sampling flag: %q",
			test.traceID,
			test.parentSpanID,
			test.samplingFlag,
		}

		if !assert.Equal(t, test.err, err, info...) {
			continue
		}

		assert.Equal(t, test.expected, sc, info...)
	}
}
