// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zpages

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
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
	errorSpan.RecordError(context.DeadlineExceeded) //nolint:forbidigo // existing usage of Span.RecordError
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
			wantContains: []string{},
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
			req := httptest.NewRequest(http.MethodGet, tt.url, http.NoBody)
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
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/tracez", http.NoBody)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			resp := w.Result()
			_ = resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
		}()
	}

	wg.Wait()
}
