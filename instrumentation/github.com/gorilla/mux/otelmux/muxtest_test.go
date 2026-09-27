// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelmux_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

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

// waitOrFail blocks on ch and fails the test instead of hanging past the
// package timeout.
func waitOrFail(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
	}
}

// TestClientDisconnect verifies that a real client disconnect sets error.type
// regardless of the status code the handler selects afterwards.
func TestClientDisconnect(t *testing.T) {
	testCases := []struct {
		name           string
		respond        func(http.ResponseWriter)
		wantStatusCode int
	}{
		{
			name:           "handler writes 500",
			respond:        func(w http.ResponseWriter) { w.WriteHeader(http.StatusInternalServerError) },
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			// The wrapper defaults to 200 when nothing is written.
			name:           "handler writes nothing",
			respond:        func(http.ResponseWriter) {},
			wantStatusCode: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
			reader := sdkmetric.NewManualReader()
			meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

			handlerStarted := make(chan struct{})
			router := mux.NewRouter()
			router.Use(otelmux.Middleware("foobar",
				otelmux.WithTracerProvider(provider),
				otelmux.WithMeterProvider(meterProvider),
			))
			router.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
				close(handlerStarted)
				<-r.Context().Done()
				tc.respond(w)
			})

			srv := httptest.NewServer(router)
			defer srv.Close()

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/hello", http.NoBody)
			require.NoError(t, err)

			requestDone := make(chan struct{})
			go func() {
				defer close(requestDone)
				resp, doErr := srv.Client().Do(req)
				if doErr == nil {
					_ = resp.Body.Close()
				}
			}()

			waitOrFail(t, handlerStarted, "the handler to start")
			cancel()
			waitOrFail(t, requestDone, "the request to finish")

			require.Eventually(t, func() bool {
				return len(sr.Ended()) == 1
			}, 5*time.Second, 10*time.Millisecond, "server span never ended")

			span := sr.Ended()[0]
			assert.Equal(t, codes.Error, span.Status().Code)
			assert.Contains(t, span.Attributes(), attribute.Int("http.response.status_code", tc.wantStatusCode))
			assert.Contains(t, span.Attributes(), semconv.ErrorType(context.Canceled))
			// Metrics are recorded before the span ends, so they are complete here.
			assertMetricErrorType(t, reader, semconv.ErrorType(context.Canceled).Value.AsString())
		})
	}
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
		// Codes >= 600 are invalid and marked as errors by Status.
		{600, codes.Error, "600"},
	}
	for _, tc := range testCases {
		t.Run(strconv.Itoa(tc.httpStatusCode), func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
			reader := sdkmetric.NewManualReader()
			meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

			router := mux.NewRouter()
			router.Use(otelmux.Middleware("foobar",
				otelmux.WithTracerProvider(provider),
				otelmux.WithMeterProvider(meterProvider),
			))
			router.HandleFunc("/user/{id}", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.httpStatusCode)
			})

			router.ServeHTTP(httptest.NewRecorder(),
				httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/user/123", http.NoBody))

			require.Len(t, sr.Ended(), 1)
			span := sr.Ended()[0]
			assert.Equal(t, tc.wantSpanStatus, span.Status().Code)
			if tc.wantErrorType != "" {
				assert.Contains(t, span.Attributes(), semconv.ErrorTypeKey.String(tc.wantErrorType))
			} else {
				assertNoErrorType(t, span.Attributes())
			}
			assertMetricErrorType(t, reader, tc.wantErrorType)
		})
	}
}

// TestMetricAttributesFnErrorTypeTakesPrecedence verifies that an error.type
// returned by WithMetricAttributesFn overrides the one otelmux detects.
func TestMetricAttributesFnErrorTypeTakesPrecedence(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	reader := sdkmetric.NewManualReader()
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	router := mux.NewRouter()
	router.Use(otelmux.Middleware("foobar",
		otelmux.WithTracerProvider(provider),
		otelmux.WithMeterProvider(meterProvider),
		otelmux.WithMetricAttributesFn(func(*http.Request) []attribute.KeyValue {
			return []attribute.KeyValue{semconv.ErrorTypeKey.String("custom")}
		}),
	))
	router.HandleFunc("/user/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	router.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/user/123", http.NoBody))

	require.Len(t, sr.Ended(), 1)
	assert.Contains(t, sr.Ended()[0].Attributes(), semconv.ErrorTypeKey.String("500"), "span keeps the detected error.type")
	assertMetricErrorType(t, reader, "custom")
}

// testError has a distinct ErrorType() so tests can tell which cause was
// selected for error.type.
type testError string

func (e testError) Error() string     { return string(e) }
func (e testError) ErrorType() string { return string(e) }

// failingWriter is a ResponseWriter whose body writes fail, as when the peer
// has gone away.
type failingWriter struct {
	*httptest.ResponseRecorder
	err error
}

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

// failingBody is a request body whose reads fail, as when the client's upload
// breaks mid-stream.
type failingBody struct{ err error }

func (b failingBody) Read([]byte) (int, error) { return 0, b.err }
func (failingBody) Close() error               { return nil }

// TestErrorCause verifies that a detected cause sets span status to Error and
// error.type on a 200 response, in priority order: response write error,
// request body read error, request context error.
func TestErrorCause(t *testing.T) {
	writeErr := testError("write_error")
	readErr := testError("read_error")

	testCases := []struct {
		name          string
		writeErr      error
		body          io.Reader
		cancelCtx     bool
		wantErrorType string
	}{
		{name: "write error", writeErr: writeErr, wantErrorType: "write_error"},
		{name: "read error", body: failingBody{err: readErr}, wantErrorType: "read_error"},
		// A fully read body leaves io.EOF as the last read error.
		{name: "body fully read", body: strings.NewReader("hello")},
		{name: "write over read", writeErr: writeErr, body: failingBody{err: readErr}, wantErrorType: "write_error"},
		{name: "write over context", writeErr: writeErr, cancelCtx: true, wantErrorType: "write_error"},
		{name: "read over context", body: failingBody{err: readErr}, cancelCtx: true, wantErrorType: "read_error"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
			reader := sdkmetric.NewManualReader()
			meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

			router := mux.NewRouter()
			router.Use(otelmux.Middleware("foobar",
				otelmux.WithTracerProvider(provider),
				otelmux.WithMeterProvider(meterProvider),
			))
			router.HandleFunc("/user/{id}", func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.ReadAll(r.Body)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("payload"))
			})

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if tc.cancelCtx {
				cancel()
			}
			body := tc.body
			if body == nil {
				body = http.NoBody
			}
			var w http.ResponseWriter = httptest.NewRecorder()
			if tc.writeErr != nil {
				w = failingWriter{ResponseRecorder: httptest.NewRecorder(), err: tc.writeErr}
			}

			router.ServeHTTP(w, httptest.NewRequestWithContext(ctx, http.MethodPost, "/user/123", body))

			require.Len(t, sr.Ended(), 1)
			span := sr.Ended()[0]
			assert.Contains(t, span.Attributes(), attribute.Int("http.response.status_code", http.StatusOK))
			if tc.wantErrorType != "" {
				assert.Equal(t, codes.Error, span.Status().Code)
				assert.Equal(t, tc.wantErrorType, span.Status().Description)
				assert.Contains(t, span.Attributes(), semconv.ErrorTypeKey.String(tc.wantErrorType))
			} else {
				assert.Equal(t, codes.Unset, span.Status().Code)
				assertNoErrorType(t, span.Attributes())
			}
			assertMetricErrorType(t, reader, tc.wantErrorType)
		})
	}
}

// assertMetricErrorType asserts that every server metric data point has
// error.type set to want, or has no error.type when want is empty.
func assertMetricErrorType(t *testing.T, reader sdkmetric.Reader, want string) {
	t.Helper()

	rm := metricdata.ResourceMetrics{}
	require.NoError(t, reader.Collect(t.Context(), &rm))
	require.Len(t, rm.ScopeMetrics, 1)
	require.Len(t, rm.ScopeMetrics[0].Metrics, 3)

	for _, m := range rm.ScopeMetrics[0].Metrics {
		var attrs attribute.Set
		switch d := m.Data.(type) {
		case metricdata.Histogram[int64]:
			require.Len(t, d.DataPoints, 1, m.Name)
			attrs = d.DataPoints[0].Attributes
		case metricdata.Histogram[float64]:
			require.Len(t, d.DataPoints, 1, m.Name)
			attrs = d.DataPoints[0].Attributes
		default:
			t.Fatalf("unexpected metric type for %s", m.Name)
		}

		got, ok := attrs.Value(semconv.ErrorTypeKey)
		if want == "" {
			assert.False(t, ok, "%s: error.type must not be set, got %q", m.Name, got.AsString())
		} else {
			assert.Equal(t, want, got.AsString(), m.Name)
		}
	}
}

func assertNoErrorType(t *testing.T, attrs []attribute.KeyValue) {
	t.Helper()
	for _, attr := range attrs {
		assert.NotEqual(t, semconv.ErrorTypeKey, attr.Key, "error.type must not be set")
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
				router.ServeHTTP(rr, r)
			}
		})
	}
}
