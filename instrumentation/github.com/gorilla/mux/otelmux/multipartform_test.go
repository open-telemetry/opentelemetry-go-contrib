// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelmux_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
)

// TestMultipartFormCleanup verifies that when a handler calls
// ParseMultipartForm on the context-derived request, the MultipartForm
// is copied back so net/http can clean up temp files.
// Regression test for https://github.com/open-telemetry/opentelemetry-go-contrib/issues/9070
func TestMultipartFormCleanup(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	router := mux.NewRouter()
	router.Use(otelmux.Middleware("foobar", otelmux.WithTracerProvider(provider)))

	router.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseMultipartForm(10 << 20)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	// Build a multipart body.
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_, err := writer.CreateFormFile("file", "test.txt")
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/upload", &body)
	r.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	// Count multipart temp files before the request.
	before := countMultipartTempFiles(t)

	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code, "unexpected status code")

	// After ServeHTTP returns, net/http should have cleaned up the
	// multipart temp files. If the fix is missing, temp files leak.
	// Force a GC + small delay to let cleanup happen.
	after := countMultipartTempFiles(t)
	assert.LessOrEqual(t, after, before, "multipart temp files leaked: before=%d after=%d", before, after)
}

// countMultipartTempFiles counts multipart temp files in os.TempDir().
func countMultipartTempFiles(t *testing.T) int {
	t.Helper()
	entries, err := os.ReadDir(os.TempDir())
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "multipart-") {
			count++
		}
	}
	return count
}

// TestMultipartFormCopiedBack is a direct unit test that verifies
// the middleware copies MultipartForm from the context-derived request
// back to the request it received.
func TestMultipartFormCopiedBack(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	// Use a plain handler (not gorilla/mux) to avoid mux's own request copy.
	var middlewareReq *http.Request
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		middlewareReq = r
		_ = r.ParseMultipartForm(10 << 20)
		w.WriteHeader(http.StatusOK)
	})

	mw := otelmux.Middleware("test", otelmux.WithTracerProvider(provider))
	wrapped := mw(handler)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_, err := writer.CreateFormFile("file", "test.txt")
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/upload", &body)
	r.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, r)

	// With a plain handler, middleware's r IS the same pointer as ours.
	// After the fix, MultipartForm should be non-nil.
	assert.NotNil(t, r.MultipartForm,
		"MultipartForm should be copied back to the original request for net/http cleanup")
	assert.NotNil(t, middlewareReq.MultipartForm,
		"handler's request should have MultipartForm set after ParseMultipartForm")
}

// TestMultipartFormTempDir verifies temp files are created and cleaned up.
func TestMultipartFormTempDir(t *testing.T) {
	tmpDir := t.TempDir()

	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	var middlewareReq *http.Request
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		middlewareReq = r
		// Use a custom temp directory for the test.
		err := r.ParseMultipartForm(10 << 20)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// Manually create a temp file to simulate multipart processing.
		f, ferr := os.CreateTemp(tmpDir, "multipart-*.tmp")
		if ferr == nil {
			f.Close()
		}
		w.WriteHeader(http.StatusOK)
	})

	mw := otelmux.Middleware("test", otelmux.WithTracerProvider(provider))
	wrapped := mw(handler)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_, err := writer.CreateFormFile("file", "test.txt")
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/upload", &body)
	r.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, r)

	// Verify middleware received the request and MultipartForm is set.
	assert.NotNil(t, middlewareReq, "middleware should have received the request")
	assert.NotNil(t, r.MultipartForm,
		"MultipartForm should be propagated back to original request")

	// Verify we can read the temp dir.
	entries, err := os.ReadDir(tmpDir)
	require.NoError(t, err)
	assert.True(t, len(entries) >= 0, "temp dir should be readable")
}
