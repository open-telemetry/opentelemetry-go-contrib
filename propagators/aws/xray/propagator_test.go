// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xray

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/baggage"
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

		_, sc, err := extract(context.Background(), headerVal)

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

func TestAwsXrayExtractWithLineage(t *testing.T) {
	testData := []struct {
		lineage string

		expectedBaggage map[string]string
	}{
		{
			lineage: "32767:e65a2c4d:255",
			expectedBaggage: map[string]string{
				"Lineage": "32767:e65a2c4d:255",
			},
		},
		{
			lineage: "32767:e65a2c4d:255",
			expectedBaggage: map[string]string{
				"Lineage": "32767:e65a2c4d:255",
				"cat":     "bark",
				"dog":     "meow",
			},
		},
		{
			lineage:         "1::",
			expectedBaggage: map[string]string{},
		},
		{
			lineage:         "1",
			expectedBaggage: map[string]string{},
		},
		{
			lineage:         "",
			expectedBaggage: map[string]string{},
		},
		{
			lineage:         ":",
			expectedBaggage: map[string]string{},
		},
		{
			lineage:         "::",
			expectedBaggage: map[string]string{},
		},
		{
			lineage:         "1:badc0de:13",
			expectedBaggage: map[string]string{},
		},
		{
			lineage:         ":fbadc0de:13",
			expectedBaggage: map[string]string{},
		},
		{
			lineage:         "1:fbadc0de:",
			expectedBaggage: map[string]string{},
		},
		{
			lineage:         "1::1",
			expectedBaggage: map[string]string{},
		},
		{
			lineage:         "65535:fbadc0de:255",
			expectedBaggage: map[string]string{},
		},
		{
			lineage:         "-213:e65a2c4d:255",
			expectedBaggage: map[string]string{},
		},
		{
			lineage:         "213:e65a2c4d:-22",
			expectedBaggage: map[string]string{},
		},
	}

	p := Propagator{}

	for _, test := range testData {
		carrier := make(map[string]string)
		members := []baggage.Member{}
		expectedSpanContext := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    traceID,
			SpanID:     parentSpanID,
			TraceFlags: trace.TraceFlags(0x00),
			Remote:     true,
		})

		for key, value := range test.expectedBaggage {
			member, _ := baggage.NewMember(key, value)
			members = append(members, member)
		}

		expectedBaggage, _ := baggage.New(members...)
		carrier[traceHeaderKey] = "Root=1-8a3c60f7-d188f8fa79d48a391a778fa6;Parent=53995c3f42cd8ad8;Sampled=0;Lineage=" + test.lineage

		ctx := baggage.ContextWithBaggage(context.Background(), expectedBaggage)

		if len(test.expectedBaggage) == 0 {
			ctx = context.Background()
		}

		actualContext := p.Extract(ctx, propagation.MapCarrier(carrier))
		spanContext := trace.SpanContextFromContext(actualContext)

		assert.Equal(t, baggage.FromContext(actualContext), expectedBaggage)
		assert.Equal(t, spanContext, expectedSpanContext)
	}
}

func TestAwsXrayInjectWithLineage(t *testing.T) {
	testData := []struct {
		expectedBaggage map[string]string
	}{
		{
			expectedBaggage: map[string]string{
				"Lineage": "32767:e65a2c4d:255",
				"cat":     "bark",
				"dog":     "meow",
			},
		},
		{
			expectedBaggage: map[string]string{
				"Lineage": "32767:e65a2c4d:255",
			},
		},
	}

	p := Propagator{}

	for _, test := range testData {
		carrier := make(map[string]string)
		members := []baggage.Member{}

		for key, value := range test.expectedBaggage {
			member, _ := baggage.NewMember(key, value)
			members = append(members, member)
		}

		expectedBaggage, _ := baggage.New(members...)

		carrier[traceHeaderKey] = "Root=1-8a3c60f7-d188f8fa79d48a391a778fa6;Parent=53995c3f42cd8ad8;Sampled=0;Lineage=32767:e65a2c4d:255"

		p.Inject(baggage.ContextWithBaggage(context.Background(), expectedBaggage), propagation.MapCarrier(carrier))

		assert.Equal(t, carrier[traceHeaderKey], "Root=1-8a3c60f7-d188f8fa79d48a391a778fa6;Parent=53995c3f42cd8ad8;Sampled=0;Lineage=32767:e65a2c4d:255")
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
