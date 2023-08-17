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

// Based on https://github.com/DataDog/dd-trace-go/blob/8fb554ff7cf694267f9077ae35e27ce4689ed8b6/contrib/gin-gonic/gin/gintrace_test.go

package test

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"

	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func init() {
	gin.SetMode(gin.ReleaseMode) // silence annoying log msgs
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
	response := w.Result()
	require.Equal(t, http.StatusOK, response.StatusCode)

	// verify traces look good
	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	assert.Equal(t, "/user/:id", span.Name())
	assert.Equal(t, oteltrace.SpanKindServer, span.SpanKind())
	attr := span.Attributes()
	assert.Contains(t, attr, attribute.String("net.host.name", "foobar"))
	assert.Contains(t, attr, attribute.Int("http.status_code", http.StatusOK))
	assert.Contains(t, attr, attribute.String("http.method", "GET"))
	assert.Contains(t, attr, attribute.String("http.route", "/user/:id"))
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
		_ = c.AbortWithError(http.StatusInternalServerError, errors.New("oh no"))
	})
	r := httptest.NewRequest("GET", "/server_err", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	response := w.Result()
	assert.Equal(t, http.StatusInternalServerError, response.StatusCode)

	// verify the errors and status are correct
	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	assert.Equal(t, "/server_err", span.Name())
	attr := span.Attributes()
	assert.Contains(t, attr, attribute.String("net.host.name", "foobar"))
	assert.Contains(t, attr, attribute.Int("http.status_code", http.StatusInternalServerError))
	assert.Contains(t, attr, attribute.String("gin.errors", "Error #01: oh no\n"))
	// server errors set the status
	assert.Equal(t, codes.Error, span.Status().Code)
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
			assert.Equal(t, sr.Ended()[0].Status().Code, tc.wantSpanStatus, "should only set Error status for HTTP statuses >= 500")
		})
	}
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
			assert.Equal(t, sr.Ended()[0].Name(), tc.wantSpanName, "span name not correct")
		})
	}
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
		router.GET("/healthcheck", func(c *gin.Context) {})

		r := httptest.NewRequest("GET", "/healthcheck", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)
		assert.Len(t, sr.Ended(), 0)
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

func TestMetrics(t *testing.T) {
	tests := []struct {
		name       string
		request    *http.Request
		httpStatus int
		attrs      []attribute.KeyValue
	}{
		{
			name:       "GET 200",
			request:    httptest.NewRequest("GET", "/user/123", nil),
			httpStatus: 200,
			attrs: []attribute.KeyValue{
				semconv.HTTPMethod("GET"),
				semconv.HTTPScheme("http"),
				semconv.HTTPFlavorHTTP11,
				semconv.HTTPRoute("/user/:id"),
				semconv.HTTPStatusCode(200),
				semconv.NetHostName("foobar"),
			},
		},
		{
			name:       "POST 200",
			request:    httptest.NewRequest("POST", "/user/234", nil),
			httpStatus: 200,
			attrs: []attribute.KeyValue{
				semconv.HTTPMethod("POST"),
				semconv.HTTPScheme("http"),
				semconv.HTTPFlavorHTTP11,
				semconv.HTTPRoute("/user/:id"),
				semconv.HTTPStatusCode(200),
				semconv.NetHostName("foobar"),
			},
		},
		{
			name:       "GET 302",
			request:    httptest.NewRequest("GET", "/user/123", nil),
			httpStatus: 302,
			attrs: []attribute.KeyValue{
				semconv.HTTPMethod("GET"),
				semconv.HTTPScheme("http"),
				semconv.HTTPFlavorHTTP11,
				semconv.HTTPRoute("/user/:id"),
				semconv.HTTPStatusCode(302),
				semconv.NetHostName("foobar"),
			},
		},
		{
			name:       "404",
			request:    httptest.NewRequest("GET", "/whereami", nil),
			httpStatus: 404,
			attrs: []attribute.KeyValue{
				semconv.HTTPMethod("GET"),
				semconv.HTTPScheme("http"),
				semconv.HTTPFlavorHTTP11,
				semconv.HTTPStatusCode(404),
				semconv.NetHostName("foobar"),
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			reader := sdkmetric.NewManualReader()
			provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

			router := gin.New()
			router.Use(otelgin.Middleware("foobar", otelgin.WithMeterProvider(provider)))
			router.Any("/user/:id", func(c *gin.Context) {
				id := c.Param("id")
				c.Status(tc.httpStatus)
				_, _ = c.Writer.Write([]byte(id))
			})

			w := httptest.NewRecorder()

			// do and verify the request
			router.ServeHTTP(w, tc.request)
			response := w.Result()
			require.Equal(t, tc.httpStatus, response.StatusCode)

			checkMetrics(t, reader, tc.attrs)
		})
	}
}

func checkMetrics(t *testing.T, reader sdkmetric.Reader, expAttrs []attribute.KeyValue) {
	t.Helper()

	rm := metricdata.ResourceMetrics{}
	err := reader.Collect(context.Background(), &rm)
	assert.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	require.Len(t, rm.ScopeMetrics[0].Metrics, 1)
	require.IsType(t, rm.ScopeMetrics[0].Metrics[0].Data, metricdata.Histogram[int64]{})
	data := rm.ScopeMetrics[0].Metrics[0].Data.(metricdata.Histogram[int64])

	for _, dpt := range data.DataPoints {
		attr := dpt.Attributes.ToSlice()
		assert.ElementsMatch(t, expAttrs, attr)
	}
}
