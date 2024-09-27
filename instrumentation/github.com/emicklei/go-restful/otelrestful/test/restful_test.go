// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/emicklei/go-restful/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/contrib/instrumentation/github.com/emicklei/go-restful/otelrestful"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestChildSpanFromGlobalTracer(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr)))

	handlerFunc := func(req *restful.Request, resp *restful.Response) {
		resp.WriteHeader(http.StatusOK)
	}
	ws := &restful.WebService{}
	ws.Route(ws.GET("/user/{id}").To(handlerFunc).
		Returns(http.StatusOK, "OK", nil).
		Returns(http.StatusNotFound, "Not Found", nil))
	container := restful.NewContainer()
	container.Filter(otelrestful.OTelFilter("my-service"))
	container.Add(ws)

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	container.ServeHTTP(w, r)
	assert.Len(t, sr.Ended(), 1)
}

func TestChildSpanFromCustomTracer(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	handlerFunc := func(req *restful.Request, resp *restful.Response) {
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
	assert.Len(t, sr.Ended(), 1)
}

func TestChildSpanNames(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

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
	spans := sr.Ended()
	require.Len(t, spans, 1)
	assertSpan(
		t,
		spans[0],
		"/user/{id:[0-9]+}",
		attribute.String("net.host.name", "foobar"),
		attribute.Int("http.status_code", http.StatusOK),
		attribute.String("http.method", "GET"),
		attribute.String("http.route", "/user/{id:[0-9]+}"),
	)

	r = httptest.NewRequest("GET", "/book/foo", nil)
	w = httptest.NewRecorder()
	container.ServeHTTP(w, r)
	spans = sr.Ended()
	require.Len(t, spans, 2)
	assertSpan(
		t,
		spans[1],
		"/book/{title}",
		attribute.String("net.host.name", "foobar"),
		attribute.Int("http.status_code", http.StatusOK),
		attribute.String("http.method", "GET"),
		attribute.String("http.route", "/book/{title}"),
	)
}

func TestMultiFilters(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	retOK := func(req *restful.Request, resp *restful.Response) { resp.WriteHeader(http.StatusOK) }
	ws1 := &restful.WebService{}
	ws1.Path("/user")
	ws1.Route(ws1.GET("/{id}").
		Filter(otelrestful.OTelFilter("my-service", otelrestful.WithTracerProvider(provider))).
		To(retOK))
	ws1.Route(ws1.GET("/{id}/books").
		Filter(otelrestful.OTelFilter("book-service", otelrestful.WithTracerProvider(provider))).
		To(retOK))

	ws2 := &restful.WebService{}
	ws2.Path("/library")
	ws2.Filter(otelrestful.OTelFilter("library-service", otelrestful.WithTracerProvider(provider)))
	ws2.Route(ws2.GET("/{name}").To(retOK))

	container := restful.NewContainer()
	container.Add(ws1)
	container.Add(ws2)

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, r)
	spans := sr.Ended()
	require.Len(t, spans, 1)
	assertSpan(t, spans[0], "/user/{id}")

	r = httptest.NewRequest("GET", "/user/123/books", nil)
	w = httptest.NewRecorder()
	container.ServeHTTP(w, r)
	spans = sr.Ended()
	require.Len(t, spans, 2)
	assertSpan(t, spans[1], "/user/{id}/books")

	r = httptest.NewRequest("GET", "/library/metropolitan", nil)
	w = httptest.NewRecorder()
	container.ServeHTTP(w, r)
	spans = sr.Ended()
	require.Len(t, spans, 3)
	assertSpan(t, spans[2], "/library/{name}")
}

func TestSpanStatus(t *testing.T) {
	testCases := []struct {
		httpStatusCode int
		wantSpanStatus codes.Code
	}{
		{http.StatusOK, codes.Unset},
		{http.StatusBadRequest, codes.Unset},
		{http.StatusInternalServerError, codes.Error},
	}
	for _, tc := range testCases {
		t.Run(strconv.Itoa(tc.httpStatusCode), func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider()
			provider.RegisterSpanProcessor(sr)
			handlerFunc := func(req *restful.Request, resp *restful.Response) {
				resp.WriteHeader(tc.httpStatusCode)
			}
			ws := &restful.WebService{}
			ws.Route(ws.GET("/").To(handlerFunc))
			container := restful.NewContainer()
			container.Filter(otelrestful.OTelFilter("my-service", otelrestful.WithTracerProvider(provider)))
			container.Add(ws)

			container.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))

			require.Len(t, sr.Ended(), 1, "should emit a span")
			assert.Equal(t, tc.wantSpanStatus, sr.Ended()[0].Status().Code, "should only set Error status for HTTP statuses >= 500")
		})
	}
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

func assertSpan(t *testing.T, span sdktrace.ReadOnlySpan, name string, attrs ...attribute.KeyValue) {
	assert.Equal(t, name, span.Name())
	assert.Equal(t, oteltrace.SpanKindServer, span.SpanKind())

	gotA := span.Attributes()
	for _, a := range attrs {
		assert.Contains(t, gotA, a)
	}
}
