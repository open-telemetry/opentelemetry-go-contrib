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

package macaron

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/macaron.v1"

	mocktrace "go.opentelemetry.io/contrib/internal/trace"
	otelglobal "go.opentelemetry.io/otel/api/global"
	otelpropagation "go.opentelemetry.io/otel/api/propagation"
	oteltrace "go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
)

func TestChildSpanFromGlobalTracer(t *testing.T) {
	otelglobal.SetTraceProvider(&mocktrace.Provider{})

	m := macaron.Classic()
	m.Use(Middleware("foobar"))
	m.Get("/user/:id", func(ctx *macaron.Context) {
		span := oteltrace.SpanFromContext(ctx.Req.Request.Context())
		_, ok := span.(*mocktrace.Span)
		assert.True(t, ok)
		spanTracer := span.Tracer()
		mockTracer, ok := spanTracer.(*mocktrace.Tracer)
		require.True(t, ok)
		assert.Equal(t, "go.opentelemetry.io/contrib/instrumentation/macaron", mockTracer.Name)
		ctx.Resp.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	m.ServeHTTP(w, r)
}

func TestChildSpanFromCustomTracer(t *testing.T) {
	tracer := mocktrace.NewTracer("test-tracer")

	m := macaron.Classic()
	m.Use(Middleware("foobar", WithTracer(tracer)))
	m.Get("/user/:id", func(ctx *macaron.Context) {
		span := oteltrace.SpanFromContext(ctx.Req.Request.Context())
		_, ok := span.(*mocktrace.Span)
		assert.True(t, ok)
		spanTracer := span.Tracer()
		mockTracer, ok := spanTracer.(*mocktrace.Tracer)
		require.True(t, ok)
		assert.Equal(t, "test-tracer", mockTracer.Name)
		ctx.Resp.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	m.ServeHTTP(w, r)
}

func TestChildSpanNames(t *testing.T) {
	tracer := mocktrace.NewTracer("test-tracer")

	m := macaron.Classic()
	m.Use(Middleware("foobar", WithTracer(tracer)))
	m.Get("/user/:id", func(ctx *macaron.Context) {
		ctx.Resp.WriteHeader(http.StatusOK)
	})
	m.Get("/book/:title", func(ctx *macaron.Context) {
		_, err := ctx.Resp.Write(([]byte)("ok"))
		if err != nil {
			t.Error(err)
		}
	})

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)
	spans := tracer.EndedSpans()
	require.Len(t, spans, 1)
	span := spans[0]
	assert.Equal(t, "/user/123", span.Name) // TODO: span name should show router template, eg /user/:id
	assert.Equal(t, oteltrace.SpanKindServer, span.Kind)
	assert.Equal(t, label.StringValue("foobar"), span.Attributes["http.server_name"])
	assert.Equal(t, label.IntValue(http.StatusOK), span.Attributes["http.status_code"])
	assert.Equal(t, label.StringValue("GET"), span.Attributes["http.method"])
	assert.Equal(t, label.StringValue("/user/123"), span.Attributes["http.target"])
	// TODO: span name should show router template, eg /user/:id
	//assert.Equal(t, label.StringValue("/user/:id"), span.Attributes["http.route"])

	r = httptest.NewRequest("GET", "/book/foo", nil)
	w = httptest.NewRecorder()
	m.ServeHTTP(w, r)
	spans = tracer.EndedSpans()
	require.Len(t, spans, 1)
	span = spans[0]
	assert.Equal(t, "/book/foo", span.Name) // TODO: span name should show router template, eg /book/:title
	assert.Equal(t, oteltrace.SpanKindServer, span.Kind)
	assert.Equal(t, label.StringValue("foobar"), span.Attributes["http.server_name"])
	assert.Equal(t, label.IntValue(http.StatusOK), span.Attributes["http.status_code"])
	assert.Equal(t, label.StringValue("GET"), span.Attributes["http.method"])
	assert.Equal(t, label.StringValue("/book/foo"), span.Attributes["http.target"])
	// TODO: span name should show router template, eg /book/:title
	//assert.Equal(t, label.StringValue("/book/:title"), span.Attributes["http.route"])
}

func TestGetSpanNotInstrumented(t *testing.T) {
	m := macaron.Classic()
	m.Get("/user/:id", func(ctx *macaron.Context) {
		span := oteltrace.SpanFromContext(ctx.Req.Request.Context())
		_, ok := span.(oteltrace.NoopSpan)
		assert.True(t, ok)
		ctx.Resp.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	m.ServeHTTP(w, r)
}

func TestPropagationWithGlobalPropagators(t *testing.T) {
	tracer := mocktrace.NewTracer("test-tracer")

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	ctx, pspan := tracer.Start(context.Background(), "test")
	otelpropagation.InjectHTTP(ctx, otelglobal.Propagators(), r.Header)

	m := macaron.Classic()
	m.Use(Middleware("foobar", WithTracer(tracer)))
	m.Get("/user/:id", func(ctx *macaron.Context) {
		span := oteltrace.SpanFromContext(ctx.Req.Request.Context())
		mspan, ok := span.(*mocktrace.Span)
		require.True(t, ok)
		assert.Equal(t, pspan.SpanContext().TraceID, mspan.SpanContext().TraceID)
		assert.Equal(t, pspan.SpanContext().SpanID, mspan.ParentSpanID)
		ctx.Resp.WriteHeader(http.StatusOK)
	})

	m.ServeHTTP(w, r)
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

	m := macaron.Classic()
	m.Use(Middleware("foobar", WithTracer(tracer), WithPropagators(props)))
	m.Get("/user/:id", func(ctx *macaron.Context) {
		span := oteltrace.SpanFromContext(ctx.Req.Request.Context())
		mspan, ok := span.(*mocktrace.Span)
		require.True(t, ok)
		assert.Equal(t, pspan.SpanContext().TraceID, mspan.SpanContext().TraceID)
		assert.Equal(t, pspan.SpanContext().SpanID, mspan.ParentSpanID)
		ctx.Resp.WriteHeader(http.StatusOK)
	})

	m.ServeHTTP(w, r)
}
