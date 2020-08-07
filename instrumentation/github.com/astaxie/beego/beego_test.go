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

package beego

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/standard"

	"github.com/stretchr/testify/require"

	"github.com/astaxie/beego"

	mockmeter "go.opentelemetry.io/contrib/internal/metric"
	mocktrace "go.opentelemetry.io/contrib/internal/trace"
)

// ------------------------------------------ Test Controller

const defaultReply = "hello world"

type testController struct {
	beego.Controller
}

func (c *testController) BasicHello() {
	c.Ctx.ResponseWriter.Write([]byte(defaultReply))
}

func (c *testController) HelloWithName() {
	name := c.GetString("name")
	if name == "" {
		c.Ctx.ResponseWriter.WriteHeader(http.StatusBadRequest)
		c.Ctx.ResponseWriter.Write([]byte("missing query param \"name\""))
		return
	}
	c.Ctx.ResponseWriter.Write([]byte(fmt.Sprintf("%s said hello.", name)))
}

func newTestRouterWithController() *beego.ControllerRegister {
	router := beego.NewControllerRegister()
	controller := &testController{}
	router.Add("/", controller, "get:BasicHello")
	router.Add("/greet", controller, "post:HelloWithName")
	return router
}

// ------------------------------------------ Test Case

const middleWareName = "test-router"

type testCase struct {
	name               string
	mwName             string
	method             string
	url                string
	options            []Option
	hasSpan            bool
	expectedSpanName   string
	expectedHTTPStatus int
	expectedResponse   string
	expectedAttributes []kv.KeyValue
}

var testCases = []*testCase{
	{
		name:               "GET/__All default options",
		method:             http.MethodGet,
		url:                "http://localhost/",
		options:            []Option{},
		hasSpan:            true,
		expectedSpanName:   "/",
		expectedHTTPStatus: http.StatusOK,
		expectedResponse:   defaultReply,
		expectedAttributes: []kv.KeyValue{},
	},
	{
		name:               "POST/greet?name=test__All default options",
		method:             http.MethodPost,
		url:                "http://localhost/greet?name=test",
		options:            []Option{},
		hasSpan:            true,
		expectedSpanName:   "/greet",
		expectedHTTPStatus: http.StatusOK,
		expectedResponse:   "test said hello.",
		expectedAttributes: []kv.KeyValue{},
	},
	{
		name:   "GET/__Custom filter filtering route",
		method: http.MethodGet,
		url:    "http://localhost/",
		options: []Option{
			WithFilter(Filter(func(req *http.Request) bool {
				return req.URL.Path != "/"
			})),
		},
		hasSpan:            false,
		expectedHTTPStatus: http.StatusOK,
		expectedResponse:   defaultReply,
	},
	{
		name:   "GET/__Custom filter not filtering route",
		method: http.MethodGet,
		url:    "http://localhost/",
		options: []Option{
			WithFilter(Filter(func(req *http.Request) bool {
				return req.URL.Path != "/greet"
			})),
		},
		hasSpan:            true,
		expectedSpanName:   "/",
		expectedHTTPStatus: http.StatusOK,
		expectedResponse:   defaultReply,
		expectedAttributes: []kv.KeyValue{},
	},
	{
		name:               "POST/greet__Default options, bad request",
		method:             http.MethodPost,
		url:                "http://localhost/greet",
		options:            []Option{},
		hasSpan:            true,
		expectedSpanName:   "/greet",
		expectedHTTPStatus: http.StatusBadRequest,
		expectedResponse:   "missing query param \"name\"",
		expectedAttributes: []kv.KeyValue{},
	},
	{
		name:   "POST/greet?name=test__Custom span name formatter",
		method: http.MethodPost,
		url:    "http://localhost/greet?name=test",
		options: []Option{
			WithSpanNameFormatter(SpanNameFormatter(func(opp string, req *http.Request) string {
				return "Test Span Name"
			})),
		},
		hasSpan:            true,
		expectedSpanName:   "Test Span Name",
		expectedHTTPStatus: http.StatusOK,
		expectedResponse:   "test said hello.",
		expectedAttributes: []kv.KeyValue{},
	},
}

func TestHandler(t *testing.T) {
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tracer := mocktrace.NewTracer("beego-test")
			meterimpl, meter := mockmeter.NewMeter()
			router := newTestRouterWithController()

			rr := httptest.NewRecorder()
			req, err := http.NewRequest(tc.method, tc.url, nil)
			require.NoError(t, err)

			tc.expectedAttributes = append(tc.expectedAttributes, defaultAttributes()...)

			mw := NewOTelBeegoMiddleWare(
				middleWareName,
				append(
					tc.options,
					WithTracer(tracer),
					WithMeter(meter),
				)...,
			)

			mw(router).ServeHTTP(rr, req)

			require.Equal(t, tc.expectedHTTPStatus, rr.Result().StatusCode)
			body, err := ioutil.ReadAll(rr.Result().Body)
			require.NoError(t, err)
			require.Equal(t, tc.expectedResponse, string(body))

			spans := tracer.EndedSpans()
			if tc.hasSpan {
				require.Len(t, spans, 1)
				result := rr.Result()
				assertSpan(t, spans[0], tc, result)
			} else {
				require.Len(t, spans, 0)
			}
			assertMetrics(t, meterimpl.MeasurementBatches, tc)

		})
	}
}

// ------------------------------------------ Utilities

func defaultAttributes() []kv.KeyValue {
	return []kv.KeyValue{
		standard.HTTPServerNameKey.String(middleWareName),
		standard.HTTPSchemeHTTP,
		standard.HTTPHostKey.String("localhost"),
	}
}

func assertSpan(t *testing.T, span *mocktrace.Span, tc *testCase, res *http.Response) {
	require.Equal(t, tc.expectedSpanName, span.Name)
	for _, att := range tc.expectedAttributes {
		require.Equal(t, att.Value.AsInterface(), span.Attributes[att.Key].AsInterface())
	}
}

func assertMetrics(t *testing.T, batches []mockmeter.Batch, tc *testCase) {
	for _, batch := range batches {
		for _, att := range tc.expectedAttributes {
			require.Contains(t, batch.Labels, att)
		}
	}
}
