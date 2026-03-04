// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package otelecho_test

import (
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
)

func ExampleMiddleware() {
	/* curl -v -d "a painting" http://localhost:7777/hello/bob/ross
	...
	* upload completely sent off: 10 out of 10 bytes
	< HTTP/1.1 200 OK
	< Traceparent: 00-76ae040ee5753f38edf1c2bd9bd128bd-dd394138cfd7a3dc-01
	< Date: Fri, 04 Oct 2019 02:33:08 GMT
	< Content-Length: 45
	< Content-Type: text/plain; charset=utf-8
	<
	Hello, bob/ross!
	You sent me this:
	a painting
	*/

	// Create a new Echo instance
	e := echo.New()

	// Use the otelecho middleware with options
	e.Use(otelecho.Middleware("server",
		otelecho.WithSkipper(func(c echo.Context) bool {
			// Skip tracing for health check endpoints
			return c.Path() == "/health"
		}),
	))

	// Define a route with a handler that demonstrates tracing
	e.POST("/hello/:name", func(c echo.Context) error {
		ctx := c.Request().Context()

		// Get the current span from context
		span := trace.SpanFromContext(ctx)

		// Create a child span for processing the name
		ctx, nameSpan := span.TracerProvider().Tracer("exampleTracer").Start(ctx, "processName")

		// Get the name parameter using Echo's built-in functionality
		name := c.Param("name")

		// Add the name as a span attribute
		nameSpan.SetAttributes(attribute.String("name", name))
		nameSpan.End()

		// Read the request body
		d, err := io.ReadAll(c.Request().Body)
		if err != nil {
			log.Println("error reading body: ", err)
			// Record the error in the span and set its status
			span.SetStatus(codes.Error, "failed to read request body")
			return c.String(http.StatusBadRequest, "Bad request")
		}

		// Create another child span for processing the response
		_, responseSpan := span.TracerProvider().Tracer("exampleTracer").Start(ctx, "createResponse")

		// Create the response
		response := "Hello, " + name + "!\nYou sent me this:\n" + string(d)

		// Add information about the response size
		responseSpan.SetAttributes(attribute.Int("response.size", len(response)))
		responseSpan.End()

		// Set the status of the main span to OK
		span.SetStatus(codes.Ok, "")

		return c.String(http.StatusOK, response)
	})

	// Add a health check endpoint that will be skipped by the tracer
	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	// Start the server
	err := e.Start(":7777")
	if !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func ExampleMiddleware_withMetrics() {
	// This example shows how to use the otelecho middleware with custom metrics attributes
	// The middleware will automatically collect HTTP server metrics including:
	// - http.server.request.duration
	// - http.server.request.body.size
	// - http.server.response.body.size

	// Create a new Echo instance
	e := echo.New()

	// Use the otelecho middleware with metrics and custom attributes
	e.Use(otelecho.Middleware("api-server",
		otelecho.WithMetricAttributeFn(func(r *http.Request) []attribute.KeyValue {
			// Add custom attributes from HTTP request
			return []attribute.KeyValue{
				attribute.Bool("custom.has_request_body", r.ContentLength != 0),
				attribute.Bool("custom.has_content_type", r.Header.Get("Content-Type") != ""),
			}
		}),
		otelecho.WithEchoMetricAttributeFn(func(c echo.Context) []attribute.KeyValue {
			// Add custom attributes from Echo context
			// If attributes are duplicated between this method and `WithMetricAttributeFn`, the attributes in this method will be used.
			return []attribute.KeyValue{
				attribute.Bool("custom.has_request_body", c.Request().ContentLength != 0),
				attribute.Bool("custom.has_content_type", c.Request().Header.Get("Content-Type") != ""),
			}
		}),
	))

	// Define routes
	e.GET("/api/users/:id", func(c echo.Context) error {
		userID := c.Param("id")
		return c.JSON(http.StatusOK, map[string]any{
			"id":   userID,
			"name": "User " + userID,
		})
	})

	e.POST("/api/users", func(c echo.Context) error {
		var user struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}

		if err := c.Bind(&user); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
		}

		return c.JSON(http.StatusCreated, map[string]any{
			"id":    "12345",
			"name":  user.Name,
			"email": user.Email,
		})
	})

	// Output:
}
