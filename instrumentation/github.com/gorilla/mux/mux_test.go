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

package mux

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mocktrace "go.opentelemetry.io/contrib/internal/trace"
	otelglobal "go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/kv"
	otelpropagation "go.opentelemetry.io/otel/api/propagation"
	oteltrace "go.opentelemetry.io/otel/api/trace"
)

func TestChildSpanFromGlobalTracer(t *testing.T) {
	otelglobal.SetTraceProvider(&mocktrace.Provider{})

	router := mux.NewRouter()
	router.Use(Middleware("foobar"))
	router.HandleFunc("/user/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := oteltrace.SpanFromContext(r.Context())
		_, ok := span.(*mocktrace.Span)
		assert.True(t, ok)
		spanTracer := span.Tracer()
		mockTracer, ok := spanTracer.(*mocktrace.Tracer)
		require.True(t, ok)
		assert.Equal(t, "go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux", mockTracer.Name)
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)
}

func TestChildSpanFromCustomTracer(t *testing.T) {
	tracer := mocktrace.NewTracer("test-tracer")

	router := mux.NewRouter()
	router.Use(Middleware("foobar", WithTracer(tracer)))
	router.HandleFunc("/user/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := oteltrace.SpanFromContext(r.Context())
		_, ok := span.(*mocktrace.Span)
		assert.True(t, ok)
		spanTracer := span.Tracer()
		mockTracer, ok := spanTracer.(*mocktrace.Tracer)
		require.True(t, ok)
		assert.Equal(t, "test-tracer", mockTracer.Name)
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)
}

func TestChildSpanNames(t *testing.T) {
	tracer := mocktrace.NewTracer("test-tracer")

	router := mux.NewRouter()
	router.Use(Middleware("foobar", WithTracer(tracer)))
	router.HandleFunc("/user/{id:[0-9]+}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	router.HandleFunc("/book/{title}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(([]byte)("ok"))
	}))

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	spans := tracer.EndedSpans()
	require.Len(t, spans, 1)
	span := spans[0]
	assert.Equal(t, "/user/{id:[0-9]+}", span.Name)
	assert.Equal(t, oteltrace.SpanKindServer, span.Kind)
	assert.Equal(t, kv.StringValue("foobar"), span.Attributes["http.server_name"])
	assert.Equal(t, kv.IntValue(http.StatusOK), span.Attributes["http.status_code"])
	assert.Equal(t, kv.StringValue("GET"), span.Attributes["http.method"])
	assert.Equal(t, kv.StringValue("/user/123"), span.Attributes["http.target"])
	assert.Equal(t, kv.StringValue("/user/{id:[0-9]+}"), span.Attributes["http.route"])

	r = httptest.NewRequest("GET", "/book/foo", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	spans = tracer.EndedSpans()
	require.Len(t, spans, 1)
	span = spans[0]
	assert.Equal(t, "/book/{title}", span.Name)
	assert.Equal(t, oteltrace.SpanKindServer, span.Kind)
	assert.Equal(t, kv.StringValue("foobar"), span.Attributes["http.server_name"])
	assert.Equal(t, kv.IntValue(http.StatusOK), span.Attributes["http.status_code"])
	assert.Equal(t, kv.StringValue("GET"), span.Attributes["http.method"])
	assert.Equal(t, kv.StringValue("/book/foo"), span.Attributes["http.target"])
	assert.Equal(t, kv.StringValue("/book/{title}"), span.Attributes["http.route"])
}

func TestGetSpanNotInstrumented(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/user/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := oteltrace.SpanFromContext(r.Context())
		_, ok := span.(oteltrace.NoopSpan)
		assert.True(t, ok)
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)
}

func TestPropagationWithGlobalPropagators(t *testing.T) {
	tracer := mocktrace.NewTracer("test-tracer")

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	ctx, pspan := tracer.Start(context.Background(), "test")
	otelpropagation.InjectHTTP(ctx, otelglobal.Propagators(), r.Header)

	router := mux.NewRouter()
	router.Use(Middleware("foobar", WithTracer(tracer)))
	router.HandleFunc("/user/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := oteltrace.SpanFromContext(r.Context())
		mspan, ok := span.(*mocktrace.Span)
		require.True(t, ok)
		assert.Equal(t, pspan.SpanContext().TraceID, mspan.SpanContext().TraceID)
		assert.Equal(t, pspan.SpanContext().SpanID, mspan.ParentSpanID)
		w.WriteHeader(http.StatusOK)
	}))

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

	router := mux.NewRouter()
	router.Use(Middleware("foobar", WithTracer(tracer), WithPropagators(props)))
	router.HandleFunc("/user/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := oteltrace.SpanFromContext(r.Context())
		mspan, ok := span.(*mocktrace.Span)
		require.True(t, ok)
		assert.Equal(t, pspan.SpanContext().TraceID, mspan.SpanContext().TraceID)
		assert.Equal(t, pspan.SpanContext().SpanID, mspan.ParentSpanID)
		w.WriteHeader(http.StatusOK)
	}))

	router.ServeHTTP(w, r)
}
