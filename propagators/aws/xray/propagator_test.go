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

package xray

import (
	"context"
	"net/http"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/stretchr/testify/assert"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var (
	traceID                    = trace.TraceID{0x8a, 0x3c, 0x60, 0xf7, 0xd1, 0x88, 0xf8, 0xfa, 0x79, 0xd4, 0x8a, 0x39, 0x1a, 0x77, 0x8f, 0xa6}
	rootSpanID                 = trace.SpanID{0x8a, 0x3c, 0x60, 0xf7, 0xd1, 0x88, 0xf8, 0xfa}
	xrayTraceID                = "1-8a3c60f7-d188f8fa79d48a391a778fa6"
	xrayTraceIDIncorrectLength = "1-82138-1203123"
	parentID64Str              = "53995c3f42cd8ad8"
	parentSpanID               = trace.SpanID{0x53, 0x99, 0x5c, 0x3f, 0x42, 0xcd, 0x8a, 0xd8}
	zeroSpanIDStr              = "0000000000000000"
	wrongVersionTraceHeaderID  = "5b00000000b000000000000000000000000"
)

func TestAwsXrayExtract(t *testing.T) {
	testData := []struct {
		name         string
		traceID      string
		parentSpanID string
		samplingFlag string
		expected     trace.SpanContextConfig
		err          error
	}{
		{
			"only root",
			xrayTraceID, "", "",
			trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     rootSpanID,
				TraceFlags: traceFlagNone,
			},
			nil,
		},
		{
			"only root and sampled",
			xrayTraceID, "", isSampled,
			trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     rootSpanID,
				TraceFlags: trace.FlagsSampled,
			},
			nil,
		},
		{
			"not sampled",
			xrayTraceID, parentID64Str, notSampled,
			trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     parentSpanID,
				TraceFlags: traceFlagNone,
			},
			nil,
		},
		{
			"sampled",
			xrayTraceID, parentID64Str, isSampled,
			trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     parentSpanID,
				TraceFlags: traceFlagSampled,
			},
			nil,
		},
		{
			"zero parent id",
			xrayTraceID, zeroSpanIDStr, isSampled,
			trace.SpanContextConfig{},
			errInvalidSpanIDLength,
		},
		{
			"wrong trace id length",
			xrayTraceIDIncorrectLength, parentID64Str, isSampled,
			trace.SpanContextConfig{},
			errLengthTraceIDHeader,
		},
		{
			"wrong trace id version",
			wrongVersionTraceHeaderID, parentID64Str, isSampled,
			trace.SpanContextConfig{},
			errInvalidTraceIDVersion,
		},
	}

	for _, test := range testData {
		t.Run(test.name, func(t *testing.T) {
			headerVal := makeTraceHeaderVal(test.traceID, test.parentSpanID, test.samplingFlag)

			sc, err := extract(headerVal)

			info := []interface{}{
				"trace ID: %q, parent span ID: %q, sampling flag: %q",
				test.traceID,
				test.parentSpanID,
				test.samplingFlag,
			}

			assert.Equal(t, test.err, err, info...)
			assert.Equal(t, trace.NewSpanContext(test.expected), sc, info...)
			if test.err == nil {
				assert.True(t, sc.IsValid())
			}
		})
	}
}

func BenchmarkPropagatorExtract(b *testing.B) {
	propagator := Propagator{}

	ctx := context.Background()

	headers := make(http.Header)
	headers.Set(traceHeaderKey,
		makeTraceHeaderVal(xrayTraceID, parentID64Str, isSampled))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = propagator.Extract(ctx, propagation.HeaderCarrier(headers))
	}
}

func BenchmarkPropagatorInject(b *testing.B) {
	propagator := Propagator{}

	otel.SetTracerProvider(sdktrace.NewTracerProvider(
		sdktrace.WithIDGenerator(NewIDGenerator()),
	))

	tracer := otel.Tracer("test")

	headers := make(http.Header)

	ctx, _ := tracer.Start(context.Background(), "Parent operation...")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		propagator.Inject(ctx, propagation.HeaderCarrier(headers))
	}
}
