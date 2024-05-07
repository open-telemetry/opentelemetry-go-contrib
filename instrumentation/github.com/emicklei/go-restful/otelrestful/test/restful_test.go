// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/emicklei/go-restful/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/contrib/instrumentation/github.com/emicklei/go-restful/otelrestful"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestChildSpanFromGlobalTracer(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr)))

	handlerFunc := func(req *restful.Request, resp *restful.Response) {
		resp.WriteHeader(http.StatusOK)
	}
	ws := &restful.WebService{}
	ws.Route(ws.GET("/user/{id}").To(handlerFunc).
		Returns(http.StatusOK, "OK", nil).
		Returns(http.StatusNotFound, "Not Found", nil))
	container := restful.NewContainer()
	container.Filter(otelrestful.OTelFilter("my-service"))
	container.Add(ws)

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	container.ServeHTTP(w, r)
	assert.Len(t, sr.Ended(), 1)
}

func TestChildSpanFromCustomTracer(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	handlerFunc := func(req *restful.Request, resp *restful.Response) {
		resp.WriteHeader(http.StatusOK)
	}
	ws := &restful.WebService{}
	ws.Route(ws.GET("/user/{id}").To(handlerFunc))

	container := restful.NewContainer()
	container.Filter(otelrestful.OTelFilter("my-service", otelrestful.WithTracerProvider(provider)))
	container.Add(ws)

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	container.ServeHTTP(w, r)
	assert.Len(t, sr.Ended(), 1)
}

func TestChildSpanNames(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	handlerFunc := func(req *restful.Request, resp *restful.Response) {
		resp.WriteHeader(http.StatusOK)
	}
	ws := &restful.WebService{}
	ws.Route(ws.GET("/user/{id:[0-9]+}").To(handlerFunc))

	container := restful.NewContainer()
	container.Filter(otelrestful.OTelFilter("foobar", otelrestful.WithTracerProvider(provider)))
	container.Add(ws)

	ws.Route(ws.GET("/book/{title}").To(func(req *restful.Request, resp *restful.Response) {
		_, _ = resp.Write(([]byte)("ok"))
	}))

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	container.ServeHTTP(w, r)
	spans := sr.Ended()
	require.Len(t, spans, 1)
	assertSpan(
		t,
		spans[0],
		"/user/{id:[0-9]+}",
		attribute.String("net.host.name", "foobar"),
		attribute.Int("http.status_code", http.StatusOK),
		attribute.String("http.method", "GET"),
		attribute.String("http.route", "/user/{id:[0-9]+}"),
	)

	r = httptest.NewRequest("GET", "/book/foo", nil)
	w = httptest.NewRecorder()
	container.ServeHTTP(w, r)
	spans = sr.Ended()
	require.Len(t, spans, 2)
	assertSpan(
		t,
		spans[1],
		"/book/{title}",
		attribute.String("net.host.name", "foobar"),
		attribute.Int("http.status_code", http.StatusOK),
		attribute.String("http.method", "GET"),
		attribute.String("http.route", "/book/{title}"),
	)
}

func TestMultiFilters(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	retOK := func(req *restful.Request, resp *restful.Response) { resp.WriteHeader(http.StatusOK) }
	ws1 := &restful.WebService{}
	ws1.Path("/user")
	ws1.Route(ws1.GET("/{id}").
		Filter(otelrestful.OTelFilter("my-service", otelrestful.WithTracerProvider(provider))).
		To(retOK))
	ws1.Route(ws1.GET("/{id}/books").
		Filter(otelrestful.OTelFilter("book-service", otelrestful.WithTracerProvider(provider))).
		To(retOK))

	ws2 := &restful.WebService{}
	ws2.Path("/library")
	ws2.Filter(otelrestful.OTelFilter("library-service", otelrestful.WithTracerProvider(provider)))
	ws2.Route(ws2.GET("/{name}").To(retOK))

	container := restful.NewContainer()
	container.Add(ws1)
	container.Add(ws2)

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, r)
	spans := sr.Ended()
	require.Len(t, spans, 1)
	assertSpan(t, spans[0], "/user/{id}")

	r = httptest.NewRequest("GET", "/user/123/books", nil)
	w = httptest.NewRecorder()
	container.ServeHTTP(w, r)
	spans = sr.Ended()
	require.Len(t, spans, 2)
	assertSpan(t, spans[1], "/user/{id}/books")

	r = httptest.NewRequest("GET", "/library/metropolitan", nil)
	w = httptest.NewRecorder()
	container.ServeHTTP(w, r)
	spans = sr.Ended()
	require.Len(t, spans, 3)
	assertSpan(t, spans[2], "/library/{name}")
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
			provider := sdktrace.NewTracerProvider()
			provider.RegisterSpanProcessor(sr)
			handlerFunc := func(req *restful.Request, resp *restful.Response) {
				resp.WriteHeader(tc.httpStatusCode)
			}
			ws := &restful.WebService{}
			ws.Route(ws.GET("/").To(handlerFunc))
			container := restful.NewContainer()
			container.Filter(otelrestful.OTelFilter("my-service", otelrestful.WithTracerProvider(provider)))
			container.Add(ws)

			container.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))

			require.Len(t, sr.Ended(), 1, "should emit a span")
			assert.Equal(t, sr.Ended()[0].Status().Code, tc.wantSpanStatus, "should only set Error status for HTTP statuses >= 500")
		})
	}
}

func assertSpan(t *testing.T, span sdktrace.ReadOnlySpan, name string, attrs ...attribute.KeyValue) {
	assert.Equal(t, name, span.Name())
	assert.Equal(t, oteltrace.SpanKindServer, span.SpanKind())

	gotA := span.Attributes()
	for _, a := range attrs {
		assert.Contains(t, gotA, a)
	}
}
