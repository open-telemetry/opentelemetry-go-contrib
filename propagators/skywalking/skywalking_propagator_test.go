// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package skywalking

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

var (
	traceID = trace.TraceID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}
	spanID  = trace.SpanID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
)

func TestSkyWalkingPropagator_Interface(_ *testing.T) {
	var _ propagation.TextMapPropagator = &Skywalking{}
}

func TestSkyWalkingPropagator_Fields(t *testing.T) {
	p := Skywalking{}
	fields := p.Fields()

	assert.Contains(t, fields, sw8Header)
	assert.Contains(t, fields, sw8CorrelationHeader)
	assert.Contains(t, fields, sw8ExtensionHeader)
	assert.Len(t, fields, 3)
}

func TestSkyWalkingPropagator_Inject_EmptyContext(t *testing.T) {
	p := Skywalking{}
	carrier := make(propagation.MapCarrier)

	// Inject with empty context should not set any headers
	p.Inject(context.Background(), carrier)

	assert.Empty(t, carrier.Get(sw8Header))
}

func TestSkyWalkingPropagator_Inject_ValidContext(t *testing.T) {
	p := Skywalking{}
	carrier := make(propagation.MapCarrier)

	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithRemoteSpanContext(context.Background(), sc)

	p.Inject(ctx, carrier)

	// Should set the sw8 header
	sw8Value := carrier.Get(sw8Header)
	assert.NotEmpty(t, sw8Value)

	// The header should be in the correct format with base64 encoded fields
	// Check that it starts with "1" (sampled flag) and has the right number of fields
	fields := strings.Split(sw8Value, "-")
	assert.Len(t, fields, 8, "sw8 header should have 8 fields")
	assert.Equal(t, "1", fields[0], "first field should be sample flag = 1")
}

func TestSkyWalkingPropagator_Extract_EmptyCarrier(t *testing.T) {
	p := Skywalking{}
	carrier := make(propagation.MapCarrier)

	ctx := p.Extract(context.Background(), carrier)

	// Should return the original context
	assert.Equal(t, context.Background(), ctx)
}

func TestSkyWalkingPropagator_Extract_InvalidHeader(t *testing.T) {
	p := Skywalking{}
	carrier := make(propagation.MapCarrier)

	// Set an invalid sw8 header
	carrier.Set(sw8Header, "invalid-header")

	ctx := p.Extract(context.Background(), carrier)

	// Should return the original context
	assert.Equal(t, context.Background(), ctx)
}

func TestSkyWalkingPropagator_RoundTrip(t *testing.T) {
	p := Skywalking{}

	// Create a span context
	originalSC := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
	originalCtx := trace.ContextWithRemoteSpanContext(context.Background(), originalSC)

	// Inject into carrier
	carrier := make(propagation.MapCarrier)
	p.Inject(originalCtx, carrier)

	// Extract from carrier
	extractedCtx := p.Extract(context.Background(), carrier)
	extractedSC := trace.SpanContextFromContext(extractedCtx)

	// TODO: This test will need adjustment once the exact specification is implemented
	// For now, we just verify that some context was extracted
	assert.True(t, extractedSC.IsValid(), "extracted span context should be valid")

	// The trace ID should be preserved
	assert.Equal(t, originalSC.TraceID(), extractedSC.TraceID())

	// TODO: Verify other fields once the specification is complete
}

// TestSkyWalkingPropagator_ExtractWithMinimalHeader tests extraction with a minimal valid header.
func TestSkyWalkingPropagator_ExtractWithMinimalHeader(t *testing.T) {
	p := Skywalking{}
	carrier := make(propagation.MapCarrier)

	// Create a minimal valid sw8 header in the correct format
	// Format: {sample-flag}-{trace-id}-{segment-id}-{span-id}-{parent-service}-{parent-service-instance}-{parent-endpoint}-{address-used-at-client}
	sw8Value := strings.Join([]string{
		"1", // sample flag
		base64.StdEncoding.EncodeToString([]byte(traceID.String())), // trace ID
		base64.StdEncoding.EncodeToString([]byte(spanID.String())),  // segment ID
		"123", // span ID as integer
		base64.StdEncoding.EncodeToString([]byte("test-service")),  // parent service
		base64.StdEncoding.EncodeToString([]byte("test-instance")), // parent service instance
		base64.StdEncoding.EncodeToString([]byte("test-endpoint")), // parent endpoint
		base64.StdEncoding.EncodeToString([]byte("test-address")),  // address
	}, "-")
	carrier.Set(sw8Header, sw8Value)

	ctx := p.Extract(context.Background(), carrier)
	sc := trace.SpanContextFromContext(ctx)

	require.True(t, sc.IsValid())
	assert.Equal(t, traceID, sc.TraceID())
	assert.Equal(t, spanID, sc.SpanID())
	assert.True(t, sc.IsSampled(), "should be sampled based on sample flag")
}

// Benchmark tests.
func BenchmarkSkyWalkingPropagator_Inject(b *testing.B) {
	p := Skywalking{}
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithRemoteSpanContext(context.Background(), sc)

	b.ResetTimer()
	for range b.N {
		carrier := make(propagation.MapCarrier)
		p.Inject(ctx, carrier)
	}
}

func BenchmarkSkyWalkingPropagator_Extract(b *testing.B) {
	p := Skywalking{}
	carrier := make(propagation.MapCarrier)
	sw8Value := strings.Join([]string{
		"1", // sample flag
		base64.StdEncoding.EncodeToString([]byte(traceID.String())), // trace ID
		base64.StdEncoding.EncodeToString([]byte(spanID.String())),  // segment ID
		"123", // span ID as integer
		base64.StdEncoding.EncodeToString([]byte("service")),  // parent service
		base64.StdEncoding.EncodeToString([]byte("instance")), // parent service instance
		base64.StdEncoding.EncodeToString([]byte("endpoint")), // parent endpoint
		base64.StdEncoding.EncodeToString([]byte("target")),   // address
	}, "-")
	carrier.Set(sw8Header, sw8Value)

	b.ResetTimer()
	for range b.N {
		p.Extract(context.Background(), carrier)
	}
}

// Test that unknown values are used when no carrier metadata is set.
func TestSkyWalkingPropagator_InjectWithDefaultValues(t *testing.T) {
	p := Skywalking{}
	carrier := make(propagation.MapCarrier)

	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithRemoteSpanContext(context.Background(), sc)

	p.Inject(ctx, carrier)

	sw8Value := carrier.Get(sw8Header)
	assert.NotEmpty(t, sw8Value, "sw8 header should be set")

	// Parse the sw8 header to verify default "unknown" values are used
	fields := strings.Split(sw8Value, "-")
	require.Len(t, fields, 8, "sw8 header should have 8 fields")

	// Check that default "unknown" values are properly base64 encoded in the header
	serviceBytes, err := base64.StdEncoding.DecodeString(fields[4])
	require.NoError(t, err)
	assert.Equal(t, "unknown", string(serviceBytes))

	instanceBytes, err := base64.StdEncoding.DecodeString(fields[5])
	require.NoError(t, err)
	assert.Equal(t, "unknown", string(instanceBytes))

	endpointBytes, err := base64.StdEncoding.DecodeString(fields[6])
	require.NoError(t, err)
	assert.Equal(t, "unknown", string(endpointBytes))

	addressBytes, err := base64.StdEncoding.DecodeString(fields[7])
	require.NoError(t, err)
	assert.Equal(t, "unknown", string(addressBytes))
}

func TestSkyWalkingPropagator_Correlation_Inject(t *testing.T) {
	p := Skywalking{}
	carrier := make(propagation.MapCarrier)

	// Create baggage with correlation data
	member1, _ := baggage.NewMember("service.name", "test-service")
	member2, _ := baggage.NewMember("user.id", "12345")
	member3, _ := baggage.NewMember("component", "web-server")

	bags, err := baggage.New(member1, member2, member3)
	require.NoError(t, err)

	// Create context with span and baggage
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})

	ctx := trace.ContextWithSpanContext(context.Background(), sc)
	ctx = baggage.ContextWithBaggage(ctx, bags)

	p.Inject(ctx, carrier)

	// Verify sw8 header is set
	assert.NotEmpty(t, carrier.Get(sw8Header))

	// Verify sw8-correlation header is set
	correlationValue := carrier.Get(sw8CorrelationHeader)
	assert.NotEmpty(t, correlationValue)

	// Parse correlation header
	pairs := strings.Split(correlationValue, ",")
	assert.Len(t, pairs, 3)

	// Verify all pairs are present (order may vary)
	pairMap := make(map[string]string)
	for _, pair := range pairs {
		kv := strings.SplitN(pair, ":", 2)
		require.Len(t, kv, 2)

		// Decode BASE64 encoded key and value
		keyBytes, err := base64.StdEncoding.DecodeString(kv[0])
		require.NoError(t, err)
		key := string(keyBytes)

		valueBytes, err := base64.StdEncoding.DecodeString(kv[1])
		require.NoError(t, err)
		value := string(valueBytes)

		pairMap[key] = value
	}

	assert.Equal(t, "test-service", pairMap["service.name"])
	assert.Equal(t, "12345", pairMap["user.id"])
	assert.Equal(t, "web-server", pairMap["component"])
}

func TestSkyWalkingPropagator_Correlation_Extract(t *testing.T) {
	p := Skywalking{}
	carrier := make(propagation.MapCarrier)

	// Set up valid sw8 header
	sw8Value := "1-" + base64.StdEncoding.EncodeToString([]byte(traceID.String())) +
		"-" + base64.StdEncoding.EncodeToString([]byte(spanID.String())) +
		"-123-" + base64.StdEncoding.EncodeToString([]byte("unknown")) +
		"-" + base64.StdEncoding.EncodeToString([]byte("unknown")) +
		"-" + base64.StdEncoding.EncodeToString([]byte("unknown")) +
		"-" + base64.StdEncoding.EncodeToString([]byte("unknown"))
	carrier.Set(sw8Header, sw8Value)

	// Set up correlation header with BASE64 encoded values as per specification
	correlationValue := base64.StdEncoding.EncodeToString([]byte("service.name")) + ":" + base64.StdEncoding.EncodeToString([]byte("test-service")) + "," +
		base64.StdEncoding.EncodeToString([]byte("user.id")) + ":" + base64.StdEncoding.EncodeToString([]byte("12345")) + "," +
		base64.StdEncoding.EncodeToString([]byte("component")) + ":" + base64.StdEncoding.EncodeToString([]byte("web-server"))
	carrier.Set(sw8CorrelationHeader, correlationValue)

	ctx := p.Extract(context.Background(), carrier)

	// Verify span context is extracted
	sc := trace.SpanContextFromContext(ctx)
	assert.True(t, sc.IsValid())
	assert.Equal(t, traceID, sc.TraceID())

	// Verify baggage is extracted from correlation header
	bags := baggage.FromContext(ctx)
	assert.Equal(t, 3, bags.Len())

	// Verify individual baggage members
	serviceName := bags.Member("service.name")
	assert.Equal(t, "test-service", serviceName.Value())

	userID := bags.Member("user.id")
	assert.Equal(t, "12345", userID.Value())

	component := bags.Member("component")
	assert.Equal(t, "web-server", component.Value())
}

func TestSkyWalkingPropagator_Correlation_RoundTrip(t *testing.T) {
	p := Skywalking{}

	// Create original context with span and baggage
	originalSC := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})

	member1, _ := baggage.NewMember("service.name", "test-service")
	member2, _ := baggage.NewMember("user.id", "12345")
	member3, _ := baggage.NewMember("component", "web-server")

	originalBags, err := baggage.New(member1, member2, member3)
	require.NoError(t, err)

	originalCtx := trace.ContextWithSpanContext(context.Background(), originalSC)
	originalCtx = baggage.ContextWithBaggage(originalCtx, originalBags)

	// Inject into carrier
	carrier := make(propagation.MapCarrier)
	p.Inject(originalCtx, carrier)

	// Extract from carrier
	extractedCtx := p.Extract(context.Background(), carrier)

	// Verify span context round trip
	extractedSC := trace.SpanContextFromContext(extractedCtx)
	assert.True(t, extractedSC.IsValid())
	assert.Equal(t, originalSC.TraceID(), extractedSC.TraceID())
	assert.Equal(t, originalSC.IsSampled(), extractedSC.IsSampled())

	// Verify baggage round trip
	extractedBags := baggage.FromContext(extractedCtx)
	assert.Equal(t, originalBags.Len(), extractedBags.Len())

	for _, originalMember := range originalBags.Members() {
		extractedMember := extractedBags.Member(originalMember.Key())
		assert.Equal(t, originalMember.Value(), extractedMember.Value())
	}
}

func TestSkyWalkingPropagator_Correlation_EmptyBaggage(t *testing.T) {
	p := Skywalking{}
	carrier := make(propagation.MapCarrier)

	// Create context with span but no baggage
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})

	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	p.Inject(ctx, carrier)

	// Verify sw8 header is set
	assert.NotEmpty(t, carrier.Get(sw8Header))

	// Verify sw8-correlation header is not set for empty baggage
	assert.Empty(t, carrier.Get(sw8CorrelationHeader))
}

func TestSkyWalkingPropagator_Correlation_MalformedHeader(t *testing.T) {
	p := Skywalking{}
	carrier := make(propagation.MapCarrier)

	// Set up valid sw8 header
	sw8Value := "1-" + base64.StdEncoding.EncodeToString([]byte(traceID.String())) +
		"-" + base64.StdEncoding.EncodeToString([]byte(spanID.String())) +
		"-123-" + base64.StdEncoding.EncodeToString([]byte("unknown")) +
		"-" + base64.StdEncoding.EncodeToString([]byte("unknown")) +
		"-" + base64.StdEncoding.EncodeToString([]byte("unknown")) +
		"-" + base64.StdEncoding.EncodeToString([]byte("unknown"))
	carrier.Set(sw8Header, sw8Value)

	// Set up malformed correlation headers
	testCases := []struct {
		name             string
		correlationValue string
		expectedBaggage  int
	}{
		{
			name:             "missing colon",
			correlationValue: "key1value1," + base64.StdEncoding.EncodeToString([]byte("key2")) + ":" + base64.StdEncoding.EncodeToString([]byte("value2")),
			expectedBaggage:  1, // Only key2:value2 should be parsed
		},
		{
			name:             "empty pairs",
			correlationValue: base64.StdEncoding.EncodeToString([]byte("key1")) + ":" + base64.StdEncoding.EncodeToString([]byte("value1")) + ",," + base64.StdEncoding.EncodeToString([]byte("key2")) + ":" + base64.StdEncoding.EncodeToString([]byte("value2")),
			expectedBaggage:  2, // Empty pair should be skipped
		},
		{
			name:             "invalid BASE64 encoding",
			correlationValue: base64.StdEncoding.EncodeToString([]byte("key1")) + ":" + base64.StdEncoding.EncodeToString([]byte("value1")) + ",key%ZZ:value2",
			expectedBaggage:  1, // Only key1:value1 should be parsed
		},
		{
			name:             "completely malformed",
			correlationValue: "not-correlation-data",
			expectedBaggage:  0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			carrier.Set(sw8CorrelationHeader, tc.correlationValue)

			ctx := p.Extract(context.Background(), carrier)

			// Verify span context is still extracted
			sc := trace.SpanContextFromContext(ctx)
			assert.True(t, sc.IsValid())

			// Verify baggage handling
			bags := baggage.FromContext(ctx)
			assert.Equal(t, tc.expectedBaggage, bags.Len())
		})
	}
}

func TestSkyWalkingPropagator_Sw8X_Extension(t *testing.T) {
	p := Skywalking{}
	carrier := make(propagation.MapCarrier)

	// Create valid span context
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})

	// Test injection with default tracing mode
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	// Inject headers
	p.Inject(ctx, carrier)

	// Verify SW8-X extension header is set with default mode
	sw8XValue := carrier.Get(sw8ExtensionHeader)
	assert.NotEmpty(t, sw8XValue)
	assert.Equal(t, "0- ", sw8XValue) // Default tracing mode with placeholder timestamp

	// Test injection with skip analysis mode
	carrier = make(propagation.MapCarrier)
	ctx = WithTracingMode(ctx, TracingModeSkipAnalysis)
	p.Inject(ctx, carrier)

	// Verify SW8-X extension header is set with skip analysis mode
	sw8XValue = carrier.Get(sw8ExtensionHeader)
	assert.Equal(t, "1- ", sw8XValue) // Skip analysis mode with placeholder timestamp

	// Test extraction with SW8-X header
	extractCarrier := make(propagation.MapCarrier)
	extractCarrier.Set(sw8Header, carrier.Get(sw8Header))
	extractCarrier.Set(sw8ExtensionHeader, "1- ") // Skip analysis mode with placeholder

	extractedCtx := p.Extract(context.Background(), extractCarrier)

	// Verify span context is still extracted correctly
	extractedSC := trace.SpanContextFromContext(extractedCtx)
	assert.True(t, extractedSC.IsValid())
	assert.Equal(t, traceID, extractedSC.TraceID())
	assert.Equal(t, spanID, extractedSC.SpanID())

	// Verify tracing mode is extracted and stored in context
	extractedMode := TracingModeFromContext(extractedCtx)
	assert.Equal(t, TracingModeSkipAnalysis, extractedMode)
}
