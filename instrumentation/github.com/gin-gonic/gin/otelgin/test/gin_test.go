// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Based on https://github.com/DataDog/dd-trace-go/blob/8fb554ff7cf694267f9077ae35e27ce4689ed8b6/contrib/gin-gonic/gin/gintrace_test.go

package test

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	b3prop "go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func init() {
	gin.SetMode(gin.ReleaseMode) // silence annoying log msgs
}

func TestGetSpanNotInstrumented(t *testing.T) {
	router := gin.New()
	router.GET("/ping", func(c *gin.Context) {
		// Assert we don't have a span on the context.
		span := trace.SpanFromContext(c.Request.Context())
		ok := !span.SpanContext().IsValid()
		assert.True(t, ok)
		_, _ = c.Writer.Write([]byte("ok"))
	})
	r := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	response := w.Result() //nolint:bodyclose // False positive for httptest.ResponseRecorder: https://github.com/timakin/bodyclose/issues/59.
	assert.Equal(t, http.StatusOK, response.StatusCode)
}

func TestPropagationWithGlobalPropagators(t *testing.T) {
	provider := noop.NewTracerProvider()
	otel.SetTextMapPropagator(b3prop.New())

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	ctx := context.Background()
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{0x01},
		SpanID:  trace.SpanID{0x01},
	})
	ctx = trace.ContextWithRemoteSpanContext(ctx, sc)
	ctx, _ = provider.Tracer(otelgin.ScopeName).Start(ctx, "test")
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(r.Header))

	router := gin.New()
	router.Use(otelgin.Middleware("foobar", otelgin.WithTracerProvider(provider)))
	router.GET("/user/:id", func(c *gin.Context) {
		span := trace.SpanFromContext(c.Request.Context())
		assert.Equal(t, sc.TraceID(), span.SpanContext().TraceID())
		assert.Equal(t, sc.SpanID(), span.SpanContext().SpanID())
	})

	router.ServeHTTP(w, r)
}

func TestPropagationWithCustomPropagators(t *testing.T) {
	provider := noop.NewTracerProvider()
	b3 := b3prop.New()

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	ctx := context.Background()
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{0x01},
		SpanID:  trace.SpanID{0x01},
	})
	ctx = trace.ContextWithRemoteSpanContext(ctx, sc)
	ctx, _ = provider.Tracer(otelgin.ScopeName).Start(ctx, "test")
	b3.Inject(ctx, propagation.HeaderCarrier(r.Header))

	router := gin.New()
	router.Use(otelgin.Middleware("foobar", otelgin.WithTracerProvider(provider), otelgin.WithPropagators(b3)))
	router.GET("/user/:id", func(c *gin.Context) {
		span := trace.SpanFromContext(c.Request.Context())
		assert.Equal(t, sc.TraceID(), span.SpanContext().TraceID())
		assert.Equal(t, sc.SpanID(), span.SpanContext().SpanID())
	})

	router.ServeHTTP(w, r)
}

func TestChildSpanFromGlobalTracer(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr)))

	router := gin.New()
	router.Use(otelgin.Middleware("foobar"))
	router.GET("/user/:id", func(c *gin.Context) {})

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)
	assert.Len(t, sr.Ended(), 1)
}

func TestChildSpanFromCustomTracer(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	router := gin.New()
	router.Use(otelgin.Middleware("foobar", otelgin.WithTracerProvider(provider)))
	router.GET("/user/:id", func(c *gin.Context) {})

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)
	assert.Len(t, sr.Ended(), 1)
}

func TestTrace200(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	router := gin.New()
	router.Use(otelgin.Middleware("foobar", otelgin.WithTracerProvider(provider)))
	router.GET("/user/:id", func(c *gin.Context) {
		id := c.Param("id")
		_, _ = c.Writer.Write([]byte(id))
	})

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	// do and verify the request
	router.ServeHTTP(w, r)
	response := w.Result() //nolint:bodyclose // False positive for httptest.ResponseRecorder: https://github.com/timakin/bodyclose/issues/59.
	require.Equal(t, http.StatusOK, response.StatusCode)

	// verify traces look good
	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	assert.Equal(t, "/user/:id", span.Name())
	assert.Equal(t, trace.SpanKindServer, span.SpanKind())
	attr := span.Attributes()
	assert.Contains(t, attr, attribute.String("net.host.name", "foobar"))
	assert.Contains(t, attr, attribute.Int("http.status_code", http.StatusOK))
	assert.Contains(t, attr, attribute.String("http.method", "GET"))
	assert.Contains(t, attr, attribute.String("http.route", "/user/:id"))
	assert.Empty(t, span.Events())
	assert.Equal(t, codes.Unset, span.Status().Code)
	assert.Empty(t, span.Status().Description)
}

func TestError(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	// setup
	router := gin.New()
	router.Use(otelgin.Middleware("foobar", otelgin.WithTracerProvider(provider)))

	// configure a handler that returns an error and 5xx status
	// code
	router.GET("/server_err", func(c *gin.Context) {
		_ = c.Error(errors.New("oh no one"))
		_ = c.AbortWithError(http.StatusInternalServerError, errors.New("oh no two"))
	})
	r := httptest.NewRequest("GET", "/server_err", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	response := w.Result() //nolint:bodyclose // False positive for httptest.ResponseRecorder: https://github.com/timakin/bodyclose/issues/59.
	assert.Equal(t, http.StatusInternalServerError, response.StatusCode)

	// verify the errors and status are correct
	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	assert.Equal(t, "/server_err", span.Name())
	attr := span.Attributes()
	assert.Contains(t, attr, attribute.String("net.host.name", "foobar"))
	assert.Contains(t, attr, attribute.Int("http.status_code", http.StatusInternalServerError))

	// verify the error events
	events := span.Events()
	require.Len(t, events, 2)
	assert.Equal(t, "exception", events[0].Name)
	assert.Contains(t, events[0].Attributes, attribute.String("exception.type", "*errors.errorString"))
	assert.Contains(t, events[0].Attributes, attribute.String("exception.message", "oh no one"))
	assert.Equal(t, "exception", events[1].Name)
	assert.Contains(t, events[1].Attributes, attribute.String("exception.type", "*errors.errorString"))
	assert.Contains(t, events[1].Attributes, attribute.String("exception.message", "oh no two"))

	// server errors set the status
	assert.Equal(t, codes.Error, span.Status().Code)
	assert.Equal(t, "Error #01: oh no one\nError #02: oh no two\n", span.Status().Description)
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
			router := gin.New()
			router.Use(otelgin.Middleware("foobar", otelgin.WithTracerProvider(provider)))
			router.GET("/", func(c *gin.Context) {
				c.Status(tc.httpStatusCode)
			})

			router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))

			require.Len(t, sr.Ended(), 1, "should emit a span")
			assert.Equal(t, tc.wantSpanStatus, sr.Ended()[0].Status().Code, "should only set Error status for HTTP statuses >= 500")
		})
	}

	t.Run("The status code is 200, but an error is returned", func(t *testing.T) {
		sr := tracetest.NewSpanRecorder()
		provider := sdktrace.NewTracerProvider(
			sdktrace.WithSpanProcessor(sr),
		)

		router := gin.New()
		router.Use(otelgin.Middleware("foobar", otelgin.WithTracerProvider(provider)))
		router.GET("/", func(c *gin.Context) {
			_ = c.Error(errors.New("something went wrong"))
			c.JSON(http.StatusOK, nil)
		})

		router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))

		require.Len(t, sr.Ended(), 1)
		assert.Equal(t, codes.Error, sr.Ended()[0].Status().Code)
		require.Len(t, sr.Ended()[0].Events(), 1)
		assert.Contains(t, sr.Ended()[0].Events()[0].Attributes, attribute.String("exception.message", "something went wrong"))
	})
}

func TestSpanName(t *testing.T) {
	testCases := []struct {
		requestPath       string
		spanNameFormatter otelgin.SpanNameFormatter
		wantSpanName      string
	}{
		{"/user/1", nil, "/user/:id"},
		{"/user/1", func(r *http.Request) string { return r.URL.Path }, "/user/1"},
	}
	for _, tc := range testCases {
		t.Run(tc.requestPath, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider()
			provider.RegisterSpanProcessor(sr)
			router := gin.New()
			router.Use(otelgin.Middleware("foobar", otelgin.WithTracerProvider(provider), otelgin.WithSpanNameFormatter(tc.spanNameFormatter)))
			router.GET("/user/:id", func(c *gin.Context) {})

			router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", tc.requestPath, nil))

			require.Len(t, sr.Ended(), 1, "should emit a span")
			assert.Equal(t, tc.wantSpanName, sr.Ended()[0].Name(), "span name not correct")
		})
	}
}

func TestHTTPRouteWithSpanNameFormatter(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	router := gin.New()
	router.Use(otelgin.Middleware("foobar",
		otelgin.WithTracerProvider(provider),
		otelgin.WithSpanNameFormatter(func(r *http.Request) string {
			return r.URL.Path
		}),
	),
	)
	router.GET("/user/:id", func(c *gin.Context) {
		id := c.Param("id")
		_, _ = c.Writer.Write([]byte(id))
	})

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	// do and verify the request
	router.ServeHTTP(w, r)
	response := w.Result() //nolint:bodyclose // False positive for httptest.ResponseRecorder: https://github.com/timakin/bodyclose/issues/59.
	require.Equal(t, http.StatusOK, response.StatusCode)

	// verify traces look good
	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	assert.Equal(t, "/user/123", span.Name())
	assert.Equal(t, trace.SpanKindServer, span.SpanKind())
	attr := span.Attributes()
	assert.Contains(t, attr, attribute.String("http.method", "GET"))
	assert.Contains(t, attr, attribute.String("http.route", "/user/:id"))
}

func TestHTML(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	// setup
	router := gin.New()
	router.Use(otelgin.Middleware("foobar", otelgin.WithTracerProvider(provider)))

	// add a template
	tmpl := template.Must(template.New("hello").Parse("hello {{.}}"))
	router.SetHTMLTemplate(tmpl)

	// a handler with an error and make the requests
	router.GET("/hello", func(c *gin.Context) {
		otelgin.HTML(c, http.StatusOK, "hello", "world")
	})
	r := httptest.NewRequest("GET", "/hello", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	response := w.Result() //nolint:bodyclose // False positive for httptest.ResponseRecorder: https://github.com/timakin/bodyclose/issues/59.
	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Equal(t, "hello world", w.Body.String())

	// verify the errors and status are correct
	spans := sr.Ended()
	require.Len(t, spans, 2)
	var tspan sdktrace.ReadOnlySpan
	for _, s := range spans {
		// we need to pick up the span we're searching for, as the
		// order is not guaranteed within the buffer
		if s.Name() == "gin.renderer.html" {
			tspan = s
			break
		}
	}
	require.NotNil(t, tspan)
	assert.Contains(t, tspan.Attributes(), attribute.String("go.template", "hello"))
}

func TestWithFilter(t *testing.T) {
	t.Run("custom filter filtering route", func(t *testing.T) {
		sr := tracetest.NewSpanRecorder()
		otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr)))

		router := gin.New()
		f := func(req *http.Request) bool { return req.URL.Path != "/healthcheck" }
		router.Use(otelgin.Middleware("foobar", otelgin.WithFilter(f)))
		router.GET("/healthcheck", func(c *gin.Context) {})

		r := httptest.NewRequest("GET", "/healthcheck", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)
		assert.Empty(t, sr.Ended())
	})

	t.Run("custom filter not filtering route", func(t *testing.T) {
		sr := tracetest.NewSpanRecorder()
		otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr)))

		router := gin.New()
		f := func(req *http.Request) bool { return req.URL.Path != "/healthcheck" }
		router.Use(otelgin.Middleware("foobar", otelgin.WithFilter(f)))
		router.GET("/user/:id", func(c *gin.Context) {})

		r := httptest.NewRequest("GET", "/user/123", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)
		assert.Len(t, sr.Ended(), 1)
	})
}

func TestWithGinFilter(t *testing.T) {
	t.Run("custom filter filtering route", func(t *testing.T) {
		sr := tracetest.NewSpanRecorder()
		otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr)))

		router := gin.New()
		f := func(c *gin.Context) bool { return c.Request.URL.Path != "/healthcheck" }
		router.Use(otelgin.Middleware("foobar", otelgin.WithGinFilter(f)))
		router.GET("/healthcheck", func(c *gin.Context) {})

		r := httptest.NewRequest("GET", "/healthcheck", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)
		assert.Empty(t, sr.Ended())
	})

	t.Run("custom filter not filtering route", func(t *testing.T) {
		sr := tracetest.NewSpanRecorder()
		otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr)))

		router := gin.New()
		f := func(c *gin.Context) bool { return c.Request.URL.Path != "/user/:id" }
		router.Use(otelgin.Middleware("foobar", otelgin.WithGinFilter(f)))
		router.GET("/user/:id", func(c *gin.Context) {})

		r := httptest.NewRequest("GET", "/user/123", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)
		assert.Len(t, sr.Ended(), 1)
	})
}

func TestMetrics(t *testing.T) {
	tests := []struct {
		name                     string
		metricAttributeExtractor func(*http.Request) []attribute.KeyValue
	}{
		{"default", nil},
		{"with metric attributes callback", func(req *http.Request) []attribute.KeyValue {
			return []attribute.KeyValue{
				attribute.String("key1", "value1"),
				attribute.String("key2", "value"),
				attribute.String("method", strings.ToUpper(req.Method)),
			}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := sdkmetric.NewManualReader()
			meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

			router := gin.New()
			router.Use(otelgin.Middleware("foobar",
				otelgin.WithMeterProvider(meterProvider),
				otelgin.WithMetricAttributeFn(tt.metricAttributeExtractor),
			))
			router.GET("/user/:id", func(c *gin.Context) {
				id := c.Param("id")
				assert.Equal(t, "123", id)
				_, _ = c.Writer.Write([]byte(id))
			})

			r := httptest.NewRequest("GET", "/user/123", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)

			// verify metrics
			rm := metricdata.ResourceMetrics{}
			require.NoError(t, reader.Collect(context.Background(), &rm))

			require.Len(t, rm.ScopeMetrics, 1)
			sm := rm.ScopeMetrics[0]
			assert.Equal(t, otelgin.ScopeName, sm.Scope.Name)
			assert.Equal(t, otelgin.Version(), sm.Scope.Version)

			attrs := []attribute.KeyValue{
				semconv.NetHostName("foobar"),
				semconv.HTTPSchemeHTTP,
				semconv.NetProtocolName("http"),
				semconv.NetProtocolVersion(fmt.Sprintf("1.%d", r.ProtoMinor)),
				semconv.HTTPMethod(http.MethodGet),
				semconv.HTTPStatusCode(200),
			}

			if tt.metricAttributeExtractor != nil {
				attrs = append(attrs, tt.metricAttributeExtractor(r)...)
			}

			metricdatatest.AssertEqual(t, metricdata.Metrics{
				Name:        "http.server.request.size",
				Description: "Measures the size of HTTP request messages.",
				Unit:        "By",
				Data: metricdata.Sum[int64]{
					DataPoints: []metricdata.DataPoint[int64]{{
						Attributes: attribute.NewSet(attrs...), Value: 0,
					}},
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: true,
				},
			}, sm.Metrics[0], metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())

			metricdatatest.AssertEqual(t, metricdata.Metrics{
				Name:        "http.server.response.size",
				Description: "Measures the size of HTTP response messages.",
				Unit:        "By",
				Data: metricdata.Sum[int64]{
					DataPoints: []metricdata.DataPoint[int64]{{
						Attributes: attribute.NewSet(attrs...), Value: 3,
					}},
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: true,
				},
			}, sm.Metrics[1], metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())

			metricdatatest.AssertEqual(t, metricdata.Metrics{
				Name:        "http.server.duration",
				Description: "Measures the duration of inbound HTTP requests.",
				Unit:        "ms",
				Data: metricdata.Histogram[float64]{
					DataPoints:  []metricdata.HistogramDataPoint[float64]{{Attributes: attribute.NewSet(attrs...)}},
					Temporality: metricdata.CumulativeTemporality,
				},
			}, sm.Metrics[2], metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue(), metricdatatest.IgnoreExemplars())
		})
	}
}
