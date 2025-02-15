// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelmux

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

var sc = trace.NewSpanContext(trace.SpanContextConfig{
	TraceID:    [16]byte{1},
	SpanID:     [8]byte{1},
	Remote:     true,
	TraceFlags: trace.FlagsSampled,
})

func TestPassthroughSpanFromGlobalTracer(t *testing.T) {
	var called bool
	router := mux.NewRouter()
	router.Use(Middleware("foobar"))
	// The default global TracerProvider provides "pass through" spans for any
	// span context in the incoming request context.
	router.HandleFunc("/user/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		got := trace.SpanFromContext(r.Context()).SpanContext()
		assert.Equal(t, sc, got)
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/user/123", nil)
	r = r.WithContext(trace.ContextWithRemoteSpanContext(context.Background(), sc))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)
	assert.True(t, called, "failed to run test")
}

func TestPropagationWithGlobalPropagators(t *testing.T) {
	defer func(p propagation.TextMapPropagator) {
		otel.SetTextMapPropagator(p)
	}(otel.GetTextMapPropagator())

	prop := propagation.TraceContext{}
	otel.SetTextMapPropagator(prop)

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	ctx := trace.ContextWithRemoteSpanContext(context.Background(), sc)
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(r.Header))

	var called bool
	router := mux.NewRouter()
	router.Use(Middleware("foobar"))
	router.HandleFunc("/user/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		span := trace.SpanFromContext(r.Context())
		assert.Equal(t, sc, span.SpanContext())
		w.WriteHeader(http.StatusOK)
	}))

	router.ServeHTTP(w, r)
	assert.True(t, called, "failed to run test")
}

func TestPropagationWithCustomPropagators(t *testing.T) {
	prop := propagation.TraceContext{}

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	ctx := trace.ContextWithRemoteSpanContext(context.Background(), sc)
	prop.Inject(ctx, propagation.HeaderCarrier(r.Header))

	var called bool
	router := mux.NewRouter()
	router.Use(Middleware("foobar", WithPropagators(prop)))
	router.HandleFunc("/user/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		span := trace.SpanFromContext(r.Context())
		assert.Equal(t, sc, span.SpanContext())
		w.WriteHeader(http.StatusOK)
	}))

	router.ServeHTTP(w, r)
	assert.True(t, called, "failed to run test")
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

// implement Hijacker.
func (rw *testResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, nil
}

// implement Pusher.
func (rw *testResponseWriter) Push(target string, opts *http.PushOptions) error {
	return nil
}

// implement Flusher.
func (rw *testResponseWriter) Flush() {
}

// implement io.ReaderFrom.
func (rw *testResponseWriter) ReadFrom(r io.Reader) (n int64, err error) {
	return 0, nil
}

func TestResponseWriterInterfaces(t *testing.T) {
	// make sure the recordingResponseWriter preserves interfaces implemented by the wrapped writer
	router := mux.NewRouter()
	router.Use(Middleware("foobar"))
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

func TestFilter(t *testing.T) {
	prop := propagation.TraceContext{}

	router := mux.NewRouter()
	var calledHealth, calledTest int
	router.Use(Middleware("foobar", WithFilter(func(r *http.Request) bool {
		return r.URL.Path != "/health"
	})))
	router.HandleFunc("/health", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calledHealth++
		span := trace.SpanFromContext(r.Context())
		assert.NotEqual(t, sc, span.SpanContext())
		w.WriteHeader(http.StatusOK)
	}))
	router.HandleFunc("/test", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calledTest++
		span := trace.SpanFromContext(r.Context())
		assert.Equal(t, sc, span.SpanContext())
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/health", nil)
	ctx := trace.ContextWithRemoteSpanContext(context.Background(), sc)
	prop.Inject(ctx, propagation.HeaderCarrier(r.Header))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	r = httptest.NewRequest("GET", "/test", nil)
	ctx = trace.ContextWithRemoteSpanContext(context.Background(), sc)
	prop.Inject(ctx, propagation.HeaderCarrier(r.Header))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)

	assert.Equal(t, 1, calledHealth, "failed to run test")
	assert.Equal(t, 1, calledTest, "failed to run test")
}

func TestPassthroughSpanFromGlobalTracerWithBody(t *testing.T) {
	expectedBody := `{"message":"successfully"}`
	router := mux.NewRouter()
	router.Use(Middleware("foobar"))

	var called bool
	router.HandleFunc("/user", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		got := trace.SpanFromContext(r.Context()).SpanContext()
		assert.Equal(t, sc, got)

		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		defer r.Body.Close()

		assert.JSONEq(t, `{"name":"John Doe","age":30}`, string(body), "request body does not match")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, err = w.Write([]byte(expectedBody))
		assert.NoError(t, err)
	})).Methods(http.MethodPost)

	r := httptest.NewRequest(http.MethodPost, "/user", strings.NewReader(`{"name":"John Doe","age":30}`))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(trace.ContextWithRemoteSpanContext(context.Background(), sc))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	// Validate the assertions
	assert.True(t, called, "failed to run test")
	assert.Equal(t, http.StatusCreated, w.Code, "unexpected status code")
	assert.JSONEq(t, expectedBody, w.Body.String(), "unexpected response body")
}

func TestHeaderAlreadyWrittenWhenFlushing(t *testing.T) {
	var called bool

	router := mux.NewRouter()
	router.Use(Middleware("foobar"))

	router.HandleFunc("/user/{id}", func(w http.ResponseWriter, r *http.Request) {
		called = true

		w.WriteHeader(http.StatusBadRequest)
		f := w.(http.Flusher)
		f.Flush()
	})

	r := httptest.NewRequest("GET", "/user/123", nil)
	r = r.WithContext(trace.ContextWithRemoteSpanContext(context.Background(), sc))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	// Assertions
	assert.True(t, called, "failed to run test")
	assert.Equal(t, http.StatusBadRequest, w.Code, "Header was not set before flushing")
}
