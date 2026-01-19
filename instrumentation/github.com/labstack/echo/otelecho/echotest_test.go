// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Based on https://github.com/DataDog/dd-trace-go/blob/8fb554ff7cf694267f9077ae35e27ce4689ed8b6/contrib/gin-gonic/gin/gintrace_test.go

package otelecho_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	oteltrace "go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
)

func TestChildSpanFromGlobalTracer(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := trace.NewTracerProvider(trace.WithSpanProcessor(sr))
	otel.SetTracerProvider(provider)

	router := echo.New()
	router.Use(otelecho.Middleware("foobar"))
	router.GET("/user/:id", func(c *echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	r := httptest.NewRequest(http.MethodGet, "/user/123", http.NoBody)
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
	router.GET("/user/:id", func(c *echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	r := httptest.NewRequest(http.MethodGet, "/user/123", http.NoBody)
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
	router.GET("/user/:id", func(c *echo.Context) error {
		id := c.Param("id")
		return c.String(http.StatusOK, id)
	})

	r := httptest.NewRequest(http.MethodGet, "/user/123", http.NoBody)
	w := httptest.NewRecorder()

	// do and verify the request
	router.ServeHTTP(w, r)
	response := w.Result()
	require.Equal(t, http.StatusOK, response.StatusCode)

	// verify traces look good
	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	assert.Equal(t, "GET /user/:id", span.Name())
	assert.Equal(t, oteltrace.SpanKindServer, span.SpanKind())
	attrs := span.Attributes()
	assert.Contains(t, attrs, attribute.String("server.address", "foobar"))
	assert.Contains(t, attrs, attribute.Int("http.response.status_code", http.StatusOK))
	assert.Contains(t, attrs, attribute.String("http.request.method", "GET"))
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
	router.GET("/server_err", func(*echo.Context) error {
		return wantErr
	})
	r := httptest.NewRequest(http.MethodGet, "/server_err", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	response := w.Result()
	assert.Equal(t, http.StatusInternalServerError, response.StatusCode)

	// verify the errors and status are correct
	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	assert.Equal(t, "GET /server_err", span.Name())
	attrs := span.Attributes()
	assert.Contains(t, attrs, attribute.String("server.address", "foobar"))
	assert.Contains(t, attrs, attribute.Int("http.response.status_code", http.StatusInternalServerError))
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
		handler    func(c *echo.Context) error
	}{
		{
			name:       "StandardError",
			echoError:  "oh no",
			statusCode: http.StatusInternalServerError,
			spanCode:   codes.Error,
			handler: func(*echo.Context) error {
				return errors.New("oh no")
			},
		},
		{
			name:       "EchoHTTPServerError",
			echoError:  "code=500, message=my error message",
			statusCode: http.StatusInternalServerError,
			spanCode:   codes.Error,
			handler: func(*echo.Context) error {
				return echo.NewHTTPError(http.StatusInternalServerError, "my error message")
			},
		},
		{
			name:       "EchoHTTPClientError",
			echoError:  "code=400, message=my error message",
			statusCode: http.StatusBadRequest,
			spanCode:   codes.Unset,
			handler: func(*echo.Context) error {
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
			r := httptest.NewRequest(http.MethodGet, "/err", http.NoBody)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)

			spans := sr.Ended()
			require.Len(t, spans, 1)
			span := spans[0]
			assert.Equal(t, "GET /err", span.Name())
			assert.Equal(t, tc.spanCode, span.Status().Code)

			attrs := span.Attributes()
			assert.Contains(t, attrs, attribute.String("server.address", "foobar"))
			assert.Contains(t, attrs, attribute.String("http.route", "/err"))
			assert.Contains(t, attrs, attribute.String("http.request.method", "GET"))
			assert.Contains(t, attrs, attribute.Int("http.response.status_code", tc.statusCode))
			assert.Contains(t, attrs, attribute.String("echo.error", tc.echoError))
		})
	}
}

func TestErrorNotSwallowedByMiddleware(t *testing.T) {
	e := echo.New()
	r := httptest.NewRequest(http.MethodGet, "/err", http.NoBody)
	w := httptest.NewRecorder()
	c := e.NewContext(r, w)
	h := otelecho.Middleware("foobar")(echo.HandlerFunc(func(*echo.Context) error {
		return assert.AnError
	}))

	err := h(c)
	assert.Equal(t, assert.AnError, err)
}

func TestSpanNameFormatter(t *testing.T) {
	imsb := tracetest.NewInMemoryExporter()
	provider := trace.NewTracerProvider(trace.WithSyncer(imsb))

	tests := []struct {
		name     string
		method   string
		path     string
		url      string
		expected string
	}{
		// Test for standard methods
		{"standard method of GET", http.MethodGet, "/user/:id", "/user/123", "GET /user/:id"},
		{"standard method of HEAD", http.MethodHead, "/user/:id", "/user/123", "HEAD /user/:id"},
		{"standard method of POST", http.MethodPost, "/user/:id", "/user/123", "POST /user/:id"},
		{"standard method of PUT", http.MethodPut, "/user/:id", "/user/123", "PUT /user/:id"},
		{"standard method of PATCH", http.MethodPatch, "/user/:id", "/user/123", "PATCH /user/:id"},
		{"standard method of DELETE", http.MethodDelete, "/user/:id", "/user/123", "DELETE /user/:id"},
		{"standard method of CONNECT", http.MethodConnect, "/user/:id", "/user/123", "CONNECT /user/:id"},
		{"standard method of OPTIONS", http.MethodOptions, "/user/:id", "/user/123", "OPTIONS /user/:id"},
		{"standard method of TRACE", http.MethodTrace, "/user/:id", "/user/123", "TRACE /user/:id"},
		{"standard method of GET, but it's another route.", http.MethodGet, "/", "/", "GET /"},

		// Test for no route
		{"no route", http.MethodGet, "/", "/user/id", "GET"},

		// Test for case-insensitive method
		{"all lowercase", "get", "/user/123", "/user/123", "GET /user/123"},
		{"partial capitalization", "Get", "/user/123", "/user/123", "GET /user/123"},
		{"full capitalization", "GET", "/user/:id", "/user/123", "GET /user/:id"},

		// Test for invalid method
		{"invalid method", "INVALID", "/user/123", "/user/123", "HTTP /user/123"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer imsb.Reset()

			router := echo.New()
			router.Use(otelecho.Middleware("foobar", otelecho.WithTracerProvider(provider)))
			router.Add(test.method, test.path, func(c *echo.Context) error {
				return c.NoContent(http.StatusOK)
			})

			r := httptest.NewRequest(test.method, test.url, http.NoBody)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)

			spans := imsb.GetSpans()
			require.Len(t, spans, 1)
			assert.Equal(t, test.expected, spans[0].Name)
		})
	}
}
