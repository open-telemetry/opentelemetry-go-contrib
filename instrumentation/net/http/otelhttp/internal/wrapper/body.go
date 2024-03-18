// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package wrapper // import "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp/internal/wrapper"

import (
	"io"
	"sync/atomic"
)

var _ io.ReadCloser = &Body{}

// Body wraps an http.Request.Body (an io.ReadCloser) to track the
// number of bytes read and the last error.
type Body struct {
	io.ReadCloser
	OnRecordFn func(n int64) // must not be nil

	read atomic.Int64
	Err  error
}

func (w *Body) Read(b []byte) (int, error) {
	n, err := w.ReadCloser.Read(b)
	n1 := int64(n)
	w.read.Add(n1)
	w.Err = err
	w.OnRecordFn(n1)
	return n, err
}

func (w *Body) ReadLength() int64 {
	return w.read.Load()
}

func (w *Body) Close() error {
	return w.ReadCloser.Close()
}
