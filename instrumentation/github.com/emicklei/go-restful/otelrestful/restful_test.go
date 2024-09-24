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
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/contrib/instrumentation/github.com/emicklei/go-restful/otelrestful"
	b3prop "go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
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

func TestWithPublicEndpoint(t *testing.T) {
	spanRecorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(spanRecorder),
	)
	remoteSpan := oteltrace.SpanContextConfig{
		TraceID: oteltrace.TraceID{0x01},
		SpanID:  oteltrace.SpanID{0x01},
		Remote:  true,
	}
	prop := propagation.TraceContext{}

	handlerFunc := func(req *restful.Request, resp *restful.Response) {
		s := oteltrace.SpanFromContext(req.Request.Context())
		sc := s.SpanContext()

		// Should be with new root trace.
		assert.True(t, sc.IsValid())
		assert.False(t, sc.IsRemote())
		assert.NotEqual(t, remoteSpan.TraceID, sc.TraceID())
	}

	ws := &restful.WebService{}
	ws.Route(ws.GET("/user/{id}").To(handlerFunc))

	container := restful.NewContainer()
	container.Filter(otelrestful.OTelFilter("test_handler",
		otelrestful.WithPublicEndpoint(),
		otelrestful.WithPropagators(prop),
		otelrestful.WithTracerProvider(provider)),
	)
	container.Add(ws)

	r, err := http.NewRequest(http.MethodGet, "http://localhost/user/123", nil)
	require.NoError(t, err)

	sc := oteltrace.NewSpanContext(remoteSpan)
	ctx := oteltrace.ContextWithSpanContext(context.Background(), sc)
	prop.Inject(ctx, propagation.HeaderCarrier(r.Header))

	rr := httptest.NewRecorder()
	container.ServeHTTP(rr, r)
	assert.Equal(t, 200, rr.Result().StatusCode) //nolint:bodyclose // False positive for httptest.ResponseRecorder: https://github.com/timakin/bodyclose/issues/59.

	// Recorded span should be linked with an incoming span context.
	assert.NoError(t, spanRecorder.ForceFlush(ctx))
	done := spanRecorder.Ended()
	require.Len(t, done, 1)
	require.Len(t, done[0].Links(), 1, "should contain link")
	require.True(t, sc.Equal(done[0].Links()[0].SpanContext), "should link incoming span context")
}

func TestWithPublicEndpointFn(t *testing.T) {
	remoteSpan := oteltrace.SpanContextConfig{
		TraceID:    oteltrace.TraceID{0x01},
		SpanID:     oteltrace.SpanID{0x01},
		TraceFlags: oteltrace.FlagsSampled,
		Remote:     true,
	}
	prop := propagation.TraceContext{}

	for _, tt := range []struct {
		name          string
		fn            func(*http.Request) bool
		handlerAssert func(*testing.T, oteltrace.SpanContext)
		spansAssert   func(*testing.T, oteltrace.SpanContext, []sdktrace.ReadOnlySpan)
	}{
		{
			name: "with the method returning true",
			fn: func(r *http.Request) bool {
				return true
			},
			handlerAssert: func(t *testing.T, sc oteltrace.SpanContext) {
				// Should be with new root trace.
				assert.True(t, sc.IsValid())
				assert.False(t, sc.IsRemote())
				assert.NotEqual(t, remoteSpan.TraceID, sc.TraceID())
			},
			spansAssert: func(t *testing.T, sc oteltrace.SpanContext, spans []sdktrace.ReadOnlySpan) {
				require.Len(t, spans, 1)
				require.Len(t, spans[0].Links(), 1, "should contain link")
				require.True(t, sc.Equal(spans[0].Links()[0].SpanContext), "should link incoming span context")
			},
		},
		{
			name: "with the method returning false",
			fn: func(r *http.Request) bool {
				return false
			},
			handlerAssert: func(t *testing.T, sc oteltrace.SpanContext) {
				// Should have remote span as parent
				assert.True(t, sc.IsValid())
				assert.False(t, sc.IsRemote())
				assert.Equal(t, remoteSpan.TraceID, sc.TraceID())
			},
			spansAssert: func(t *testing.T, _ oteltrace.SpanContext, spans []sdktrace.ReadOnlySpan) {
				require.Len(t, spans, 1)
				require.Empty(t, spans[0].Links(), "should not contain link")
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			spanRecorder := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(
				sdktrace.WithSpanProcessor(spanRecorder),
			)

			handlerFunc := func(req *restful.Request, resp *restful.Response) {
				s := oteltrace.SpanFromContext(req.Request.Context())
				tt.handlerAssert(t, s.SpanContext())
			}

			ws := &restful.WebService{}
			ws.Route(ws.GET("/user/{id}").To(handlerFunc))

			container := restful.NewContainer()
			container.Filter(otelrestful.OTelFilter("test_handler",
				otelrestful.WithPublicEndpointFn(tt.fn),
				otelrestful.WithPropagators(prop),
				otelrestful.WithTracerProvider(provider)),
			)
			container.Add(ws)

			r, err := http.NewRequest(http.MethodGet, "http://localhost/user/123", nil)
			require.NoError(t, err)

			sc := oteltrace.NewSpanContext(remoteSpan)
			ctx := oteltrace.ContextWithSpanContext(context.Background(), sc)
			prop.Inject(ctx, propagation.HeaderCarrier(r.Header))

			rr := httptest.NewRecorder()
			container.ServeHTTP(rr, r)
			assert.Equal(t, http.StatusOK, rr.Result().StatusCode) //nolint:bodyclose // False positive for httptest.ResponseRecorder: https://github.com/timakin/bodyclose/issues/59.

			// Recorded span should be linked with an incoming span context.
			assert.NoError(t, spanRecorder.ForceFlush(ctx))
			spans := spanRecorder.Ended()
			tt.spansAssert(t, sc, spans)
		})
	}
}
