// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xray

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/trace"
)

var (
	traceID                    = trace.TraceID{0x8a, 0x3c, 0x60, 0xf7, 0xd1, 0x88, 0xf8, 0xfa, 0x79, 0xd4, 0x8a, 0x39, 0x1a, 0x77, 0x8f, 0xa6}
	xrayTraceID                = "1-8a3c60f7-d188f8fa79d48a391a778fa6"
	xrayTraceIDIncorrectLength = "1-82138-1203123"
	parentID64Str              = "53995c3f42cd8ad8"
	parentSpanID               = trace.SpanID{0x53, 0x99, 0x5c, 0x3f, 0x42, 0xcd, 0x8a, 0xd8}
	zeroSpanIDStr              = "0000000000000000"
	wrongVersionTraceHeaderID  = "5b00000000b000000000000000000000000"
)

func TestAwsXrayExtract(t *testing.T) {
	testData := []struct {
		traceID      string
		parentSpanID string
		samplingFlag string
		expected     trace.SpanContextConfig
		err          error
	}{
		{
			xrayTraceID, parentID64Str, notSampled,
			trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     parentSpanID,
				TraceFlags: traceFlagNone,
			},
			nil,
		},
		{
			xrayTraceID, parentID64Str, isSampled,
			trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     parentSpanID,
				TraceFlags: traceFlagSampled,
			},
			nil,
		},
		{
			xrayTraceID, zeroSpanIDStr, isSampled,
			trace.SpanContextConfig{},
			errInvalidSpanIDLength,
		},
		{
			xrayTraceIDIncorrectLength, parentID64Str, isSampled,
			trace.SpanContextConfig{},
			errLengthTraceIDHeader,
		},
		{
			wrongVersionTraceHeaderID, parentID64Str, isSampled,
			trace.SpanContextConfig{},
			errInvalidTraceIDVersion,
		},
	}

	for _, test := range testData {
		headerVal := strings.Join([]string{
			traceIDKey, kvDelimiter, test.traceID, traceHeaderDelimiter, parentIDKey, kvDelimiter,
			test.parentSpanID, traceHeaderDelimiter, sampleFlagKey, kvDelimiter, test.samplingFlag,
		}, "")

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

		assert.Equal(t, trace.NewSpanContext(test.expected), sc, info...)
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

func TestXrayExtract(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name            string
		headerValue     string
		expectedTraceID string
		expectedSpanID  string
		expectedFlags   trace.TraceFlags
		expectValid     bool
	}{
		{
			name:            "Valid X-Ray header with sampling enabled",
			headerValue:     "Root=1-5759e988-bd862e3fe1be46a994272793;Parent=53995c3f42cd8ad8;Sampled=1",
			expectedTraceID: "5759e988bd862e3fe1be46a994272793",
			expectedSpanID:  "53995c3f42cd8ad8",
			expectedFlags:   trace.FlagsSampled,
			expectValid:     true,
		},
		{
			name:            "Valid X-Ray header with sampling disabled",
			headerValue:     "Root=1-5759e988-bd862e3fe1be46a994272793;Parent=53995c3f42cd8ad8;Sampled=0",
			expectedTraceID: "5759e988bd862e3fe1be46a994272793",
			expectedSpanID:  "53995c3f42cd8ad8",
			expectedFlags:   traceFlagNone,
			expectValid:     true,
		},
		{
			name:        "Missing header",
			headerValue: "",
			expectValid: false,
		},
		{
			name:        "Malformed trace ID",
			headerValue: "Root=1x5759e988xbd862e3fe1be46a994272793;Parent=53995c3f42cd8ad8;Sampled=1",
			expectValid: false,
		},
		{
			name:        "Invalid version in trace ID",
			headerValue: "Root=2-5759e988-bd862e3fe1be46a994272793;Parent=53995c3f42cd8ad8;Sampled=1",
			expectValid: false,
		},
		{
			name:        "Missing parent ID",
			headerValue: "Root=1-5759e988-bd862e3fe1be46a994272793;Sampled=1",
			expectValid: false,
		},
	}
	for _, testCase := range testCases {

		// Create propagator
		prop := &Propagator{}

		// Create original context
		originalCtx := context.Background()

		// Create carrier with header
		headers := map[string]string{}
		if testCase.headerValue != "" {
			headers[traceHeaderKey] = testCase.headerValue
		}
		carrier := NewMockCarrier(headers)

		// Call Extract
		resultCtx := prop.Extract(originalCtx, carrier)

		// Verify results
		if testCase.expectValid {
			// Get span context from result context
			sc := trace.SpanFromContext(resultCtx).SpanContext()

			// Convert expected trace ID to trace.TraceID
			expectedTraceID, _ := trace.TraceIDFromHex(testCase.expectedTraceID)
			expectedSpanID, _ := trace.SpanIDFromHex(testCase.expectedSpanID)

			// Assert values match
			assert.Equal(t, expectedTraceID, sc.TraceID())
			assert.Equal(t, expectedSpanID, sc.SpanID())
			assert.Equal(t, testCase.expectedFlags, sc.TraceFlags())
		} else {
			// For invalid cases, context should be unchanged
			assert.Equal(t, originalCtx, resultCtx)
		}

	}
}

type MockCarrier struct {
	headers map[string]string
}

func NewMockCarrier(h map[string]string) *MockCarrier {
	return &MockCarrier{
		headers: h,
	}
}

func (m *MockCarrier) Get(key string) string {
	return m.headers[key]
}

func (m *MockCarrier) Set(key string, value string) {

}

func (m *MockCarrier) Keys() []string {
	return []string{}
}
