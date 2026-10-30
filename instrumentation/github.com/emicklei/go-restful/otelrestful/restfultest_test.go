// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelrestful_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/emicklei/go-restful/v3"
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
	otelsemconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	oteltrace "go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/contrib/instrumentation/github.com/emicklei/go-restful/otelrestful"
)

func TestChildSpanFromGlobalTracer(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr)))

	handlerFunc := func(_ *restful.Request, resp *restful.Response) {
		resp.WriteHeader(http.StatusOK)
	}
	ws := &restful.WebService{}
	ws.Route(ws.GET("/user/{id}").To(handlerFunc).
		Returns(http.StatusOK, "OK", nil).
		Returns(http.StatusNotFound, "Not Found", nil))
	container := restful.NewContainer()
	container.Filter(otelrestful.OTelFilter("my-service"))
	container.Add(ws)

	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/user/123", http.NoBody)
	w := httptest.NewRecorder()

	container.ServeHTTP(w, r)
	assert.Len(t, sr.Ended(), 1)
}

func TestChildSpanFromCustomTracer(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	handlerFunc := func(_ *restful.Request, resp *restful.Response) {
		resp.WriteHeader(http.StatusOK)
	}
	ws := &restful.WebService{}
	ws.Route(ws.GET("/user/{id}").To(handlerFunc))

	container := restful.NewContainer()
	container.Filter(otelrestful.OTelFilter("my-service", otelrestful.WithTracerProvider(provider)))
	container.Add(ws)

	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/user/123", http.NoBody)
	w := httptest.NewRecorder()

	container.ServeHTTP(w, r)
	assert.Len(t, sr.Ended(), 1)
}

func TestChildSpanNames(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	handlerFunc := func(_ *restful.Request, resp *restful.Response) {
		resp.WriteHeader(http.StatusOK)
	}
	ws := &restful.WebService{}
	ws.Route(ws.GET("/user/{id:[0-9]+}").To(handlerFunc))

	container := restful.NewContainer()
	container.Filter(otelrestful.OTelFilter("foobar", otelrestful.WithTracerProvider(provider)))
	container.Add(ws)

	ws.Route(ws.GET("/book/{title}").To(func(_ *restful.Request, resp *restful.Response) {
		_, _ = resp.Write([]byte("ok"))
	}))

	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/user/123", http.NoBody)
	w := httptest.NewRecorder()

	container.ServeHTTP(w, r)
	spans := sr.Ended()
	require.Len(t, spans, 1)
	assertSpan(
		t,
		spans[0],
		"/user/{id:[0-9]+}",
		attribute.String("server.address", "foobar"),
		attribute.Int("http.response.status_code", http.StatusOK),
		attribute.String("http.request.method", "GET"),
		attribute.String("http.route", "/user/{id:[0-9]+}"),
	)

	r = httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/book/foo", http.NoBody)
	w = httptest.NewRecorder()
	container.ServeHTTP(w, r)
	spans = sr.Ended()
	require.Len(t, spans, 2)
	assertSpan(
		t,
		spans[1],
		"/book/{title}",
		attribute.String("server.address", "foobar"),
		attribute.Int("http.response.status_code", http.StatusOK),
		attribute.Int("http.response.body.size", 2),
		attribute.String("http.request.method", "GET"),
		attribute.String("http.route", "/book/{title}"),
	)
}

func TestMultiFilters(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	retOK := func(_ *restful.Request, resp *restful.Response) { resp.WriteHeader(http.StatusOK) }
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

	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/user/123", http.NoBody)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, r)
	spans := sr.Ended()
	require.Len(t, spans, 1)
	assertSpan(t, spans[0], "/user/{id}")

	r = httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/user/123/books", http.NoBody)
	w = httptest.NewRecorder()
	container.ServeHTTP(w, r)
	spans = sr.Ended()
	require.Len(t, spans, 2)
	assertSpan(t, spans[1], "/user/{id}/books")

	r = httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/library/metropolitan", http.NoBody)
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
		wantErrorType  string
	}{
		{http.StatusOK, codes.Unset, ""},
		{http.StatusBadRequest, codes.Unset, ""},
		{http.StatusInternalServerError, codes.Error, "500"},
	}
	for _, tc := range testCases {
		t.Run(strconv.Itoa(tc.httpStatusCode), func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider()
			provider.RegisterSpanProcessor(sr)
			handlerFunc := func(_ *restful.Request, resp *restful.Response) {
				resp.WriteHeader(tc.httpStatusCode)
			}
			ws := &restful.WebService{}
			ws.Route(ws.GET("/").To(handlerFunc))
			container := restful.NewContainer()
			container.Filter(otelrestful.OTelFilter("my-service", otelrestful.WithTracerProvider(provider)))
			container.Add(ws)

			container.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody))

			require.Len(t, sr.Ended(), 1, "should emit a span")
			assert.Equal(t, tc.wantSpanStatus, sr.Ended()[0].Status().Code, "should only set Error status for HTTP statuses >= 500")

			if tc.wantErrorType != "" {
				assert.Contains(t, sr.Ended()[0].Attributes(), attribute.String("error.type", tc.wantErrorType))
			} else {
				for _, attr := range sr.Ended()[0].Attributes() {
					assert.NotEqual(t, attribute.Key("error.type"), attr.Key)
				}
			}
		})
	}
}

func TestClientDisconnect(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	reader := sdkmetric.NewManualReader()
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	handlerStarted := make(chan struct{})
	handlerDone := make(chan struct{})
	handlerFunc := func(req *restful.Request, resp *restful.Response) {
		close(handlerStarted)
		<-req.Request.Context().Done()
		resp.WriteHeader(http.StatusInternalServerError)
		close(handlerDone)
	}

	ws := &restful.WebService{}
	ws.Route(ws.GET("/hello").To(handlerFunc))
	container := restful.NewContainer()
	container.Filter(otelrestful.OTelFilter(
		"srv",
		otelrestful.WithTracerProvider(provider),
		otelrestful.WithMeterProvider(meterProvider),
	))
	container.Add(ws)

	ts := httptest.NewServer(container)
	defer ts.Close()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/hello", http.NoBody)
	require.NoError(t, err)

	client := ts.Client()
	go func() {
		select {
		case <-handlerStarted:
			cancel()
		case <-time.After(5 * time.Second):
		}
	}()

	resp, _ := client.Do(req)
	if resp != nil {
		_ = resp.Body.Close()
	}
	select {
	case <-handlerDone:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for handler")
	}

	require.Eventually(t, func() bool {
		return len(sr.Ended()) == 1
	}, 1*time.Second, 10*time.Millisecond)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	assert.Equal(t, codes.Error, span.Status().Code)
	assert.Contains(t, span.Attributes(), attribute.Int("http.response.status_code", http.StatusInternalServerError))
	assert.Contains(t, span.Attributes(), otelsemconv.ErrorType(context.Canceled))

	rm := metricdata.ResourceMetrics{}
	require.NoError(t, reader.Collect(t.Context(), &rm))
	require.Len(t, rm.ScopeMetrics, 1)
	for _, m := range rm.ScopeMetrics[0].Metrics {
		switch data := m.Data.(type) {
		case metricdata.Histogram[float64]:
			require.Len(t, data.DataPoints, 1)
			errVal, ok := data.DataPoints[0].Attributes.Value(otelsemconv.ErrorTypeKey)
			require.True(t, ok, "expected error.type on %s", m.Name)
			assert.Equal(t, otelsemconv.ErrorType(context.Canceled).Value.AsString(), errVal.AsString())
		case metricdata.Histogram[int64]:
			require.Len(t, data.DataPoints, 1)
			errVal, ok := data.DataPoints[0].Attributes.Value(otelsemconv.ErrorTypeKey)
			require.True(t, ok, "expected error.type on %s", m.Name)
			assert.Equal(t, otelsemconv.ErrorType(context.Canceled).Value.AsString(), errVal.AsString())
		}
	}
}

func TestDownstreamContextCancellationIgnored(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	scopedContextFilter := func(
		req *restful.Request,
		resp *restful.Response,
		chain *restful.FilterChain,
	) {
		ctx, cancel := context.WithCancel(req.Request.Context())
		defer cancel()
		req.Request = req.Request.WithContext(ctx)
		chain.ProcessFilter(req, resp)
	}

	ws := &restful.WebService{}
	ws.Route(ws.GET("/").To(func(
		_ *restful.Request,
		resp *restful.Response,
	) {
		resp.WriteHeader(http.StatusOK)
	}))

	container := restful.NewContainer()
	container.Filter(otelrestful.OTelFilter(
		"test",
		otelrestful.WithTracerProvider(provider),
	))
	container.Filter(scopedContextFilter)
	container.Add(ws)

	container.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequestWithContext(
			t.Context(),
			http.MethodGet,
			"/",
			http.NoBody,
		),
	)

	require.Len(t, sr.Ended(), 1)
	span := sr.Ended()[0]
	assert.Equal(t, codes.Unset, span.Status().Code)
	for _, attr := range span.Attributes() {
		assert.NotEqual(t, otelsemconv.ErrorTypeKey, attr.Key)
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

	handlerFunc := func(req *restful.Request, _ *restful.Response) {
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
	container.Filter(
		otelrestful.OTelFilter("test_handler",
			otelrestful.WithPublicEndpoint(),
			otelrestful.WithPropagators(prop),
			otelrestful.WithTracerProvider(provider)),
	)
	container.Add(ws)

	r, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://localhost/user/123", http.NoBody)
	require.NoError(t, err)

	sc := oteltrace.NewSpanContext(remoteSpan)
	ctx := oteltrace.ContextWithSpanContext(t.Context(), sc)
	prop.Inject(ctx, propagation.HeaderCarrier(r.Header))

	rr := httptest.NewRecorder()
	container.ServeHTTP(rr, r)
	assert.Equal(t, 200, rr.Result().StatusCode)

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
			fn: func(*http.Request) bool {
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
			fn: func(*http.Request) bool {
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

			handlerFunc := func(req *restful.Request, _ *restful.Response) {
				s := oteltrace.SpanFromContext(req.Request.Context())
				tt.handlerAssert(t, s.SpanContext())
			}

			ws := &restful.WebService{}
			ws.Route(ws.GET("/user/{id}").To(handlerFunc))

			container := restful.NewContainer()
			container.Filter(
				otelrestful.OTelFilter("test_handler",
					otelrestful.WithPublicEndpointFn(tt.fn),
					otelrestful.WithPropagators(prop),
					otelrestful.WithTracerProvider(provider)),
			)
			container.Add(ws)

			r, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://localhost/user/123", http.NoBody)
			require.NoError(t, err)

			sc := oteltrace.NewSpanContext(remoteSpan)
			ctx := oteltrace.ContextWithSpanContext(t.Context(), sc)
			prop.Inject(ctx, propagation.HeaderCarrier(r.Header))

			rr := httptest.NewRecorder()
			container.ServeHTTP(rr, r)
			assert.Equal(t, http.StatusOK, rr.Result().StatusCode)

			// Recorded span should be linked with an incoming span context.
			assert.NoError(t, spanRecorder.ForceFlush(ctx))
			spans := spanRecorder.Ended()
			tt.spansAssert(t, sc, spans)
		})
	}
}

type multipartParseResult struct {
	tmpPath string
	err     error
}

func multipartDiskBackedTmpPath(r *http.Request) (string, error) {
	if err := r.ParseMultipartForm(0); err != nil {
		return "", err
	}
	files := r.MultipartForm.File["file"]
	if len(files) == 0 {
		return "", errors.New("missing file part")
	}
	f, err := files[0].Open()
	if err != nil {
		return "", err
	}
	name := ""
	if of, ok := f.(*os.File); ok {
		name = of.Name()
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	if name == "" {
		return "", errors.New("file part should be disk-backed given maxMemory of 0")
	}
	return name, nil
}

func TestMultipartFormTempFileCleanup(t *testing.T) {
	resultCh := make(chan multipartParseResult, 1)
	ws := &restful.WebService{}
	ws.Route(ws.POST("/upload").To(func(req *restful.Request, resp *restful.Response) {
		path, err := multipartDiskBackedTmpPath(req.Request)
		resultCh <- multipartParseResult{tmpPath: path, err: err}
		if err != nil {
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}
		resp.WriteHeader(http.StatusOK)
	}))

	container := restful.NewContainer()
	container.DoNotRecover(false)
	container.Filter(otelrestful.OTelFilter("test_handler"))
	container.Add(ws)

	srv := httptest.NewServer(container)
	defer srv.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "test.txt")
	require.NoError(t, err)
	_, err = part.Write([]byte("hello"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, srv.URL+"/upload", &body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := srv.Client().Do(req)
	require.NoError(t, err)
	_, _ = io.Copy(io.Discard, resp.Body)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result multipartParseResult
	select {
	case result = <-resultCh:
	case <-time.After(5 * time.Second):
		t.Fatal("handler never reported temp file path")
	}
	require.NoError(t, result.err)
	require.NotEmpty(t, result.tmpPath)
	tmpPath := result.tmpPath
	t.Cleanup(func() { _ = os.Remove(tmpPath) })

	require.Eventually(t, func() bool {
		_, statErr := os.Stat(tmpPath)
		return os.IsNotExist(statErr)
	}, 2*time.Second, 20*time.Millisecond, "multipart temp file was not cleaned up: %s", tmpPath)
}

func TestMultipartFormTempFileCleanupOnPanic(t *testing.T) {
	resultCh := make(chan multipartParseResult, 1)
	ws := &restful.WebService{}
	ws.Route(ws.POST("/upload").To(func(req *restful.Request, _ *restful.Response) {
		path, err := multipartDiskBackedTmpPath(req.Request)
		resultCh <- multipartParseResult{tmpPath: path, err: err}
		panic("test panic after multipart parse")
	}))

	container := restful.NewContainer()
	container.DoNotRecover(false)
	container.Filter(otelrestful.OTelFilter("test_handler"))
	container.Add(ws)

	srv := httptest.NewServer(container)
	defer srv.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "test.txt")
	require.NoError(t, err)
	_, err = part.Write([]byte("hello"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, srv.URL+"/upload", &body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := srv.Client().Do(req)
	require.NoError(t, err)
	_, _ = io.Copy(io.Discard, resp.Body)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var result multipartParseResult
	select {
	case result = <-resultCh:
	case <-time.After(5 * time.Second):
		t.Fatal("handler never reported temp file path")
	}
	require.NoError(t, result.err)
	require.NotEmpty(t, result.tmpPath)
	tmpPath := result.tmpPath
	t.Cleanup(func() { _ = os.Remove(tmpPath) })

	require.Eventually(t, func() bool {
		_, statErr := os.Stat(tmpPath)
		return os.IsNotExist(statErr)
	}, 5*time.Second, 20*time.Millisecond, "multipart temp file was not cleaned up after panic: %s", tmpPath)
}

func TestMetrics(t *testing.T) {
	tests := []struct {
		name              string
		useGlobalProvider bool
		method            string
		target            string
		reqBody           string
		unknownReqSize    bool
		respStatus        int
		respBody          string
		wantRouteAttr     string
		wantErrorType     string
		wantReqBodyMetric bool
		wantReqBodyBytes  int64
		wantRespBodyBytes int64
	}{
		{
			name:              "matched route with request and response bodies",
			method:            http.MethodPost,
			target:            "/users/123",
			reqBody:           "hello",
			respStatus:        http.StatusOK,
			respBody:          "world!!",
			wantRouteAttr:     "/users/{id}",
			wantReqBodyMetric: true,
			wantReqBodyBytes:  5,
			wantRespBodyBytes: 7,
		},
		{
			name:              "global meter provider fallback",
			useGlobalProvider: true,
			method:            http.MethodGet,
			target:            "/users/123",
			respStatus:        http.StatusOK,
			respBody:          "ok",
			wantRouteAttr:     "/users/{id}",
			wantReqBodyMetric: true,
			wantReqBodyBytes:  0,
			wantRespBodyBytes: 2,
		},
		{
			name:              "unmatched 404 route omits http.route",
			method:            http.MethodGet,
			target:            "/users/123/unknown",
			respStatus:        http.StatusNotFound,
			wantRouteAttr:     "",
			wantReqBodyMetric: true,
			wantReqBodyBytes:  0,
			wantRespBodyBytes: 19,
		},
		{
			name:              "unknown request content length (-1) omits request body size metric",
			method:            http.MethodPost,
			target:            "/users/123",
			reqBody:           "chunked",
			unknownReqSize:    true,
			respStatus:        http.StatusOK,
			respBody:          "ok",
			wantRouteAttr:     "/users/{id}",
			wantReqBodyMetric: false,
			wantRespBodyBytes: 2,
		},
		{
			name:              "500 status records error.type attribute",
			method:            http.MethodGet,
			target:            "/users/500",
			respStatus:        http.StatusInternalServerError,
			respBody:          "fail",
			wantRouteAttr:     "/users/{id}",
			wantErrorType:     "500",
			wantReqBodyMetric: true,
			wantReqBodyBytes:  0,
			wantRespBodyBytes: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := sdkmetric.NewManualReader()
			meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

			var opts []otelrestful.Option
			if tt.useGlobalProvider {
				orig := otel.GetMeterProvider()
				otel.SetMeterProvider(meterProvider)
				t.Cleanup(func() { otel.SetMeterProvider(orig) })
				// Passing nil should be ignored and fall back to global provider.
				opts = append(opts, otelrestful.WithMeterProvider(nil))
			} else {
				opts = append(opts, otelrestful.WithMeterProvider(meterProvider))
			}

			ws := &restful.WebService{}
			ws.Path("/users")
			ws.Route(ws.GET("/{id}").To(func(_ *restful.Request, resp *restful.Response) {
				resp.WriteHeader(tt.respStatus)
				if tt.respBody != "" {
					_, _ = resp.Write([]byte(tt.respBody))
				}
			}))
			ws.Route(ws.POST("/{id}").To(func(_ *restful.Request, resp *restful.Response) {
				resp.WriteHeader(tt.respStatus)
				if tt.respBody != "" {
					_, _ = resp.Write([]byte(tt.respBody))
				}
			}))

			container := restful.NewContainer()
			container.Filter(otelrestful.OTelFilter("foobar", opts...))
			container.Add(ws)

			r := httptest.NewRequestWithContext(t.Context(), tt.method, tt.target, strings.NewReader(tt.reqBody))
			if tt.unknownReqSize {
				r.ContentLength = -1
			}
			w := httptest.NewRecorder()
			container.ServeHTTP(w, r)
			assert.Equal(t, tt.respStatus, w.Code)

			rm := metricdata.ResourceMetrics{}
			require.NoError(t, reader.Collect(t.Context(), &rm))
			require.Len(t, rm.ScopeMetrics, 1)
			sm := rm.ScopeMetrics[0]
			assert.Equal(t, otelrestful.ScopeName, sm.Scope.Name)
			assert.Equal(t, otelrestful.Version, sm.Scope.Version)

			attrs := []attribute.KeyValue{
				attribute.String("http.request.method", tt.method),
				attribute.Int64("http.response.status_code", int64(tt.respStatus)),
				attribute.String("network.protocol.name", "http"),
				attribute.String("network.protocol.version", "1.1"),
				attribute.String("server.address", "foobar"),
				attribute.String("url.scheme", "http"),
			}
			if tt.wantRouteAttr != "" {
				attrs = append(attrs, attribute.String("http.route", tt.wantRouteAttr))
			}
			if tt.wantErrorType != "" {
				attrs = append(attrs, attribute.String("error.type", tt.wantErrorType))
			}

			metricsByName := make(map[string]metricdata.Metrics, len(sm.Metrics))
			for _, m := range sm.Metrics {
				metricsByName[m.Name] = m
			}

			if tt.wantReqBodyMetric {
				require.Len(t, sm.Metrics, 3)
				gotReq := metricsByName["http.server.request.body.size"]
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
				}, gotReq, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue(), metricdatatest.IgnoreExemplars())
				assert.Equal(t, tt.wantReqBodyBytes, gotReq.Data.(metricdata.Histogram[int64]).DataPoints[0].Sum)
			} else {
				require.Len(t, sm.Metrics, 2)
				assert.NotContains(t, metricsByName, "http.server.request.body.size")
			}

			gotResp := metricsByName["http.server.response.body.size"]
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
			}, gotResp, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue(), metricdatatest.IgnoreExemplars())
			assert.Equal(t, tt.wantRespBodyBytes, gotResp.Data.(metricdata.Histogram[int64]).DataPoints[0].Sum)

			gotDur := metricsByName["http.server.request.duration"]
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
			}, gotDur, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue(), metricdatatest.IgnoreExemplars())
		})
	}
}

func assertSpan(t *testing.T, span sdktrace.ReadOnlySpan, name string, attrs ...attribute.KeyValue) {
	t.Helper()

	assert.Equal(t, name, span.Name())
	assert.Equal(t, oteltrace.SpanKindServer, span.SpanKind())

	gotA := span.Attributes()
	for _, a := range attrs {
		assert.Contains(t, gotA, a)
	}
}
