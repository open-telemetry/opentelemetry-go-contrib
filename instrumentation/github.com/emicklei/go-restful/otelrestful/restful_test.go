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

package otelrestful_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emicklei/go-restful/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/contrib/instrumentation/github.com/emicklei/go-restful/otelrestful"
	b3prop "go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	otelkv "go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/oteltest"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const tracerName = "go.opentelemetry.io/contrib/instrumentation/github.com/emicklei/go-restful/otelrestful"

func TestChildSpanFromGlobalTracer(t *testing.T) {
	otel.SetTracerProvider(oteltest.NewTracerProvider())

	handlerFunc := func(req *restful.Request, resp *restful.Response) {
		span := oteltrace.SpanFromContext(req.Request.Context())
		_, ok := span.(*oteltest.Span)
		assert.True(t, ok)
		spanTracer := span.Tracer()
		mockTracer, ok := spanTracer.(*oteltest.Tracer)
		require.True(t, ok)
		assert.Equal(t, tracerName, mockTracer.Name)
		resp.WriteHeader(http.StatusOK)
	}
	ws := &restful.WebService{}
	ws.Route(ws.GET("/user/{id}").To(handlerFunc).
		Returns(200, "OK", nil).
		Returns(404, "Not Found", nil))
	container := restful.NewContainer()
	container.Filter(otelrestful.OTelFilter("my-service"))
	container.Add(ws)

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	container.ServeHTTP(w, r)
}

func TestChildSpanFromCustomTracer(t *testing.T) {
	provider := oteltest.NewTracerProvider()

	handlerFunc := func(req *restful.Request, resp *restful.Response) {
		span := oteltrace.SpanFromContext(req.Request.Context())
		_, ok := span.(*oteltest.Span)
		assert.True(t, ok)
		spanTracer := span.Tracer()
		mockTracer, ok := spanTracer.(*oteltest.Tracer)
		require.True(t, ok)
		assert.Equal(t, tracerName, mockTracer.Name)
		resp.WriteHeader(http.StatusOK)
	}
	ws := &restful.WebService{}
	ws.Route(ws.GET("/user/{id}").To(handlerFunc))

	container := restful.NewContainer()
	container.Filter(otelrestful.OTelFilter("my-service", otelrestful.WithTracerProvider(provider)))
	container.Add(ws)

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	container.ServeHTTP(w, r)
}

func TestChildSpanNames(t *testing.T) {
	sr := new(oteltest.StandardSpanRecorder)
	provider := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))

	handlerFunc := func(req *restful.Request, resp *restful.Response) {
		resp.WriteHeader(http.StatusOK)
	}
	ws := &restful.WebService{}
	ws.Route(ws.GET("/user/{id:[0-9]+}").To(handlerFunc))

	container := restful.NewContainer()
	container.Filter(otelrestful.OTelFilter("foobar", otelrestful.WithTracerProvider(provider)))
	container.Add(ws)

	ws.Route(ws.GET("/book/{title}").To(func(req *restful.Request, resp *restful.Response) {
		_, _ = resp.Write(([]byte)("ok"))
	}))

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	container.ServeHTTP(w, r)
	spans := sr.Completed()
	require.Len(t, spans, 1)
	span := spans[0]
	assert.Equal(t, "/user/{id:[0-9]+}", span.Name())
	assert.Equal(t, oteltrace.SpanKindServer, span.SpanKind())
	assert.Equal(t, otelkv.StringValue("foobar"), span.Attributes()["http.server_name"])
	assert.Equal(t, otelkv.IntValue(http.StatusOK), span.Attributes()["http.status_code"])
	assert.Equal(t, otelkv.StringValue("GET"), span.Attributes()["http.method"])
	assert.Equal(t, otelkv.StringValue("/user/123"), span.Attributes()["http.target"])
	assert.Equal(t, otelkv.StringValue("/user/{id:[0-9]+}"), span.Attributes()["http.route"])

	r = httptest.NewRequest("GET", "/book/foo", nil)
	w = httptest.NewRecorder()
	container.ServeHTTP(w, r)
	spans = sr.Completed()
	require.Len(t, spans, 2)
	span = spans[1]
	assert.Equal(t, "/book/{title}", span.Name())
	assert.Equal(t, oteltrace.SpanKindServer, span.SpanKind())
	assert.Equal(t, otelkv.StringValue("foobar"), span.Attributes()["http.server_name"])
	assert.Equal(t, otelkv.IntValue(http.StatusOK), span.Attributes()["http.status_code"])
	assert.Equal(t, otelkv.StringValue("GET"), span.Attributes()["http.method"])
	assert.Equal(t, otelkv.StringValue("/book/foo"), span.Attributes()["http.target"])
	assert.Equal(t, otelkv.StringValue("/book/{title}"), span.Attributes()["http.route"])
}

func TestGetSpanNotInstrumented(t *testing.T) {
	handlerFunc := func(req *restful.Request, resp *restful.Response) {
		span := oteltrace.SpanFromContext(req.Request.Context())
		ok := !span.SpanContext().IsValid()
		assert.True(t, ok)
		resp.WriteHeader(http.StatusOK)
	}
	ws := &restful.WebService{}
	ws.Route(ws.GET("/user/{id}").To(handlerFunc))
	container := restful.NewContainer()
	container.Add(ws)

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	container.ServeHTTP(w, r)
}

func TestPropagationWithGlobalPropagators(t *testing.T) {
	provider := oteltest.NewTracerProvider()
	otel.SetTextMapPropagator(propagation.TraceContext{})

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	ctx, pspan := provider.Tracer(tracerName).Start(context.Background(), "test")
	otel.GetTextMapPropagator().Inject(ctx, r.Header)

	handlerFunc := func(req *restful.Request, resp *restful.Response) {
		span := oteltrace.SpanFromContext(req.Request.Context())
		mspan, ok := span.(*oteltest.Span)
		require.True(t, ok)
		assert.Equal(t, pspan.SpanContext().TraceID, mspan.SpanContext().TraceID)
		assert.Equal(t, pspan.SpanContext().SpanID, mspan.ParentSpanID())
		w.WriteHeader(http.StatusOK)
	}
	ws := &restful.WebService{}
	ws.Route(ws.GET("/user/{id}").To(handlerFunc))

	container := restful.NewContainer()
	container.Filter(otelrestful.OTelFilter("foobar", otelrestful.WithTracerProvider(provider)))
	container.Add(ws)

	container.ServeHTTP(w, r)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator())
}

func TestPropagationWithCustomPropagators(t *testing.T) {
	provider := oteltest.NewTracerProvider()
	b3 := b3prop.B3{}

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	ctx, pspan := provider.Tracer(tracerName).Start(context.Background(), "test")
	b3.Inject(ctx, r.Header)

	handlerFunc := func(req *restful.Request, resp *restful.Response) {
		span := oteltrace.SpanFromContext(req.Request.Context())
		mspan, ok := span.(*oteltest.Span)
		require.True(t, ok)
		assert.Equal(t, pspan.SpanContext().TraceID, mspan.SpanContext().TraceID)
		assert.Equal(t, pspan.SpanContext().SpanID, mspan.ParentSpanID())
		w.WriteHeader(http.StatusOK)
	}
	ws := &restful.WebService{}
	ws.Route(ws.GET("/user/{id}").To(handlerFunc))

	container := restful.NewContainer()
	container.Filter(otelrestful.OTelFilter("foobar",
		otelrestful.WithTracerProvider(provider),
		otelrestful.WithPropagators(b3)))
	container.Add(ws)

	container.ServeHTTP(w, r)
}

func TestMultiFilters(t *testing.T) {
	sr := new(oteltest.StandardSpanRecorder)
	provider := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))

	wrappedFunc := func(tracerName string) restful.RouteFunction {
		return func(req *restful.Request, resp *restful.Response) {
			span := oteltrace.SpanFromContext(req.Request.Context())
			_, ok := span.(*oteltest.Span)
			assert.True(t, ok)
			spanTracer := span.Tracer()
			mockTracer, ok := spanTracer.(*oteltest.Tracer)
			require.True(t, ok)
			assert.Equal(t, tracerName, mockTracer.Name)
			resp.WriteHeader(http.StatusOK)
		}
	}
	ws1 := &restful.WebService{}
	ws1.Path("/user")
	ws1.Route(ws1.GET("/{id}").
		Filter(otelrestful.OTelFilter("my-service", otelrestful.WithTracerProvider(provider))).
		To(wrappedFunc(tracerName)))
	ws1.Route(ws1.GET("/{id}/books").
		Filter(otelrestful.OTelFilter("book-service", otelrestful.WithTracerProvider(provider))).
		To(wrappedFunc(tracerName)))

	ws2 := &restful.WebService{}
	ws2.Path("/library")
	ws2.Filter(otelrestful.OTelFilter("library-service", otelrestful.WithTracerProvider(provider)))
	ws2.Route(ws2.GET("/{name}").To(wrappedFunc(tracerName)))

	container := restful.NewContainer()
	container.Add(ws1)
	container.Add(ws2)

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, r)

	r = httptest.NewRequest("GET", "/user/123/books", nil)
	w = httptest.NewRecorder()
	container.ServeHTTP(w, r)

	r = httptest.NewRequest("GET", "/library/metropolitan", nil)
	w = httptest.NewRecorder()
	container.ServeHTTP(w, r)
}
