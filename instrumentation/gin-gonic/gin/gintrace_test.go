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

package gin

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	mocktrace "go.opentelemetry.io/contrib/internal/trace"
	otelglobal "go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/kv/value"
	otelpropagation "go.opentelemetry.io/otel/api/propagation"
	oteltrace "go.opentelemetry.io/otel/api/trace"
)

func init() {
	gin.SetMode(gin.ReleaseMode) // silence annoying log msgs
}

func TestChildSpanFromGlobalTracer(t *testing.T) {
	otelglobal.SetTraceProvider(&mocktrace.Provider{})

	router := gin.New()
	router.Use(Middleware("foobar"))
	router.GET("/user/:id", func(c *gin.Context) {
		span := oteltrace.SpanFromContext(c.Request.Context())
		_, ok := span.(*mocktrace.Span)
		assert.True(t, ok)
		spanTracer := span.Tracer()
		mockTracer, ok := spanTracer.(*mocktrace.Tracer)
		require.True(t, ok)
		assert.Equal(t, "go.opentelemetry.io/contrib/instrumentation/gin-gonic/gin", mockTracer.Name)
	})

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)
}

func TestChildSpanFromCustomTracer(t *testing.T) {
	tracer := mocktrace.NewTracer("test-tracer")

	router := gin.New()
	router.Use(Middleware("foobar", WithTracer(tracer)))
	router.GET("/user/:id", func(c *gin.Context) {
		span := oteltrace.SpanFromContext(c.Request.Context())
		_, ok := span.(*mocktrace.Span)
		assert.True(t, ok)
		spanTracer := span.Tracer()
		mockTracer, ok := spanTracer.(*mocktrace.Tracer)
		require.True(t, ok)
		assert.Equal(t, "test-tracer", mockTracer.Name)
	})

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)
}

func TestTrace200(t *testing.T) {
	tracer := mocktrace.NewTracer("test-tracer")

	router := gin.New()
	router.Use(Middleware("foobar", WithTracer(tracer)))
	router.GET("/user/:id", func(c *gin.Context) {
		span := oteltrace.SpanFromContext(c.Request.Context())
		mspan, ok := span.(*mocktrace.Span)
		require.True(t, ok)
		assert.Equal(t, value.String("foobar"), mspan.Attributes["http.server_name"])
		id := c.Param("id")
		_, _ = c.Writer.Write([]byte(id))
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
	assert.Equal(t, value.String("foobar"), span.Attributes["http.server_name"])
	assert.Equal(t, value.Int(http.StatusOK), span.Attributes["http.status_code"])
	assert.Equal(t, value.String("GET"), span.Attributes["http.method"])
	assert.Equal(t, value.String("/user/123"), span.Attributes["http.target"])
	assert.Equal(t, value.String("/user/:id"), span.Attributes["http.route"])
}

func TestError(t *testing.T) {
	tracer := mocktrace.NewTracer("test-tracer")

	// setup
	router := gin.New()
	router.Use(Middleware("foobar", WithTracer(tracer)))

	// configure a handler that returns an error and 5xx status
	// code
	router.GET("/server_err", func(c *gin.Context) {
		_ = c.AbortWithError(http.StatusInternalServerError, errors.New("oh no"))
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
	assert.Equal(t, value.String("foobar"), span.Attributes["http.server_name"])
	assert.Equal(t, value.Int(http.StatusInternalServerError), span.Attributes["http.status_code"])
	assert.Equal(t, value.String("Error #01: oh no\n"), span.Attributes["gin.errors"])
	// server errors set the status
	assert.Equal(t, codes.Internal, span.Status)
}

func TestHTML(t *testing.T) {
	tracer := mocktrace.NewTracer("test-tracer")

	// setup
	router := gin.New()
	router.Use(Middleware("foobar", WithTracer(tracer)))

	// add a template
	tmpl := template.Must(template.New("hello").Parse("hello {{.}}"))
	router.SetHTMLTemplate(tmpl)

	// a handler with an error and make the requests
	router.GET("/hello", func(c *gin.Context) {
		HTML(c, 200, "hello", "world")
	})
	r := httptest.NewRequest("GET", "/hello", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	response := w.Result()
	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Equal(t, "hello world", w.Body.String())

	// verify the errors and status are correct
	spans := tracer.EndedSpans()
	require.Len(t, spans, 2)
	var tspan *mocktrace.Span
	for _, s := range spans {
		// we need to pick up the span we're searching for, as the
		// order is not guaranteed within the buffer
		if s.Name == "gin.renderer.html" {
			tspan = s
			break
		}
	}
	require.NotNil(t, tspan)
	assert.Equal(t, value.String("hello"), tspan.Attributes["go.template"])
}

func TestGetSpanNotInstrumented(t *testing.T) {
	router := gin.New()
	router.GET("/ping", func(c *gin.Context) {
		// Assert we don't have a span on the context.
		span := oteltrace.SpanFromContext(c.Request.Context())
		_, ok := span.(oteltrace.NoopSpan)
		assert.True(t, ok)
		_, _ = c.Writer.Write([]byte("ok"))
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

	router := gin.New()
	router.Use(Middleware("foobar", WithTracer(tracer)))
	router.GET("/user/:id", func(c *gin.Context) {
		span := oteltrace.SpanFromContext(c.Request.Context())
		mspan, ok := span.(*mocktrace.Span)
		require.True(t, ok)
		assert.Equal(t, pspan.SpanContext().TraceID, mspan.SpanContext().TraceID)
		assert.Equal(t, pspan.SpanContext().SpanID, mspan.ParentSpanID)
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

	router := gin.New()
	router.Use(Middleware("foobar", WithTracer(tracer), WithPropagators(props)))
	router.GET("/user/:id", func(c *gin.Context) {
		span := oteltrace.SpanFromContext(c.Request.Context())
		mspan, ok := span.(*mocktrace.Span)
		require.True(t, ok)
		assert.Equal(t, pspan.SpanContext().TraceID, mspan.SpanContext().TraceID)
		assert.Equal(t, pspan.SpanContext().SpanID, mspan.ParentSpanID)
	})

	router.ServeHTTP(w, r)
}
