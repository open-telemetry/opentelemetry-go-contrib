// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package otelecho_test

import (
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"

	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
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
			span.RecordError(err)
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
