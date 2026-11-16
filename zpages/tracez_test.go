// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zpages

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func TestNewTracezHandler(t *testing.T) {
	sp := NewSpanProcessor()
	handler := NewTracezHandler(sp)
	require.NotNil(t, handler, "NewTracezHandler returned nil")
}

func TestTracezHandler_ServeHTTP(t *testing.T) {
	// Setup common test infrastructure
	sp := NewSpanProcessor()
	defer func() {
		require.NoError(t, sp.Shutdown(t.Context()))
	}()

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sp),
	)
	defer func() {
		require.NoError(t, tp.Shutdown(t.Context()))
	}()

	tracer := tp.Tracer("test-tracer")
	ctx := t.Context()

	_, completedSpan := tracer.Start(ctx, "completed-span")
	completedSpan.End()

	_, errorSpan := tracer.Start(ctx, "error-span")
	errorSpan.SetStatus(codes.Error, context.DeadlineExceeded.Error())
	errorSpan.End()

	_, querySpan := tracer.Start(ctx, "query-span")
	querySpan.End()

	for range 5 {
		_, span := tracer.Start(ctx, "multi-span")
		span.End()
	}

	handler := NewTracezHandler(sp)

	tests := []struct {
		name           string
		url            string
		wantStatus     int
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:         "basic response with no query params",
			url:          "/tracez",
			wantStatus:   http.StatusOK,
			wantContains: []string{"completed-span", "error-span", "query-span", "multi-span"},
		},
		{
			name:         "query for existing span",
			url:          "/tracez?zspanname=query-span",
			wantStatus:   http.StatusOK,
			wantContains: []string{"query-span"},
		},
		{
			name:       "query for non-existing span",
			url:        "/tracez?zspanname=non-existing",
			wantStatus: http.StatusOK,
			// Note: The handler displays the queried span name in the response
			// (e.g., "Span Name: non-existing") even when no matching spans exist.
			// This verifies the query parameter was processed, but doesn't verify
			// that the span was actually found (which it shouldn't be).
			wantContains: []string{"non-existing"},
		},
		{
			name:         "latency spans (type=1)",
			url:          "/tracez?zspanname=completed-span&ztype=1",
			wantStatus:   http.StatusOK,
			wantContains: []string{},
		},
		{
			name:         "error spans (type=2)",
			url:          "/tracez?zspanname=error-span&ztype=2",
			wantStatus:   http.StatusOK,
			wantContains: []string{"1 Requests", context.DeadlineExceeded.Error()},
		},
		{
			name:       "with latency bucket",
			url:        "/tracez?zspanname=completed-span&ztype=1&zlatencybucket=0",
			wantStatus: http.StatusOK,
		},
		{
			name:         "all parameters",
			url:          "/tracez?zspanname=completed-span&ztype=1&zlatencybucket=2",
			wantStatus:   http.StatusOK,
			wantContains: []string{},
		},
		{
			name:       "type only",
			url:        "/tracez?ztype=1",
			wantStatus: http.StatusOK,
		},
		{
			name:       "latency bucket only",
			url:        "/tracez?zlatencybucket=3",
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid type value",
			url:        "/tracez?ztype=invalid",
			wantStatus: http.StatusOK,
		},
		{
			name:       "negative latency bucket",
			url:        "/tracez?zlatencybucket=-1",
			wantStatus: http.StatusOK,
		},
		{
			name:         "multiple spans with same name",
			url:          "/tracez?zspanname=multi-span&ztype=1",
			wantStatus:   http.StatusOK,
			wantContains: []string{"multi-span"},
		},
		{
			name:       "invalid form encoding",
			url:        "/tracez?%zzz",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, tt.url, http.NoBody)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.wantStatus, resp.StatusCode)

			// Verify content type for successful responses
			if resp.StatusCode == http.StatusOK && tt.url == "/tracez" {
				assert.Equal(t, "text/html; charset=utf-8", resp.Header.Get("Content-Type"))
			}

			body := w.Body.String()

			// Only check body content for successful responses
			if resp.StatusCode == http.StatusOK {
				assert.NotEmpty(t, body, "expected non-empty response body")

				for _, want := range tt.wantContains {
					assert.Contains(t, body, want)
				}
				for _, notWant := range tt.wantNotContain {
					assert.NotContains(t, body, notWant)
				}
			}
		})
	}
}

func TestTracezHandler_FormatsAttributesUsingSpecString(t *testing.T) {
	// attribute.Value.Emit is deprecated in favor of attribute.Value.String,
	// which follows the OpenTelemetry AnyValue representation for non-OTLP
	// protocols. The two representations disagree for:
	//   - FLOAT64SLICE containing NaN or Inf: Emit produces invalid output
	//     (e.g. "invalid: [NaN 1.5]") while String produces a valid JSON
	//     array (e.g. ["NaN",1.5]).
	//   - FLOAT64 containing +/-Inf: Emit renders "+Inf"/"-Inf" while String
	//     renders the spec's "Infinity"/"-Infinity".
	//   - BOOLSLICE: Emit space-separates elements (Go's %v on a slice)
	//     while String comma-separates them as a JSON array.
	testCases := []struct {
		name  string
		key   string
		value attribute.Value
		want  string
	}{
		{
			name:  "float64 slice with NaN",
			key:   "nums",
			value: attribute.Float64SliceValue([]float64{math.NaN(), 1.5}),
			want:  `nums=[&#34;NaN&#34;,1.5]`,
		},
		{
			name:  "float64 slice with +Inf",
			key:   "nums",
			value: attribute.Float64SliceValue([]float64{1.5, math.Inf(1)}),
			want:  `nums=[1.5,&#34;Infinity&#34;]`,
		},
		{
			name:  "float64 +Inf",
			key:   "num",
			value: attribute.Float64Value(math.Inf(1)),
			want:  `num=Infinity`,
		},
		{
			name:  "float64 -Inf",
			key:   "num",
			value: attribute.Float64Value(math.Inf(-1)),
			want:  `num=-Infinity`,
		},
		{
			name:  "bool slice",
			key:   "flags",
			value: attribute.BoolSliceValue([]bool{true, false}),
			want:  `flags=[true,false]`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sp := NewSpanProcessor()
			defer func() {
				require.NoError(t, sp.Shutdown(t.Context()))
			}()

			tp := sdktrace.NewTracerProvider(
				sdktrace.WithSpanProcessor(sp),
			)
			defer func() {
				require.NoError(t, tp.Shutdown(t.Context()))
			}()

			tracer := tp.Tracer("test-tracer")
			ctx := t.Context()

			// Use an error span rather than a latency-bucketed one: which
			// latency bucket a near-instant span lands in is
			// timing-dependent and flaky across environments, whereas
			// error spans aren't bucketed.
			_, span := tracer.Start(ctx, "attribute-span")
			span.SetAttributes(attribute.KeyValue{Key: attribute.Key(tc.key), Value: tc.value})
			span.SetStatus(codes.Error, "boom")
			span.End()

			handler := NewTracezHandler(sp)

			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/tracez?zspanname=attribute-span&ztype=2", http.NoBody)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			require.Equal(t, http.StatusOK, resp.StatusCode)

			body := w.Body.String()
			assert.Contains(t, body, tc.want)
			assert.NotContains(t, body, "invalid:")
		})
	}
}

func TestTracezHandler_ConcurrentSafe(t *testing.T) {
	sp := NewSpanProcessor()
	defer func() {
		require.NoError(t, sp.Shutdown(t.Context()))
	}()

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sp),
	)
	defer func() {
		require.NoError(t, tp.Shutdown(t.Context()))
	}()

	tracer := tp.Tracer("test-tracer")
	ctx := t.Context()

	for range 10 {
		_, span := tracer.Start(ctx, "concurrent-span")
		span.End()
	}

	handler := NewTracezHandler(sp)

	var wg sync.WaitGroup
	for range 5 {
		wg.Go(func() {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/tracez", http.NoBody)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			resp := w.Result()
			_ = resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})
	}

	wg.Wait()
}

func TestTracezHandler_ActiveOnlySpan(t *testing.T) {
	sp := NewSpanProcessor()
	defer func() {
		require.NoError(t, sp.Shutdown(t.Context()))
	}()

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sp),
	)
	defer func() {
		require.NoError(t, tp.Shutdown(t.Context()))
	}()

	tracer := tp.Tracer("test-tracer")
	ctx := t.Context()

	_, activeSpan := tracer.Start(ctx, "active-only-span")
	defer activeSpan.End()

	handler := NewTracezHandler(sp)

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/tracez", http.NoBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	body := w.Body.String()

	// Verify the active span is shown with active count = 1.
	assert.Contains(t, body, `<a href="tracez?zspanname=active-only-span&ztype=0">1</a>`)

	// Verify all 9 latency buckets are rendered as cells with 0 count.
	for i := range defaultBoundaries.numBuckets() {
		expectedCell := `<a href="tracez?zspanname=active-only-span&ztype=1&zlatencybucket=` + strconv.Itoa(i) + `">0</a>`
		assert.Contains(t, body, expectedCell)
	}

	// Verify error cell renders with closing </a> tag.
	assert.Contains(t, body, `<a href="tracez?zspanname=active-only-span&ztype=2&zlatencybucket=0">0</a></td>`)
}

func TestSpanRows_DoesNotMutateSpan(t *testing.T) {
	sp := NewSpanProcessor()
	defer func() {
		require.NoError(t, sp.Shutdown(t.Context()))
	}()

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sp),
	)
	defer func() {
		require.NoError(t, tp.Shutdown(t.Context()))
	}()

	tracer := tp.Tracer("test-tracer")
	ctx := t.Context()

	origAttrs := []attribute.KeyValue{
		attribute.String("z-key", "z-val"),
		attribute.String("a-key", "a-val"),
		attribute.String("m-key", "m-val"),
	}

	eventAttrs := []attribute.KeyValue{
		attribute.String("y-key", "y-val"),
		attribute.String("b-key", "b-val"),
	}

	t0 := time.Unix(1000, 0)
	_, span := tracer.Start(ctx, "mutation-test-span", trace.WithTimestamp(t0))
	span.SetAttributes(origAttrs...)
	// Add events with timestamps out of chronological order to ensure
	// sorting events in spanRows does not mutate the span's events slice in-place.
	span.AddEvent("second-event", trace.WithTimestamp(t0.Add(time.Second)), trace.WithAttributes(eventAttrs...))
	span.AddEvent("first-event", trace.WithTimestamp(t0))
	span.End(trace.WithTimestamp(t0.Add(2 * time.Second)))

	spans := sp.spansByLatency("mutation-test-span", 0)
	// If it ended up in another latency bucket, find it
	if len(spans) == 0 {
		for i := range defaultBoundaries.numBuckets() {
			spans = sp.spansByLatency("mutation-test-span", i)
			if len(spans) > 0 {
				break
			}
		}
	}
	require.NotEmpty(t, spans)
	roSpan := spans[0]

	// Copy the attributes slice to compare order before and after
	attrsBefore := append([]attribute.KeyValue(nil), roSpan.Attributes()...)
	eventsBefore := append([]sdktrace.Event(nil), roSpan.Events()...)
	eventAttrsBefore := append([]attribute.KeyValue(nil), roSpan.Events()[0].Attributes...)

	rows := spanRows(roSpan)
	require.NotEmpty(t, rows)

	// Verify the original span's attributes and event attributes were not sorted in-place
	assert.Equal(t, attrsBefore, roSpan.Attributes(), "span.Attributes() was mutated in-place")
	assert.Equal(t, eventsBefore, roSpan.Events(), "span.Events() was mutated in-place")
	assert.Equal(t, eventAttrsBefore, roSpan.Events()[0].Attributes, "event.Attributes was mutated in-place")
}
