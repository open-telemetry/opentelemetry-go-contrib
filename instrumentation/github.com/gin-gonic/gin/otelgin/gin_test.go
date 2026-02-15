// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Based on https://github.com/DataDog/dd-trace-go/blob/8fb554ff7cf694267f9077ae35e27ce4689ed8b6/contrib/gin-gonic/gin/gintrace_test.go

package otelgin_test

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	b3prop "go.opentelemetry.io/contrib/propagators/b3"
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
		_, _ = c.Writer.WriteString("ok")
	})
	r := httptest.NewRequest(http.MethodGet, "/ping", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	response := w.Result()
	assert.Equal(t, http.StatusOK, response.StatusCode)
}

func TestPropagationWithGlobalPropagators(t *testing.T) {
	provider := noop.NewTracerProvider()
	otel.SetTextMapPropagator(b3prop.New())

	r := httptest.NewRequest(http.MethodGet, "/user/123", http.NoBody)
	w := httptest.NewRecorder()

	ctx := t.Context()
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

	r := httptest.NewRequest(http.MethodGet, "/user/123", http.NoBody)
	w := httptest.NewRecorder()

	ctx := t.Context()
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
	router.GET("/user/:id", func(*gin.Context) {})

	r := httptest.NewRequest(http.MethodGet, "/user/123", http.NoBody)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)
	assert.Len(t, sr.Ended(), 1)
}

func TestChildSpanFromCustomTracer(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	router := gin.New()
	router.Use(otelgin.Middleware("foobar", otelgin.WithTracerProvider(provider)))
	router.GET("/user/:id", func(*gin.Context) {})

	r := httptest.NewRequest(http.MethodGet, "/user/123", http.NoBody)
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
		_, _ = c.Writer.WriteString(id)
	})

	r := httptest.NewRequest(http.MethodGet, "/user/123", http.NoBody)
	w := httptest.NewRecorder()

	// do and verify the request
	router.ServeHTTP(w, r)
	response := w.Result()
	require.Equal(t, http.StatusOK, response.StatusCode)

	// verify traces look good
	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	assert.Equal(t, "GET /user/:id", span.Name())
	assert.Equal(t, trace.SpanKindServer, span.SpanKind())
	attr := span.Attributes()
	assert.Contains(t, attr, attribute.String("server.address", "foobar"))
	assert.Contains(t, attr, attribute.Int("http.response.status_code", http.StatusOK))
	assert.Contains(t, attr, attribute.String("http.request.method", "GET"))
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
	r := httptest.NewRequest(http.MethodGet, "/server_err", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	response := w.Result()
	assert.Equal(t, http.StatusInternalServerError, response.StatusCode)

	// verify the errors and status are correct
	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	assert.Equal(t, "GET /server_err", span.Name())
	attr := span.Attributes()
	assert.Contains(t, attr, attribute.String("server.address", "foobar"))
	assert.Contains(t, attr, attribute.Int("http.response.status_code", http.StatusInternalServerError))

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

			router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", http.NoBody))

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

		router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", http.NoBody))

		require.Len(t, sr.Ended(), 1)
		assert.Equal(t, codes.Error, sr.Ended()[0].Status().Code)
		require.Len(t, sr.Ended()[0].Events(), 1)
		assert.Contains(t, sr.Ended()[0].Events()[0].Attributes, attribute.String("exception.message", "something went wrong"))
	})
}

func TestWithSpanOptions_CustomAttributesAndSpanKind(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	customAttr := attribute.String("custom.key", "custom.value")

	router := gin.New()
	router.Use(otelgin.Middleware("foobar",
		otelgin.WithTracerProvider(provider),
		otelgin.WithSpanStartOptions(trace.WithAttributes(customAttr)),
	))
	router.GET("/test", func(*gin.Context) {})

	r := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	spans := sr.Ended()
	require.Len(t, spans, 1)

	span := spans[0]
	assert.Contains(t, span.Attributes(), customAttr)
	assert.Equal(t, trace.SpanKindServer, span.SpanKind())
}

func TestSpanName(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sr),
	)

	testCases := []struct {
		method            string
		route             string
		requestPath       string
		spanNameFormatter otelgin.SpanNameFormatter
		wantSpanName      string
	}{
		// Test for standard methods
		{http.MethodGet, "/user/:id", "/user/1", nil, "GET /user/:id"},
		{http.MethodPost, "/user/:id", "/user/1", nil, "POST /user/:id"},
		{http.MethodPut, "/user/:id", "/user/1", nil, "PUT /user/:id"},
		{http.MethodPatch, "/user/:id", "/user/1", nil, "PATCH /user/:id"},
		{http.MethodDelete, "/user/:id", "/user/1", nil, "DELETE /user/:id"},
		{http.MethodConnect, "/user/:id", "/user/1", nil, "CONNECT /user/:id"},
		{http.MethodOptions, "/user/:id", "/user/1", nil, "OPTIONS /user/:id"},
		{http.MethodTrace, "/user/:id", "/user/1", nil, "TRACE /user/:id"},
		// Test for no route
		{http.MethodGet, "", "/user/1", nil, "GET"},
		// Test for invalid method
		{"INVALID", "/user/:id", "/user/1", nil, "HTTP /user/:id"},
		// Test for custom span name formatter
		{http.MethodGet, "/user/:id", "/user/1", func(c *gin.Context) string { return c.Request.URL.Path }, "/user/1"},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("method: %s, route: %s, requestPath: %s", tc.method, tc.route, tc.requestPath), func(t *testing.T) {
			defer sr.Reset()

			router := gin.New()
			router.Use(otelgin.Middleware("foobar", otelgin.WithTracerProvider(provider), otelgin.WithSpanNameFormatter(tc.spanNameFormatter)))
			router.Handle(tc.method, tc.route, func(*gin.Context) {})

			router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(tc.method, tc.requestPath, http.NoBody))

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
		otelgin.WithSpanNameFormatter(func(c *gin.Context) string {
			return c.Request.URL.Path
		}),
	),
	)
	router.GET("/user/:id", func(c *gin.Context) {
		id := c.Param("id")
		_, _ = c.Writer.WriteString(id)
	})

	r := httptest.NewRequest(http.MethodGet, "/user/123", http.NoBody)
	w := httptest.NewRecorder()

	// do and verify the request
	router.ServeHTTP(w, r)
	response := w.Result()
	require.Equal(t, http.StatusOK, response.StatusCode)

	// verify traces look good
	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	assert.Equal(t, "/user/123", span.Name())
	assert.Equal(t, trace.SpanKindServer, span.SpanKind())
	attr := span.Attributes()
	assert.Contains(t, attr, attribute.String("http.request.method", "GET"))
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
	r := httptest.NewRequest(http.MethodGet, "/hello", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	response := w.Result()
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
		router.GET("/healthcheck", func(*gin.Context) {})

		r := httptest.NewRequest(http.MethodGet, "/healthcheck", http.NoBody)
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
		router.GET("/user/:id", func(*gin.Context) {})

		r := httptest.NewRequest(http.MethodGet, "/user/123", http.NoBody)
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
		router.GET("/healthcheck", func(*gin.Context) {})

		r := httptest.NewRequest(http.MethodGet, "/healthcheck", http.NoBody)
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
		router.GET("/user/:id", func(*gin.Context) {})

		r := httptest.NewRequest(http.MethodGet, "/user/123", http.NoBody)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)
		assert.Len(t, sr.Ended(), 1)
	})
}

func TestTemporaryFormFileRemove(t *testing.T) {
	if runtime.GOOS == "windows" {
		// Windows sometimes refuses to remove a file that was just closed.
		t.Skip("https://go.dev/issue/25965")
	}
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	router := gin.New()
	router.MaxMultipartMemory = 1

	// We register three middlewares here, with the otel one in the middle.
	// When the response is returned, the post-handler execution order is:
	// MiddlewareC -> otel -> MiddlewareA (reverse of registration order).
	// In MiddlewareC, the multipart form file still exists, so opening it succeeds.
	// However, by the time MiddlewareA runs, the otel middleware has already
	// removed the temporary multipart files, so opening the file results in a
	// "file does not exist" error.

	// MiddlewareA
	router.Use(func(c *gin.Context) {
		c.Next()
		form, _ := c.MultipartForm()
		files := form.File["files"]
		require.Len(t, files, 1)
		_, err := files[0].Open()
		require.ErrorIs(t, err, fs.ErrNotExist)

	})

	router.Use(otelgin.Middleware("foobar", otelgin.WithTracerProvider(provider)))

	// MiddlewareC
	router.Use(func(c *gin.Context) {
		c.Next()
		form, _ := c.MultipartForm()
		files := form.File["files"]
		require.Len(t, files, 1)
		_, err := files[0].Open()
		require.NoError(t, err)
	})

	var fileHeader *multipart.FileHeader
	router.POST("/upload", func(c *gin.Context) {
		_, err := c.FormFile("files")
		require.NoError(t, err)
		fileHeader = c.Request.MultipartForm.File["files"][0]
		_, err = fileHeader.Open()
		require.NoError(t, err)
		c.JSON(http.StatusOK, nil)
	})

	var body bytes.Buffer

	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("files", "file")
	require.NoError(t, err)

	_, err = fw.Write([]byte("hello world"))
	require.NoError(t, err)
	err = mw.Close()
	require.NoError(t, err)
	r := httptest.NewRequest("POST", "/upload", &body)
	r.Header.Add("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)
	assert.Len(t, sr.Ended(), 1)
	require.Equal(t, http.StatusOK, w.Code)
	_, err = fileHeader.Open()
	require.ErrorIs(t, err, fs.ErrNotExist)
}

func TestMetrics(t *testing.T) {
	tests := []struct {
		name                        string
		metricAttributeExtractor    func(*http.Request) []attribute.KeyValue
		ginMetricAttributeExtractor func(*gin.Context) []attribute.KeyValue
		requestTarget               string
		wantRouteAttr               string
		wantStatus                  int64
	}{
		{
			name:                        "default",
			metricAttributeExtractor:    nil,
			ginMetricAttributeExtractor: nil,
			requestTarget:               "/user/123",
			wantRouteAttr:               "/user/:id",
			wantStatus:                  200,
		},
		{
			name:                        "request target not exist",
			metricAttributeExtractor:    nil,
			ginMetricAttributeExtractor: nil,
			requestTarget:               "/abc/123",
			wantStatus:                  404,
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
			ginMetricAttributeExtractor: func(*gin.Context) []attribute.KeyValue {
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

			router := gin.New()
			router.Use(otelgin.Middleware("foobar",
				otelgin.WithMeterProvider(meterProvider),
				otelgin.WithMetricAttributeFn(tt.metricAttributeExtractor),
				otelgin.WithGinMetricAttributeFn(tt.ginMetricAttributeExtractor),
			))
			router.GET("/user/:id", func(c *gin.Context) {
				id := c.Param("id")
				assert.Equal(t, "123", id)
				_, _ = c.Writer.WriteString(id)
			})

			r := httptest.NewRequest(http.MethodGet, tt.requestTarget, http.NoBody)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = r
			router.ServeHTTP(w, r)

			// verify metrics
			rm := metricdata.ResourceMetrics{}
			require.NoError(t, reader.Collect(t.Context(), &rm))

			require.Len(t, rm.ScopeMetrics, 1)
			sm := rm.ScopeMetrics[0]
			assert.Equal(t, otelgin.ScopeName, sm.Scope.Name)
			assert.Equal(t, otelgin.Version, sm.Scope.Version)

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
			if tt.ginMetricAttributeExtractor != nil {
				attrs = append(attrs, tt.ginMetricAttributeExtractor(c)...)
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
