// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelmux_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
)

func TestDefaultTrace(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	router := mux.NewRouter()
	router.Use(otelmux.Middleware("foobar", otelmux.WithTracerProvider(provider)))

	router.HandleFunc("/user/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/user/123", http.NoBody)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code, "unexpected status code")

	spans := sr.Ended()

	require.Len(t, sr.Ended(), 1)
	span := spans[0]
	attr := span.Attributes()
	assert.True(t, ensurePrefix(http.MethodGet, spans[0].Name()))
	assert.Equal(t, "GET /user/{id}", span.Name())
	assert.Equal(t, trace.SpanKindServer, span.SpanKind())
	assert.Contains(t, attr, attribute.Int("http.response.status_code", http.StatusOK))
	assert.Contains(t, attr, attribute.String("http.request.method", "GET"))
	assert.Contains(t, attr, attribute.String("http.route", "/user/{id}"))
	assert.Equal(t, codes.Unset, span.Status().Code)
	assert.Empty(t, span.Status().Description)
}

func TestCustomSpanNameFormatter(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()

	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))

	routeTpl := "/user/{id}"

	testdata := []struct {
		spanNameFormatter func(string, *http.Request) string
		want              string
	}{
		{nil, setDefaultName(http.MethodGet, routeTpl)},
		{
			func(string, *http.Request) string { return "custom" },
			"custom",
		},
		{
			func(name string, r *http.Request) string {
				return fmt.Sprintf("%s %s", r.Method, name)
			},
			"GET " + routeTpl,
		},
	}

	for i, d := range testdata {
		t.Run(fmt.Sprintf("%d_%s", i, d.want), func(t *testing.T) {
			router := mux.NewRouter()
			router.Use(otelmux.Middleware(
				"foobar",
				otelmux.WithTracerProvider(tp),
				otelmux.WithSpanNameFormatter(d.spanNameFormatter),
			))
			router.HandleFunc(routeTpl, func(http.ResponseWriter, *http.Request) {})

			r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/user/123", http.NoBody)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, r)

			spans := exporter.GetSpans()
			require.Len(t, spans, 1)
			assert.Equal(t, d.want, spans[0].Name)

			exporter.Reset()
		})
	}
}

func ok(http.ResponseWriter, *http.Request) {}
func notfound(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "not found", http.StatusNotFound)
}

func TestSDKIntegration(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider()
	provider.RegisterSpanProcessor(sr)

	reader := sdkmetric.NewManualReader()
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	router := mux.NewRouter()
	router.Use(otelmux.Middleware("foobar",
		otelmux.WithTracerProvider(provider),
		otelmux.WithMeterProvider(meterProvider)))

	router.HandleFunc("/user/{id:[0-9]+}", ok)
	router.HandleFunc("/book/{title}", ok)

	tests := []struct {
		name         string
		method       string
		path         string
		reqFunc      func(r *http.Request)
		wantSpanName string
		wantMethod   string
		wantRoute    string
	}{
		{
			name:         "user route",
			method:       http.MethodGet,
			path:         "/user/123",
			reqFunc:      nil,
			wantSpanName: "GET /user/{id:[0-9]+}",
			wantMethod:   http.MethodGet,
			wantRoute:    "/user/{id:[0-9]+}",
		},
		{
			name:         "POST book route",
			method:       http.MethodPost,
			path:         "/book/foo",
			reqFunc:      nil,
			wantSpanName: "POST /book/{title}",
			wantMethod:   http.MethodPost,
			wantRoute:    "/book/{title}",
		},
		{
			name:         "book route with custom pattern",
			method:       http.MethodGet,
			path:         "/book/bar",
			reqFunc:      func(r *http.Request) { r.Pattern = "/book/{custom}" },
			wantSpanName: "GET /book/{custom}",
			wantMethod:   http.MethodGet,
			wantRoute:    "/book/{custom}",
		},
		{
			name:         "Invalid HTTP Method",
			method:       "INVALID",
			path:         "/book/bar",
			reqFunc:      func(r *http.Request) { r.Pattern = "/book/{custom}" },
			wantSpanName: "HTTP /book/{custom}",
			wantMethod:   "_OTHER",
			wantRoute:    "/book/{custom}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer sr.Reset()

			r := httptest.NewRequestWithContext(t.Context(), tt.method, tt.path, http.NoBody)
			if tt.reqFunc != nil {
				tt.reqFunc(r)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			spans := sr.Ended()

			require.Len(t, spans, 1)
			assertSpan(
				t, sr.Ended()[0],
				tt.wantSpanName,
				trace.SpanKindServer,
				attribute.String("server.address", "foobar"),
				attribute.Int("http.response.status_code", http.StatusOK),
				attribute.String("http.request.method", tt.wantMethod),
				attribute.String("http.route", tt.wantRoute),
			)
		})
	}
}

func TestNotFoundIsNotError(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider()
	provider.RegisterSpanProcessor(sr)

	router := mux.NewRouter()
	router.Use(otelmux.Middleware("foobar", otelmux.WithTracerProvider(provider)))
	router.HandleFunc("/does/not/exist", notfound)

	r0 := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/does/not/exist", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r0)

	require.Len(t, sr.Ended(), 1)
	assertSpan(
		t, sr.Ended()[0],
		"GET /does/not/exist",
		trace.SpanKindServer,
		attribute.String("server.address", "foobar"),
		attribute.Int("http.response.status_code", http.StatusNotFound),
		attribute.String("http.request.method", "GET"),
		attribute.String("http.route", "/does/not/exist"),
	)
	assert.Equal(t, codes.Unset, sr.Ended()[0].Status().Code)
}

func assertSpan(t *testing.T, span sdktrace.ReadOnlySpan, name string, kind trace.SpanKind, attrs ...attribute.KeyValue) {
	t.Helper()

	assert.Equal(t, name, span.Name())
	assert.Equal(t, kind, span.SpanKind())

	got := make(map[attribute.Key]attribute.Value, len(span.Attributes()))
	for _, a := range span.Attributes() {
		got[a.Key] = a.Value
	}
	for _, want := range attrs {
		if !assert.Contains(t, got, want.Key) {
			continue
		}
		assert.Equal(t, want.Value, got[want.Key])
	}
}

func TestWithPublicEndpoint(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider()
	provider.RegisterSpanProcessor(sr)

	remoteSpan := trace.SpanContextConfig{
		TraceID: trace.TraceID{0x01},
		SpanID:  trace.SpanID{0x01},
		Remote:  true,
	}
	prop := propagation.TraceContext{}

	router := mux.NewRouter()
	router.Use(otelmux.Middleware(
		"foobar",
		otelmux.WithPublicEndpoint(),
		otelmux.WithPropagators(prop),
		otelmux.WithTracerProvider(provider),
	))
	router.HandleFunc("/with/public/endpoint", func(_ http.ResponseWriter, r *http.Request) {
		s := trace.SpanFromContext(r.Context())
		sc := s.SpanContext()

		// Should be with new root trace.
		assert.True(t, sc.IsValid())
		assert.False(t, sc.IsRemote())
		assert.NotEqual(t, remoteSpan.TraceID, sc.TraceID())
	})

	r0 := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/with/public/endpoint", http.NoBody)
	w := httptest.NewRecorder()

	sc := trace.NewSpanContext(remoteSpan)
	ctx := trace.ContextWithSpanContext(t.Context(), sc)
	prop.Inject(ctx, propagation.HeaderCarrier(r0.Header))

	router.ServeHTTP(w, r0)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)

	// Recorded span should be linked with an incoming span context.
	assert.NoError(t, sr.ForceFlush(ctx))
	done := sr.Ended()
	require.Len(t, done, 1)
	require.Len(t, done[0].Links(), 1, "should contain link")
	require.True(t, sc.Equal(done[0].Links()[0].SpanContext), "should link incoming span context")
}

func TestWithPublicEndpointFn(t *testing.T) {
	remoteSpan := trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x01},
		SpanID:     trace.SpanID{0x01},
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	}
	prop := propagation.TraceContext{}

	testdata := []struct {
		name          string
		fn            func(*http.Request) bool
		handlerAssert func(*testing.T, trace.SpanContext)
		spansAssert   func(*testing.T, trace.SpanContext, []sdktrace.ReadOnlySpan)
	}{
		{
			name: "with the method returning true",
			fn: func(*http.Request) bool {
				return true
			},
			handlerAssert: func(t *testing.T, sc trace.SpanContext) {
				// Should be with new root trace.
				assert.True(t, sc.IsValid())
				assert.False(t, sc.IsRemote())
				assert.NotEqual(t, remoteSpan.TraceID, sc.TraceID())
			},
			spansAssert: func(t *testing.T, sc trace.SpanContext, spans []sdktrace.ReadOnlySpan) {
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
			handlerAssert: func(t *testing.T, sc trace.SpanContext) {
				// Should have remote span as parent
				assert.True(t, sc.IsValid())
				assert.False(t, sc.IsRemote())
				assert.Equal(t, remoteSpan.TraceID, sc.TraceID())
			},
			spansAssert: func(t *testing.T, _ trace.SpanContext, spans []sdktrace.ReadOnlySpan) {
				require.Len(t, spans, 1)
				require.Empty(t, spans[0].Links(), "should not contain link")
			},
		},
	}

	for _, tt := range testdata {
		t.Run(tt.name, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider()
			provider.RegisterSpanProcessor(sr)

			router := mux.NewRouter()
			router.Use(otelmux.Middleware(
				"foobar",
				otelmux.WithPublicEndpointFn(tt.fn),
				otelmux.WithPropagators(prop),
				otelmux.WithTracerProvider(provider),
			))
			router.HandleFunc("/with/public/endpointfn", func(_ http.ResponseWriter, r *http.Request) {
				s := trace.SpanFromContext(r.Context())
				tt.handlerAssert(t, s.SpanContext())
			})

			r0 := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/with/public/endpointfn", http.NoBody)
			w := httptest.NewRecorder()

			sc := trace.NewSpanContext(remoteSpan)
			ctx := trace.ContextWithSpanContext(t.Context(), sc)
			prop.Inject(ctx, propagation.HeaderCarrier(r0.Header))

			router.ServeHTTP(w, r0)
			assert.Equal(t, http.StatusOK, w.Result().StatusCode)

			// Recorded span should be linked with an incoming span context.
			assert.NoError(t, sr.ForceFlush(ctx))
			spans := sr.Ended()
			tt.spansAssert(t, sc, spans)
		})
	}
}

func TestDefaultMetricAttributes(t *testing.T) {
	defaultMetricAttributes := []attribute.KeyValue{
		attribute.String("http.route", "/user/{id:[0-9]+}"),
		attribute.String("server.address", "foobar"),
	}

	reader := sdkmetric.NewManualReader()
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	router := mux.NewRouter()
	router.Use(otelmux.Middleware(
		"foobar",
		otelmux.WithMeterProvider(meterProvider),
	))

	router.HandleFunc("/user/{id:[0-9]+}", ok)
	r, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://localhost/user/123", http.NoBody)
	require.NoError(t, err)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, r)

	rm := metricdata.ResourceMetrics{}
	err = reader.Collect(t.Context(), &rm)
	require.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	assert.Len(t, rm.ScopeMetrics[0].Metrics, 3)

	// Verify that the additional attribute is present in the metrics.
	for _, m := range rm.ScopeMetrics[0].Metrics {
		switch d := m.Data.(type) {
		case metricdata.Histogram[int64]:
			assert.Len(t, d.DataPoints, 1)
			containsAttributes(t, d.DataPoints[0].Attributes, defaultMetricAttributes)
		case metricdata.Histogram[float64]:
			assert.Len(t, d.DataPoints, 1)
			containsAttributes(t, d.DataPoints[0].Attributes, defaultMetricAttributes)
		default:
			// Intentional failure to keep the test updated with changes in metrics
			t.Errorf("Unexpected metric type")
		}
	}
}

func TestHandlerWithMetricAttributesFn(t *testing.T) {
	const (
		serverRequestSize  = "http.server.request.body.size"
		serverResponseSize = "http.server.response.body.size"
		serverDuration     = "http.server.request.duration"
	)
	testCases := []struct {
		name                    string
		fn                      func(r *http.Request) []attribute.KeyValue
		wantAdditionalAttribute []attribute.KeyValue
	}{
		{
			name:                    "With a nil function",
			fn:                      nil,
			wantAdditionalAttribute: []attribute.KeyValue{},
		},
		{
			name: "With a function that returns an additional attribute",
			fn: func(*http.Request) []attribute.KeyValue {
				return []attribute.KeyValue{
					attribute.String("fooKey", "fooValue"),
					attribute.String("barKey", "barValue"),
				}
			},
			wantAdditionalAttribute: []attribute.KeyValue{
				attribute.String("fooKey", "fooValue"),
				attribute.String("barKey", "barValue"),
			},
		},
	}

	for _, tc := range testCases {
		reader := sdkmetric.NewManualReader()
		meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

		router := mux.NewRouter()
		router.Use(otelmux.Middleware(
			"foobar",
			otelmux.WithMeterProvider(meterProvider),
			otelmux.WithMetricAttributesFn(tc.fn),
		))

		router.HandleFunc("/user/{id:[0-9]+}", ok)
		r, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://localhost/user/123", http.NoBody)
		require.NoError(t, err)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, r)

		rm := metricdata.ResourceMetrics{}
		err = reader.Collect(t.Context(), &rm)
		require.NoError(t, err)
		require.Len(t, rm.ScopeMetrics, 1)
		assert.Len(t, rm.ScopeMetrics[0].Metrics, 3)

		// Verify that the additional attribute is present in the metrics.
		for _, m := range rm.ScopeMetrics[0].Metrics {
			switch m.Name {
			case serverRequestSize, serverResponseSize:
				d, ok := m.Data.(metricdata.Histogram[int64])
				assert.True(t, ok)
				assert.Len(t, d.DataPoints, 1)
				containsAttributes(t, d.DataPoints[0].Attributes, testCases[0].wantAdditionalAttribute)
			case serverDuration:
				d, ok := m.Data.(metricdata.Histogram[float64])
				assert.True(t, ok)
				assert.Len(t, d.DataPoints, 1)
				containsAttributes(t, d.DataPoints[0].Attributes, testCases[0].wantAdditionalAttribute)
			default:
				// Intentional failure to keep the test updated with changes in metrics
				t.Errorf("Unexpected metric name")
			}
		}
	}
}

func containsAttributes(t *testing.T, attrSet attribute.Set, expected []attribute.KeyValue) {
	for _, att := range expected {
		actualValue, ok := attrSet.Value(att.Key)
		assert.True(t, ok)
		assert.Equal(t, att.Value.AsString(), actualValue.AsString())
	}
}

func setDefaultName(method, path string) string {
	return method + " " + path
}

func ensurePrefix(prefix, s string) bool {
	return strings.HasPrefix(s, prefix)
}

// testError has a distinct ErrorType() to tell which cause was selected.
type testError string

func (e testError) Error() string     { return string(e) }
func (e testError) ErrorType() string { return string(e) }

// failingWriter is a ResponseWriter whose body writes fail.
type failingWriter struct {
	*httptest.ResponseRecorder
	err error
}

func (w failingWriter) Write([]byte) (int, error)         { return 0, w.err }
func (w failingWriter) ReadFrom(io.Reader) (int64, error) { return 0, w.err }

// failingBody is a request body whose reads fail.
type failingBody struct{ err error }

func (b failingBody) Read([]byte) (int, error) { return 0, b.err }
func (failingBody) Close() error               { return nil }

func errorType(err error) string {
	return semconv.ErrorType(err).Value.AsString()
}

// newTestHandler serves h at /user/{id} behind the middleware and returns the
// recorded telemetry.
func newTestHandler(
	h http.HandlerFunc,
	opts ...otelmux.Option,
) (http.Handler, *tracetest.SpanRecorder, sdkmetric.Reader) {
	sr := tracetest.NewSpanRecorder()
	reader := sdkmetric.NewManualReader()
	router := mux.NewRouter()
	router.Use(otelmux.Middleware("foobar", append([]otelmux.Option{
		otelmux.WithTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))),
		otelmux.WithMeterProvider(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))),
	}, opts...)...))
	router.HandleFunc("/user/{id}", h)
	return router, sr, reader
}

// TestClientDisconnect verifies that a client disconnect is recorded as the
// cause, not the 500 the handler writes afterwards.
func TestClientDisconnect(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	router, sr, reader := newTestHandler(func(w http.ResponseWriter, r *http.Request) {
		cancel()
		<-r.Context().Done()
		w.WriteHeader(http.StatusInternalServerError)
	})
	srv := httptest.NewServer(router)
	defer srv.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/user/123", http.NoBody)
	require.NoError(t, err)
	resp, err := srv.Client().Do(req)
	if err == nil {
		_ = resp.Body.Close()
	}
	require.ErrorIs(t, err, context.Canceled)
	// Close waits for the handler, so the span has ended.
	srv.Close()

	require.Len(t, sr.Ended(), 1)
	span := sr.Ended()[0]
	assert.Equal(t, codes.Error, span.Status().Code)
	assert.Equal(t, context.Canceled.Error(), span.Status().Description)
	assert.Contains(t, span.Attributes(), attribute.Int("http.response.status_code", http.StatusInternalServerError))
	assertErrorType(t, attribute.NewSet(span.Attributes()...), errorType(context.Canceled))
	assertMetricErrorType(t, reader, errorType(context.Canceled))
}

// TestErrorType verifies that a detected cause wins over the status code, in
// priority order: write error, read error, context error.
func TestErrorType(t *testing.T) {
	errWrite := testError("write_error")
	errRead := testError("read_error")
	wrappedEOF := fmt.Errorf("upload failed: %w", io.EOF)

	testCases := []struct {
		name          string
		status        int
		writeErr      error
		bodyErr       error
		body          io.Reader
		cancelCtx     bool
		wantErrorType string
	}{
		{name: "200", status: http.StatusOK},
		{name: "400", status: http.StatusBadRequest},
		{name: "500", status: http.StatusInternalServerError, wantErrorType: "500"},
		// Codes >= 600 are invalid and marked as errors by Status.
		{name: "600", status: 600, wantErrorType: "600"},
		{name: "write error", status: http.StatusOK, writeErr: errWrite, wantErrorType: "write_error"},
		{name: "read error", status: http.StatusOK, bodyErr: errRead, wantErrorType: "read_error"},
		{name: "wrapped EOF is read error", status: http.StatusOK, bodyErr: wrappedEOF, wantErrorType: errorType(wrappedEOF)},
		// A fully read body leaves io.EOF as the last read error.
		{name: "body fully read", status: http.StatusOK, body: strings.NewReader("hello")},
		{name: "context canceled", status: http.StatusOK, cancelCtx: true, wantErrorType: errorType(context.Canceled)},
		{name: "cause wins over 5xx", status: http.StatusInternalServerError, writeErr: errWrite, wantErrorType: "write_error"},
		{name: "write over read", status: http.StatusOK, writeErr: errWrite, bodyErr: errRead, wantErrorType: "write_error"},
		{name: "write over context", status: http.StatusOK, writeErr: errWrite, cancelCtx: true, wantErrorType: "write_error"},
		{name: "read over context", status: http.StatusOK, bodyErr: errRead, cancelCtx: true, wantErrorType: "read_error"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router, sr, reader := newTestHandler(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.ReadAll(r.Body)
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte("payload"))
			})

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if tc.cancelCtx {
				cancel()
			}
			var body io.Reader = http.NoBody
			switch {
			case tc.bodyErr != nil:
				body = failingBody{err: tc.bodyErr}
			case tc.body != nil:
				body = tc.body
			}
			var w http.ResponseWriter = httptest.NewRecorder()
			if tc.writeErr != nil {
				w = failingWriter{ResponseRecorder: httptest.NewRecorder(), err: tc.writeErr}
			}

			router.ServeHTTP(w, httptest.NewRequestWithContext(ctx, http.MethodPost, "/user/123", body))

			require.Len(t, sr.Ended(), 1)
			span := sr.Ended()[0]
			wantSpanStatus := codes.Unset
			if tc.wantErrorType != "" {
				wantSpanStatus = codes.Error
			}
			assert.Equal(t, wantSpanStatus, span.Status().Code)
			assert.Contains(t, span.Attributes(), attribute.Int("http.response.status_code", tc.status))
			assertErrorType(t, attribute.NewSet(span.Attributes()...), tc.wantErrorType)
			assertMetricErrorType(t, reader, tc.wantErrorType)
		})
	}
}

// TestDownstreamContextCancelIsNotError verifies that a request context
// replaced and canceled by the handler is not recorded as an error.
func TestDownstreamContextCancelIsNotError(t *testing.T) {
	router, sr, reader := newTestHandler(func(_ http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithCancel(r.Context())
		*r = *r.WithContext(ctx)
		cancel()
	})

	router.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/user/123", http.NoBody))

	require.Len(t, sr.Ended(), 1)
	assert.Equal(t, codes.Unset, sr.Ended()[0].Status().Code)
	assertErrorType(t, attribute.NewSet(sr.Ended()[0].Attributes()...), "")
	assertMetricErrorType(t, reader, "")
}

// TestMetricAttributesFnErrorTypeTakesPrecedence verifies that error.type from
// WithMetricAttributesFn overrides the detected one on metrics.
func TestMetricAttributesFnErrorTypeTakesPrecedence(t *testing.T) {
	router, sr, reader := newTestHandler(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}, otelmux.WithMetricAttributesFn(func(*http.Request) []attribute.KeyValue {
		return []attribute.KeyValue{semconv.ErrorTypeKey.String("custom")}
	}))

	router.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/user/123", http.NoBody))

	require.Len(t, sr.Ended(), 1)
	assert.Contains(t, sr.Ended()[0].Attributes(), semconv.ErrorTypeKey.String("500"), "span keeps the detected error.type")
	assertMetricErrorType(t, reader, "custom")
}

// TestReadFromWriteError verifies that an io.ReaderFrom write error is
// recorded like a Write error.
func TestReadFromWriteError(t *testing.T) {
	errReadFrom := testError("read_from_error")
	router, sr, reader := newTestHandler(func(w http.ResponseWriter, _ *http.Request) {
		// Larger than the prefix the wrapper copies through Write, so the rest
		// reaches ReadFrom. Hiding strings.Reader's WriteTo makes io.Copy use it.
		_, err := io.Copy(w, struct{ io.Reader }{strings.NewReader(strings.Repeat("a", 1024))})
		assert.ErrorIs(t, err, errReadFrom)
	})

	router.ServeHTTP(failingWriter{ResponseRecorder: httptest.NewRecorder(), err: errReadFrom},
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/user/123", http.NoBody))

	require.Len(t, sr.Ended(), 1)
	assert.Equal(t, codes.Error, sr.Ended()[0].Status().Code)
	assertErrorType(t, attribute.NewSet(sr.Ended()[0].Attributes()...), "read_from_error")
	assertMetricErrorType(t, reader, "read_from_error")
}

// assertErrorType asserts error.type is want, or absent when want is empty.
func assertErrorType(t *testing.T, attrs attribute.Set, want string) {
	t.Helper()

	got, ok := attrs.Value(semconv.ErrorTypeKey)
	assert.Equal(t, want != "", ok, "error.type presence")
	assert.Equal(t, want, got.AsString(), "error.type")
}

func assertMetricErrorType(t *testing.T, reader sdkmetric.Reader, want string) {
	t.Helper()

	rm := metricdata.ResourceMetrics{}
	require.NoError(t, reader.Collect(t.Context(), &rm))
	require.Len(t, rm.ScopeMetrics, 1)
	require.Len(t, rm.ScopeMetrics[0].Metrics, 3)

	for _, m := range rm.ScopeMetrics[0].Metrics {
		switch d := m.Data.(type) {
		case metricdata.Histogram[int64]:
			require.Len(t, d.DataPoints, 1, m.Name)
			assertErrorType(t, d.DataPoints[0].Attributes, want)
		case metricdata.Histogram[float64]:
			require.Len(t, d.DataPoints, 1, m.Name)
			assertErrorType(t, d.DataPoints[0].Attributes, want)
		default:
			t.Fatalf("unexpected metric type for %s", m.Name)
		}
	}
}

func BenchmarkMiddleware(b *testing.B) {
	tp := sdktrace.NewTracerProvider()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(sdkmetric.NewManualReader()))

	r, err := http.NewRequestWithContext(b.Context(), http.MethodGet, "/user/123", http.NoBody)
	require.NoError(b, err)

	for _, bb := range []struct {
		name   string
		status int
	}{
		{name: "ok", status: http.StatusOK},
		{name: "internal server error", status: http.StatusInternalServerError},
	} {
		b.Run(bb.name, func(b *testing.B) {
			router := mux.NewRouter()
			router.Use(otelmux.Middleware("foobar",
				otelmux.WithTracerProvider(tp),
				otelmux.WithMeterProvider(mp),
			))
			router.HandleFunc("/user/{id}", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(bb.status)
				_, _ = w.Write([]byte("Hello World"))
			})
			rr := httptest.NewRecorder()

			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				rr.Body.Reset()
				router.ServeHTTP(rr, r)
			}
		})
	}
}
