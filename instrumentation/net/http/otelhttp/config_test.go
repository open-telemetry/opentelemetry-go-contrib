// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelhttp_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestBasicFilter(t *testing.T) {
	rr := httptest.NewRecorder()

	spanRecorder := tracetest.NewSpanRecorder()
	provider := trace.NewTracerProvider(trace.WithSpanProcessor(spanRecorder))

	h := otelhttp.NewHandler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, err := io.WriteString(w, "hello world"); err != nil {
				t.Fatal(err)
			}
		}), "test_handler",
		otelhttp.WithTracerProvider(provider),
		otelhttp.WithFilter(func(r *http.Request) bool {
			return false
		}),
	)

	r, err := http.NewRequest(http.MethodGet, "http://localhost/", nil)
	if err != nil {
		t.Fatal(err)
	}
	h.ServeHTTP(rr, r)
	if got, expected := rr.Result().StatusCode, http.StatusOK; got != expected { //nolint:bodyclose // False positive for httptest.ResponseRecorder: https://github.com/timakin/bodyclose/issues/59.
		t.Fatalf("got %d, expected %d", got, expected)
	}
	if got := rr.Header().Get("Traceparent"); got != "" {
		t.Fatal("expected empty trace header")
	}
	if got, expected := len(spanRecorder.Ended()), 0; got != expected {
		t.Fatalf("got %d recorded spans, expected %d", got, expected)
	}
	d, err := io.ReadAll(rr.Result().Body) //nolint:bodyclose // False positive for httptest.ResponseRecorder: https://github.com/timakin/bodyclose/issues/59.
	if err != nil {
		t.Fatal(err)
	}
	if got, expected := string(d), "hello world"; got != expected {
		t.Fatalf("got %q, expected %q", got, expected)
	}
}

func TestSpanNameFormatter(t *testing.T) {
	testCases := []struct {
		name      string
		formatter otelhttp.SpanNameFormatter
		operation string
		expected  string
	}{
		{
			name:      "default handler formatter",
			formatter: otelhttp.SpanNameFromOperation,
			operation: "test_operation",
			expected:  "test_operation",
		},
		{
			name:      "default transport formatter",
			formatter: otelhttp.SpanNameFromMethod,
			expected:  "HTTP GET",
		},
		{
			name:      "request pattern formatter",
			formatter: otelhttp.SpanNameFromPattern,
			expected:  "GET /hello/{thing}",
		},
		{
			name: "custom formatter",
			formatter: func(s string, r *http.Request) string {
				return r.URL.Path
			},
			operation: "",
			expected:  "/hello/world",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()

			spanRecorder := tracetest.NewSpanRecorder()
			provider := trace.NewTracerProvider(trace.WithSpanProcessor(spanRecorder))
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if _, err := io.WriteString(w, "hello world"); err != nil {
					t.Fatal(err)
				}
			})
			mux := http.NewServeMux()
			mux.Handle("GET /hello/{thing}", handler)
			h := otelhttp.NewHandler(
				mux,
				tc.operation,
				otelhttp.WithTracerProvider(provider),
				otelhttp.WithSpanNameFormatter(tc.formatter),
			)
			r, err := http.NewRequest(http.MethodGet, "http://localhost/hello/world", nil)
			if err != nil {
				t.Fatal(err)
			}
			h.ServeHTTP(rr, r)
			if got, expected := rr.Result().StatusCode, http.StatusOK; got != expected { //nolint:bodyclose // False positive for httptest.ResponseRecorder: https://github.com/timakin/bodyclose/issues/59.
				t.Fatalf("got %d, expected %d", got, expected)
			}

			spans := spanRecorder.Ended()
			if assert.Len(t, spans, 1) {
				assert.Equal(t, tc.expected, spans[0].Name())
			}
		})
	}
}
