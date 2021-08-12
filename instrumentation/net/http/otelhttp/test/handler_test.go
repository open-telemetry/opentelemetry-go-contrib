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
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/metrictest"
	"go.opentelemetry.io/otel/oteltest"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

func assertMetricAttributes(t *testing.T, expectedAttributes []attribute.KeyValue, measurementBatches []metrictest.Batch) {
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
	meterimpl, meterProvider := metrictest.NewMeterProvider()

	operation := "test_handler"

	h := otelhttp.NewHandler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			l, _ := otelhttp.LabelerFromContext(r.Context())
			l.Add(attribute.String("test", "attribute"))

			if _, err := io.WriteString(w, "hello world"); err != nil {
				t.Fatal(err)
			}
		}), operation,
		otelhttp.WithTracerProvider(provider),
		otelhttp.WithMeterProvider(meterProvider),
		otelhttp.WithPropagators(propagation.TraceContext{}),
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
