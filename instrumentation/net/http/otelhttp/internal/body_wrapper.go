// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp/internal"

import (
	"io"
	"sync/atomic"
)

var _ io.ReadCloser = &BodyWrapper{}

// BodyWrapper wraps a http.Request.Body (an io.ReadCloser) to track the number
// of bytes read and the last error.
type BodyWrapper struct {
	io.ReadCloser
	OnRead func(n int64) // must not be nil

	read atomic.Int64
	err  atomic.Value
}

// NewBodyWrapper creates a new BodyWrapper.
func NewBodyWrapper(rc io.ReadCloser, onRead func(int64)) *BodyWrapper {
	return &BodyWrapper{
		ReadCloser: rc,
		OnRead:     onRead,
	}
}

// Read reads the data from the io.ReadCloser, and stores the number of bytes
// read and the error.
func (w *BodyWrapper) Read(b []byte) (int, error) {
	n, err := w.ReadCloser.Read(b)
	n1 := int64(n)
	w.read.Add(n1)
	if err != nil {
		w.err.Store(err)
	}
	w.OnRead(n1)
	return n, err
}

// Closes closes the io.ReadCloser.
func (w *BodyWrapper) Close() error {
	return w.ReadCloser.Close()
}

// BytesRead returns the number of bytes read up to this point.
func (w *BodyWrapper) BytesRead() int64 {
	return w.read.Load()
}

// Error returns the last error.
func (w *BodyWrapper) Error() error {
	err, ok := w.err.Load().(error)
	if !ok {
		return nil
	}
	return err
}
