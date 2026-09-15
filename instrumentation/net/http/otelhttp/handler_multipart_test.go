// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelhttp

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestHandlerMultipartTempFileCleanup verifies that a disk-backed multipart
// upload parsed downstream is cleaned up by net/http after the request
// completes. The middleware passes a context-derived request copy downstream,
// so the parsed MultipartForm must be copied back onto the original request
// for net/http's finishRequest to find and remove its temp files.
func TestHandlerMultipartTempFileCleanup(t *testing.T) {
	pathCh := make(chan string, 1)
	handler := NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// maxMemory of 0 forces the file part to spill to a disk-backed
		// temp file instead of staying in memory.
		require.NoError(t, r.ParseMultipartForm(0))
		f, err := r.MultipartForm.File["file"][0].Open()
		require.NoError(t, err)
		name := ""
		if of, ok := f.(*os.File); ok {
			name = of.Name()
		}
		require.NoError(t, f.Close())
		require.NotEmpty(t, name, "file part should be disk-backed given maxMemory of 0")
		pathCh <- name
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

	var tmpPath string
	select {
	case tmpPath = <-pathCh:
	case <-time.After(5 * time.Second):
		t.Fatal("handler never reported temp file path")
	}

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
