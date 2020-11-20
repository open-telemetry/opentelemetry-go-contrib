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
package otelhttp

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/api/metric/metrictest"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/api/trace/tracetest"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/propagators"
	"go.opentelemetry.io/otel/semconv"
)

func assertMetricLabels(t *testing.T, expectedLabels []label.KeyValue, measurementBatches []metrictest.Batch) {
	for _, batch := range measurementBatches {
		assert.ElementsMatch(t, expectedLabels, batch.Labels)
	}
}

func TestHandlerBasics(t *testing.T) {
	rr := httptest.NewRecorder()

	spanRecorder := new(tracetest.StandardSpanRecorder)
	provider := tracetest.NewTracerProvider(
		tracetest.WithSpanRecorder(spanRecorder),
	)
	meterimpl, meterProvider := metrictest.NewMeterProvider()

	operation := "test_handler"

	h := NewHandler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			l, _ := LabelerFromContext(r.Context())
			l.Add(label.String("test", "label"))

			if _, err := io.WriteString(w, "hello world"); err != nil {
				t.Fatal(err)
			}
		}), operation,
		WithTracerProvider(provider),
		WithMeterProvider(meterProvider),
		WithPropagators(propagators.TraceContext{}),
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
		label.String("test", "label"),
	}

	assertMetricLabels(t, labelsToVerify, meterimpl.MeasurementBatches)

	if got, expected := rr.Result().StatusCode, http.StatusOK; got != expected {
		t.Fatalf("got %d, expected %d", got, expected)
	}
	if got := rr.Header().Get("Traceparent"); got == "" {
		t.Fatal("expected non empty trace header")
	}

	spans := spanRecorder.Completed()
	if got, expected := len(spans), 1; got != expected {
		t.Fatalf("got %d spans, expected %d", got, expected)
	}
	expectSpanID := trace.SpanID{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2} // we expect the span ID to be incremented by one
	if got, expected := spans[0].SpanContext().SpanID, expectSpanID; got != expected {
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
	provider := tracetest.NewTracerProvider()

	operation := "test_handler"
	var span trace.Span

	h := NewHandler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			span = trace.SpanFromContext(r.Context())
		}), operation,
		WithTracerProvider(provider),
		WithPropagators(propagators.TraceContext{}),
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
	expectSpanID := trace.SpanID{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2} // we expect the span ID to be incremented by one
	if got, expected := span.SpanContext().SpanID, expectSpanID; got != expected {
		t.Fatalf("got %d, expected %d", got, expected)
	}
	if mockSpan, ok := span.(*tracetest.Span); ok {
		if got, expected := mockSpan.StatusCode(), codes.Unset; got != expected {
			t.Fatalf("got %q, expected %q", got, expected)
		}
	} else {
		t.Fatalf("Expected *moctrace.MockSpan, got %T", span)
	}
}

func TestResponseWriterOptionalInterfaces(t *testing.T) {
	rr := httptest.NewRecorder()

	provider := tracetest.NewTracerProvider()

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
		WithTracerProvider(provider))

	r, err := http.NewRequest(http.MethodGet, "http://localhost/", nil)
	if err != nil {
		t.Fatal(err)
	}
	h.ServeHTTP(rr, r)
	if !isFlusher {
		t.Fatal("http.Flusher interface not exposed")
	}
}
