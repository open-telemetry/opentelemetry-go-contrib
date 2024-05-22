// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelhttp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRespWriterWriteHeader(t *testing.T) {
	rw := &respWriterWrapper{
		ResponseWriter: &httptest.ResponseRecorder{},
		record:         func(int64) {},
	}

	rw.WriteHeader(http.StatusTeapot)
	assert.Equal(t, http.StatusTeapot, rw.statusCode)
	assert.True(t, rw.wroteHeader)

	rw.WriteHeader(http.StatusGone)
	assert.Equal(t, http.StatusTeapot, rw.statusCode)
}

func TestRespWriterFlush(t *testing.T) {
	rw := &respWriterWrapper{
		ResponseWriter: &httptest.ResponseRecorder{},
		record:         func(int64) {},
	}

	rw.Flush()
	assert.Equal(t, http.StatusOK, rw.statusCode)
	assert.True(t, rw.wroteHeader)
}
