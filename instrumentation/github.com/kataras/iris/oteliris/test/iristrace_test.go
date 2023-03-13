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

// Based on https://github.com/DataDog/dd-trace-go/blob/8fb554ff7cf694267f9077ae35e27ce4689ed8b6/contrib/kataras/iris/gintrace_test.go

package test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/kataras/iris/v12"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/contrib/instrumentation/github.com/kataras/iris/oteliris"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestChildSpanFromGlobalTracer(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr)))

	router := iris.New()
	router.Use(oteliris.Middleware("foobar"))
	router.Get("/user/{id}", func(ctx iris.Context) {})

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	if err := router.Build(); err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(w, r)
	assert.Len(t, sr.Ended(), 1)
}

func TestChildSpanFromCustomTracer(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	router := iris.New()
	router.Use(oteliris.Middleware("foobar", oteliris.WithTracerProvider(provider)))
	router.Get("/user/{id}", func(ctx iris.Context) {})

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	if err := router.Build(); err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(w, r)
	assert.Len(t, sr.Ended(), 1)
}

func TestTrace200(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	router := iris.New()
	router.Use(oteliris.Middleware("foobar", oteliris.WithTracerProvider(provider)))
	router.Get("/user/{id}", func(ctx iris.Context) {
		id := ctx.Params().Get("id")
		_, _ = ctx.Write([]byte(id))
	})

	r := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	// do and verify the request
	if err := router.Build(); err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(w, r)
	response := w.Result()
	require.Equal(t, iris.StatusOK, response.StatusCode)

	// verify traces look good
	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	assert.Equal(t, "/user/{id}", span.Name())
	assert.Equal(t, oteltrace.SpanKindServer, span.SpanKind())
	attr := span.Attributes()
	assert.Contains(t, attr, attribute.String("net.host.name", "foobar"))
	assert.Contains(t, attr, attribute.Int("http.status_code", iris.StatusOK))
	assert.Contains(t, attr, attribute.String("http.method", "GET"))
	assert.Contains(t, attr, attribute.String("http.route", "/user/{id}"))
}

func TestError(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	// setup
	router := iris.New()
	router.Use(oteliris.Middleware("foobar", oteliris.WithTracerProvider(provider)))

	// configure a handler that returns an error and 5xx status
	// code
	router.Get("/server_err", func(ctx iris.Context) {
		ctx.StopWithError(iris.StatusInternalServerError, errors.New("oh no"))
	})
	r := httptest.NewRequest("GET", "/server_err", nil)
	w := httptest.NewRecorder()

	if err := router.Build(); err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(w, r)
	response := w.Result()
	assert.Equal(t, iris.StatusInternalServerError, response.StatusCode)

	// verify the errors and status are correct
	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	assert.Equal(t, "/server_err", span.Name())
	attr := span.Attributes()
	assert.Contains(t, attr, attribute.String("net.host.name", "foobar"))
	assert.Contains(t, attr, attribute.Int("http.status_code", iris.StatusInternalServerError))
	assert.Contains(t, attr, attribute.String("iris.errors", "oh no"))
	// server errors set the status
	assert.Equal(t, codes.Error, span.Status().Code)
}

func TestSpanStatus(t *testing.T) {
	testCases := []struct {
		httpStatusCode int
		wantSpanStatus codes.Code
	}{
		{iris.StatusOK, codes.Unset},
		{iris.StatusBadRequest, codes.Unset},
		{iris.StatusInternalServerError, codes.Error},
	}
	for _, tc := range testCases {
		t.Run(strconv.Itoa(tc.httpStatusCode), func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider()
			provider.RegisterSpanProcessor(sr)
			router := iris.New()
			router.Use(oteliris.Middleware("foobar", oteliris.WithTracerProvider(provider)))
			router.Get("/", func(ctx iris.Context) {
				ctx.StatusCode(tc.httpStatusCode)
			})

			if err := router.Build(); err != nil {
				t.Fatal(err)
			}
			router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))

			require.Len(t, sr.Ended(), 1, "should emit a span")
			assert.Equal(t, sr.Ended()[0].Status().Code, tc.wantSpanStatus, "should only set Error status for HTTP statuses >= 500")
		})
	}
}

func TestSpanName(t *testing.T) {
	testCases := []struct {
		requestPath       string
		spanNameFormatter oteliris.SpanNameFormatter
		wantSpanName      string
	}{
		{"/user/1", nil, "/user/{id}"},
		{"/user/1", func(r *http.Request) string { return r.URL.Path }, "/user/1"},
	}
	for _, tc := range testCases {
		t.Run(tc.requestPath, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider()
			provider.RegisterSpanProcessor(sr)
			router := iris.New()
			router.Use(oteliris.Middleware("foobar", oteliris.WithTracerProvider(provider), oteliris.WithSpanNameFormatter(tc.spanNameFormatter)))
			router.Get("/user/{id}", func(ctx iris.Context) {})

			if err := router.Build(); err != nil {
				t.Fatal(err)
			}
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
	router := iris.New()
	router.Use(oteliris.Middleware("foobar", oteliris.WithTracerProvider(provider)))

	// add a template
	router.RegisterView(iris.HTML("./templates", ".html"))
	// OR
	// view := iris.HTML("", "")
	// view.ParseTemplate("hello", []byte(`hello {{.}}`), nil)
	// router.RegisterView(view)

	// a handler with an error and make the requests
	router.Get("/hello", func(ctx iris.Context) {
		oteliris.HTML(ctx, iris.StatusOK, "hello", "world")
	})
	r := httptest.NewRequest("GET", "/hello", nil)
	w := httptest.NewRecorder()

	if err := router.Build(); err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(w, r)
	response := w.Result()
	assert.Equal(t, iris.StatusOK, response.StatusCode)
	assert.Equal(t, "hello world", w.Body.String())

	// verify the errors and status are correct
	spans := sr.Ended()
	require.Len(t, spans, 2)
	var tspan sdktrace.ReadOnlySpan
	for _, s := range spans {
		// we need to pick up the span we're searching for, as the
		// order is not guaranteed within the buffer
		if s.Name() == "iris.renderer.html" {
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

		router := iris.New()
		f := func(req *http.Request) bool { return req.URL.Path != "/healthcheck" }
		router.Use(oteliris.Middleware("foobar", oteliris.WithFilter(f)))
		router.Get("/healthcheck", func(ctx iris.Context) {})

		r := httptest.NewRequest("GET", "/healthcheck", nil)
		w := httptest.NewRecorder()

		if err := router.Build(); err != nil {
			t.Fatal(err)
		}
		router.ServeHTTP(w, r)
		assert.Len(t, sr.Ended(), 0)
	})

	t.Run("custom filter not filtering route", func(t *testing.T) {
		sr := tracetest.NewSpanRecorder()
		otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr)))

		router := iris.New()
		f := func(req *http.Request) bool { return req.URL.Path != "/healthcheck" }
		router.Use(oteliris.Middleware("foobar", oteliris.WithFilter(f)))
		router.Get("/user/{id}", func(ctx iris.Context) {})

		r := httptest.NewRequest("GET", "/user/123", nil)
		w := httptest.NewRecorder()

		if err := router.Build(); err != nil {
			t.Fatal(err)
		}
		router.ServeHTTP(w, r)
		assert.Len(t, sr.Ended(), 1)
	})
}
