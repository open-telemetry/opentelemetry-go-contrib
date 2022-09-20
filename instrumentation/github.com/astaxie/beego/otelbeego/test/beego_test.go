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

package test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.opentelemetry.io/contrib/instrumentation/github.com/astaxie/beego/otelbeego"
	"go.opentelemetry.io/contrib/instrumentation/github.com/astaxie/beego/otelbeego/internal"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"

	"github.com/astaxie/beego"
	beegoCtx "github.com/astaxie/beego/context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO(#2762): Add metric integration tests for the instrumentation. These
// tests depend on
// https://github.com/open-telemetry/opentelemetry-go/issues/3031 being
// resolved.

// ------------------------------------------ Test Controller

const defaultReply = "hello world"

var tplName = ""

type testReply struct {
	Message string `json:"message"`
	Err     string `json:"error"`
}

type testController struct {
	beego.Controller
	T *testing.T
}

func (c *testController) Get() {
	reply := &testReply{
		Message: defaultReply,
	}
	c.Data["json"] = reply
	c.ServeJSON()
}

func (c *testController) Post() {
	name := c.GetString("name")
	var reply *testReply
	if name == "" {
		c.Ctx.ResponseWriter.WriteHeader(http.StatusBadRequest)
		reply = &testReply{
			Err: "missing query param \"name\"",
		}
	} else {
		reply = &testReply{
			Message: fmt.Sprintf("%s said hello.", name),
		}
	}
	c.Data["json"] = reply
	c.ServeJSON()
}

func (c *testController) Delete() {
	reply := &testReply{
		Message: "success",
	}
	c.Ctx.ResponseWriter.WriteHeader(http.StatusAccepted)
	c.Data["json"] = reply
	c.ServeJSON()
}

func (c *testController) Put() {
	reply := &testReply{
		Message: "successfully put",
	}
	c.Ctx.ResponseWriter.WriteHeader(http.StatusAccepted)
	c.Data["json"] = reply
	c.ServeJSON()
}

func (c *testController) TemplateRender() {
	c.TplName = tplName
	c.Data["name"] = "test"
	require.NoError(c.T, otelbeego.Render(&c.Controller))
}

func (c *testController) TemplateRenderString() {
	c.TplName = tplName
	c.Data["name"] = "test"
	str, err := otelbeego.RenderString(&c.Controller)
	require.NoError(c.T, err)
	c.Ctx.WriteString(str)
}

func (c *testController) TemplateRenderBytes() {
	c.TplName = tplName
	c.Data["name"] = "test"
	bytes, err := otelbeego.RenderBytes(&c.Controller)
	require.NoError(c.T, err)
	_, err = c.Ctx.ResponseWriter.Write(bytes)
	require.NoError(c.T, err)
}

func addTestRoutes(t *testing.T) {
	controller := &testController{
		T: t,
	}
	beego.Router("/", controller)
	beego.Router("/:id", controller)
	beego.Router("/greet", controller)
	beego.Router("/template/render", controller, "get:TemplateRender")
	beego.Router("/template/renderstring", controller, "get:TemplateRenderString")
	beego.Router("/template/renderbytes", controller, "get:TemplateRenderBytes")
	router := beego.NewNamespace("/api",
		beego.NSNamespace("/v1",
			beego.NSRouter("/", controller),
			beego.NSRouter("/:id", controller),
			beego.NSRouter("/greet", controller),
		),
	)
	beego.AddNamespace(router)
}

func replaceBeego() {
	beego.BeeApp = beego.NewApp()
}

// ------------------------------------------ Unit Tests

func TestHandler(t *testing.T) {
	for _, tcase := range testCases {
		tc := *tcase
		t.Run(tc.name, func(t *testing.T) {
			runTest(t, &tc, "http://localhost")
		})
	}
}

func TestHandlerWithNamespace(t *testing.T) {
	for _, tcase := range testCases {
		tc := *tcase
		t.Run(tc.name, func(t *testing.T) {
			// if using default span name, change name to NS path
			if tc.expectedSpanName != customSpanName {
				tc.expectedSpanName = fmt.Sprintf("/api/v1%s", tc.expectedSpanName)
			}
			runTest(t, &tc, "http://localhost/api/v1")
		})
	}
}

func TestWithFilters(t *testing.T) {
	for _, tcase := range testCases {
		tc := *tcase
		t.Run(tc.name, func(t *testing.T) {
			wasCalled := false
			beego.InsertFilter("/*", beego.BeforeRouter, func(ctx *beegoCtx.Context) {
				wasCalled = true
			})
			runTest(t, &tc, "http://localhost")
			require.True(t, wasCalled)
		})
	}
}

func TestStatic(t *testing.T) {
	defer replaceBeego()
	sr := tracetest.NewSpanRecorder()
	tracerProvider := trace.NewTracerProvider(trace.WithSpanProcessor(sr))
	file, err := os.CreateTemp("", "static-*.html")
	require.NoError(t, err)
	defer file.Close()
	defer os.Remove(file.Name())
	_, err = file.WriteString(beego.Htmlunquote("<h1>Hello, world!</h1>"))
	require.NoError(t, err)

	beego.SetStaticPath("/", file.Name())
	defer beego.SetStaticPath("/", "")

	mw := otelbeego.NewOTelBeegoMiddleWare(middleWareName,
		otelbeego.WithTracerProvider(tracerProvider),
	)

	rr := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "http://localhost/", nil)
	require.NoError(t, err)
	mw(beego.BeeApp.Handlers).ServeHTTP(rr, req)
	tc := &testCase{
		expectedSpanName:   "GET",
		expectedAttributes: defaultAttributes(),
	}

	require.Equal(t, http.StatusOK, rr.Result().StatusCode)
	body, err := io.ReadAll(rr.Result().Body)
	require.NoError(t, err)
	require.Equal(t, "<h1>Hello, world!</h1>", string(body))
	spans := sr.Ended()
	require.Len(t, spans, 1)
	assertSpan(t, spans[0], tc)
}

func TestRender(t *testing.T) {
	// Disable autorender to enable traced render
	beego.BConfig.WebConfig.AutoRender = false
	addTestRoutes(t)
	defer replaceBeego()
	htmlStr := "<!DOCTYPE html><html lang=\"en\">" +
		"<head><meta charset=\"UTF-8\"><title>Hello World</title></head>" +
		"<body>This is a template test. Hello {{.name}}</body></html>"

	// Create a temp directory to hold a view
	dir := t.TempDir()

	// Create the view
	file, err := os.CreateTemp(dir, "*index.tpl")
	require.NoError(t, err)
	defer file.Close()
	_, err = file.WriteString(htmlStr)
	require.NoError(t, err)
	// Add path to view path
	require.NoError(t, beego.AddViewPath(dir))
	beego.SetViewsPath(dir)
	_, tplName = filepath.Split(file.Name())

	sr := tracetest.NewSpanRecorder()
	tracerProvider := trace.NewTracerProvider(trace.WithSpanProcessor(sr))

	mw := otelbeego.NewOTelBeegoMiddleWare(
		middleWareName,
		otelbeego.WithTracerProvider(tracerProvider),
	)
	for _, str := range []string{"/render", "/renderstring", "/renderbytes"} {
		rr := httptest.NewRecorder()
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost/template%s", str), nil)
		require.NoError(t, err)
		mw(beego.BeeApp.Handlers).ServeHTTP(rr, req)
		body, err := io.ReadAll(rr.Result().Body)
		require.Equal(t, strings.Replace(htmlStr, "{{.name}}", "test", 1), string(body))
		require.NoError(t, err)
	}

	spans := sr.Ended()
	require.Len(t, spans, 6) // 3 HTTP requests, each creating 2 spans
	for _, span := range spans {
		switch span.Name() {
		case "/template/render":
		case "/template/renderstring":
		case "/template/renderbytes":
			continue
		case internal.RenderTemplateSpanName,
			internal.RenderStringSpanName,
			internal.RenderBytesSpanName:
			assert.Contains(t, span.Attributes(), internal.TemplateKey.String(tplName))
		default:
			t.Fatal("unexpected span name")
		}
	}
}

// ------------------------------------------ Utilities

func runTest(t *testing.T, tc *testCase, url string) {
	sr := tracetest.NewSpanRecorder()
	tracerProvider := trace.NewTracerProvider(trace.WithSpanProcessor(sr))
	addTestRoutes(t)
	defer replaceBeego()

	rr := httptest.NewRecorder()
	req, err := http.NewRequest(
		tc.method,
		fmt.Sprintf("%s%s", url, tc.path),
		nil,
	)
	require.NoError(t, err)

	tc.expectedAttributes = append(tc.expectedAttributes, defaultAttributes()...)

	mw := otelbeego.NewOTelBeegoMiddleWare(
		middleWareName,
		append(
			tc.options,
			otelbeego.WithTracerProvider(tracerProvider),
		)...,
	)

	mw(beego.BeeApp.Handlers).ServeHTTP(rr, req)

	require.Equal(t, tc.expectedHTTPStatus, rr.Result().StatusCode)
	body, err := io.ReadAll(rr.Result().Body)
	require.NoError(t, err)
	message := testReply{}
	require.NoError(t, json.Unmarshal(body, &message))
	require.Equal(t, tc.expectedResponse, message)

	spans := sr.Ended()
	if tc.hasSpan {
		require.Len(t, spans, 1)
		assertSpan(t, spans[0], tc)
	} else {
		require.Len(t, spans, 0)
	}
}

func defaultAttributes() []attribute.KeyValue {
	return []attribute.KeyValue{
		semconv.HTTPServerNameKey.String(middleWareName),
		semconv.HTTPSchemeHTTP,
		semconv.HTTPHostKey.String("localhost"),
	}
}

func assertSpan(t *testing.T, span trace.ReadOnlySpan, tc *testCase) {
	require.Equal(t, tc.expectedSpanName, span.Name())
	attr := span.Attributes()
	for _, att := range tc.expectedAttributes {
		assert.Contains(t, attr, att)
	}
}

// ------------------------------------------ Test Cases

const middleWareName = "test-router"

const customSpanName = "Test span name"

type testCase struct {
	name               string
	method             string
	path               string
	options            []otelbeego.Option
	hasSpan            bool
	expectedSpanName   string
	expectedHTTPStatus int
	expectedResponse   testReply
	expectedAttributes []attribute.KeyValue
}

var testCases = []*testCase{
	{
		name:               "GET/__All default options",
		method:             http.MethodGet,
		path:               "/",
		options:            []otelbeego.Option{},
		hasSpan:            true,
		expectedSpanName:   "/",
		expectedHTTPStatus: http.StatusOK,
		expectedResponse:   testReply{Message: defaultReply},
		expectedAttributes: []attribute.KeyValue{},
	},
	{
		name:               "GET/1__All default options",
		method:             http.MethodGet,
		path:               "/1",
		options:            []otelbeego.Option{},
		hasSpan:            true,
		expectedSpanName:   "/:id",
		expectedHTTPStatus: http.StatusOK,
		expectedResponse:   testReply{Message: defaultReply},
		expectedAttributes: []attribute.KeyValue{},
	},
	{
		name:               "POST/greet?name=test__All default options",
		method:             http.MethodPost,
		path:               "/greet?name=test",
		options:            []otelbeego.Option{},
		hasSpan:            true,
		expectedSpanName:   "/greet",
		expectedHTTPStatus: http.StatusOK,
		expectedResponse:   testReply{Message: "test said hello."},
		expectedAttributes: []attribute.KeyValue{},
	},
	{
		name:               "DELETE/__All default options",
		method:             http.MethodDelete,
		path:               "/",
		options:            []otelbeego.Option{},
		hasSpan:            true,
		expectedSpanName:   "/",
		expectedHTTPStatus: http.StatusAccepted,
		expectedResponse:   testReply{Message: "success"},
		expectedAttributes: []attribute.KeyValue{},
	},
	{
		name:               "PUT/__All default options",
		method:             http.MethodPut,
		path:               "/",
		options:            []otelbeego.Option{},
		hasSpan:            true,
		expectedSpanName:   "/",
		expectedHTTPStatus: http.StatusAccepted,
		expectedResponse:   testReply{Message: "successfully put"},
		expectedAttributes: []attribute.KeyValue{},
	},
	{
		name:   "GET/__Custom propagators",
		method: http.MethodGet,
		path:   "/",
		options: []otelbeego.Option{
			otelbeego.WithPropagators(propagation.NewCompositeTextMapPropagator(b3.New())),
		},
		hasSpan:            true,
		expectedSpanName:   "/",
		expectedHTTPStatus: http.StatusOK,
		expectedResponse:   testReply{Message: defaultReply},
		expectedAttributes: []attribute.KeyValue{},
	},
	{
		name:   "GET/__Custom filter filtering route",
		method: http.MethodGet,
		path:   "/",
		options: []otelbeego.Option{
			otelbeego.WithFilter(otelbeego.Filter(func(req *http.Request) bool {
				return req.URL.Path != "/"
			})),
			otelbeego.WithFilter(otelbeego.Filter(func(req *http.Request) bool {
				return req.URL.Path != "/api/v1/"
			})),
		},
		hasSpan:            false,
		expectedHTTPStatus: http.StatusOK,
		expectedResponse:   testReply{Message: defaultReply},
	},
	{
		name:   "GET/__Custom filter not filtering route",
		method: http.MethodGet,
		path:   "/",
		options: []otelbeego.Option{
			otelbeego.WithFilter(otelbeego.Filter(func(req *http.Request) bool {
				return req.URL.Path != "/greet"
			})),
		},
		hasSpan:            true,
		expectedSpanName:   "/",
		expectedHTTPStatus: http.StatusOK,
		expectedResponse:   testReply{Message: defaultReply},
		expectedAttributes: []attribute.KeyValue{},
	},
	{
		name:               "POST/greet__Default options, bad request",
		method:             http.MethodPost,
		path:               "/greet",
		options:            []otelbeego.Option{},
		hasSpan:            true,
		expectedSpanName:   "/greet",
		expectedHTTPStatus: http.StatusBadRequest,
		expectedResponse:   testReply{Err: "missing query param \"name\""},
		expectedAttributes: []attribute.KeyValue{},
	},
	{
		name:   "POST/greet?name=test__Custom span name formatter",
		method: http.MethodPost,
		path:   "/greet?name=test",
		options: []otelbeego.Option{
			otelbeego.WithSpanNameFormatter(otelbeego.SpanNameFormatter(func(opp string, req *http.Request) string {
				return customSpanName
			})),
		},
		hasSpan:            true,
		expectedSpanName:   customSpanName,
		expectedHTTPStatus: http.StatusOK,
		expectedResponse:   testReply{Message: "test said hello."},
		expectedAttributes: []attribute.KeyValue{},
	},
	{
		name:   "POST/greet?name=test__Custom span name formatter and custom filter",
		method: http.MethodPost,
		path:   "/greet?name=test",
		options: []otelbeego.Option{
			otelbeego.WithFilter(otelbeego.Filter(func(req *http.Request) bool {
				return !strings.Contains(req.URL.Path, "greet")
			})),
			otelbeego.WithSpanNameFormatter(otelbeego.SpanNameFormatter(func(opp string, req *http.Request) string {
				return customSpanName
			})),
		},
		hasSpan:            false,
		expectedHTTPStatus: http.StatusOK,
		expectedResponse:   testReply{Message: "test said hello."},
		expectedAttributes: []attribute.KeyValue{},
	},
}
