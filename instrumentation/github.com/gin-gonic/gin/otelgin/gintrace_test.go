// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Based on https://github.com/DataDog/dd-trace-go/blob/8fb554ff7cf694267f9077ae35e27ce4689ed8b6/contrib/gin-gonic/gin/gintrace_test.go

package otelgin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	b3prop "go.opentelemetry.io/contrib/propagators/b3"
)

func init() {
	gin.SetMode(gin.ReleaseMode) // silence annoying log msgs
}

func TestGetSpanNotInstrumented(t *testing.T) {
	router := gin.New()
	router.GET("/ping", func(c *gin.Context) {
		// Assert we don't have a span on the context.
		span := trace.SpanFromContext(c.Request.Context())
		ok := !span.SpanContext().IsValid()
		assert.True(t, ok)
		_, _ = c.Writer.Write([]byte("ok"))
	})
	r := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	response := w.Result() //nolint:bodyclose // False positive for httptest.ResponseRecorder: https://github.com/timakin/bodyclose/issues/59.
	assert.Equal(t, http.StatusOK, response.StatusCode)
}

func TestPropagationWithGlobalPropagators(t *testing.T) {
	provider := noop.NewTracerProvider()
	otel.SetTextMapPropagator(b3prop.New())

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	ctx := context.Background()
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{0x01},
		SpanID:  trace.SpanID{0x01},
	})
	ctx = trace.ContextWithRemoteSpanContext(ctx, sc)
	ctx, _ = provider.Tracer(ScopeName).Start(ctx, "test")
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(r.Header))

	router := gin.New()
	router.Use(Middleware("foobar", WithTracerProvider(provider)))
	router.GET("/user/:id", func(c *gin.Context) {
		span := trace.SpanFromContext(c.Request.Context())
		assert.Equal(t, sc.TraceID(), span.SpanContext().TraceID())
		assert.Equal(t, sc.SpanID(), span.SpanContext().SpanID())
	})

	router.ServeHTTP(w, r)
}

func TestPropagationWithCustomPropagators(t *testing.T) {
	provider := noop.NewTracerProvider()
	b3 := b3prop.New()

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	ctx := context.Background()
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{0x01},
		SpanID:  trace.SpanID{0x01},
	})
	ctx = trace.ContextWithRemoteSpanContext(ctx, sc)
	ctx, _ = provider.Tracer(ScopeName).Start(ctx, "test")
	b3.Inject(ctx, propagation.HeaderCarrier(r.Header))

	router := gin.New()
	router.Use(Middleware("foobar", WithTracerProvider(provider), WithPropagators(b3)))
	router.GET("/user/:id", func(c *gin.Context) {
		span := trace.SpanFromContext(c.Request.Context())
		assert.Equal(t, sc.TraceID(), span.SpanContext().TraceID())
		assert.Equal(t, sc.SpanID(), span.SpanContext().SpanID())
	})

	router.ServeHTTP(w, r)
}

func TestSpanRecordError(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()

	router := gin.New()
	router.Use(Middleware("foobar", WithTracerProvider(
		tracesdk.NewTracerProvider(
			tracesdk.WithSyncer(exporter),
		)),
	))

	t.Run("test success", func(t *testing.T) {
		defer exporter.Reset()
		assert.Empty(t, exporter.GetSpans())

		router.GET("/success", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})
		r := httptest.NewRequest("GET", "/success", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Len(t, exporter.GetSpans(), 1)

		// Assert span status
		span := exporter.GetSpans()[0]
		assert.Equal(t, "/success", span.Name)
		assert.NotEqual(t, codes.Error, span.Status.Code)
		assert.Empty(t, span.Events)
	})

	// test success
	t.Run("test error", func(t *testing.T) {
		defer exporter.Reset()
		assert.Empty(t, exporter.GetSpans())

		router.GET("/error", func(c *gin.Context) {
			assert.Error(t, c.AbortWithError(http.StatusInternalServerError, assert.AnError))
		})
		r := httptest.NewRequest("GET", "/error", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Len(t, exporter.GetSpans(), 1)

		// Assert span
		span := exporter.GetSpans()[0]
		assert.Equal(t, "/error", span.Name)
		assert.Equal(t, codes.Error, span.Status.Code)
		assert.Len(t, span.Events, 1)

		// Assert span events
		assert.Equal(t, "exception", span.Events[0].Name)
	})
}
