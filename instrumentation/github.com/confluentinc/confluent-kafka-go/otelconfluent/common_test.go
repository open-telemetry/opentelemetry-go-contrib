package otelconfluent

import (
	"context"
	"testing"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestContextFromMessageHeaders(t *testing.T) {
	// Given
	testCases := []struct {
		name            string
		ctx             context.Context
		msg             *kafka.Message
		expectedTraceID string
		expectedSpanID  string
	}{
		{
			name:            "Nil message",
			ctx:             context.Background(),
			msg:             nil,
			expectedTraceID: "00000000000000000000000000000000",
			expectedSpanID:  "0000000000000000",
		},
		{
			name:            "Empty message",
			ctx:             context.Background(),
			msg:             &kafka.Message{},
			expectedTraceID: "00000000000000000000000000000000",
			expectedSpanID:  "0000000000000000",
		},
		{
			name: "Only trace id header",
			ctx:  context.Background(),
			msg: &kafka.Message{
				Headers: []kafka.Header{
					{Key: traceIdentifierHeaderName, Value: []byte("12345678912345678912345678912345")},
				},
			},
			expectedTraceID: "00000000000000000000000000000000",
			expectedSpanID:  "0000000000000000",
		},
		{
			name: "Only span id header",
			ctx:  context.Background(),
			msg: &kafka.Message{
				Headers: []kafka.Header{
					{Key: spanIdentifierHeaderName, Value: []byte("1234567891234567")},
				},
			},
			expectedTraceID: "00000000000000000000000000000000",
			expectedSpanID:  "0000000000000000",
		},
		{
			name: "Both trace id and span id headers",
			ctx:  context.Background(),
			msg: &kafka.Message{
				Headers: []kafka.Header{
					{Key: traceIdentifierHeaderName, Value: []byte("12345678912345678912345678912345")},
					{Key: spanIdentifierHeaderName, Value: []byte("1234567891234567")},
				},
			},
			expectedTraceID: "12345678912345678912345678912345",
			expectedSpanID:  "1234567891234567",
		},
	}

	// When - Then
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := contextFromMessageHeaders(testCase.ctx, testCase.msg)

			span := trace.SpanFromContext(ctx)

			assert.Equal(t, testCase.expectedTraceID, span.SpanContext().TraceID().String())
			assert.Equal(t, testCase.expectedSpanID, span.SpanContext().SpanID().String())
		})
	}
}

func TestReplaceOrAddSpanContextToMessageHeaders(t *testing.T) {
	// Given
	traceID, _ := oteltrace.TraceIDFromHex("12345678912345678912345678912345")
	spanID, _ := oteltrace.SpanIDFromHex("1234567891234567")

	testCases := []struct {
		name            string
		spanContext     oteltrace.SpanContext
		msg             *kafka.Message
		expectedHeaders []kafka.Header
	}{
		{
			name:            "Nil message and empty SpanContext",
			spanContext:     oteltrace.NewSpanContext(oteltrace.SpanContextConfig{}),
			msg:             nil,
			expectedHeaders: []kafka.Header{},
		},
		{
			name:        "Empty message and empty SpanContext",
			spanContext: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{}),
			msg:         &kafka.Message{},
			expectedHeaders: []kafka.Header{
				{Key: traceIdentifierHeaderName, Value: []byte("00000000000000000000000000000000")},
				{Key: spanIdentifierHeaderName, Value: []byte("0000000000000000")},
			},
		},
		{
			name: "Empty message and SpanContext: only trace id",
			spanContext: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				TraceID: traceID,
			}),
			msg: &kafka.Message{},
			expectedHeaders: []kafka.Header{
				{Key: traceIdentifierHeaderName, Value: []byte("12345678912345678912345678912345")},
				{Key: spanIdentifierHeaderName, Value: []byte("0000000000000000")},
			},
		},
		{
			name: "Empty message and SpanContext: only span id",
			spanContext: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				SpanID: spanID,
			}),
			msg: &kafka.Message{},
			expectedHeaders: []kafka.Header{
				{Key: traceIdentifierHeaderName, Value: []byte("00000000000000000000000000000000")},
				{Key: spanIdentifierHeaderName, Value: []byte("1234567891234567")},
			},
		},
		{
			name: "Empty message and SpanContext headers",
			spanContext: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				TraceID: traceID,
				SpanID:  spanID,
			}),
			msg: &kafka.Message{},
			expectedHeaders: []kafka.Header{
				{Key: traceIdentifierHeaderName, Value: []byte("12345678912345678912345678912345")},
				{Key: spanIdentifierHeaderName, Value: []byte("1234567891234567")},
			},
		},
		{
			name: "Already filled message and SpanContext headers",
			spanContext: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				TraceID: traceID,
				SpanID:  spanID,
			}),
			msg: &kafka.Message{
				Headers: []kafka.Header{
					{Key: traceIdentifierHeaderName, Value: []byte("9876543219876543219876543219876")},
					{Key: spanIdentifierHeaderName, Value: []byte("9876543219876543")},
				},
			},
			expectedHeaders: []kafka.Header{
				{Key: traceIdentifierHeaderName, Value: []byte("12345678912345678912345678912345")},
				{Key: spanIdentifierHeaderName, Value: []byte("1234567891234567")},
			},
		},
	}

	// When - Then
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			replaceOrAddSpanContextToMessageHeaders(testCase.spanContext, testCase.msg)

			if testCase.msg != nil {
				assert.Equal(t, testCase.expectedHeaders, testCase.msg.Headers)
			}
		})
	}
}
