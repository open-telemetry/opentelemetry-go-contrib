// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zpages

import (
	"context"
	"reflect"
	"sort"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func TestSpanProcessorDoNothing(t *testing.T) {
	zsp := NewSpanProcessor()
	assert.NoError(t, zsp.ForceFlush(t.Context()))
	assert.NoError(t, zsp.Shutdown(t.Context()))
}

func TestSpanProcessor(t *testing.T) {
	zsp := NewSpanProcessor()
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(zsp),
	)

	const spanName = "testSpan"
	const numSpans = 9

	tracer := tracerProvider.Tracer("test")
	spans := createActiveSpans(tracer, spanName, numSpans)
	// Sort the spans by the address pointer so we can compare.
	sort.Slice(spans, func(i, j int) bool {
		return reflect.ValueOf(spans[i]).Pointer() < reflect.ValueOf(spans[j]).Pointer()
	})
	require.Len(t, spans, numSpans)
	activeSpans := zsp.activeSpans(spanName)
	assert.Len(t, activeSpans, numSpans)
	// Sort the activeSpans by the address pointer so we can compare.
	sort.Slice(activeSpans, func(i, j int) bool {
		return reflect.ValueOf(activeSpans[i]).Pointer() < reflect.ValueOf(activeSpans[j]).Pointer()
	})
	for i := range spans {
		assert.Same(t, spans[i], activeSpans[i])
	}
	// No ended spans so there will be no error, no latency samples.
	assert.Empty(t, zsp.errorSpans(spanName))
	for i := range defaultBoundaries.numBuckets() {
		assert.Empty(t, zsp.spansByLatency(spanName, i))
	}
	spansPM := zsp.spansPerMethod()
	require.Len(t, spansPM, 1)
	assert.Equal(t, numSpans, spansPM[spanName].activeSpans)

	// End all Spans, they will end pretty fast, so we can only check that we have at least one in
	// errors and one in latency samples.
	for _, s := range spans {
		s.End()
	}
	// Test that no more active spans.
	assert.Empty(t, zsp.activeSpans(spanName))
	assert.Len(t, zsp.errorSpans(spanName), 1)
	numLatencySamples := 0
	for i := range defaultBoundaries.numBuckets() {
		numLatencySamples += len(zsp.spansByLatency(spanName, i))
	}
	assert.GreaterOrEqual(t, numLatencySamples, 1)
}

func TestSpanProcessorFuzzer(t *testing.T) {
	zsp := NewSpanProcessor()
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(zsp),
	)

	const numIterations = 200
	const numSpansPerIteration = 90
	const goroutine = 4

	var wg sync.WaitGroup
	wg.Add(goroutine)
	for g := range goroutine {
		go func(n int) {
			defer wg.Done()
			tracer := tracerProvider.Tracer("test" + strconv.Itoa(1+n))
			name := "testSpan" + strconv.Itoa(1+(n%2))
			for range numIterations {
				createEndedSpans(tracer, name, numSpansPerIteration)
			}
		}(g)
	}
	wg.Wait()

	assert.Len(t, zsp.spansPerMethod(), 2)

	assert.Empty(t, zsp.activeSpans("testSpan1"))
	assert.GreaterOrEqual(t, len(zsp.errorSpans("testSpan1")), 1)
	// Count latency samples across all buckets instead of a single bucket to avoid flakes
	numLatencySamples1 := 0
	for i := range defaultBoundaries.numBuckets() {
		numLatencySamples1 += len(zsp.spansByLatency("testSpan1", i))
	}
	assert.GreaterOrEqual(t, numLatencySamples1, 1)

	assert.Empty(t, zsp.activeSpans("testSpan2"))
	assert.GreaterOrEqual(t, len(zsp.errorSpans("testSpan2")), 1)
	// Count latency samples across all buckets instead of a single bucket to avoid flakes
	numLatencySamples2 := 0
	for i := range defaultBoundaries.numBuckets() {
		numLatencySamples2 += len(zsp.spansByLatency("testSpan2", i))
	}
	assert.GreaterOrEqual(t, numLatencySamples2, 1)
}

func TestSpanProcessorNegativeLatency(t *testing.T) {
	zsp := NewSpanProcessor()
	ts := &testSpan{
		spanContext: trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1},
			SpanID:     [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
			TraceFlags: 1,
			Remote:     false,
		}),
		name:      "test",
		startTime: time.Unix(10, 0),
		endTime:   time.Unix(5, 0),
		status: sdktrace.Status{
			Code:        codes.Ok,
			Description: "",
		},
	}
	zsp.OnStart(t.Context(), ts)

	spansPM := zsp.spansPerMethod()
	require.Len(t, spansPM, 1)
	assert.Equal(t, 1, spansPM["test"].activeSpans)

	zsp.OnEnd(ts)

	spansPM = zsp.spansPerMethod()
	require.Len(t, spansPM, 1)
	assert.Equal(t, 1, spansPM["test"].latencySpans[0])
}

func TestSpanProcessorSpansByLatencyWrongIndex(t *testing.T) {
	zsp := NewSpanProcessor()
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(zsp),
	)
	tracer := tracerProvider.Tracer("test")
	createEndedSpans(tracer, "test", 6)
	assert.Nil(t, zsp.spansByLatency("test", -1))
	assert.Nil(t, zsp.spansByLatency("test", defaultBoundaries.numBuckets()))
}

func createEndedSpans(tracer trace.Tracer, spanName string, numSpans int) {
	for i := range numSpans {
		_, span := tracer.Start(context.Background(), spanName)
		span.SetStatus(codes.Code(i%3), "")
		span.End()
	}
}

func createActiveSpans(tracer trace.Tracer, spanName string, numSpans int) []trace.Span {
	spans := make([]trace.Span, 0, numSpans)
	for i := range numSpans {
		_, span := tracer.Start(context.Background(), spanName)
		span.SetStatus(codes.Code(i%3), "")
		spans = append(spans, span)
	}
	return spans
}
