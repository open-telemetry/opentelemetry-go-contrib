// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/macaron.v1"

	"go.opentelemetry.io/contrib/instrumentation/gopkg.in/macaron.v1/otelmacaron" // nolint:staticcheck  // deprecated.
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestChildSpanFromGlobalTracer(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := trace.NewTracerProvider(trace.WithSpanProcessor(sr))
	otel.SetTracerProvider(tp)

	m := macaron.Classic()
	m.Use(otelmacaron.Middleware("foobar"))
	m.Get("/user/:id", func(ctx *macaron.Context) {
		ctx.Resp.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	m.ServeHTTP(w, r)

	assert.Len(t, sr.Ended(), 1)
}

func TestChildSpanNames(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := trace.NewTracerProvider(trace.WithSpanProcessor(sr))

	m := macaron.Classic()
	m.Use(otelmacaron.Middleware("foobar", otelmacaron.WithTracerProvider(tp)))
	m.Get("/user/:id", func(ctx *macaron.Context) {
		ctx.Resp.WriteHeader(http.StatusOK)
	})
	m.Get("/book/:title", func(ctx *macaron.Context) {
		_, err := ctx.Resp.Write(([]byte)("ok"))
		if err != nil {
			t.Error(err)
		}
	})

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)

	r = httptest.NewRequest("GET", "/book/foo", nil)
	w = httptest.NewRecorder()
	m.ServeHTTP(w, r)

	spans := sr.Ended()
	require.Len(t, spans, 2)
	span := spans[0]
	assert.Equal(t, "/user/123", span.Name()) // TODO: span name should show router template, eg /user/:id
	assert.Equal(t, oteltrace.SpanKindServer, span.SpanKind())
	attrs := span.Attributes()
	assert.Contains(t, attrs, attribute.String("net.host.name", "foobar"))
	assert.Contains(t, attrs, attribute.Int("http.status_code", http.StatusOK))
	assert.Contains(t, attrs, attribute.String("http.method", "GET"))

	span = spans[1]
	assert.Equal(t, "/book/foo", span.Name()) // TODO: span name should show router template, eg /book/:title
	assert.Equal(t, oteltrace.SpanKindServer, span.SpanKind())
	attrs = span.Attributes()
	assert.Contains(t, attrs, attribute.String("net.host.name", "foobar"))
	assert.Contains(t, attrs, attribute.Int("http.status_code", http.StatusOK))
	assert.Contains(t, attrs, attribute.String("http.method", "GET"))
}

func TestSpanStatus(t *testing.T) {
	testCases := []struct {
		httpStatusCode int
		wantSpanStatus codes.Code
	}{
		{http.StatusOK, codes.Unset},
		{http.StatusBadRequest, codes.Unset},
		{http.StatusInternalServerError, codes.Error},
	}
	for _, tc := range testCases {
		t.Run(strconv.Itoa(tc.httpStatusCode), func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			provider := trace.NewTracerProvider()
			provider.RegisterSpanProcessor(sr)
			m := macaron.Classic()
			m.Use(otelmacaron.Middleware("foobar", otelmacaron.WithTracerProvider(provider)))
			m.Get("/", func(ctx *macaron.Context) {
				ctx.Resp.WriteHeader(tc.httpStatusCode)
			})

			m.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))

			require.Len(t, sr.Ended(), 1, "should emit a span")
			assert.Equal(t, tc.wantSpanStatus, sr.Ended()[0].Status().Code, "should only set Error status for HTTP statuses >= 500")
		})
	}
}
