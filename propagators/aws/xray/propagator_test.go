// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xray

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TestExtract(t *testing.T) {
	type extractTestCase struct {
		name        string
		headerVal   string
		wantValid   bool
		wantTraceID string
		wantSpanID  string
		wantSampled bool
	}

	tests := []extractTestCase{
		{
			name:        "Valid header - sampled",
			headerVal:   "Root=1-abcdef12-1234567890abcdef12345678;Parent=1234567890abcdef;Sampled=1",
			wantValid:   true,
			wantTraceID: "abcdef121234567890abcdef12345678",
			wantSpanID:  "1234567890abcdef",
			wantSampled: true,
		},
		{
			name:        "Valid header - not sampled",
			headerVal:   "Root=1-abcdef12-1234567890abcdef12345678;Parent=1234567890abcdef;Sampled=0",
			wantValid:   true,
			wantTraceID: "abcdef121234567890abcdef12345678",
			wantSpanID:  "1234567890abcdef",
			wantSampled: false,
		},
		{
			name:      "Empty header - no trace info",
			headerVal: "",
			wantValid: false,
		},
		{
			name:      "Malformed TraceID - too short",
			headerVal: "Root=1-abc-123;Parent=1234567890abcdef;Sampled=1",
			wantValid: false,
		},
		{
			name:      "Malformed TraceID - missing delimiters",
			headerVal: "Root=1abcdef121234567890abcdef12345678;Parent=1234567890abcdef;Sampled=1",
			wantValid: false,
		},
		{
			name:      "Invalid TraceID version",
			headerVal: "Root=2-abcdef12-1234567890abcdef12345678;Parent=1234567890abcdef;Sampled=1",
			wantValid: false,
		},
		{
			name:      "Invalid SpanID format",
			headerVal: "Root=1-abcdef12-1234567890abcdef12345678;Parent=bad-spanid;Sampled=1",
			wantValid: false,
		},
		{
			name:        "Missing Sampled",
			headerVal:   "Root=1-abcdef12-1234567890abcdef12345678;Parent=1234567890abcdef",
			wantValid:   true,
			wantTraceID: "abcdef121234567890abcdef12345678",
			wantSpanID:  "1234567890abcdef",
			wantSampled: false,
		},
		{
			name:      "Malformed key-value pair - missing '='",
			headerVal: "Root=1-abcdef12-1234567890abcdef12345678;BrokenKeyValue;Sampled=1",
			wantValid: false,
		},
		{
			name:      "Only Sampled key",
			headerVal: "Sampled=1",
			wantValid: false,
		},
		{
			name:      "Missing Root key",
			headerVal: "Parent=1234567890abcdef;Sampled=1",
			wantValid: false,
		},
		{
			name:        "Trailing semicolon",
			headerVal:   "Root=1-abcdef12-1234567890abcdef12345678;Parent=1234567890abcdef;Sampled=1;",
			wantValid:   true,
			wantTraceID: "abcdef121234567890abcdef12345678",
			wantSpanID:  "1234567890abcdef",
			wantSampled: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			carrier := propagation.MapCarrier{}
			if tc.headerVal != "" {
				carrier.Set("X-Amzn-Trace-Id", tc.headerVal)
			}

			// prop := P

			ctx := Propagator{}.Extract(context.Background(), carrier)
			sc := trace.SpanContextFromContext(ctx)

			if sc.IsValid() != tc.wantValid {
				t.Fatalf("expected valid=%v, got %v", tc.wantValid, sc.IsValid())
			}

			if tc.wantValid {
				if got := sc.TraceID().String(); got != tc.wantTraceID {
					t.Errorf("expected TraceID %q, got %q", tc.wantTraceID, got)
				}
				if got := sc.SpanID().String(); got != tc.wantSpanID {
					t.Errorf("expected SpanID %q, got %q", tc.wantSpanID, got)
				}
				if got := sc.IsSampled(); got != tc.wantSampled {
					t.Log("name-->> ", tc.name)
					t.Errorf("expected sampled=%v, got %v", tc.wantSampled, got)
				}
			}
		})
	}
}

func BenchmarkPropagatorExtract(b *testing.B) {
	propagator := Propagator{}

	ctx := context.Background()
	req, _ := http.NewRequest("GET", "http://example.com", nil)

	req.Header.Set("Root", "1-8a3c60f7-d188f8fa79d48a391a778fa6")
	req.Header.Set("Parent", "53995c3f42cd8ad8")
	req.Header.Set("Sampled", "1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = propagator.Extract(ctx, propagation.HeaderCarrier(req.Header))
	}
}

func BenchmarkPropagatorInject(b *testing.B) {
	propagator := Propagator{}
	tracer := otel.Tracer("test")

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	ctx, _ := tracer.Start(context.Background(), "Parent operation...")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		propagator.Inject(ctx, propagation.HeaderCarrier(req.Header))
	}
}

func TestPropagatorFields(t *testing.T) {
	propagator := Propagator{}
	assert.Len(t, propagator.Fields(), 1, "Fields() should return exactly one field")
	assert.Equal(t, []string{traceHeaderKey}, propagator.Fields())
}
