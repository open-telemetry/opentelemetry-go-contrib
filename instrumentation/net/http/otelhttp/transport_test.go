// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package otelhttp

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TestTransportFormatter(t *testing.T) {
	var httpMethods = []struct {
		name     string
		method   string
		expected string
	}{
		{
			"GET method",
			http.MethodGet,
			"HTTP GET",
		},
		{
			"HEAD method",
			http.MethodHead,
			"HTTP HEAD",
		},
		{
			"POST method",
			http.MethodPost,
			"HTTP POST",
		},
		{
			"PUT method",
			http.MethodPut,
			"HTTP PUT",
		},
		{
			"PATCH method",
			http.MethodPatch,
			"HTTP PATCH",
		},
		{
			"DELETE method",
			http.MethodDelete,
			"HTTP DELETE",
		},
		{
			"CONNECT method",
			http.MethodConnect,
			"HTTP CONNECT",
		},
		{
			"OPTIONS method",
			http.MethodOptions,
			"HTTP OPTIONS",
		},
		{
			"TRACE method",
			http.MethodTrace,
			"HTTP TRACE",
		},
	}

	for _, tc := range httpMethods {
		t.Run(tc.name, func(t *testing.T) {
			r, err := http.NewRequest(tc.method, "http://localhost/", nil)
			if err != nil {
				t.Fatal(err)
			}
			formattedName := "HTTP " + r.Method

			if formattedName != tc.expected {
				t.Fatalf("unexpected name: got %s, expected %s", formattedName, tc.expected)
			}
		})
	}

}

func TestTransportBasics(t *testing.T) {
	prop := propagation.TraceContext{}
	content := []byte("Hello, world!")

	ctx := context.Background()
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{0x01},
		SpanID:  trace.SpanID{0x01},
	})
	ctx = trace.ContextWithRemoteSpanContext(ctx, sc)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := prop.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		span := trace.SpanContextFromContext(ctx)
		if span.SpanID() != sc.SpanID() {
			t.Fatalf("testing remote SpanID: got %s, expected %s", span.SpanID(), sc.SpanID())
		}
		if _, err := w.Write(content); err != nil {
			t.Fatal(err)
		}
	}))
	defer ts.Close()

	r, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	tr := NewTransport(http.DefaultTransport, WithPropagators(prop))

	c := http.Client{Transport: tr}
	res, err := c.Do(r)
	if err != nil {
		t.Fatal(err)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(body, content) {
		t.Fatalf("unexpected content: got %s, expected %s", body, content)
	}
}

func TestNilTransport(t *testing.T) {
	prop := propagation.TraceContext{}
	content := []byte("Hello, world!")

	ctx := context.Background()
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{0x01},
		SpanID:  trace.SpanID{0x01},
	})
	ctx = trace.ContextWithRemoteSpanContext(ctx, sc)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := prop.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		span := trace.SpanContextFromContext(ctx)
		if span.SpanID() != sc.SpanID() {
			t.Fatalf("testing remote SpanID: got %s, expected %s", span.SpanID(), sc.SpanID())
		}
		if _, err := w.Write(content); err != nil {
			t.Fatal(err)
		}
	}))
	defer ts.Close()

	r, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	tr := NewTransport(nil, WithPropagators(prop))

	c := http.Client{Transport: tr}
	res, err := c.Do(r)
	if err != nil {
		t.Fatal(err)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(body, content) {
		t.Fatalf("unexpected content: got %s, expected %s", body, content)
	}
}

const readSize = 42

type readCloser struct {
	readErr, closeErr error
}

func (rc readCloser) Read(p []byte) (n int, err error) {
	return readSize, rc.readErr
}
func (rc readCloser) Close() error {
	return rc.closeErr
}

type span struct {
	trace.Span

	ended       bool
	recordedErr error

	statusCode codes.Code
	statusDesc string
}

func (s *span) End(...trace.SpanEndOption) {
	s.ended = true
}

func (s *span) RecordError(err error, _ ...trace.EventOption) {
	s.recordedErr = err
}

func (s *span) SetStatus(c codes.Code, d string) {
	s.statusCode, s.statusDesc = c, d
}

func (s *span) assert(t *testing.T, ended bool, err error, c codes.Code, d string) {
	if ended {
		assert.True(t, s.ended, "not ended")
	} else {
		assert.False(t, s.ended, "ended")
	}

	if err == nil {
		assert.NoError(t, s.recordedErr, "recorded an error")
	} else {
		assert.Equal(t, err, s.recordedErr)
	}

	assert.Equal(t, c, s.statusCode, "status codes not equal")
	assert.Equal(t, d, s.statusDesc, "status description not equal")
}

func TestWrappedBodyRead(t *testing.T) {
	s := new(span)
	wb := &wrappedBody{span: trace.Span(s), body: readCloser{}}
	n, err := wb.Read([]byte{})
	assert.Equal(t, readSize, n, "wrappedBody returned wrong bytes")
	assert.NoError(t, err)
	s.assert(t, false, nil, codes.Unset, "")
}

func TestWrappedBodyReadEOFError(t *testing.T) {
	s := new(span)
	wb := &wrappedBody{span: trace.Span(s), body: readCloser{readErr: io.EOF}}
	n, err := wb.Read([]byte{})
	assert.Equal(t, readSize, n, "wrappedBody returned wrong bytes")
	assert.Equal(t, io.EOF, err)
	s.assert(t, true, nil, codes.Unset, "")
}

func TestWrappedBodyReadError(t *testing.T) {
	s := new(span)
	expectedErr := errors.New("test")
	wb := &wrappedBody{span: trace.Span(s), body: readCloser{readErr: expectedErr}}
	n, err := wb.Read([]byte{})
	assert.Equal(t, readSize, n, "wrappedBody returned wrong bytes")
	assert.Equal(t, expectedErr, err)
	s.assert(t, false, expectedErr, codes.Error, expectedErr.Error())
}

func TestWrappedBodyClose(t *testing.T) {
	s := new(span)
	wb := &wrappedBody{span: trace.Span(s), body: readCloser{}}
	assert.NoError(t, wb.Close())
	s.assert(t, true, nil, codes.Unset, "")
}

func TestWrappedBodyCloseError(t *testing.T) {
	s := new(span)
	expectedErr := errors.New("test")
	wb := &wrappedBody{span: trace.Span(s), body: readCloser{closeErr: expectedErr}}
	assert.Equal(t, expectedErr, wb.Close())
	s.assert(t, true, nil, codes.Unset, "")
}
