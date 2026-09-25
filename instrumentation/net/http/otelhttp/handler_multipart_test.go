// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelhttp

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// multipartParseResult carries the outcome of parsing a multipart request
// inside an HTTP handler back to the test goroutine. Handlers must not call
// require directly because they run outside the test goroutine.
type multipartParseResult struct {
	tmpPath string
	err     error
}

// multipartDiskBackedTmpPath parses the request with maxMemory of 0 so the
// file part spills to a disk-backed temp file, and returns that file's path.
func multipartDiskBackedTmpPath(r *http.Request) (string, error) {
	if err := r.ParseMultipartForm(0); err != nil {
		return "", err
	}
	files := r.MultipartForm.File["file"]
	if len(files) == 0 {
		return "", errors.New("missing file part")
	}
	f, err := files[0].Open()
	if err != nil {
		return "", err
	}
	name := ""
	if of, ok := f.(*os.File); ok {
		name = of.Name()
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	if name == "" {
		return "", errors.New("file part should be disk-backed given maxMemory of 0")
	}
	return name, nil
}

// TestHandlerMultipartTempFileCleanup verifies that a disk-backed multipart
// upload parsed downstream is cleaned up by net/http after the request
// completes. The middleware passes a context-derived request copy downstream,
// so the parsed MultipartForm must be copied back onto the original request
// for net/http's finishRequest to find and remove its temp files.
func TestHandlerMultipartTempFileCleanup(t *testing.T) {
	resultCh := make(chan multipartParseResult, 1)
	handler := NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, err := multipartDiskBackedTmpPath(r)
		resultCh <- multipartParseResult{tmpPath: path, err: err}
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}), "test_handler")

	srv := httptest.NewServer(handler)
	defer srv.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "test.txt")
	require.NoError(t, err)
	_, err = part.Write([]byte("hello"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, srv.URL+"/upload", &body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := srv.Client().Do(req)
	require.NoError(t, err)
	_, _ = io.Copy(io.Discard, resp.Body)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result multipartParseResult
	select {
	case result = <-resultCh:
	case <-time.After(5 * time.Second):
		t.Fatal("handler never reported temp file path")
	}
	require.NoError(t, result.err)
	require.NotEmpty(t, result.tmpPath)
	tmpPath := result.tmpPath

	// finishRequest runs on the server before the response completes, but
	// allow scheduling slack.
	deadline := time.Now().Add(2 * time.Second)
	for {
		_, statErr := os.Stat(tmpPath)
		if os.IsNotExist(statErr) {
			break
		}
		require.NoError(t, statErr)
		if time.Now().After(deadline) {
			_ = os.Remove(tmpPath)
			t.Fatalf("multipart temp file was not cleaned up: %s", tmpPath)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TestHandlerMultipartTempFileCleanupOnPanic verifies that the deferred
// MultipartForm copy-back preserves net/http temp file cleanup when the
// downstream handler panics. An outer recovery middleware recovers the panic
// so the server still runs finishRequest; without the deferred copy-back the
// original request would have a nil MultipartForm and the temp file would
// leak.
func TestHandlerMultipartTempFileCleanupOnPanic(t *testing.T) {
	resultCh := make(chan multipartParseResult, 1)
	inner := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		path, err := multipartDiskBackedTmpPath(r)
		resultCh <- multipartParseResult{tmpPath: path, err: err}
		panic("test panic after multipart parse")
	})
	otelHandler := NewHandler(inner, "test_handler")
	recovering := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()
		otelHandler.ServeHTTP(w, r)
	})

	srv := httptest.NewServer(recovering)
	defer srv.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "test.txt")
	require.NoError(t, err)
	_, err = part.Write([]byte("hello"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, srv.URL+"/upload", &body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := srv.Client().Do(req)
	require.NoError(t, err)
	_, _ = io.Copy(io.Discard, resp.Body)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var result multipartParseResult
	select {
	case result = <-resultCh:
	case <-time.After(5 * time.Second):
		t.Fatal("handler never reported temp file path")
	}
	require.NoError(t, result.err)
	require.NotEmpty(t, result.tmpPath)
	tmpPath := result.tmpPath

	// finishRequest runs on the server before the recovery response
	// completes, but allow scheduling slack.
	deadline := time.Now().Add(5 * time.Second)
	for {
		_, statErr := os.Stat(tmpPath)
		if os.IsNotExist(statErr) {
			break
		}
		require.NoError(t, statErr)
		if time.Now().After(deadline) {
			_ = os.Remove(tmpPath)
			t.Fatalf("multipart temp file was not cleaned up after panic: %s", tmpPath)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
