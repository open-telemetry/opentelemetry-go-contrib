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

package echo

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/codes"

	mocktrace "go.opentelemetry.io/contrib/internal/trace"
	otelglobal "go.opentelemetry.io/otel/api/global"
	otelpropagation "go.opentelemetry.io/otel/api/propagation"
	oteltrace "go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
)

func TestChildSpanFromGlobalTracer(t *testing.T) {
	otelglobal.SetTraceProvider(&mocktrace.Provider{})

	router := echo.New()
	router.Use(Middleware("foobar"))
	router.GET("/user/:id", func(c echo.Context) error {
		span := oteltrace.SpanFromContext(c.Request().Context())
		_, ok := span.(*mocktrace.Span)
		assert.True(t, ok)
		spanTracer := span.Tracer()
		mockTracer, ok := spanTracer.(*mocktrace.Tracer)
		require.True(t, ok)
		assert.Equal(t, "go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo", mockTracer.Name)
		return c.NoContent(200)
	})

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)
}

func TestChildSpanFromCustomTracer(t *testing.T) {
	tracer := mocktrace.NewTracer("test-tracer")

	router := echo.New()
	router.Use(Middleware("foobar", WithTracer(tracer)))
	router.GET("/user/:id", func(c echo.Context) error {
		span := oteltrace.SpanFromContext(c.Request().Context())
		_, ok := span.(*mocktrace.Span)
		assert.True(t, ok)
		spanTracer := span.Tracer()
		mockTracer, ok := spanTracer.(*mocktrace.Tracer)
		require.True(t, ok)
		assert.Equal(t, "test-tracer", mockTracer.Name)
		return c.NoContent(200)
	})

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)
}

func TestTrace200(t *testing.T) {
	tracer := mocktrace.NewTracer("test-tracer")

	router := echo.New()
	router.Use(Middleware("foobar", WithTracer(tracer)))
	router.GET("/user/:id", func(c echo.Context) error {
		span := oteltrace.SpanFromContext(c.Request().Context())
		mspan, ok := span.(*mocktrace.Span)
		require.True(t, ok)
		assert.Equal(t, label.StringValue("foobar"), mspan.Attributes["http.server_name"])
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
	spans := tracer.EndedSpans()
	require.Len(t, spans, 1)
	span := spans[0]
	assert.Equal(t, "/user/:id", span.Name)
	assert.Equal(t, oteltrace.SpanKindServer, span.Kind)
	assert.Equal(t, label.StringValue("foobar"), span.Attributes["http.server_name"])
	assert.Equal(t, label.IntValue(http.StatusOK), span.Attributes["http.status_code"])
	assert.Equal(t, label.StringValue("GET"), span.Attributes["http.method"])
	assert.Equal(t, label.StringValue("/user/123"), span.Attributes["http.target"])
	assert.Equal(t, label.StringValue("/user/:id"), span.Attributes["http.route"])
}

func TestError(t *testing.T) {
	tracer := mocktrace.NewTracer("test-tracer")

	// setup
	router := echo.New()
	router.Use(Middleware("foobar", WithTracer(tracer)))
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
	spans := tracer.EndedSpans()
	require.Len(t, spans, 1)
	span := spans[0]
	assert.Equal(t, "/server_err", span.Name)
	assert.Equal(t, label.StringValue("foobar"), span.Attributes["http.server_name"])
	assert.Equal(t, label.IntValue(http.StatusInternalServerError), span.Attributes["http.status_code"])
	assert.Equal(t, label.StringValue("oh no"), span.Attributes["echo.error"])
	// server errors set the status
	assert.Equal(t, codes.Internal, span.Status)
}

func TestGetSpanNotInstrumented(t *testing.T) {
	router := echo.New()
	router.GET("/ping", func(c echo.Context) error {
		// Assert we don't have a span on the context.
		span := oteltrace.SpanFromContext(c.Request().Context())
		_, ok := span.(oteltrace.NoopSpan)
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
	tracer := mocktrace.NewTracer("test-tracer")

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	ctx, pspan := tracer.Start(context.Background(), "test")
	otelpropagation.InjectHTTP(ctx, otelglobal.Propagators(), r.Header)

	router := echo.New()
	router.Use(Middleware("foobar", WithTracer(tracer)))
	router.GET("/user/:id", func(c echo.Context) error {
		span := oteltrace.SpanFromContext(c.Request().Context())
		mspan, ok := span.(*mocktrace.Span)
		require.True(t, ok)
		assert.Equal(t, pspan.SpanContext().TraceID, mspan.SpanContext().TraceID)
		assert.Equal(t, pspan.SpanContext().SpanID, mspan.ParentSpanID)
		return c.NoContent(200)
	})

	router.ServeHTTP(w, r)
}

func TestPropagationWithCustomPropagators(t *testing.T) {
	tracer := mocktrace.NewTracer("test-tracer")
	b3 := oteltrace.B3{}
	props := otelpropagation.New(
		otelpropagation.WithExtractors(b3),
		otelpropagation.WithInjectors(b3),
	)

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	ctx, pspan := tracer.Start(context.Background(), "test")
	otelpropagation.InjectHTTP(ctx, props, r.Header)

	router := echo.New()
	router.Use(Middleware("foobar", WithTracer(tracer), WithPropagators(props)))
	router.GET("/user/:id", func(c echo.Context) error {
		span := oteltrace.SpanFromContext(c.Request().Context())
		mspan, ok := span.(*mocktrace.Span)
		require.True(t, ok)
		assert.Equal(t, pspan.SpanContext().TraceID, mspan.SpanContext().TraceID)
		assert.Equal(t, pspan.SpanContext().SpanID, mspan.ParentSpanID)
		return c.NoContent(200)
	})

	router.ServeHTTP(w, r)
}
