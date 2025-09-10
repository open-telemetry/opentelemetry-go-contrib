// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package skywalking

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type extractTest struct {
	name        string
	headers     map[string]string
	expected    trace.SpanContextConfig
	tracingMode string
}

type injectTest struct {
	name        string
	scc         trace.SpanContextConfig
	tracingMode string
	baggage     map[string]string
	wantHeaders map[string]string
}

var extractHeaders = []extractTest{
	{
		name: "valid sw8 header with normal tracing mode",
		headers: map[string]string{
			"sw8": "1-" + base64.StdEncoding.EncodeToString([]byte(traceID.String())) +
				"-" + base64.StdEncoding.EncodeToString([]byte(spanID.String())) +
				"-123-" + base64.StdEncoding.EncodeToString([]byte("service")) +
				"-" + base64.StdEncoding.EncodeToString([]byte("instance")) +
				"-" + base64.StdEncoding.EncodeToString([]byte("endpoint")) +
				"-" + base64.StdEncoding.EncodeToString([]byte("target")),
			"sw8-x": "0- ",
		},
		expected: trace.SpanContextConfig{
			TraceID:    traceID,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
			Remote:     true,
		},
		tracingMode: TracingModeNormal,
	},
	{
		name: "valid sw8 header with skip analysis mode",
		headers: map[string]string{
			"sw8": "1-" + base64.StdEncoding.EncodeToString([]byte(traceID.String())) +
				"-" + base64.StdEncoding.EncodeToString([]byte(spanID.String())) +
				"-123-" + base64.StdEncoding.EncodeToString([]byte("service")) +
				"-" + base64.StdEncoding.EncodeToString([]byte("instance")) +
				"-" + base64.StdEncoding.EncodeToString([]byte("endpoint")) +
				"-" + base64.StdEncoding.EncodeToString([]byte("target")),
			"sw8-x": "1- ",
		},
		expected: trace.SpanContextConfig{
			TraceID:    traceID,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
			Remote:     true,
		},
		tracingMode: TracingModeSkipAnalysis,
	},
	{
		name: "valid sw8 header without sw8-x extension",
		headers: map[string]string{
			"sw8": "0-" + base64.StdEncoding.EncodeToString([]byte(traceID.String())) +
				"-" + base64.StdEncoding.EncodeToString([]byte(spanID.String())) +
				"-123-" + base64.StdEncoding.EncodeToString([]byte("service")) +
				"-" + base64.StdEncoding.EncodeToString([]byte("instance")) +
				"-" + base64.StdEncoding.EncodeToString([]byte("endpoint")) +
				"-" + base64.StdEncoding.EncodeToString([]byte("target")),
		},
		expected: trace.SpanContextConfig{
			TraceID:    traceID,
			SpanID:     spanID,
			TraceFlags: 0, // Not sampled
			Remote:     true,
		},
		tracingMode: TracingModeNormal, // Default when sw8-x is missing
	},
	{
		name: "sw8 header with correlation data",
		headers: map[string]string{
			"sw8": "1-" + base64.StdEncoding.EncodeToString([]byte(traceID.String())) +
				"-" + base64.StdEncoding.EncodeToString([]byte(spanID.String())) +
				"-123-" + base64.StdEncoding.EncodeToString([]byte("service")) +
				"-" + base64.StdEncoding.EncodeToString([]byte("instance")) +
				"-" + base64.StdEncoding.EncodeToString([]byte("endpoint")) +
				"-" + base64.StdEncoding.EncodeToString([]byte("target")),
			"sw8-correlation": base64.StdEncoding.EncodeToString([]byte("user.id")) + ":" + base64.StdEncoding.EncodeToString([]byte("12345")),
		},
		expected: trace.SpanContextConfig{
			TraceID:    traceID,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
			Remote:     true,
		},
		tracingMode: TracingModeNormal,
	},
}

var invalidExtractHeaders = []extractTest{
	{
		name: "missing sw8 header",
		headers: map[string]string{
			"sw8-x": "0- ",
		},
		expected:    trace.SpanContextConfig{},
		tracingMode: TracingModeNormal,
	},
	{
		name: "malformed sw8 header",
		headers: map[string]string{
			"sw8": "invalid-format",
		},
		expected:    trace.SpanContextConfig{},
		tracingMode: TracingModeNormal,
	},
	{
		name: "insufficient fields in sw8 header",
		headers: map[string]string{
			"sw8": "1-" + base64.StdEncoding.EncodeToString([]byte(traceID.String())),
		},
		expected:    trace.SpanContextConfig{},
		tracingMode: TracingModeNormal,
	},
}

var injectHeaders = []injectTest{
	{
		name: "sampled span with normal tracing mode",
		scc: trace.SpanContextConfig{
			TraceID:    traceID,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
		},
		tracingMode: TracingModeNormal,
		wantHeaders: map[string]string{
			"sw8":   "1-", // Should start with sampled flag
			"sw8-x": "0- ",
		},
	},
	{
		name: "sampled span with skip analysis mode",
		scc: trace.SpanContextConfig{
			TraceID:    traceID,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
		},
		tracingMode: TracingModeSkipAnalysis,
		wantHeaders: map[string]string{
			"sw8":   "1-", // Should start with sampled flag
			"sw8-x": "1- ",
		},
	},
	{
		name: "not sampled span",
		scc: trace.SpanContextConfig{
			TraceID:    traceID,
			SpanID:     spanID,
			TraceFlags: 0,
		},
		tracingMode: TracingModeNormal,
		wantHeaders: map[string]string{
			"sw8":   "0-", // Should start with not sampled flag
			"sw8-x": "0- ",
		},
	},
	{
		name: "span with correlation baggage",
		scc: trace.SpanContextConfig{
			TraceID:    traceID,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
		},
		tracingMode: TracingModeNormal,
		baggage: map[string]string{
			"user.id":      "12345",
			"service.name": "test-service",
		},
		wantHeaders: map[string]string{
			"sw8":             "1-", // Should start with sampled flag
			"sw8-x":           "0- ",
			"sw8-correlation": "", // Will be validated separately
		},
	},
}

func TestExtractSkyWalking(t *testing.T) {
	testGroup := []struct {
		name  string
		tests []extractTest
	}{
		{
			name:  "valid extract headers",
			tests: extractHeaders,
		},
		{
			name:  "invalid extract headers",
			tests: invalidExtractHeaders,
		},
	}

	for _, tg := range testGroup {
		propagator := Skywalking{}

		for _, tt := range tg.tests {
			t.Run(tt.name, func(t *testing.T) {
				header := make(http.Header, len(tt.headers))
				for h, v := range tt.headers {
					header.Set(h, v)
				}

				ctx := context.Background()
				ctx = propagator.Extract(ctx, propagation.HeaderCarrier(header))
				gotSc := trace.SpanContextFromContext(ctx)

				comparer := cmp.Comparer(func(a, b trace.SpanContext) bool {
					// Do not compare remote field, it is unset on empty
					// SpanContext.
					newA := a.WithRemote(b.IsRemote())
					return newA.Equal(b)
				})
				if diff := cmp.Diff(gotSc, trace.NewSpanContext(tt.expected), comparer); diff != "" {
					t.Errorf("%s: %s: -got +want %s", tg.name, tt.name, diff)
				}

				// Verify tracing mode is extracted correctly
				assert.Equal(t, tt.tracingMode, TracingModeFromContext(ctx))

				// If correlation header is present, verify baggage is extracted
				if correlationHeader, ok := tt.headers["sw8-correlation"]; ok && correlationHeader != "" {
					bags := baggage.FromContext(ctx)
					assert.Positive(t, bags.Len(), "baggage should be extracted from correlation header")
				}
			})
		}
	}
}

func TestInjectSkyWalking(t *testing.T) {
	for _, tt := range injectHeaders {
		propagator := Skywalking{}
		t.Run(tt.name, func(t *testing.T) {
			header := http.Header{}
			ctx := trace.ContextWithSpanContext(
				context.Background(),
				trace.NewSpanContext(tt.scc),
			)
			ctx = WithTracingMode(ctx, tt.tracingMode)

			// Add baggage if specified
			if tt.baggage != nil {
				var members []baggage.Member
				for k, v := range tt.baggage {
					member, err := baggage.NewMember(k, v)
					require.NoError(t, err)
					members = append(members, member)
				}
				bags, err := baggage.New(members...)
				require.NoError(t, err)
				ctx = baggage.ContextWithBaggage(ctx, bags)
			}

			propagator.Inject(ctx, propagation.HeaderCarrier(header))

			for h, expectedPrefix := range tt.wantHeaders {
				got := header.Get(h)
				if h == "sw8-correlation" {
					if tt.baggage != nil {
						assert.NotEmpty(t, got, "correlation header should be set when baggage is present")
					} else {
						assert.Empty(t, got, "correlation header should not be set when no baggage")
					}
				} else if expectedPrefix != "" {
					assert.True(t, strings.HasPrefix(got, expectedPrefix),
						"header %s should start with %s, got: %s", h, expectedPrefix, got)
				}
			}
		})
	}
}

func TestSkyWalkingPropagator_Fields_Integration(t *testing.T) {
	propagator := Skywalking{}
	want := []string{
		"sw8",
		"sw8-correlation",
		"sw8-x",
	}

	got := propagator.Fields()
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("Fields: -got +want %s", diff)
	}
}

func TestSkyWalkingTracingModeRoundTrip(t *testing.T) {
	propagator := Skywalking{}

	testCases := []struct {
		name        string
		tracingMode string
	}{
		{
			name:        "normal tracing mode",
			tracingMode: TracingModeNormal,
		},
		{
			name:        "skip analysis mode",
			tracingMode: TracingModeSkipAnalysis,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create original context with span and tracing mode
			originalSC := trace.NewSpanContext(trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: trace.FlagsSampled,
			})

			originalCtx := trace.ContextWithSpanContext(context.Background(), originalSC)
			originalCtx = WithTracingMode(originalCtx, tc.tracingMode)

			// Inject into carrier
			carrier := make(propagation.MapCarrier)
			propagator.Inject(originalCtx, carrier)

			// Extract from carrier
			extractedCtx := propagator.Extract(context.Background(), carrier)

			// Verify span context round trip
			extractedSC := trace.SpanContextFromContext(extractedCtx)
			assert.True(t, extractedSC.IsValid())
			assert.Equal(t, originalSC.TraceID(), extractedSC.TraceID())
			assert.Equal(t, originalSC.SpanID(), extractedSC.SpanID())
			assert.Equal(t, originalSC.IsSampled(), extractedSC.IsSampled())

			// Verify tracing mode round trip
			extractedMode := TracingModeFromContext(extractedCtx)
			assert.Equal(t, tc.tracingMode, extractedMode)
		})
	}
}

func TestSkyWalkingCompleteIntegration(t *testing.T) {
	propagator := Skywalking{}

	// Create original context with span, tracing mode, and baggage
	originalSC := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})

	member1, _ := baggage.NewMember("user.id", "12345")
	member2, _ := baggage.NewMember("service.name", "test-service")
	originalBags, err := baggage.New(member1, member2)
	require.NoError(t, err)

	originalCtx := trace.ContextWithSpanContext(context.Background(), originalSC)
	originalCtx = WithTracingMode(originalCtx, TracingModeSkipAnalysis)
	originalCtx = baggage.ContextWithBaggage(originalCtx, originalBags)

	// Inject into carrier
	carrier := make(propagation.MapCarrier)
	propagator.Inject(originalCtx, carrier)

	// Verify all headers are set
	assert.NotEmpty(t, carrier.Get("sw8"))
	assert.NotEmpty(t, carrier.Get("sw8-correlation"))
	assert.Equal(t, "1- ", carrier.Get("sw8-x")) // Skip analysis mode with placeholder timestamp

	// Extract from carrier
	extractedCtx := propagator.Extract(context.Background(), carrier)

	// Verify complete round trip
	extractedSC := trace.SpanContextFromContext(extractedCtx)
	assert.True(t, extractedSC.IsValid())
	assert.Equal(t, originalSC.TraceID(), extractedSC.TraceID())
	assert.Equal(t, originalSC.SpanID(), extractedSC.SpanID())
	assert.Equal(t, originalSC.IsSampled(), extractedSC.IsSampled())

	extractedMode := TracingModeFromContext(extractedCtx)
	assert.Equal(t, TracingModeSkipAnalysis, extractedMode)

	extractedBags := baggage.FromContext(extractedCtx)
	assert.Equal(t, 2, extractedBags.Len())
	assert.Equal(t, "12345", extractedBags.Member("user.id").Value())
	assert.Equal(t, "test-service", extractedBags.Member("service.name").Value())
}

func TestSkyWalkingTimestampIntegration(t *testing.T) {
	propagator := Skywalking{}

	testCases := []struct {
		name      string
		timestamp int64
		expected  string
	}{
		{
			name:      "no timestamp",
			timestamp: 0,
			expected:  "0- ",
		},
		{
			name:      "with timestamp",
			timestamp: 1602743904804,
			expected:  "0-1602743904804",
		},
		{
			name:      "with current time",
			timestamp: 1640995200000,
			expected:  "0-1640995200000",
		},
		{
			name:      "skip analysis with timestamp",
			timestamp: 1602743904804,
			expected:  "1-1602743904804",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create span context
			sc := trace.NewSpanContext(trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: trace.FlagsSampled,
			})

			ctx := trace.ContextWithSpanContext(context.Background(), sc)

			// Set tracing mode based on test expectation
			if strings.HasPrefix(tc.expected, "1") {
				ctx = WithTracingMode(ctx, TracingModeSkipAnalysis)
			}

			// Set timestamp if provided
			if tc.timestamp > 0 {
				ctx = WithTimestamp(ctx, tc.timestamp)
			}

			// Inject
			carrier := make(propagation.MapCarrier)
			propagator.Inject(ctx, carrier)

			// Verify SW8-X header format
			sw8XValue := carrier.Get("sw8-x")
			assert.Equal(t, tc.expected, sw8XValue)

			// Extract and verify round trip
			extractedCtx := propagator.Extract(context.Background(), carrier)

			// Verify timestamp round trip
			extractedTimestamp := TimestampFromContext(extractedCtx)
			assert.Equal(t, tc.timestamp, extractedTimestamp)

			// Verify tracing mode round trip
			expectedMode := TracingModeNormal
			if strings.HasPrefix(tc.expected, "1") {
				expectedMode = TracingModeSkipAnalysis
			}
			extractedMode := TracingModeFromContext(extractedCtx)
			assert.Equal(t, expectedMode, extractedMode)
		})
	}
}

func TestSkyWalkingTimestampExtraction(t *testing.T) {
	propagator := Skywalking{}

	testCases := []struct {
		name              string
		sw8XValue         string
		expectedTimestamp int64
		expectedMode      string
	}{
		{
			name:              "empty header",
			sw8XValue:         "",
			expectedTimestamp: 0,
			expectedMode:      TracingModeNormal,
		},
		{
			name:              "only tracing mode",
			sw8XValue:         "1",
			expectedTimestamp: 0,
			expectedMode:      TracingModeSkipAnalysis,
		},
		{
			name:              "tracing mode with empty timestamp",
			sw8XValue:         "0- ",
			expectedTimestamp: 0,
			expectedMode:      TracingModeNormal,
		},
		{
			name:              "tracing mode with timestamp",
			sw8XValue:         "1-1602743904804",
			expectedTimestamp: 1602743904804,
			expectedMode:      TracingModeSkipAnalysis,
		},
		{
			name:              "invalid timestamp format",
			sw8XValue:         "0-invalid",
			expectedTimestamp: 0,
			expectedMode:      TracingModeNormal,
		},
		{
			name:              "negative timestamp",
			sw8XValue:         "0- -123",
			expectedTimestamp: 0, // Malformed input (double separator) should not parse
			expectedMode:      TracingModeNormal,
		},
		{
			name:              "malformed negative timestamp",
			sw8XValue:         "1--123", // This creates parsing ambiguity with double separator
			expectedTimestamp: 0,        // Cannot parse due to ambiguous format
			expectedMode:      TracingModeSkipAnalysis,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			carrier := make(propagation.MapCarrier)

			// Add a valid SW8 header for SW8-X extraction to work
			validSW8 := "1-" + base64.StdEncoding.EncodeToString([]byte(traceID.String())) +
				"-" + base64.StdEncoding.EncodeToString([]byte(spanID.String())) +
				"-123-" + base64.StdEncoding.EncodeToString([]byte("service")) +
				"-" + base64.StdEncoding.EncodeToString([]byte("instance")) +
				"-" + base64.StdEncoding.EncodeToString([]byte("endpoint")) +
				"-" + base64.StdEncoding.EncodeToString([]byte("target"))
			carrier.Set("sw8", validSW8)

			if tc.sw8XValue != "" {
				carrier.Set("sw8-x", tc.sw8XValue)
			}

			extractedCtx := propagator.Extract(context.Background(), carrier)

			extractedTimestamp := TimestampFromContext(extractedCtx)
			assert.Equal(t, tc.expectedTimestamp, extractedTimestamp)

			extractedMode := TracingModeFromContext(extractedCtx)
			assert.Equal(t, tc.expectedMode, extractedMode)
		})
	}
}
