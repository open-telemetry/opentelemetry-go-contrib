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

package otelmux

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mocktrace "go.opentelemetry.io/contrib/internal/trace"
	b3prop "go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	otelglobal "go.opentelemetry.io/otel/api/global"
	oteltrace "go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/propagators"
)

func TestChildSpanFromGlobalTracer(t *testing.T) {
	otelglobal.SetTracerProvider(&mocktrace.TracerProvider{})

	router := mux.NewRouter()
	router.Use(Middleware("foobar"))
	router.HandleFunc("/user/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := oteltrace.SpanFromContext(r.Context())
		_, ok := span.(*mocktrace.Span)
		assert.True(t, ok)
		spanTracer := span.Tracer()
		mockTracer, ok := spanTracer.(*mocktrace.Tracer)
		require.True(t, ok)
		assert.Equal(t, "go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux", mockTracer.Name)
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)
}

func TestChildSpanFromCustomTracer(t *testing.T) {
	provider, _ := mocktrace.NewTracerProviderAndTracer(tracerName)

	router := mux.NewRouter()
	router.Use(Middleware("foobar", WithTracerProvider(provider)))
	router.HandleFunc("/user/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := oteltrace.SpanFromContext(r.Context())
		_, ok := span.(*mocktrace.Span)
		assert.True(t, ok)
		spanTracer := span.Tracer()
		mockTracer, ok := spanTracer.(*mocktrace.Tracer)
		require.True(t, ok)
		assert.Equal(t, tracerName, mockTracer.Name)
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)
}

func TestChildSpanNames(t *testing.T) {
	provider, tracer := mocktrace.NewTracerProviderAndTracer(tracerName)

	router := mux.NewRouter()
	router.Use(Middleware("foobar", WithTracerProvider(provider)))
	router.HandleFunc("/user/{id:[0-9]+}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	router.HandleFunc("/book/{title}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(([]byte)("ok"))
	}))

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	spans := tracer.EndedSpans()
	require.Len(t, spans, 1)
	span := spans[0]
	assert.Equal(t, "/user/{id:[0-9]+}", span.Name)
	assert.Equal(t, oteltrace.SpanKindServer, span.Kind)
	assert.Equal(t, label.StringValue("foobar"), span.Attributes["http.server_name"])
	assert.Equal(t, label.IntValue(http.StatusOK), span.Attributes["http.status_code"])
	assert.Equal(t, label.StringValue("GET"), span.Attributes["http.method"])
	assert.Equal(t, label.StringValue("/user/123"), span.Attributes["http.target"])
	assert.Equal(t, label.StringValue("/user/{id:[0-9]+}"), span.Attributes["http.route"])

	r = httptest.NewRequest("GET", "/book/foo", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	spans = tracer.EndedSpans()
	require.Len(t, spans, 1)
	span = spans[0]
	assert.Equal(t, "/book/{title}", span.Name)
	assert.Equal(t, oteltrace.SpanKindServer, span.Kind)
	assert.Equal(t, label.StringValue("foobar"), span.Attributes["http.server_name"])
	assert.Equal(t, label.IntValue(http.StatusOK), span.Attributes["http.status_code"])
	assert.Equal(t, label.StringValue("GET"), span.Attributes["http.method"])
	assert.Equal(t, label.StringValue("/book/foo"), span.Attributes["http.target"])
	assert.Equal(t, label.StringValue("/book/{title}"), span.Attributes["http.route"])
}

func TestGetSpanNotInstrumented(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/user/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := oteltrace.SpanFromContext(r.Context())
		ok := !span.SpanContext().IsValid()
		assert.True(t, ok)
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)
}

func TestPropagationWithGlobalPropagators(t *testing.T) {
	provider, tracer := mocktrace.NewTracerProviderAndTracer(tracerName)
	otelglobal.SetTextMapPropagator(propagators.TraceContext{})

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	ctx, pspan := tracer.Start(context.Background(), "test")
	otelglobal.TextMapPropagator().Inject(ctx, r.Header)

	router := mux.NewRouter()
	router.Use(Middleware("foobar", WithTracerProvider(provider)))
	router.HandleFunc("/user/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := oteltrace.SpanFromContext(r.Context())
		mspan, ok := span.(*mocktrace.Span)
		require.True(t, ok)
		assert.Equal(t, pspan.SpanContext().TraceID, mspan.SpanContext().TraceID)
		assert.Equal(t, pspan.SpanContext().SpanID, mspan.ParentSpanID)
		w.WriteHeader(http.StatusOK)
	}))

	router.ServeHTTP(w, r)
	otelglobal.SetTextMapPropagator(otel.NewCompositeTextMapPropagator())
}

func TestPropagationWithCustomPropagators(t *testing.T) {
	provider, tracer := mocktrace.NewTracerProviderAndTracer(tracerName)

	b3 := b3prop.B3{}

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	ctx, pspan := tracer.Start(context.Background(), "test")
	b3.Inject(ctx, r.Header)

	router := mux.NewRouter()
	router.Use(Middleware("foobar", WithTracerProvider(provider), WithPropagators(b3)))
	router.HandleFunc("/user/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := oteltrace.SpanFromContext(r.Context())
		mspan, ok := span.(*mocktrace.Span)
		require.True(t, ok)
		assert.Equal(t, pspan.SpanContext().TraceID, mspan.SpanContext().TraceID)
		assert.Equal(t, pspan.SpanContext().SpanID, mspan.ParentSpanID)
		w.WriteHeader(http.StatusOK)
	}))

	router.ServeHTTP(w, r)
}

type testResponseWriter struct {
	writer http.ResponseWriter
}

func (rw *testResponseWriter) Header() http.Header {
	return rw.writer.Header()
}
func (rw *testResponseWriter) Write(b []byte) (int, error) {
	return rw.writer.Write(b)
}
func (rw *testResponseWriter) WriteHeader(statusCode int) {
	rw.writer.WriteHeader(statusCode)
}

// implement Hijacker
func (rw *testResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, nil
}

// implement Pusher
func (rw *testResponseWriter) Push(target string, opts *http.PushOptions) error {
	return nil
}

// implement Flusher
func (rw *testResponseWriter) Flush() {
}

// implement io.ReaderFrom
func (rw *testResponseWriter) ReadFrom(r io.Reader) (n int64, err error) {
	return 0, nil
}

func TestResponseWriterInterfaces(t *testing.T) {
	// make sure the recordingResponseWriter preserves interfaces implemented by the wrapped writer
	provider, _ := mocktrace.NewTracerProviderAndTracer(tracerName)

	router := mux.NewRouter()
	router.Use(Middleware("foobar", WithTracerProvider(provider)))
	router.HandleFunc("/user/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Implements(t, (*http.Hijacker)(nil), w)
		assert.Implements(t, (*http.Pusher)(nil), w)
		assert.Implements(t, (*http.Flusher)(nil), w)
		assert.Implements(t, (*io.ReaderFrom)(nil), w)
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := &testResponseWriter{
		writer: httptest.NewRecorder(),
	}

	router.ServeHTTP(w, r)
}
