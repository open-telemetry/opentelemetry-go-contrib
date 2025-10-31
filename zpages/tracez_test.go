// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zpages

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestNewTracezHandler(t *testing.T) {
	sp := NewSpanProcessor()
	handler := NewTracezHandler(sp)

	if handler == nil {
		t.Fatal("NewTracezHandler returned nil")
	}

	var _ = handler
}

func TestTracezHandler_ServeHTTP_BasicResponse(t *testing.T) {
	sp := NewSpanProcessor()
	handler := NewTracezHandler(sp)

	req := httptest.NewRequest(http.MethodGet, "/tracez", http.NoBody)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status OK, got %v", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("expected Content-Type 'text/html; charset=utf-8', got %q", contentType)
	}

	body := w.Body.String()
	if body == "" {
		t.Error("expected non-empty response body")
	}
}

func TestTracezHandler_ServeHTTP_WithRealSpans(t *testing.T) {
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
	_, span1 := tracer.Start(ctx, "test-span-1")
	span1.End()

	_, span2 := tracer.Start(ctx, "test-span-2")
	span2.End()

	handler := NewTracezHandler(sp)

	req := httptest.NewRequest(http.MethodGet, "/tracez", http.NoBody)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status OK, got %v", resp.StatusCode)
	}

	body := w.Body.String()
	if !strings.Contains(body, "test-span-1") {
		t.Error("expected response to contain span name 'test-span-1'")
	}
	if !strings.Contains(body, "test-span-2") {
		t.Error("expected response to contain span name 'test-span-2'")
	}
}

func TestTracezHandler_ServeHTTP_WithSpanNameQuery(t *testing.T) {
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

	_, span := tracer.Start(ctx, "query-span")
	span.End()

	handler := NewTracezHandler(sp)

	tests := []struct {
		name      string
		queryPath string
		wantSpan  bool
	}{
		{
			name:      "query for existing span",
			queryPath: "/tracez?zspanname=query-span",
			wantSpan:  true,
		},
		{
			name:      "query for non-existing span",
			queryPath: "/tracez?zspanname=non-existing",
			wantSpan:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.queryPath, http.NoBody)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected status OK, got %v", resp.StatusCode)
			}

			body := w.Body.String()
			containsSpan := strings.Contains(body, "query-span")
			if tt.wantSpan && !containsSpan {
				t.Error("expected response to contain span name")
			}
		})
	}
}

func TestTracezHandler_ServeHTTP_SpanTypes(t *testing.T) {
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
	errorSpan.RecordError(context.DeadlineExceeded)
	errorSpan.End()

	handler := NewTracezHandler(sp)

	tests := []struct {
		name       string
		spanName   string
		spanType   string
		wantStatus int
	}{
		{
			name:       "latency spans (type=1)",
			spanName:   "completed-span",
			spanType:   "1",
			wantStatus: http.StatusOK,
		},
		{
			name:       "error spans (type=2)",
			spanName:   "error-span",
			spanType:   "2",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/tracez?zspanname="+tt.spanName+"&ztype="+tt.spanType, http.NoBody)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("expected status %v, got %v", tt.wantStatus, resp.StatusCode)
			}
		})
	}
}

func TestTracezHandler_ServeHTTP_LatencyBucket(t *testing.T) {
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

	_, span := tracer.Start(ctx, "latency-span")
	span.End()

	handler := NewTracezHandler(sp)

	req := httptest.NewRequest(http.MethodGet, "/tracez?zspanname=latency-span&ztype=1&zlatencybucket=0", http.NoBody)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status OK, got %v", resp.StatusCode)
	}
}

func TestTracezHandler_ServeHTTP_InvalidForm(t *testing.T) {
	sp := NewSpanProcessor()
	handler := NewTracezHandler(sp)

	req := httptest.NewRequest(http.MethodGet, "/tracez?%zzz", http.NoBody)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status OK or BadRequest, got %v", resp.StatusCode)
	}
}

func TestTracezHandler_ServeHTTP_MultipleSpans(t *testing.T) {
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

	for range 5 {
		_, span := tracer.Start(ctx, "multi-span")
		span.End()
	}

	handler := NewTracezHandler(sp)

	req := httptest.NewRequest(http.MethodGet, "/tracez?zspanname=multi-span&ztype=1", http.NoBody)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status OK, got %v", resp.StatusCode)
	}

	body := w.Body.String()
	if !strings.Contains(body, "multi-span") {
		t.Error("expected response to contain span name")
	}
}

func TestTracezHandler_ServeHTTP_AllQueryParameters(t *testing.T) {
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

	_, span := tracer.Start(ctx, "param-test")
	span.End()

	handler := NewTracezHandler(sp)

	tests := []struct {
		name string
		url  string
	}{
		{"with all parameters", "/tracez?zspanname=param-test&ztype=1&zlatencybucket=2"},
		{"with type only", "/tracez?ztype=1"},
		{"with latency bucket only", "/tracez?zlatencybucket=3"},
		{"with invalid type value", "/tracez?ztype=invalid"},
		{"with negative latency bucket", "/tracez?zlatencybucket=-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, http.NoBody)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected status OK, got %v", resp.StatusCode)
			}
		})
	}
}

func TestTracezHandler_ServeHTTP_ConcurrentRequests(t *testing.T) {
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

	done := make(chan bool, 5)
	for range 5 {
		go func() {
			req := httptest.NewRequest(http.MethodGet, "/tracez", http.NoBody)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			resp := w.Result()
			_ = resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			done <- true
		}()
	}

	for range 5 {
		<-done
	}
}

func TestTracezHandler_Integration(t *testing.T) {
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

	otel.SetTracerProvider(tp)

	tracer := otel.Tracer("integration-test")
	ctx := t.Context()

	_, normalSpan := tracer.Start(ctx, "normal-operation")
	normalSpan.End()

	_, errorSpan := tracer.Start(ctx, "error-operation")
	errorSpan.RecordError(context.Canceled)
	errorSpan.End()

	handler := NewTracezHandler(sp)

	req := httptest.NewRequest(http.MethodGet, "/tracez", http.NoBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	body := w.Body.String()
	require.Contains(t, body, "normal-operation")
	require.Contains(t, body, "error-operation")
}
