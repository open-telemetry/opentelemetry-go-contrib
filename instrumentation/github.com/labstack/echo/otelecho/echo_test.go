// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Based on https://github.com/DataDog/dd-trace-go/blob/8fb554ff7cf694267f9077ae35e27ce4689ed8b6/contrib/gin-gonic/gin/gintrace_test.go

package otelecho

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	b3prop "go.opentelemetry.io/contrib/propagators/b3"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
)

func TestGetSpanNotInstrumented(t *testing.T) {
	router := echo.New()
	router.GET("/ping", func(c echo.Context) error {
		// Assert we don't have a span on the context.
		span := trace.SpanFromContext(c.Request().Context())
		ok := !span.SpanContext().IsValid()
		assert.True(t, ok)
		return c.String(http.StatusOK, "ok")
	})
	r := httptest.NewRequest(http.MethodGet, "/ping", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	response := w.Result()
	assert.Equal(t, http.StatusOK, response.StatusCode)
}

func TestPropagationWithGlobalPropagators(t *testing.T) {
	provider := noop.NewTracerProvider()
	otel.SetTextMapPropagator(propagation.TraceContext{})

	r := httptest.NewRequest(http.MethodGet, "/user/123", http.NoBody)
	w := httptest.NewRecorder()

	ctx := t.Context()
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{0x01},
		SpanID:  trace.SpanID{0x01},
	})
	ctx = trace.ContextWithRemoteSpanContext(ctx, sc)
	ctx, _ = provider.Tracer(ScopeName).Start(ctx, "test")
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(r.Header))

	router := echo.New()
	router.Use(Middleware("foobar", WithTracerProvider(provider)))
	router.GET("/user/:id", func(c echo.Context) error {
		span := trace.SpanFromContext(c.Request().Context())
		assert.Equal(t, sc.TraceID(), span.SpanContext().TraceID())
		assert.Equal(t, sc.SpanID(), span.SpanContext().SpanID())
		return c.NoContent(http.StatusOK)
	})

	router.ServeHTTP(w, r)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator())
	assert.Equal(t, http.StatusOK, w.Result().StatusCode, "should call the 'user' handler")
}

func TestPropagationWithCustomPropagators(t *testing.T) {
	provider := noop.NewTracerProvider()

	b3 := b3prop.New()

	r := httptest.NewRequest(http.MethodGet, "/user/123", http.NoBody)
	w := httptest.NewRecorder()

	ctx := t.Context()
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{0x01},
		SpanID:  trace.SpanID{0x01},
	})
	ctx = trace.ContextWithRemoteSpanContext(ctx, sc)
	ctx, _ = provider.Tracer(ScopeName).Start(ctx, "test")
	b3.Inject(ctx, propagation.HeaderCarrier(r.Header))

	router := echo.New()
	router.Use(Middleware("foobar", WithTracerProvider(provider), WithPropagators(b3)))
	router.GET("/user/:id", func(c echo.Context) error {
		span := trace.SpanFromContext(c.Request().Context())
		assert.Equal(t, sc.TraceID(), span.SpanContext().TraceID())
		assert.Equal(t, sc.SpanID(), span.SpanContext().SpanID())
		return c.NoContent(http.StatusOK)
	})

	router.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode, "should call the 'user' handler")
}

func TestSkipper(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/ping", http.NoBody)
	w := httptest.NewRecorder()

	skipper := func(c echo.Context) bool {
		return c.Request().RequestURI == "/ping"
	}

	router := echo.New()
	router.Use(Middleware("foobar", WithSkipper(skipper)))
	router.GET("/ping", func(c echo.Context) error {
		span := trace.SpanFromContext(c.Request().Context())
		assert.False(t, span.SpanContext().HasSpanID())
		assert.False(t, span.SpanContext().HasTraceID())
		return c.NoContent(http.StatusOK)
	})

	router.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode, "should call the 'ping' handler")
}

func TestMetrics(t *testing.T) {
	tests := []struct {
		name                         string
		metricAttributeExtractor     func(*http.Request) []attribute.KeyValue
		echoMetricAttributeExtractor func(echo.Context) []attribute.KeyValue
		requestTarget                string
		wantRouteAttr                string
		wantStatus                   int64
	}{
		{
			name:                         "default",
			metricAttributeExtractor:     nil,
			echoMetricAttributeExtractor: nil,
			requestTarget:                "/user/123",
			wantRouteAttr:                "/user/:id",
			wantStatus:                   200,
		},
		{
			name:                         "request target not exist",
			metricAttributeExtractor:     nil,
			echoMetricAttributeExtractor: nil,
			requestTarget:                "/abc/123",
			wantStatus:                   404,
		},
		{
			name: "with metric attributes callback",
			metricAttributeExtractor: func(r *http.Request) []attribute.KeyValue {
				return []attribute.KeyValue{
					attribute.String("key1", "value1"),
					attribute.String("key2", "value"),
					attribute.String("method", strings.ToUpper(r.Method)),
				}
			},
			echoMetricAttributeExtractor: func(_ echo.Context) []attribute.KeyValue {
				return []attribute.KeyValue{
					attribute.String("key3", "value3"),
				}
			},
			requestTarget: "/user/123",
			wantRouteAttr: "/user/:id",
			wantStatus:    200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := sdkmetric.NewManualReader()
			meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

			e := echo.New()
			e.Use(Middleware("foobar",
				WithMeterProvider(meterProvider),
				WithMetricAttributeFn(tt.metricAttributeExtractor),
				WithEchoMetricAttributeFn(tt.echoMetricAttributeExtractor),
			))
			e.GET("/user/:id", func(c echo.Context) error {
				id := c.Param("id")
				assert.Equal(t, "123", id)
				return c.String(http.StatusOK, id)
			})

			r := httptest.NewRequest(http.MethodGet, tt.requestTarget, http.NoBody)
			w := httptest.NewRecorder()
			e.ServeHTTP(w, r)

			// verify metrics
			rm := metricdata.ResourceMetrics{}
			require.NoError(t, reader.Collect(t.Context(), &rm))

			require.Len(t, rm.ScopeMetrics, 1)
			sm := rm.ScopeMetrics[0]
			assert.Equal(t, ScopeName, sm.Scope.Name)
			assert.Equal(t, Version, sm.Scope.Version)

			attrs := []attribute.KeyValue{
				attribute.String("http.request.method", "GET"),
				attribute.Int64("http.response.status_code", tt.wantStatus),
				attribute.String("network.protocol.name", "http"),
				attribute.String("network.protocol.version", fmt.Sprintf("1.%d", r.ProtoMinor)),
				attribute.String("server.address", "foobar"),
				attribute.String("url.scheme", "http"),
			}
			if tt.wantRouteAttr != "" {
				attrs = append(attrs, attribute.String("http.route", tt.wantRouteAttr))
			}

			if tt.metricAttributeExtractor != nil {
				attrs = append(attrs, tt.metricAttributeExtractor(r)...)
			}
			if tt.echoMetricAttributeExtractor != nil {
				// Create a mock context to get echo attributes
				mockCtx := echo.New().NewContext(r, httptest.NewRecorder())
				mockCtx.SetParamNames("id")
				mockCtx.SetParamValues("123")
				mockCtx.SetPath("/user/:id")
				attrs = append(attrs, tt.echoMetricAttributeExtractor(mockCtx)...)
			}

			metricdatatest.AssertEqual(t, metricdata.Metrics{
				Name:        "http.server.request.body.size",
				Description: "Size of HTTP server request bodies.",
				Unit:        "By",
				Data: metricdata.Histogram[int64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.HistogramDataPoint[int64]{
						{
							Attributes: attribute.NewSet(attrs...),
						},
					},
				},
			}, sm.Metrics[0], metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue(), metricdatatest.IgnoreExemplars())

			metricdatatest.AssertEqual(t, metricdata.Metrics{
				Name:        "http.server.response.body.size",
				Description: "Size of HTTP server response bodies.",
				Unit:        "By",
				Data: metricdata.Histogram[int64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.HistogramDataPoint[int64]{
						{
							Attributes: attribute.NewSet(attrs...),
						},
					},
				},
			}, sm.Metrics[1], metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue(), metricdatatest.IgnoreExemplars())

			metricdatatest.AssertEqual(t, metricdata.Metrics{
				Name:        "http.server.request.duration",
				Description: "Duration of HTTP server requests.",
				Unit:        "s",
				Data: metricdata.Histogram[float64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.HistogramDataPoint[float64]{
						{
							Attributes: attribute.NewSet(attrs...),
						},
					},
				},
			}, sm.Metrics[2], metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue(), metricdatatest.IgnoreExemplars())
		})
	}
}

func TestWithMetricAttributeFn(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	e := echo.New()
	e.Use(Middleware("test-service",
		WithMeterProvider(meterProvider),
		WithMetricAttributeFn(func(r *http.Request) []attribute.KeyValue {
			return []attribute.KeyValue{
				attribute.String("custom.header", r.Header.Get("X-Test-Header")),
			}
		}),
	))

	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "test response")
	})

	r := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	r.Header.Set("X-Test-Header", "test-value")
	w := httptest.NewRecorder()
	e.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)

	// verify metrics
	rm := metricdata.ResourceMetrics{}
	require.NoError(t, reader.Collect(t.Context(), &rm))
	require.Len(t, rm.ScopeMetrics, 1)
	sm := rm.ScopeMetrics[0]
	require.Len(t, sm.Metrics, 3)

	// Check that custom attribute is present
	found := false
	for _, metric := range sm.Metrics {
		if metric.Name == "http.server.request.duration" {
			histogram := metric.Data.(metricdata.Histogram[float64])
			require.Len(t, histogram.DataPoints, 1)
			attrs := histogram.DataPoints[0].Attributes.ToSlice()
			for _, attr := range attrs {
				if attr.Key == "custom.header" && attr.Value.AsString() == "test-value" {
					found = true
					break
				}
			}
		}
	}
	assert.True(t, found, "custom attribute should be found in metrics")
}

func TestWithEchoMetricAttributeFn(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	e := echo.New()
	e.Use(Middleware("test-service",
		WithMeterProvider(meterProvider),
		WithEchoMetricAttributeFn(func(c echo.Context) []attribute.KeyValue {
			return []attribute.KeyValue{
				// avoid high cardinality metrics in production code
				attribute.String("echo.param.id", c.Param("id")),
				attribute.String("echo.path", c.Path()),
			}
		}),
	))

	e.GET("/user/:id", func(c echo.Context) error {
		return c.String(http.StatusOK, "user: "+c.Param("id"))
	})

	r := httptest.NewRequest(http.MethodGet, "/user/456", http.NoBody)
	w := httptest.NewRecorder()
	e.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)

	// verify metrics
	rm := metricdata.ResourceMetrics{}
	require.NoError(t, reader.Collect(t.Context(), &rm))
	require.Len(t, rm.ScopeMetrics, 1)
	sm := rm.ScopeMetrics[0]
	require.Len(t, sm.Metrics, 3)

	// Check that custom attributes are present
	foundID := false
	foundPath := false
	for _, metric := range sm.Metrics {
		if metric.Name == "http.server.request.duration" {
			histogram := metric.Data.(metricdata.Histogram[float64])
			require.Len(t, histogram.DataPoints, 1)
			attrs := histogram.DataPoints[0].Attributes.ToSlice()
			for _, attr := range attrs {
				if attr.Key == "echo.param.id" && attr.Value.AsString() == "456" {
					foundID = true
				}
				if attr.Key == "echo.path" && attr.Value.AsString() == "/user/:id" {
					foundPath = true
				}
			}
		}
	}
	assert.True(t, foundID, "echo param id attribute should be found")
	assert.True(t, foundPath, "echo path attribute should be found")
}

func TestWithOnError(t *testing.T) {
	tests := []struct {
		name              string
		opt               Option
		wantHandlerCalled int
	}{
		{
			name:              "without WithOnError option (default)",
			opt:               nil,
			wantHandlerCalled: 2,
		},
		{
			name:              "nil WithOnError option",
			opt:               WithOnError(nil),
			wantHandlerCalled: 2,
		},
		{
			name: "custom WithOnError with c.Error call",
			opt: WithOnError(func(c echo.Context, err error) {
				err = fmt.Errorf("call from OnError: %w", err)
				c.Error(err)
			}),
			wantHandlerCalled: 2,
		},
		{
			name: "custom onError without c.Error call",
			opt: WithOnError(func(_ echo.Context, err error) {
				t.Logf("Inside custom OnError: %v", err)
			}),
			wantHandlerCalled: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/ping", http.NoBody)
			w := httptest.NewRecorder()

			router := echo.New()
			if tt.opt != nil {
				router.Use(Middleware("foobar", tt.opt))
			} else {
				router.Use(Middleware("foobar"))
			}

			router.GET("/ping", func(_ echo.Context) error {
				return assert.AnError
			})

			handlerCalled := 0
			router.HTTPErrorHandler = func(err error, c echo.Context) {
				handlerCalled++
				assert.ErrorIs(t, err, assert.AnError, "test error is expected in error handler")
				assert.NoError(t, c.NoContent(http.StatusTeapot))
			}

			router.ServeHTTP(w, r)
			assert.Equal(t, http.StatusTeapot, w.Result().StatusCode, "should call the 'ping' handler")
			assert.Equal(t, tt.wantHandlerCalled, handlerCalled, "handler called times mismatch")
		})
	}
}
