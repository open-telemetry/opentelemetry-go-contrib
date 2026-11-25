// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zpages

import (
	"context"
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func TestSpanProcessorForceFlush(t *testing.T) {
	zsp := NewSpanProcessor()
	assert.NoError(t, zsp.ForceFlush(t.Context()))
	assert.NoError(t, zsp.Shutdown(t.Context()))
	assert.NoError(t, zsp.ForceFlush(t.Context()))
}

func activeStoreLen(zsp *SpanProcessor) int {
	count := 0
	zsp.activeSpansStore.Range(func(_, _ any) bool {
		count++
		return true
	})
	return count
}

func sampleStoreLen(zsp *SpanProcessor) int {
	count := 0
	zsp.spanSampleStores.Range(func(_, _ any) bool {
		count++
		return true
	})
	return count
}

func totalLatencySamples(zsp *SpanProcessor, name string) int {
	total := 0
	for i := range defaultBoundaries.numBuckets() {
		total += len(zsp.spansByLatency(name, i))
	}
	return total
}

func TestSpanProcessorShutdownAcrossTracerProviderShutdown(t *testing.T) {
	zsp := NewSpanProcessor()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(zsp),
	)
	tracer := tp.Tracer("test")
	ctx := t.Context()

	_, completedLatency := tracer.Start(ctx, "mixed")
	completedLatency.End()

	_, completedErr := tracer.Start(ctx, "error-only")
	completedErr.SetStatus(codes.Error, "failed")
	completedErr.End()

	_, activeOnly := tracer.Start(ctx, "active-only")
	_, activeMixed := tracer.Start(ctx, "mixed")

	require.Equal(t, 2, activeStoreLen(zsp))
	require.Equal(t, 2, sampleStoreLen(zsp))
	require.Len(t, zsp.activeSpans("active-only"), 1)
	require.Len(t, zsp.activeSpans("mixed"), 1)
	require.Equal(t, 1, totalLatencySamples(zsp, "mixed"))
	require.Len(t, zsp.errorSpans("error-only"), 1)

	require.NoError(t, tp.Shutdown(ctx))

	// Both active and completed span state must be released on Shutdown.
	assert.Zero(t, activeStoreLen(zsp))
	assert.Zero(t, sampleStoreLen(zsp))
	assert.Empty(t, zsp.activeSpans("active-only"))
	assert.Empty(t, zsp.activeSpans("mixed"))
	assert.Zero(t, totalLatencySamples(zsp, "mixed"))
	assert.Empty(t, zsp.errorSpans("error-only"))
	assert.Empty(t, zsp.spansPerMethod())

	// Ending spans that were active across provider shutdown must not
	// resurrect active or completed state.
	activeOnly.End()
	activeMixed.End()

	assert.Zero(t, activeStoreLen(zsp))
	assert.Zero(t, sampleStoreLen(zsp))
	assert.Empty(t, zsp.activeSpans("active-only"))
	assert.Empty(t, zsp.activeSpans("mixed"))
	assert.Empty(t, zsp.spansPerMethod())
}

func TestSpanProcessorShutdownIgnoresSubsequentCalls(t *testing.T) {
	zsp := NewSpanProcessor()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(zsp),
	)
	defer func() {
		require.NoError(t, tp.Shutdown(t.Context()))
	}()

	tracer := tp.Tracer("test")
	ctx := t.Context()

	_, beforeCompleted := tracer.Start(ctx, "before-shutdown")
	beforeCompleted.End()

	_, beforeActive := tracer.Start(ctx, "active-before-shutdown")
	require.Equal(t, 1, activeStoreLen(zsp))
	require.Equal(t, 1, sampleStoreLen(zsp))

	// Shut down the SpanProcessor directly while the TracerProvider is still active
	// so subsequent span lifecycle hooks still invoke OnStart and OnEnd on zsp.
	require.NoError(t, zsp.Shutdown(ctx))
	// Subsequent Shutdown calls are no-ops.
	require.NoError(t, zsp.Shutdown(ctx))

	assert.Zero(t, activeStoreLen(zsp))
	assert.Zero(t, sampleStoreLen(zsp))
	assert.Empty(t, zsp.activeSpans("active-before-shutdown"))
	assert.Empty(t, zsp.spansPerMethod())

	// Ending the previously active span triggers zsp.OnEnd and must be ignored.
	beforeActive.End()

	// Starting and ending new spans (both new names and prior names) must be ignored.
	_, afterActive := tracer.Start(ctx, "after-shutdown-active")
	defer afterActive.End()

	_, afterCompleted := tracer.Start(ctx, "after-shutdown-completed")
	afterCompleted.End()

	_, afterPriorNameErr := tracer.Start(ctx, "before-shutdown")
	afterPriorNameErr.SetStatus(codes.Error, "post-shutdown error")
	afterPriorNameErr.End()

	assert.Zero(t, activeStoreLen(zsp))
	assert.Zero(t, sampleStoreLen(zsp))
	assert.Empty(t, zsp.activeSpans("after-shutdown-active"))
	assert.Empty(t, zsp.errorSpans("before-shutdown"))
	assert.Empty(t, zsp.spansPerMethod())
}

func TestSpanProcessorShutdownOnUnregister(t *testing.T) {
	zsp := NewSpanProcessor()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(zsp),
	)
	defer func() {
		require.NoError(t, tp.Shutdown(t.Context()))
	}()

	tracer := tp.Tracer("test")
	ctx := t.Context()

	_, doneSpan := tracer.Start(ctx, "done")
	doneSpan.End()

	_, activeSpan := tracer.Start(ctx, "in-flight")
	require.Equal(t, 1, activeStoreLen(zsp))
	require.Equal(t, 1, sampleStoreLen(zsp))

	tp.UnregisterSpanProcessor(zsp)

	assert.Zero(t, activeStoreLen(zsp))
	assert.Zero(t, sampleStoreLen(zsp))
	assert.Empty(t, zsp.activeSpans("in-flight"))
	assert.Empty(t, zsp.spansPerMethod())

	activeSpan.End()
	assert.Zero(t, activeStoreLen(zsp))
	assert.Zero(t, sampleStoreLen(zsp))
	assert.Empty(t, zsp.spansPerMethod())
}

func TestSpanProcessorShutdownConcurrentSafe(t *testing.T) {
	zsp := NewSpanProcessor()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(zsp),
	)
	defer func() {
		require.NoError(t, tp.Shutdown(t.Context()))
	}()

	handler := NewTracezHandler(zsp)
	tracer := tp.Tracer("test")
	ctx := t.Context()

	const numProducers = 4
	activeSpans := make([][]trace.Span, numProducers)

	var producersReady sync.WaitGroup
	producersReady.Add(numProducers)

	var wg sync.WaitGroup
	for i := range numProducers {
		name := "concurrent-" + strconv.Itoa(i)
		wg.Go(func() {
			// Populate both completed and un-ended active spans before
			// signaling Shutdown so Shutdown contends with live state.
			for j := range 5 {
				_, active := tracer.Start(ctx, name)
				activeSpans[i] = append(activeSpans[i], active)

				_, done := tracer.Start(ctx, name)
				if j%2 == 0 {
					done.SetStatus(codes.Error, "err")
				}
				done.End()
			}
			producersReady.Done()

			// Continue starting both ended and un-ended spans concurrently
			// with Shutdown.
			for j := range 100 {
				_, span := tracer.Start(ctx, name)
				if j%4 == 0 {
					activeSpans[i] = append(activeSpans[i], span)
					continue
				}
				if j%2 == 0 {
					span.SetStatus(codes.Error, "err")
				}
				span.End()
			}
		})
	}

	urls := []string{
		"/tracez",
		"/tracez?zspanname=concurrent-0&ztype=0",
		"/tracez?zspanname=concurrent-0&ztype=1&zlatencybucket=0",
		"/tracez?zspanname=concurrent-0&ztype=2",
	}
	for range 2 {
		wg.Go(func() {
			producersReady.Wait()
			for j := range 40 {
				req := httptest.NewRequestWithContext(ctx, http.MethodGet, urls[j%len(urls)], http.NoBody)
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, req)
				resp := w.Result()
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				_ = resp.Body.Close()
			}
		})
	}
	for range 2 {
		wg.Go(func() {
			producersReady.Wait()
			assert.NoError(t, zsp.Shutdown(ctx))
		})
	}

	wg.Wait()

	// Verify both active and completed span state were released even though held
	// active spans have not called End() yet, and that no concurrent OnStart or
	// OnEnd leaked entries after Shutdown.
	assert.Zero(t, activeStoreLen(zsp))
	assert.Zero(t, sampleStoreLen(zsp))
	assert.Empty(t, zsp.spansPerMethod())
	for i := range numProducers {
		assert.Empty(t, zsp.activeSpans("concurrent-"+strconv.Itoa(i)))
	}

	// Ending held spans after Shutdown must leave both stores empty.
	for _, spans := range activeSpans {
		for _, span := range spans {
			span.End()
		}
	}
	assert.Zero(t, activeStoreLen(zsp))
	assert.Zero(t, sampleStoreLen(zsp))
}

func BenchmarkSpanProcessorOnStartOnEnd(b *testing.B) {
	newBenchSpan := func(id uint64) *testSpan {
		var spanID trace.SpanID
		binary.BigEndian.PutUint64(spanID[:], id)
		return &testSpan{
			spanContext: trace.NewSpanContext(trace.SpanContextConfig{
				TraceID:    [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1},
				SpanID:     spanID,
				TraceFlags: trace.FlagsSampled,
			}),
			name:      "bench-span",
			startTime: time.Unix(10, 0),
			endTime:   time.Unix(10, int64(time.Millisecond)),
			status:    sdktrace.Status{Code: codes.Ok},
		}
	}

	b.Run("Serial", func(b *testing.B) {
		zsp := NewSpanProcessor()
		ts := newBenchSpan(1)
		ctx := b.Context()
		b.ReportAllocs()
		for b.Loop() {
			zsp.OnStart(ctx, ts)
			zsp.OnEnd(ts)
		}
	})

	b.Run("Parallel", func(b *testing.B) {
		zsp := NewSpanProcessor()
		ctx := b.Context()
		var nextID atomic.Uint64
		b.ReportAllocs()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			ts := newBenchSpan(nextID.Add(1))
			for pb.Next() {
				zsp.OnStart(ctx, ts)
				zsp.OnEnd(ts)
			}
		})
	})
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
	assert.Equal(t, make([]int, defaultBoundaries.numBuckets()), spansPM[spanName].latencySpans)

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
