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

package restful_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emicklei/go-restful/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	restfultrace "go.opentelemetry.io/contrib/instrumentation/github.com/emicklei/go-restful"

	mocktrace "go.opentelemetry.io/contrib/internal/trace"
	otelglobal "go.opentelemetry.io/otel/api/global"
	otelpropagation "go.opentelemetry.io/otel/api/propagation"
	oteltrace "go.opentelemetry.io/otel/api/trace"
	otelkv "go.opentelemetry.io/otel/label"
)

func TestChildSpanFromGlobalTracer(t *testing.T) {
	otelglobal.SetTraceProvider(&mocktrace.Provider{})

	handlerFunc := func(req *restful.Request, resp *restful.Response) {
		span := oteltrace.SpanFromContext(req.Request.Context())
		_, ok := span.(*mocktrace.Span)
		assert.True(t, ok)
		spanTracer := span.Tracer()
		mockTracer, ok := spanTracer.(*mocktrace.Tracer)
		require.True(t, ok)
		assert.Equal(t, "go.opentelemetry.io/contrib/instrumentation/github.com/emicklei/go-restful", mockTracer.Name)
		resp.WriteHeader(http.StatusOK)
	}
	ws := &restful.WebService{}
	ws.Route(ws.GET("/user/{id}").To(handlerFunc).
		Returns(200, "OK", nil).
		Returns(404, "Not Found", nil))
	container := restful.NewContainer()
	container.Filter(restfultrace.OTelFilter("my-service"))
	container.Add(ws)

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	container.ServeHTTP(w, r)
}

func TestChildSpanFromCustomTracer(t *testing.T) {
	tracer := mocktrace.NewTracer("test-tracer")

	handlerFunc := func(req *restful.Request, resp *restful.Response) {
		span := oteltrace.SpanFromContext(req.Request.Context())
		_, ok := span.(*mocktrace.Span)
		assert.True(t, ok)
		spanTracer := span.Tracer()
		mockTracer, ok := spanTracer.(*mocktrace.Tracer)
		require.True(t, ok)
		assert.Equal(t, "test-tracer", mockTracer.Name)
		resp.WriteHeader(http.StatusOK)
	}
	ws := &restful.WebService{}
	ws.Route(ws.GET("/user/{id}").To(handlerFunc))

	container := restful.NewContainer()
	container.Filter(restfultrace.OTelFilter("my-service", restfultrace.WithTracer(tracer)))
	container.Add(ws)

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	container.ServeHTTP(w, r)
}

func TestChildSpanNames(t *testing.T) {
	tracer := mocktrace.NewTracer("test-tracer")

	handlerFunc := func(req *restful.Request, resp *restful.Response) {
		resp.WriteHeader(http.StatusOK)
	}
	ws := &restful.WebService{}
	ws.Route(ws.GET("/user/{id:[0-9]+}").To(handlerFunc))

	container := restful.NewContainer()
	container.Filter(restfultrace.OTelFilter("foobar", restfultrace.WithTracer(tracer)))
	container.Add(ws)

	ws.Route(ws.GET("/book/{title}").To(func(req *restful.Request, resp *restful.Response) {
		_, _ = resp.Write(([]byte)("ok"))
	}))

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	container.ServeHTTP(w, r)
	spans := tracer.EndedSpans()
	require.Len(t, spans, 1)
	span := spans[0]
	assert.Equal(t, "/user/{id:[0-9]+}", span.Name)
	assert.Equal(t, oteltrace.SpanKindServer, span.Kind)
	assert.Equal(t, otelkv.StringValue("foobar"), span.Attributes["http.server_name"])
	assert.Equal(t, otelkv.IntValue(http.StatusOK), span.Attributes["http.status_code"])
	assert.Equal(t, otelkv.StringValue("GET"), span.Attributes["http.method"])
	assert.Equal(t, otelkv.StringValue("/user/123"), span.Attributes["http.target"])
	assert.Equal(t, otelkv.StringValue("/user/{id:[0-9]+}"), span.Attributes["http.route"])

	r = httptest.NewRequest("GET", "/book/foo", nil)
	w = httptest.NewRecorder()
	container.ServeHTTP(w, r)
	spans = tracer.EndedSpans()
	require.Len(t, spans, 1)
	span = spans[0]
	assert.Equal(t, "/book/{title}", span.Name)
	assert.Equal(t, oteltrace.SpanKindServer, span.Kind)
	assert.Equal(t, otelkv.StringValue("foobar"), span.Attributes["http.server_name"])
	assert.Equal(t, otelkv.IntValue(http.StatusOK), span.Attributes["http.status_code"])
	assert.Equal(t, otelkv.StringValue("GET"), span.Attributes["http.method"])
	assert.Equal(t, otelkv.StringValue("/book/foo"), span.Attributes["http.target"])
	assert.Equal(t, otelkv.StringValue("/book/{title}"), span.Attributes["http.route"])
}

func TestGetSpanNotInstrumented(t *testing.T) {
	handlerFunc := func(req *restful.Request, resp *restful.Response) {
		span := oteltrace.SpanFromContext(req.Request.Context())
		_, ok := span.(oteltrace.NoopSpan)
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
	tracer := mocktrace.NewTracer("test-tracer")

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	ctx, pspan := tracer.Start(context.Background(), "test")
	otelpropagation.InjectHTTP(ctx, otelglobal.Propagators(), r.Header)

	handlerFunc := func(req *restful.Request, resp *restful.Response) {
		span := oteltrace.SpanFromContext(req.Request.Context())
		mspan, ok := span.(*mocktrace.Span)
		require.True(t, ok)
		assert.Equal(t, pspan.SpanContext().TraceID, mspan.SpanContext().TraceID)
		assert.Equal(t, pspan.SpanContext().SpanID, mspan.ParentSpanID)
		w.WriteHeader(http.StatusOK)
	}
	ws := &restful.WebService{}
	ws.Route(ws.GET("/user/{id}").To(handlerFunc))

	container := restful.NewContainer()
	container.Filter(restfultrace.OTelFilter("foobar", restfultrace.WithTracer(tracer)))
	container.Add(ws)

	container.ServeHTTP(w, r)
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

	handlerFunc := func(req *restful.Request, resp *restful.Response) {
		span := oteltrace.SpanFromContext(req.Request.Context())
		mspan, ok := span.(*mocktrace.Span)
		require.True(t, ok)
		assert.Equal(t, pspan.SpanContext().TraceID, mspan.SpanContext().TraceID)
		assert.Equal(t, pspan.SpanContext().SpanID, mspan.ParentSpanID)
		w.WriteHeader(http.StatusOK)
	}
	ws := &restful.WebService{}
	ws.Route(ws.GET("/user/{id}").To(handlerFunc))

	container := restful.NewContainer()
	container.Filter(restfultrace.OTelFilter("foobar",
		restfultrace.WithTracer(tracer),
		restfultrace.WithPropagators(props)))
	container.Add(ws)

	container.ServeHTTP(w, r)
}

func TestMultiFilters(t *testing.T) {
	tracer1 := mocktrace.NewTracer("tracer1")
	tracer2 := mocktrace.NewTracer("tracer2")
	tracer3 := mocktrace.NewTracer("tracer3")

	wrappedFunc := func(tracerName string) restful.RouteFunction {
		return func(req *restful.Request, resp *restful.Response) {
			span := oteltrace.SpanFromContext(req.Request.Context())
			_, ok := span.(*mocktrace.Span)
			assert.True(t, ok)
			spanTracer := span.Tracer()
			mockTracer, ok := spanTracer.(*mocktrace.Tracer)
			require.True(t, ok)
			assert.Equal(t, tracerName, mockTracer.Name)
			resp.WriteHeader(http.StatusOK)
		}
	}
	ws1 := &restful.WebService{}
	ws1.Path("/user")
	ws1.Route(ws1.GET("/{id}").
		Filter(restfultrace.OTelFilter("my-service", restfultrace.WithTracer(tracer1))).
		To(wrappedFunc("tracer1")))
	ws1.Route(ws1.GET("/{id}/books").
		Filter(restfultrace.OTelFilter("book-service", restfultrace.WithTracer(tracer2))).
		To(wrappedFunc("tracer2")))

	ws2 := &restful.WebService{}
	ws2.Path("/library")
	ws2.Filter(restfultrace.OTelFilter("library-service", restfultrace.WithTracer(tracer3)))
	ws2.Route(ws2.GET("/{name}").To(wrappedFunc("tracer3")))

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
