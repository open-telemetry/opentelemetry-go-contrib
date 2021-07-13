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

package otelfiber

import (
	"context"
	"errors"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	b3prop "go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/oteltest"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestChildSpanFromGlobalTracer(t *testing.T) {
	otel.SetTracerProvider(oteltest.NewTracerProvider())

	var gotSpan oteltrace.Span

	app := fiber.New()
	app.Use(Middleware("foobar"))
	app.Get("/user/:id", func(ctx *fiber.Ctx) error {
		gotSpan = oteltrace.SpanFromContext(ctx.UserContext())
		return ctx.SendStatus(http.StatusNoContent)
	})

	_, _ = app.Test(httptest.NewRequest("GET", "/user/123", nil))

	_, ok := gotSpan.(*oteltest.Span)
	assert.True(t, ok)
}

func TestChildSpanFromCustomTracer(t *testing.T) {
	provider := oteltest.NewTracerProvider()
	var gotSpan oteltrace.Span

	app := fiber.New()
	app.Use(Middleware("foobar", WithTracerProvider(provider)))
	app.Get("/user/:id", func(ctx *fiber.Ctx) error {
		gotSpan = oteltrace.SpanFromContext(ctx.UserContext())
		return ctx.SendStatus(http.StatusNoContent)
	})

	_, _ = app.Test(httptest.NewRequest("GET", "/user/123", nil))

	_, ok := gotSpan.(*oteltest.Span)
	assert.True(t, ok)
}

func TestTrace200(t *testing.T) {
	sr := new(oteltest.SpanRecorder)
	provider := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))

	var gotSpan oteltrace.Span

	app := fiber.New()
	app.Use(Middleware("foobar", WithTracerProvider(provider)))
	app.Get("/user/:id", func(ctx *fiber.Ctx) error {
		gotSpan = oteltrace.SpanFromContext(ctx.UserContext())
		id := ctx.Params("id")
		return ctx.SendString(id)
	})

	resp, _ := app.Test(httptest.NewRequest("GET", "/user/123", nil), 3000)

	// do and verify the request
	require.Equal(t, http.StatusOK, resp.StatusCode)

	mspan, ok := gotSpan.(*oteltest.Span)
	require.True(t, ok)
	assert.Equal(t, attribute.StringValue("foobar"), mspan.Attributes()[semconv.HTTPServerNameKey])

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
	app := fiber.New()
	app.Use(Middleware("foobar", WithTracerProvider(provider)))
	// configure a handler that returns an error and 5xx status
	// code
	app.Get("/server_err", func(ctx *fiber.Ctx) error {
		return errors.New("oh no")
	})
	resp, _ := app.Test(httptest.NewRequest("GET", "/server_err", nil))
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	// verify the errors and status are correct
	spans := sr.Completed()
	require.Len(t, spans, 1)
	span := spans[0]
	assert.Equal(t, "/server_err", span.Name())
	assert.Equal(t, attribute.StringValue("foobar"), span.Attributes()["http.server_name"])
	assert.Equal(t, attribute.IntValue(http.StatusInternalServerError), span.Attributes()["http.status_code"])
	assert.Equal(t, attribute.StringValue("oh no"), span.Attributes()["fiber.error"])
	// server errors set the status
	assert.Equal(t, codes.Error, span.StatusCode())
}

func TestErrorOnlyHandledOnce(t *testing.T) {
	timesHandlingError := 0
	app := fiber.New(fiber.Config{
		ErrorHandler: func(ctx *fiber.Ctx, err error) error {
			timesHandlingError++
			return fiber.NewError(http.StatusInternalServerError, err.Error())
		},
	})
	app.Use(Middleware("test-service"))
	app.Get("/", func(ctx *fiber.Ctx) error {
		return errors.New("mock error")
	})
	_, _ = app.Test(httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, 1, timesHandlingError)
}

func TestGetSpanNotInstrumented(t *testing.T) {
	var gotSpan oteltrace.Span

	app := fiber.New()
	app.Get("/ping", func(ctx *fiber.Ctx) error {
		// Assert we don't have a span on the context.
		gotSpan = oteltrace.SpanFromContext(ctx.UserContext())
		return ctx.SendString("ok")
	})
	resp, _ := app.Test(httptest.NewRequest("GET", "/ping", nil))
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	ok := !gotSpan.SpanContext().IsValid()
	assert.True(t, ok)
}

func TestPropagationWithGlobalPropagators(t *testing.T) {
	sr := new(oteltest.SpanRecorder)
	provider := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))
	otel.SetTextMapPropagator(propagation.TraceContext{})
	defer otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator())
	var gotSpan oteltrace.Span

	r := httptest.NewRequest("GET", "/user/123", nil)

	ctx, pspan := provider.Tracer(tracerName).Start(context.Background(), "test")
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(r.Header))

	app := fiber.New()
	app.Use(Middleware("foobar", WithTracerProvider(provider)))
	app.Get("/user/:id", func(ctx *fiber.Ctx) error {
		gotSpan = oteltrace.SpanFromContext(ctx.UserContext())
		return ctx.SendStatus(http.StatusNoContent)
	})

	_, _ = app.Test(r)

	mspan, ok := gotSpan.(*oteltest.Span)
	require.True(t, ok)
	assert.Equal(t, pspan.SpanContext().TraceID(), mspan.SpanContext().TraceID())
	assert.Equal(t, pspan.SpanContext().SpanID(), mspan.ParentSpanID())
}

func TestPropagationWithCustomPropagators(t *testing.T) {
	sr := new(oteltest.SpanRecorder)
	provider := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))
	var gotSpan oteltrace.Span

	b3 := b3prop.B3{}

	r := httptest.NewRequest("GET", "/user/123", nil)

	ctx, pspan := provider.Tracer(tracerName).Start(context.Background(), "test")
	b3.Inject(ctx, propagation.HeaderCarrier(r.Header))

	app := fiber.New()
	app.Use(Middleware("foobar", WithTracerProvider(provider), WithPropagators(b3)))
	app.Get("/user/:id", func(ctx *fiber.Ctx) error {
		gotSpan = oteltrace.SpanFromContext(ctx.UserContext())
		return ctx.SendStatus(http.StatusNoContent)
	})

	_, _ = app.Test(r)

	mspan, ok := gotSpan.(*oteltest.Span)
	require.True(t, ok)
	assert.Equal(t, pspan.SpanContext().TraceID(), mspan.SpanContext().TraceID())
	assert.Equal(t, pspan.SpanContext().SpanID(), mspan.ParentSpanID())
}
