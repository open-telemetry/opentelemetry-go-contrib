// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelrestful_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emicklei/go-restful/v3"
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/contrib/instrumentation/github.com/emicklei/go-restful/otelrestful"
	b3prop "go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

const tracerName = "go.opentelemetry.io/contrib/instrumentation/github.com/emicklei/go-restful/otelrestful"

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
	defer func(p propagation.TextMapPropagator) {
		otel.SetTextMapPropagator(p)
	}(otel.GetTextMapPropagator())
	provider := noop.NewTracerProvider()
	otel.SetTextMapPropagator(propagation.TraceContext{})

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	ctx := context.Background()
	sc := oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID: oteltrace.TraceID{0x01},
		SpanID:  oteltrace.SpanID{0x01},
	})
	ctx = oteltrace.ContextWithRemoteSpanContext(ctx, sc)
	ctx, _ = provider.Tracer(tracerName).Start(ctx, "test")
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(r.Header))

	handlerFunc := func(req *restful.Request, resp *restful.Response) {
		span := oteltrace.SpanFromContext(req.Request.Context())
		assert.Equal(t, sc.TraceID(), span.SpanContext().TraceID())
		assert.Equal(t, sc.SpanID(), span.SpanContext().SpanID())
		w.WriteHeader(http.StatusOK)
	}
	ws := &restful.WebService{}
	ws.Route(ws.GET("/user/{id}").To(handlerFunc))

	container := restful.NewContainer()
	container.Filter(otelrestful.OTelFilter("foobar", otelrestful.WithTracerProvider(provider)))
	container.Add(ws)

	container.ServeHTTP(w, r)
}

func TestPropagationWithCustomPropagators(t *testing.T) {
	provider := noop.NewTracerProvider()
	b3 := b3prop.New()

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	ctx := context.Background()
	sc := oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID: oteltrace.TraceID{0x01},
		SpanID:  oteltrace.SpanID{0x01},
	})
	ctx = oteltrace.ContextWithRemoteSpanContext(ctx, sc)
	ctx, _ = provider.Tracer(tracerName).Start(ctx, "test")
	b3.Inject(ctx, propagation.HeaderCarrier(r.Header))

	handlerFunc := func(req *restful.Request, resp *restful.Response) {
		span := oteltrace.SpanFromContext(req.Request.Context())
		assert.Equal(t, sc.TraceID(), span.SpanContext().TraceID())
		assert.Equal(t, sc.SpanID(), span.SpanContext().SpanID())
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
