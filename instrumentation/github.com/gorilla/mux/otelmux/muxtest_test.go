// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelmux_test

import (
	"fmt"
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

	r := httptest.NewRequest(http.MethodGet, "/user/123", http.NoBody)
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

			r := httptest.NewRequest(http.MethodGet, "/user/123", http.NoBody)
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
			wantMethod:   http.MethodGet,
			wantRoute:    "/book/{custom}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer sr.Reset()

			r := httptest.NewRequest(tt.method, tt.path, http.NoBody)
			if tt.reqFunc != nil {
				tt.reqFunc(r)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			spans := sr.Ended()

			require.Len(t, spans, 1)
			assertSpan(t, sr.Ended()[0],
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

	r0 := httptest.NewRequest(http.MethodGet, "/does/not/exist", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r0)

	require.Len(t, sr.Ended(), 1)
	assertSpan(t, sr.Ended()[0],
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
	router.Use(otelmux.Middleware("foobar",
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

	r0 := httptest.NewRequest(http.MethodGet, "/with/public/endpoint", http.NoBody)
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
			router.Use(otelmux.Middleware("foobar",
				otelmux.WithPublicEndpointFn(tt.fn),
				otelmux.WithPropagators(prop),
				otelmux.WithTracerProvider(provider),
			))
			router.HandleFunc("/with/public/endpointfn", func(_ http.ResponseWriter, r *http.Request) {
				s := trace.SpanFromContext(r.Context())
				tt.handlerAssert(t, s.SpanContext())
			})

			r0 := httptest.NewRequest(http.MethodGet, "/with/public/endpointfn", http.NoBody)
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
	router.Use(otelmux.Middleware("foobar",
		otelmux.WithMeterProvider(meterProvider),
	))

	router.HandleFunc("/user/{id:[0-9]+}", ok)
	r, err := http.NewRequest(http.MethodGet, "http://localhost/user/123", http.NoBody)
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
		router.Use(otelmux.Middleware("foobar",
			otelmux.WithMeterProvider(meterProvider),
			otelmux.WithMetricAttributesFn(tc.fn),
		))

		router.HandleFunc("/user/{id:[0-9]+}", ok)
		r, err := http.NewRequest(http.MethodGet, "http://localhost/user/123", http.NoBody)
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
