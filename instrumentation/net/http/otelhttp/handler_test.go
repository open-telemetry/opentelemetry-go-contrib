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

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/oteltest"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/semconv"
	"go.opentelemetry.io/otel/trace"
)

func assertMetricAttributes(t *testing.T, expectedAttributes []attribute.KeyValue, measurementBatches []oteltest.Batch) {
	for _, batch := range measurementBatches {
		assert.ElementsMatch(t, expectedAttributes, batch.Labels)
	}
}

func TestHandlerBasics(t *testing.T) {
	rr := httptest.NewRecorder()

	spanRecorder := new(oteltest.SpanRecorder)
	provider := oteltest.NewTracerProvider(
		oteltest.WithSpanRecorder(spanRecorder),
	)
	meterimpl, meterProvider := oteltest.NewMeterProvider()

	operation := "test_handler"

	h := NewHandler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			l, _ := LabelerFromContext(r.Context())
			l.Add(attribute.String("test", "attribute"))

			if _, err := io.WriteString(w, "hello world"); err != nil {
				t.Fatal(err)
			}
		}), operation,
		WithTracerProvider(provider),
		WithMeterProvider(meterProvider),
		WithPropagators(propagation.TraceContext{}),
	)

	r, err := http.NewRequest(http.MethodGet, "http://localhost/", strings.NewReader("foo"))
	if err != nil {
		t.Fatal(err)
	}
	h.ServeHTTP(rr, r)

	if len(meterimpl.MeasurementBatches) == 0 {
		t.Fatalf("got 0 recorded measurements, expected 1 or more")
	}

	attributesToVerify := []attribute.KeyValue{
		semconv.HTTPServerNameKey.String(operation),
		semconv.HTTPSchemeHTTP,
		semconv.HTTPHostKey.String(r.Host),
		semconv.HTTPFlavorKey.String(fmt.Sprintf("1.%d", r.ProtoMinor)),
		attribute.String("test", "attribute"),
	}

	assertMetricAttributes(t, attributesToVerify, meterimpl.MeasurementBatches)

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
	if got, expected := spans[0].SpanContext().SpanID(), expectSpanID; got != expected {
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
	provider := oteltest.NewTracerProvider()

	operation := "test_handler"
	var span trace.Span

	h := NewHandler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			span = trace.SpanFromContext(r.Context())
		}), operation,
		WithTracerProvider(provider),
		WithPropagators(propagation.TraceContext{}),
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
	if got, expected := span.SpanContext().SpanID(), expectSpanID; got != expected {
		t.Fatalf("got %d, expected %d", got, expected)
	}
	if mockSpan, ok := span.(*oteltest.Span); ok {
		if got, expected := mockSpan.StatusCode(), codes.Unset; got != expected {
			t.Fatalf("got %q, expected %q", got, expected)
		}
	} else {
		t.Fatalf("Expected *moctrace.MockSpan, got %T", span)
	}
}

func TestResponseWriterOptionalInterfaces(t *testing.T) {
	rr := httptest.NewRecorder()

	provider := oteltest.NewTracerProvider()

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

// This use case is important as we make sure the body isn't mutated
// when it is nil. This is a common use case for tests where the request
// is directly passed to the handler.
func TestHandlerReadingNilBodySuccess(t *testing.T) {
	rr := httptest.NewRecorder()

	provider := oteltest.NewTracerProvider()

	h := NewHandler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				_, err := ioutil.ReadAll(r.Body)
				assert.NotNil(t, err)
			}
		}), "test_handler",
		WithTracerProvider(provider),
	)

	r, err := http.NewRequest(http.MethodGet, "http://localhost/", nil)
	if err != nil {
		t.Fatal(err)
	}
	h.ServeHTTP(rr, r)
	assert.Equal(t, 200, rr.Result().StatusCode)
}
