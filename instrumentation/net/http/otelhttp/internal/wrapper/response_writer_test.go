// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package wrapper

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResponseWriterWrite(t *testing.T) {
	rw := &ResponseWriter{
		ResponseWriter: &httptest.ResponseRecorder{},
		OnRecordFn:     func(int64) {},
	}

	_, err := rw.Write([]byte("hello world"))
	assert.NoError(t, err)
	assert.Equal(t, int64(11), rw.Written)
	assert.Equal(t, http.StatusOK, rw.StatusCode)
	assert.Nil(t, rw.Err)
	assert.True(t, rw.WroteHeader)
}

func TestResponseWriterOnRecordFn(t *testing.T) {
	var sizeFromFn int64

	rw := &ResponseWriter{
		ResponseWriter: &httptest.ResponseRecorder{},
		OnRecordFn: func(n int64) {
			sizeFromFn += n
		},
	}

	_, err := rw.Write([]byte("hello world"))
	assert.NoError(t, err)
	assert.Equal(t, int64(11), sizeFromFn)
}

func TestResponseWriterWriteHeader(t *testing.T) {
	rw := &ResponseWriter{
		ResponseWriter: &httptest.ResponseRecorder{},
		OnRecordFn:     func(int64) {},
	}

	rw.WriteHeader(http.StatusTeapot)
	assert.Equal(t, http.StatusTeapot, rw.StatusCode)
	assert.True(t, rw.WroteHeader)

	rw.WriteHeader(http.StatusGone)
	assert.Equal(t, http.StatusTeapot, rw.StatusCode)
}
