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
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/oteltest"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TestTransportBasics(t *testing.T) {
	prop := propagation.TraceContext{}
	provider := oteltest.NewTracerProvider()
	content := []byte("Hello, world!")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := prop.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		span := trace.SpanContextFromContext(ctx)
		tgtID, err := trace.SpanIDFromHex(fmt.Sprintf("%016x", uint(2)))
		if err != nil {
			t.Fatalf("Error converting id to SpanID: %s", err.Error())
		}
		if span.SpanID() != tgtID {
			t.Fatalf("testing remote SpanID: got %s, expected %s", span.SpanID(), tgtID)
		}
		if _, err := w.Write(content); err != nil {
			t.Fatal(err)
		}
	}))
	defer ts.Close()

	r, err := http.NewRequest(http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	tr := NewTransport(
		http.DefaultTransport,
		WithTracerProvider(provider),
		WithPropagators(prop),
	)

	c := http.Client{Transport: tr}
	res, err := c.Do(r)
	if err != nil {
		t.Fatal(err)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(body, content) {
		t.Fatalf("unexpected content: got %s, expected %s", body, content)
	}
}

func TestNilTransport(t *testing.T) {
	prop := propagation.TraceContext{}
	provider := oteltest.NewTracerProvider()
	content := []byte("Hello, world!")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := prop.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		span := trace.SpanContextFromContext(ctx)
		tgtID, err := trace.SpanIDFromHex(fmt.Sprintf("%016x", uint(2)))
		if err != nil {
			t.Fatalf("Error converting id to SpanID: %s", err.Error())
		}
		if span.SpanID() != tgtID {
			t.Fatalf("testing remote SpanID: got %s, expected %s", span.SpanID(), tgtID)
		}
		if _, err := w.Write(content); err != nil {
			t.Fatal(err)
		}
	}))
	defer ts.Close()

	r, err := http.NewRequest(http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	tr := NewTransport(
		nil,
		WithTracerProvider(provider),
		WithPropagators(prop),
	)

	c := http.Client{Transport: tr}
	res, err := c.Do(r)
	if err != nil {
		t.Fatal(err)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(body, content) {
		t.Fatalf("unexpected content: got %s, expected %s", body, content)
	}
}

func TestTransportFormatter(t *testing.T) {

	var httpMethods = []struct {
		name     string
		method   string
		expected string
	}{
		{
			"GET method",
			http.MethodGet,
			"HTTP GET",
		},
		{
			"HEAD method",
			http.MethodHead,
			"HTTP HEAD",
		},
		{
			"POST method",
			http.MethodPost,
			"HTTP POST",
		},
		{
			"PUT method",
			http.MethodPut,
			"HTTP PUT",
		},
		{
			"PATCH method",
			http.MethodPatch,
			"HTTP PATCH",
		},
		{
			"DELETE method",
			http.MethodDelete,
			"HTTP DELETE",
		},
		{
			"CONNECT method",
			http.MethodConnect,
			"HTTP CONNECT",
		},
		{
			"OPTIONS method",
			http.MethodOptions,
			"HTTP OPTIONS",
		},
		{
			"TRACE method",
			http.MethodTrace,
			"HTTP TRACE",
		},
	}

	for _, tc := range httpMethods {
		t.Run(tc.name, func(t *testing.T) {
			r, err := http.NewRequest(tc.method, "http://localhost/", nil)
			if err != nil {
				t.Fatal(err)
			}
			formattedName := defaultTransportFormatter("", r)

			if formattedName != tc.expected {
				t.Fatalf("unexpected name: got %s, expected %s", formattedName, tc.expected)
			}
		})
	}

}

func TestTransportUsesFormatter(t *testing.T) {
	prop := propagation.TraceContext{}
	spanRecorder := new(oteltest.SpanRecorder)
	provider := oteltest.NewTracerProvider(
		oteltest.WithSpanRecorder(spanRecorder),
	)
	content := []byte("Hello, world!")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := prop.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		span := trace.SpanContextFromContext(ctx)
		tgtID, err := trace.SpanIDFromHex(fmt.Sprintf("%016x", uint(2)))
		if err != nil {
			t.Fatalf("Error converting id to SpanID: %s", err.Error())
		}
		if span.SpanID() != tgtID {
			t.Fatalf("testing remote SpanID: got %s, expected %s", span.SpanID(), tgtID)
		}
		if _, err := w.Write(content); err != nil {
			t.Fatal(err)
		}
	}))
	defer ts.Close()

	r, err := http.NewRequest(http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	tr := NewTransport(
		http.DefaultTransport,
		WithTracerProvider(provider),
		WithPropagators(prop),
	)

	c := http.Client{Transport: tr}
	res, err := c.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	spans := spanRecorder.Completed()
	spanName := spans[0].Name()
	expectedName := "HTTP GET"
	if spanName != expectedName {
		t.Fatalf("unexpected name: got %s, expected %s", spanName, expectedName)
	}

}

func TestTransportErrorStatus(t *testing.T) {
	// Prepare tracing stuff.
	prop := propagation.TraceContext{}
	spanRecorder := new(oteltest.SpanRecorder)
	provider := oteltest.NewTracerProvider(
		oteltest.WithSpanRecorder(spanRecorder),
	)

	// Run a server and stop to make sure nothing is listening and force the error.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close()

	// Create our Transport and make request.
	tr := NewTransport(
		http.DefaultTransport,
		WithTracerProvider(provider),
		WithPropagators(prop),
	)
	c := http.Client{Transport: tr}
	r, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Do(r)
	if err == nil {
		t.Fatal("transport should have returned an error, it didn't")
	}

	// Check span.
	gotSpans := spanRecorder.Completed()
	if len(gotSpans) != 1 {
		t.Fatalf("expected 1 span; got: %d", len(gotSpans))
	}

	spanEnded := gotSpans[0].Ended()
	if !spanEnded {
		t.Fatalf("span should be ended; it isn't")
	}

	spanStatusCode := gotSpans[0].StatusCode()
	if spanStatusCode != codes.Error {
		t.Fatalf("expected error status code on span; got: %q", spanStatusCode)
	}

	spanStatusMessage := gotSpans[0].StatusMessage()
	if !strings.Contains(spanStatusMessage, "connect: connection refused") {
		t.Fatalf("expected error status message on span; got: %q", spanStatusMessage)
	}
}

type testErrorReadCloser struct{}

func (testErrorReadCloser) Read(p []byte) (n int, err error) { return 0, fmt.Errorf("something") }
func (testErrorReadCloser) Close() error                     { return nil }

func TestWrappedBodyReadErrorStatus(t *testing.T) {
	// Prepare tracing stuff.
	spanRecorder := new(oteltest.SpanRecorder)
	provider := oteltest.NewTracerProvider(
		oteltest.WithSpanRecorder(spanRecorder),
	)
	tracer := provider.Tracer("")
	ctx := context.Background()
	ctx, span := tracer.Start(ctx, "test")

	// Create our wrapper.
	wb := wrappedBody{
		span: span,
		body: testErrorReadCloser{},
	}
	_, err := wb.Read([]byte{})
	if err == nil {
		t.Fatalf("expected error while reading")
	}
	wb.Close()

	// Check span.
	gotSpans := spanRecorder.Completed()
	if len(gotSpans) != 1 {
		t.Fatalf("expected 1 span; got: %d", len(gotSpans))
	}

	spanEnded := gotSpans[0].Ended()
	if !spanEnded {
		t.Fatalf("span should be ended; it isn't")
	}

	spanStatusCode := gotSpans[0].StatusCode()
	if spanStatusCode != codes.Error {
		t.Fatalf("expected error status code on span; got: %q", spanStatusCode)
	}

	spanStatusMessage := gotSpans[0].StatusMessage()
	if !strings.Contains(spanStatusMessage, "something") {
		t.Fatalf("expected error status message on span; got: %q", spanStatusMessage)
	}
}
