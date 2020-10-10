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

package otelmacaron

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/macaron.v1"

	b3prop "go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	otelglobal "go.opentelemetry.io/otel/api/global"
	oteltrace "go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/api/trace/tracetest"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/propagators"
)

func TestChildSpanFromGlobalTracer(t *testing.T) {
	otelglobal.SetTracerProvider(tracetest.NewTracerProvider())

	m := macaron.Classic()
	m.Use(Middleware("foobar"))
	m.Get("/user/:id", func(ctx *macaron.Context) {
		span := oteltrace.SpanFromContext(ctx.Req.Request.Context())
		_, ok := span.(*tracetest.Span)
		assert.True(t, ok)
		spanTracer := span.Tracer()
		mockTracer, ok := spanTracer.(*tracetest.Tracer)
		require.True(t, ok)
		assert.Equal(t, instrumentationName, mockTracer.Name)
		ctx.Resp.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	m.ServeHTTP(w, r)
}

func TestChildSpanNames(t *testing.T) {
	sr := new(tracetest.StandardSpanRecorder)
	tp := tracetest.NewTracerProvider(tracetest.WithSpanRecorder(sr))

	m := macaron.Classic()
	m.Use(Middleware("foobar", WithTracerProvider(tp)))
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

	r = httptest.NewRequest("GET", "/book/foo", nil)
	w = httptest.NewRecorder()
	m.ServeHTTP(w, r)

	spans := sr.Completed()
	require.Len(t, spans, 2)
	span := spans[0]
	assert.Equal(t, "/user/123", span.Name()) // TODO: span name should show router template, eg /user/:id
	assert.Equal(t, oteltrace.SpanKindServer, span.SpanKind())
	attrs := span.Attributes()
	assert.Equal(t, label.StringValue("foobar"), attrs["http.server_name"])
	assert.Equal(t, label.IntValue(http.StatusOK), attrs["http.status_code"])
	assert.Equal(t, label.StringValue("GET"), attrs["http.method"])
	assert.Equal(t, label.StringValue("/user/123"), attrs["http.target"])
	// TODO: span name should show router template, eg /user/:id
	//assert.Equal(t, label.StringValue("/user/:id"), span.Attributes["http.route"])

	span = spans[1]
	assert.Equal(t, "/book/foo", span.Name()) // TODO: span name should show router template, eg /book/:title
	assert.Equal(t, oteltrace.SpanKindServer, span.SpanKind())
	attrs = span.Attributes()
	assert.Equal(t, label.StringValue("foobar"), attrs["http.server_name"])
	assert.Equal(t, label.IntValue(http.StatusOK), attrs["http.status_code"])
	assert.Equal(t, label.StringValue("GET"), attrs["http.method"])
	assert.Equal(t, label.StringValue("/book/foo"), attrs["http.target"])
	// TODO: span name should show router template, eg /book/:title
	//assert.Equal(t, label.StringValue("/book/:title"), span.Attributes["http.route"])
}

func TestGetSpanNotInstrumented(t *testing.T) {
	m := macaron.Classic()
	m.Get("/user/:id", func(ctx *macaron.Context) {
		span := oteltrace.SpanFromContext(ctx.Req.Request.Context())
		ok := !span.SpanContext().IsValid()
		assert.True(t, ok)
		ctx.Resp.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	m.ServeHTTP(w, r)
}

func TestPropagationWithGlobalPropagators(t *testing.T) {
	tracer := tracetest.NewTracerProvider().Tracer("test-tracer")
	otelglobal.SetTextMapPropagator(propagators.TraceContext{})

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	ctx, pspan := tracer.Start(context.Background(), "test")
	otelglobal.TextMapPropagator().Inject(ctx, r.Header)

	m := macaron.Classic()
	m.Use(Middleware("foobar"))
	m.Get("/user/:id", func(ctx *macaron.Context) {
		span := oteltrace.SpanFromContext(ctx.Req.Request.Context())
		mspan, ok := span.(*tracetest.Span)
		require.True(t, ok)
		assert.Equal(t, pspan.SpanContext().TraceID, mspan.SpanContext().TraceID)
		assert.Equal(t, pspan.SpanContext().SpanID, mspan.ParentSpanID())
		ctx.Resp.WriteHeader(http.StatusOK)
	})

	m.ServeHTTP(w, r)
	otelglobal.SetTextMapPropagator(otel.NewCompositeTextMapPropagator())
}

func TestPropagationWithCustomPropagators(t *testing.T) {
	tp := tracetest.NewTracerProvider()
	tracer := tp.Tracer("test-tracer")
	b3 := b3prop.B3{}

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	ctx, pspan := tracer.Start(context.Background(), "test")
	b3.Inject(ctx, r.Header)

	m := macaron.Classic()
	m.Use(Middleware("foobar", WithTracerProvider(tp), WithPropagators(b3)))
	m.Get("/user/:id", func(ctx *macaron.Context) {
		span := oteltrace.SpanFromContext(ctx.Req.Request.Context())
		mspan, ok := span.(*tracetest.Span)
		require.True(t, ok)
		assert.Equal(t, pspan.SpanContext().TraceID, mspan.SpanContext().TraceID)
		assert.Equal(t, pspan.SpanContext().SpanID, mspan.ParentSpanID())
		ctx.Resp.WriteHeader(http.StatusOK)
	})

	m.ServeHTTP(w, r)
}
