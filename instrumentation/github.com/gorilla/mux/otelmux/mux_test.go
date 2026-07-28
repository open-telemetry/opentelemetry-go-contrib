// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelmux

import (
	"bufio"
	"bytes"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux/internal/request"
)

var sc = trace.NewSpanContext(trace.SpanContextConfig{
	TraceID:    [16]byte{1},
	SpanID:     [8]byte{1},
	Remote:     true,
	TraceFlags: trace.FlagsSampled,
})

func TestRequestBodyWrapper(t *testing.T) {
	router := mux.NewRouter()
	router.Use(Middleware("foobar"))
	router.HandleFunc("/user/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := r.Body.(*request.BodyWrapper)
		assert.Truef(t, ok, "body should be wrapped when request is processed")

		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequestWithContext(t.Context(), "POST", "/user/123", strings.NewReader(`{"name":"John Doe","age":30}`))
	r = r.WithContext(trace.ContextWithRemoteSpanContext(t.Context(), sc))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	_, ok := r.Body.(*request.BodyWrapper)
	assert.Falsef(t, ok, "body should not be wrapped after request is processed")
}

// TestMultipartFormCopiedToOriginalRequest is a regression test for
// https://github.com/open-telemetry/opentelemetry-go-contrib/issues/9070.
// Middleware passes a context-derived copy of the request to the handler;
// if ParseMultipartForm is called on that copy, the resulting MultipartForm
// (and its temp files) must be copied back onto the request the middleware
// received so net/http's finishRequest can clean them up.
//
// The middleware is applied directly, rather than through a mux.Router, so
// that the request the middleware receives is the same one net/http would
// hand to ServeHTTP. gorilla/mux's own router makes an additional,
// unrelated context copy of its own when matching routes, which is outside
// otelmux's control.
func TestMultipartFormCopiedToOriginalRequest(t *testing.T) {
	handler := Middleware("foobar")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// maxMemory of 0 forces the file part to spill to a disk-backed temp
		// file instead of staying in memory, matching the leak this
		// regression test guards against.
		assert.NoError(t, r.ParseMultipartForm(0))
		w.WriteHeader(http.StatusOK)
	}))

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "test.txt")
	require.NoError(t, err)
	_, err = part.Write([]byte("hello"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/upload", &body)
	r.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, r.MultipartForm, "MultipartForm should be copied back to the original request so net/http can clean up its temp files")
	t.Cleanup(func() {
		require.NoError(t, r.MultipartForm.RemoveAll())
	})
	require.Len(t, r.MultipartForm.File["file"], 1)

	f, err := r.MultipartForm.File["file"][0].Open()
	require.NoError(t, err)
	defer f.Close()
	_, diskBacked := f.(*os.File)
	assert.True(t, diskBacked, "file part should be disk-backed given maxMemory of 0, otherwise this test doesn't exercise the temp-file leak")
}

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

	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/user/123", http.NoBody)
	r = r.WithContext(trace.ContextWithRemoteSpanContext(t.Context(), sc))
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

	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/user/123", http.NoBody)
	w := httptest.NewRecorder()

	ctx := trace.ContextWithRemoteSpanContext(t.Context(), sc)
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

	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/user/123", http.NoBody)
	w := httptest.NewRecorder()

	ctx := trace.ContextWithRemoteSpanContext(t.Context(), sc)
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
func (*testResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, nil
}

// implement Pusher.
func (*testResponseWriter) Push(string, *http.PushOptions) error {
	return nil
}

// implement Flusher.
func (*testResponseWriter) Flush() {
}

// implement io.ReaderFrom.
func (*testResponseWriter) ReadFrom(io.Reader) (int64, error) {
	return 0, nil
}

func TestResponseWriterInterfaces(t *testing.T) {
	// make sure the recordingResponseWriter preserves interfaces implemented by the wrapped writer
	router := mux.NewRouter()
	router.Use(Middleware("foobar"))
	router.HandleFunc("/user/{id}", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		assert.Implements(t, (*http.Hijacker)(nil), w)
		assert.Implements(t, (*http.Pusher)(nil), w)
		assert.Implements(t, (*http.Flusher)(nil), w)
		assert.Implements(t, (*io.ReaderFrom)(nil), w)
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/user/123", http.NoBody)
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

	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/health", http.NoBody)
	ctx := trace.ContextWithRemoteSpanContext(t.Context(), sc)
	prop.Inject(ctx, propagation.HeaderCarrier(r.Header))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	r = httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", http.NoBody)
	ctx = trace.ContextWithRemoteSpanContext(t.Context(), sc)
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

	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/user", strings.NewReader(`{"name":"John Doe","age":30}`))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(trace.ContextWithRemoteSpanContext(t.Context(), sc))
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

	router.HandleFunc("/user/{id}", func(w http.ResponseWriter, _ *http.Request) {
		called = true

		w.WriteHeader(http.StatusBadRequest)
		f := w.(http.Flusher)
		f.Flush()
	})

	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/user/123", http.NoBody)
	r = r.WithContext(trace.ContextWithRemoteSpanContext(t.Context(), sc))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	// Assertions
	assert.True(t, called, "failed to run test")
	assert.Equal(t, http.StatusBadRequest, w.Code, "Header was not set before flushing")
}
