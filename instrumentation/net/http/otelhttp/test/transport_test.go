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
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestTransportUsesFormatter(t *testing.T) {
	prop := propagation.TraceContext{}
	spanRecorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	content := []byte("Hello, world!")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := prop.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		span := trace.SpanContextFromContext(ctx)
		if !span.IsValid() {
			t.Fatalf("invalid span wrapping handler: %#v", span)
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

	tr := otelhttp.NewTransport(
		http.DefaultTransport,
		otelhttp.WithTracerProvider(provider),
		otelhttp.WithPropagators(prop),
	)

	c := http.Client{Transport: tr}
	res, err := c.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	require.NoError(t, res.Body.Close())

	spans := spanRecorder.Ended()
	spanName := spans[0].Name()
	expectedName := "HTTP GET"
	if spanName != expectedName {
		t.Fatalf("unexpected name: got %s, expected %s", spanName, expectedName)
	}
}

func TestTransportErrorStatus(t *testing.T) {
	// Prepare tracing stuff.
	spanRecorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))

	// Run a server and stop to make sure nothing is listening and force the error.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close()

	// Create our Transport and make request.
	tr := otelhttp.NewTransport(
		http.DefaultTransport,
		otelhttp.WithTracerProvider(provider),
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
	spans := spanRecorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span; got: %d", len(spans))
	}
	span := spans[0]

	if span.EndTime().IsZero() {
		t.Errorf("span should be ended; it isn't")
	}

	if got := span.Status().Code; got != codes.Error {
		t.Errorf("expected error status code on span; got: %q", got)
	}

	errSubstr := "connect: connection refused"
	if runtime.GOOS == "windows" {
		// tls.Dial returns an error that does not contain the substring "connection refused"
		// on Windows machines
		//
		// ref: "dial tcp 127.0.0.1:50115: connectex: No connection could be made because the target machine actively refused it."
		errSubstr = "No connection could be made because the target machine actively refused it"
	}
	if got := span.Status().Description; !strings.Contains(got, errSubstr) {
		t.Errorf("expected error status message on span; got: %q", got)
	}
}

func TestTransportRequestWithTraceContext(t *testing.T) {
	spanRecorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(spanRecorder),
	)
	content := []byte("Hello, world!")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write(content)
		require.NoError(t, err)
	}))
	defer ts.Close()

	tracer := provider.Tracer("")
	ctx, span := tracer.Start(context.Background(), "test_span")

	r, err := http.NewRequest(http.MethodGet, ts.URL, nil)
	require.NoError(t, err)

	r = r.WithContext(ctx)

	tr := otelhttp.NewTransport(
		http.DefaultTransport,
	)

	c := http.Client{Transport: tr}
	res, err := c.Do(r)
	require.NoError(t, err)

	span.End()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	require.Equal(t, content, body)

	spans := spanRecorder.Ended()
	require.Len(t, spans, 2)

	assert.Equal(t, "test_span", spans[0].Name())
	assert.Equal(t, "HTTP GET", spans[1].Name())
	assert.NotEmpty(t, spans[1].Parent().SpanID())
	assert.Equal(t, spans[0].SpanContext().SpanID(), spans[1].Parent().SpanID())
}

func TestWithHTTPTrace(t *testing.T) {
	spanRecorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(spanRecorder),
	)
	content := []byte("Hello, world!")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write(content)
		require.NoError(t, err)
	}))
	defer ts.Close()

	tracer := provider.Tracer("")
	ctx, span := tracer.Start(context.Background(), "test_span")

	r, err := http.NewRequest(http.MethodGet, ts.URL, nil)
	require.NoError(t, err)

	r = r.WithContext(ctx)

	clientTracer := func(ctx context.Context) *httptrace.ClientTrace {
		var span trace.Span
		return &httptrace.ClientTrace{
			GetConn: func(_ string) {
				_, span = trace.SpanFromContext(ctx).TracerProvider().Tracer("").Start(ctx, "httptrace.GetConn")
			},
			GotConn: func(_ httptrace.GotConnInfo) {
				if span != nil {
					span.End()
				}
			},
		}
	}

	tr := otelhttp.NewTransport(
		http.DefaultTransport,
		otelhttp.WithClientTrace(clientTracer),
	)

	c := http.Client{Transport: tr}
	res, err := c.Do(r)
	require.NoError(t, err)

	span.End()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	require.Equal(t, content, body)

	spans := spanRecorder.Ended()
	require.Len(t, spans, 3)

	assert.Equal(t, "httptrace.GetConn", spans[0].Name())
	assert.Equal(t, "test_span", spans[1].Name())
	assert.Equal(t, "HTTP GET", spans[2].Name())
	assert.NotEmpty(t, spans[0].Parent().SpanID())
	assert.NotEmpty(t, spans[2].Parent().SpanID())
	assert.Equal(t, spans[2].SpanContext().SpanID(), spans[0].Parent().SpanID())
	assert.Equal(t, spans[1].SpanContext().SpanID(), spans[2].Parent().SpanID())
}
