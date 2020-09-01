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
package http

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/codes"

	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/semconv"

	mockmeter "go.opentelemetry.io/contrib/internal/metric"
	mocktrace "go.opentelemetry.io/contrib/internal/trace"
)

func assertMetricLabels(t *testing.T, expectedLabels []label.KeyValue, measurementBatches []mockmeter.Batch) {
	for _, batch := range measurementBatches {
		assert.ElementsMatch(t, expectedLabels, batch.Labels)
	}
}

func TestHandlerBasics(t *testing.T) {
	rr := httptest.NewRecorder()

	tracerProvider, tracer := mocktrace.NewProviderAndTracer(instrumentationName)
	meterimpl, meterProvider := mockmeter.NewProvider()

	operation := "test_handler"

	h := NewHandler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, err := io.WriteString(w, "hello world"); err != nil {
				t.Fatal(err)
			}
		}), operation,
		WithTracerProvider(tracerProvider),
		WithMeterProvider(meterProvider),
	)

	r, err := http.NewRequest(http.MethodGet, "http://localhost/", strings.NewReader("foo"))
	if err != nil {
		t.Fatal(err)
	}
	h.ServeHTTP(rr, r)

	if len(meterimpl.MeasurementBatches) == 0 {
		t.Fatalf("got 0 recorded measurements, expected 1 or more")
	}

	labelsToVerify := []label.KeyValue{
		semconv.HTTPServerNameKey.String(operation),
		semconv.HTTPSchemeHTTP,
		semconv.HTTPHostKey.String(r.Host),
		semconv.HTTPFlavorKey.String(fmt.Sprintf("1.%d", r.ProtoMinor)),
	}

	assertMetricLabels(t, labelsToVerify, meterimpl.MeasurementBatches)

	if got, expected := rr.Result().StatusCode, http.StatusOK; got != expected {
		t.Fatalf("got %d, expected %d", got, expected)
	}
	if got := rr.Header().Get("Traceparent"); got == "" {
		t.Fatal("expected non empty trace header")
	}
	if got, expected := tracer.StartSpanID, uint64(1); got != expected {
		t.Fatalf("got %d, expected %d", got, expected)
	}
	d, err := ioutil.ReadAll(rr.Result().Body)
	if err != nil {
		t.Fatal(err)
	}
	if got, expected := string(d), "hello world"; got != expected {
		t.Fatalf("got %q, expected %q", got, expected)
	}
}

func TestHandlerNoWrite(t *testing.T) {
	rr := httptest.NewRecorder()

	tracerProvider, tracer := mocktrace.NewProviderAndTracer(instrumentationName)

	operation := "test_handler"
	var span trace.Span

	h := NewHandler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			span = trace.SpanFromContext(r.Context())
		}), operation,
		WithTracerProvider(tracerProvider),
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
	if got, expected := tracer.StartSpanID, uint64(1); got != expected {
		t.Fatalf("got %d, expected %d", got, expected)
	}
	if mockSpan, ok := span.(*mocktrace.Span); ok {
		if got, expected := mockSpan.Status, codes.OK; got != expected {
			t.Fatalf("got %q, expected %q", got, expected)
		}
	} else {
		t.Fatalf("Expected *moctrace.MockSpan, got %T", span)
	}
}

func TestResponseWriterOptionalInterfaces(t *testing.T) {
	rr := httptest.NewRecorder()

	tracerProvider, _ := mocktrace.NewProviderAndTracer(instrumentationName)

	// ResponseRecorder implements the Flusher interface. Make sure the
	// wrapped ResponseWriter passed to the handler still implements
	// Flusher.

	var isFlusher bool
	h := NewHandler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, isFlusher = w.(http.Flusher)
			if _, err := io.WriteString(w, "hello world"); err != nil {
				t.Fatal(err)
			}
		}), "test_handler",
		WithTracerProvider(tracerProvider))

	r, err := http.NewRequest(http.MethodGet, "http://localhost/", nil)
	if err != nil {
		t.Fatal(err)
	}
	h.ServeHTTP(rr, r)
	if !isFlusher {
		t.Fatal("http.Flusher interface not exposed")
	}
}
