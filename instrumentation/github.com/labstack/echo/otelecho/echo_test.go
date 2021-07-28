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

// Based on https://github.com/DataDog/dd-trace-go/blob/8fb554ff7cf694267f9077ae35e27ce4689ed8b6/contrib/gin-gonic/gin/gintrace_test.go

package otelecho

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/oteltest"
	"go.opentelemetry.io/otel/propagation"

	b3prop "go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestChildSpanFromGlobalTracer(t *testing.T) {
	otel.SetTracerProvider(oteltest.NewTracerProvider())

	router := echo.New()
	router.Use(Middleware("foobar"))
	router.GET("/user/:id", func(c echo.Context) error {
		span := oteltrace.SpanFromContext(c.Request().Context())
		_, ok := span.(*oteltest.Span)
		assert.True(t, ok)
		return c.NoContent(200)
	})

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode, "should call the 'user' handler")
}

func TestChildSpanFromCustomTracer(t *testing.T) {
	provider := oteltest.NewTracerProvider()

	router := echo.New()
	router.Use(Middleware("foobar", WithTracerProvider(provider)))
	router.GET("/user/:id", func(c echo.Context) error {
		span := oteltrace.SpanFromContext(c.Request().Context())
		_, ok := span.(*oteltest.Span)
		assert.True(t, ok)
		return c.NoContent(200)
	})

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode, "should call the 'user' handler")
}

func TestTrace200(t *testing.T) {
	sr := new(oteltest.SpanRecorder)
	provider := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))

	router := echo.New()
	router.Use(Middleware("foobar", WithTracerProvider(provider)))
	router.GET("/user/:id", func(c echo.Context) error {
		span := oteltrace.SpanFromContext(c.Request().Context())
		mspan, ok := span.(*oteltest.Span)
		require.True(t, ok)
		assert.Equal(t, attribute.StringValue("foobar"), mspan.Attributes()["http.server_name"])
		id := c.Param("id")
		return c.String(200, id)
	})

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	// do and verify the request
	router.ServeHTTP(w, r)
	response := w.Result()
	require.Equal(t, http.StatusOK, response.StatusCode)

	// verify traces look good
	spans := sr.Completed()
	require.Len(t, spans, 1)
	span := spans[0]
	assert.Equal(t, "/user/:id", span.Name())
	assert.Equal(t, oteltrace.SpanKindServer, span.SpanKind())
	assert.Equal(t, attribute.StringValue("foobar"), span.Attributes()["http.server_name"])
	assert.Equal(t, attribute.IntValue(http.StatusOK), span.Attributes()["http.status_code"])
	assert.Equal(t, attribute.StringValue("GET"), span.Attributes()["http.method"])
	assert.Equal(t, attribute.StringValue("/user/123"), span.Attributes()["http.target"])
	assert.Equal(t, attribute.StringValue("/user/:id"), span.Attributes()["http.route"])
}

func TestError(t *testing.T) {
	sr := new(oteltest.SpanRecorder)
	provider := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))

	// setup
	router := echo.New()
	router.Use(Middleware("foobar", WithTracerProvider(provider)))
	wantErr := errors.New("oh no")
	// configure a handler that returns an error and 5xx status
	// code
	router.GET("/server_err", func(c echo.Context) error {
		return wantErr
	})
	r := httptest.NewRequest("GET", "/server_err", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	response := w.Result()
	assert.Equal(t, http.StatusInternalServerError, response.StatusCode)

	// verify the errors and status are correct
	spans := sr.Completed()
	require.Len(t, spans, 1)
	span := spans[0]
	assert.Equal(t, "/server_err", span.Name())
	assert.Equal(t, attribute.StringValue("foobar"), span.Attributes()["http.server_name"])
	assert.Equal(t, attribute.IntValue(http.StatusInternalServerError), span.Attributes()["http.status_code"])
	assert.Equal(t, attribute.StringValue("oh no"), span.Attributes()["echo.error"])
	// server errors set the status
	assert.Equal(t, codes.Error, span.StatusCode())
}

func TestErrorOnlyHandledOnce(t *testing.T) {
	router := echo.New()
	timesHandlingError := 0
	router.HTTPErrorHandler = func(e error, c echo.Context) {
		timesHandlingError++
	}
	router.Use(Middleware("test-service"))
	router.GET("/", func(c echo.Context) error {
		return errors.New("Mock Error")
	})
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	assert.Equal(t, 1, timesHandlingError)
}

func TestGetSpanNotInstrumented(t *testing.T) {
	router := echo.New()
	router.GET("/ping", func(c echo.Context) error {
		// Assert we don't have a span on the context.
		span := oteltrace.SpanFromContext(c.Request().Context())
		ok := !span.SpanContext().IsValid()
		assert.True(t, ok)
		return c.String(200, "ok")
	})
	r := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	response := w.Result()
	assert.Equal(t, http.StatusOK, response.StatusCode)
}

func TestPropagationWithGlobalPropagators(t *testing.T) {
	sr := new(oteltest.SpanRecorder)
	provider := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))
	otel.SetTextMapPropagator(propagation.TraceContext{})

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	ctx, pspan := provider.Tracer(tracerName).Start(context.Background(), "test")
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(r.Header))

	router := echo.New()
	router.Use(Middleware("foobar", WithTracerProvider(provider)))
	router.GET("/user/:id", func(c echo.Context) error {
		span := oteltrace.SpanFromContext(c.Request().Context())
		mspan, ok := span.(*oteltest.Span)
		require.True(t, ok)
		assert.Equal(t, pspan.SpanContext().TraceID(), mspan.SpanContext().TraceID())
		assert.Equal(t, pspan.SpanContext().SpanID(), mspan.ParentSpanID())
		return c.NoContent(200)
	})

	router.ServeHTTP(w, r)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator())
	assert.Equal(t, http.StatusOK, w.Result().StatusCode, "should call the 'user' handler")
}

func TestPropagationWithCustomPropagators(t *testing.T) {
	sr := new(oteltest.SpanRecorder)
	provider := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))

	b3 := b3prop.New()

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	ctx, pspan := provider.Tracer(tracerName).Start(context.Background(), "test")
	b3.Inject(ctx, propagation.HeaderCarrier(r.Header))

	router := echo.New()
	router.Use(Middleware("foobar", WithTracerProvider(provider), WithPropagators(b3)))
	router.GET("/user/:id", func(c echo.Context) error {
		span := oteltrace.SpanFromContext(c.Request().Context())
		mspan, ok := span.(*oteltest.Span)
		require.True(t, ok)
		assert.Equal(t, pspan.SpanContext().TraceID(), mspan.SpanContext().TraceID())
		assert.Equal(t, pspan.SpanContext().SpanID(), mspan.ParentSpanID())
		return c.NoContent(200)
	})

	router.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode, "should call the 'user' handler")
}

func TestSkipper(t *testing.T) {
	r := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()

	skipper := func(c echo.Context) bool {
		return c.Request().RequestURI == "/ping"
	}

	router := echo.New()
	router.Use(Middleware("foobar", WithSkipper(skipper)))
	router.GET("/ping", func(c echo.Context) error {
		span := oteltrace.SpanFromContext(c.Request().Context())
		assert.False(t, span.SpanContext().HasSpanID())
		assert.False(t, span.SpanContext().HasTraceID())
		return c.NoContent(200)
	})

	router.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode, "should call the 'ping' handler")
}
