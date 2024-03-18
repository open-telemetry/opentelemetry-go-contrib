// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package wrapper // import "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp/internal/wrapper"

import (
	"net/http"
)

var _ http.ResponseWriter = &ResponseWriter{}

// ResponseWriter wraps a http.ResponseWriter in order to track the number of
// bytes written, the last error, and to catch the first written statusCode.
// TODO: The wrapped http.ResponseWriter doesn't implement any of the optional
// types (http.Hijacker, http.Pusher, http.CloseNotifier, http.Flusher, etc)
// that may be useful when using it in real life situations.
type ResponseWriter struct {
	http.ResponseWriter
	OnRecordFn func(n int64) // must not be nil

	Written     int64
	StatusCode  int
	Err         error
	WroteHeader bool
}

func (w *ResponseWriter) Write(p []byte) (int, error) {
	if !w.WroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(p)
	n1 := int64(n)
	w.OnRecordFn(n1)
	w.Written += n1
	w.Err = err
	return n, err
}

// WriteHeader persists initial statusCode for span attribution.
// All calls to WriteHeader will be propagated to the underlying ResponseWriter
// and will persist the statusCode from the first call.
// Blocking consecutive calls to WriteHeader alters expected behavior and will
// remove warning logs from net/http where developers will notice incorrect handler implementations.
func (w *ResponseWriter) WriteHeader(statusCode int) {
	if !w.WroteHeader {
		w.WroteHeader = true
		w.StatusCode = statusCode
	}
	w.ResponseWriter.WriteHeader(statusCode)
}
