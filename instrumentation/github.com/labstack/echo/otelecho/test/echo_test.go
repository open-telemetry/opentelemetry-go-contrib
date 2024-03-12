// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Based on https://github.com/DataDog/dd-trace-go/blob/8fb554ff7cf694267f9077ae35e27ce4689ed8b6/contrib/gin-gonic/gin/gintrace_test.go

package test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestChildSpanFromGlobalTracer(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := trace.NewTracerProvider(trace.WithSpanProcessor(sr))
	otel.SetTracerProvider(provider)

	router := echo.New()
	router.Use(otelecho.Middleware("foobar"))
	router.GET("/user/:id", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode, "should call the 'user' handler")
	assert.Len(t, sr.Ended(), 1)
}

func TestChildSpanFromCustomTracer(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := trace.NewTracerProvider(trace.WithSpanProcessor(sr))

	router := echo.New()
	router.Use(otelecho.Middleware("foobar", otelecho.WithTracerProvider(provider)))
	router.GET("/user/:id", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode, "should call the 'user' handler")
	assert.Len(t, sr.Ended(), 1)
}

func TestTrace200(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := trace.NewTracerProvider(trace.WithSpanProcessor(sr))

	router := echo.New()
	router.Use(otelecho.Middleware("foobar", otelecho.WithTracerProvider(provider)))
	router.GET("/user/:id", func(c echo.Context) error {
		id := c.Param("id")
		return c.String(http.StatusOK, id)
	})

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	// do and verify the request
	router.ServeHTTP(w, r)
	response := w.Result()
	require.Equal(t, http.StatusOK, response.StatusCode)

	// verify traces look good
	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	assert.Equal(t, "/user/:id", span.Name())
	assert.Equal(t, oteltrace.SpanKindServer, span.SpanKind())
	attrs := span.Attributes()
	assert.Contains(t, attrs, attribute.String("net.host.name", "foobar"))
	assert.Contains(t, attrs, attribute.Int("http.status_code", http.StatusOK))
	assert.Contains(t, attrs, attribute.String("http.method", "GET"))
	assert.Contains(t, attrs, attribute.String("http.route", "/user/:id"))
}

func TestError(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := trace.NewTracerProvider(trace.WithSpanProcessor(sr))

	// setup
	router := echo.New()
	router.Use(otelecho.Middleware("foobar", otelecho.WithTracerProvider(provider)))
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
	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	assert.Equal(t, "/server_err", span.Name())
	attrs := span.Attributes()
	assert.Contains(t, attrs, attribute.String("net.host.name", "foobar"))
	assert.Contains(t, attrs, attribute.Int("http.status_code", http.StatusInternalServerError))
	assert.Contains(t, attrs, attribute.String("echo.error", "oh no"))
	// server errors set the status
	assert.Equal(t, codes.Error, span.Status().Code)
}

func TestStatusError(t *testing.T) {
	for _, tc := range []struct {
		name       string
		echoError  string
		statusCode int
		spanCode   codes.Code
		handler    func(c echo.Context) error
	}{
		{
			name:       "StandardError",
			echoError:  "oh no",
			statusCode: http.StatusInternalServerError,
			spanCode:   codes.Error,
			handler: func(c echo.Context) error {
				return errors.New("oh no")
			},
		},
		{
			name:       "EchoHTTPServerError",
			echoError:  "code=500, message=my error message",
			statusCode: http.StatusInternalServerError,
			spanCode:   codes.Error,
			handler: func(c echo.Context) error {
				return echo.NewHTTPError(http.StatusInternalServerError, "my error message")
			},
		},
		{
			name:       "EchoHTTPClientError",
			echoError:  "code=400, message=my error message",
			statusCode: http.StatusBadRequest,
			spanCode:   codes.Unset,
			handler: func(c echo.Context) error {
				return echo.NewHTTPError(http.StatusBadRequest, "my error message")
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			provider := trace.NewTracerProvider(trace.WithSpanProcessor(sr))

			router := echo.New()
			router.Use(otelecho.Middleware("foobar", otelecho.WithTracerProvider(provider)))
			router.GET("/err", tc.handler)
			r := httptest.NewRequest("GET", "/err", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)

			spans := sr.Ended()
			require.Len(t, spans, 1)
			span := spans[0]
			assert.Equal(t, "/err", span.Name())
			assert.Equal(t, tc.spanCode, span.Status().Code)

			attrs := span.Attributes()
			assert.Contains(t, attrs, attribute.String("net.host.name", "foobar"))
			assert.Contains(t, attrs, attribute.String("http.route", "/err"))
			assert.Contains(t, attrs, attribute.String("http.method", "GET"))
			assert.Contains(t, attrs, attribute.Int("http.status_code", tc.statusCode))
			assert.Contains(t, attrs, attribute.String("echo.error", tc.echoError))
		})
	}
}

func TestErrorNotSwallowedByMiddleware(t *testing.T) {
	e := echo.New()
	r := httptest.NewRequest(http.MethodGet, "/err", nil)
	w := httptest.NewRecorder()
	c := e.NewContext(r, w)
	h := otelecho.Middleware("foobar")(echo.HandlerFunc(func(c echo.Context) error {
		return assert.AnError
	}))

	err := h(c)
	assert.Equal(t, assert.AnError, err)
}
