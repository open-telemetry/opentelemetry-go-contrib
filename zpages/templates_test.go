// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zpages

import (
	"html/template"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/trace"
)

func TestSpanRowFormatter(t *testing.T) {
	for _, tt := range []struct {
		name string
		row  spanRow

		expectedTemplate template.HTML
	}{
		{
			name: "with an invalid span context",
			row: spanRow{
				SpanContext: trace.NewSpanContext(trace.SpanContextConfig{}),
			},
			expectedTemplate: "",
		},
		{
			name: "with a valid span context",
			row: spanRow{
				SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
					TraceID: trace.TraceID{2, 3, 4, 5, 6, 7, 8, 9, 2, 3, 4, 5, 6, 7, 8, 9},
					SpanID:  trace.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
				}),
			},
			expectedTemplate: "trace_id: <b style=\"color:black\">02030405060708090203040506070809</b> span_id: 0102030405060708",
		},
		{
			name: "with a valid parent span context",
			row: spanRow{
				SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
					TraceID: trace.TraceID{2, 3, 4, 5, 6, 7, 8, 9, 2, 3, 4, 5, 6, 7, 8, 9},
					SpanID:  trace.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
				}),
				ParentSpanContext: trace.NewSpanContext(trace.SpanContextConfig{
					TraceID: trace.TraceID{10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25},
					SpanID:  trace.SpanID{10, 11, 12, 13, 14, 15, 16, 18},
				}),
			},
			expectedTemplate: "trace_id: <b style=\"color:black\">02030405060708090203040506070809</b> span_id: 0102030405060708 parent_span_id: 0a0b0c0d0e0f1012",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := spanRowFormatter(tt.row)
			assert.Equal(t, tt.expectedTemplate, r)
		})
	}
}
