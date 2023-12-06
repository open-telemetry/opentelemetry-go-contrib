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

package test

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
	if got, expected := rr.Result().StatusCode, http.StatusOK; got != expected {
		t.Fatalf("got %d, expected %d", got, expected)
	}
	if got := rr.Header().Get("Traceparent"); got != "" {
		t.Fatal("expected empty trace header")
	}
	if got, expected := len(spanRecorder.Ended()), 0; got != expected {
		t.Fatalf("got %d recorded spans, expected %d", got, expected)
	}
	d, err := io.ReadAll(rr.Result().Body)
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
		formatter func(s string, r *http.Request) string
		operation string
		expected  string
	}{
		{
			name: "default handler formatter",
			formatter: func(operation string, _ *http.Request) string {
				return operation
			},
			operation: "test_operation",
			expected:  "test_operation",
		},
		{
			name: "default transport formatter",
			formatter: func(_ string, r *http.Request) string {
				return "HTTP " + r.Method
			},
			expected: "HTTP GET",
		},
		{
			name: "custom formatter",
			formatter: func(s string, r *http.Request) string {
				return r.URL.Path
			},
			operation: "",
			expected:  "/hello",
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
			h := otelhttp.NewHandler(
				handler,
				tc.operation,
				otelhttp.WithTracerProvider(provider),
				otelhttp.WithSpanNameFormatter(tc.formatter),
			)
			r, err := http.NewRequest(http.MethodGet, "http://localhost/hello", nil)
			if err != nil {
				t.Fatal(err)
			}
			h.ServeHTTP(rr, r)
			if got, expected := rr.Result().StatusCode, http.StatusOK; got != expected {
				t.Fatalf("got %d, expected %d", got, expected)
			}

			spans := spanRecorder.Ended()
			if assert.Len(t, spans, 1) {
				assert.Equal(t, tc.expected, spans[0].Name())
			}
		})
	}
}
