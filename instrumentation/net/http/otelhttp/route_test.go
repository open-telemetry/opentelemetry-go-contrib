// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelhttp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

type routeTestContextKey struct{}

func copyRouteRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), routeTestContextKey{}, "copied")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func TestRouteMiddlewareRequestCopy(t *testing.T) {
	for _, tt := range []struct {
		name    string
		pattern string
		method  string
		clone   bool
	}{
		{name: "path", pattern: "/items/{id}", method: http.MethodGet},
		{name: "clone", pattern: "/items/{id}", method: http.MethodGet, clone: true},
		{name: "method", pattern: "GET /items/{id}", method: http.MethodGet},
		{name: "host", pattern: "example.com/items/{id}", method: http.MethodGet},
		{name: "method and host", pattern: "GET example.com/items/{id}", method: http.MethodGet},
		{name: "nonstandard method", pattern: "CUSTOM /items/{id}", method: "CUSTOM"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("OTEL_METRICS_EXEMPLAR_FILTER", "always_off")
			reader := sdkmetric.NewManualReader()
			mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
			recorder := tracetest.NewSpanRecorder()
			tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
			mux := http.NewServeMux()
			mux.HandleFunc(tt.pattern, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "copied", r.Context().Value(routeTestContextKey{}))
				labeler, found := LabelerFromContext(r.Context())
				assert.True(t, found)
				labeler.Add(attribute.String("application", "test"))
				w.WriteHeader(http.StatusOK)
			})
			var metricPattern, metricID string
			copyRequest := copyRouteRequest
			if tt.clone {
				copyRequest = func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						ctx := context.WithValue(r.Context(), routeTestContextKey{}, "copied")
						next.ServeHTTP(w, r.Clone(ctx))
					})
				}
			}
			h := NewHandler(copyRequest(RouteMiddleware(mux)), "server",
				WithTracerProvider(tp), WithMeterProvider(mp),
				WithMetricAttributesFn(func(r *http.Request) []attribute.KeyValue {
					metricPattern, metricID = r.Pattern, r.PathValue("id")
					return nil
				}))
			r := httptest.NewRequestWithContext(t.Context(), tt.method, "http://example.com/items/42", http.NoBody)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, r)
			require.Equal(t, http.StatusOK, rr.Code)
			require.Len(t, recorder.Ended(), 1)
			wantMethod := tt.method
			if tt.method == "CUSTOM" {
				wantMethod = "HTTP"
			}
			assert.Equal(t, wantMethod+" /items/{id}", recorder.Ended()[0].Name())
			assert.Contains(t, recorder.Ended()[0].Attributes(), attribute.String("http.route", "/items/{id}"))
			assert.Equal(t, tt.pattern, metricPattern)
			assert.Equal(t, "42", metricID)
			assertRouteMetrics(t, reader, "/items/{id}", attribute.String("application", "test"))
			assert.Empty(t, r.Pattern, "instrumentation must not change the original request")
		})
	}
}

func assertRouteMetrics(t *testing.T, reader *sdkmetric.ManualReader, route string, extra ...attribute.KeyValue) {
	t.Helper()
	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(t.Context(), &rm))
	require.Len(t, rm.ScopeMetrics, 1)
	require.Len(t, rm.ScopeMetrics[0].Metrics, 3)
	for _, m := range rm.ScopeMetrics[0].Metrics {
		var attrs []attribute.KeyValue
		switch data := m.Data.(type) {
		case metricdata.Histogram[int64]:
			require.Len(t, data.DataPoints, 1)
			attrs = data.DataPoints[0].Attributes.ToSlice()
		case metricdata.Histogram[float64]:
			require.Len(t, data.DataPoints, 1)
			attrs = data.DataPoints[0].Attributes.ToSlice()
		default:
			t.Fatalf("unexpected metric type %T", m.Data)
		}
		if route != "" {
			assert.Contains(t, attrs, attribute.String("http.route", route), "metric %s", m.Name)
		} else {
			for _, attr := range attrs {
				assert.NotEqual(t, attribute.Key("http.route"), attr.Key, "metric %s", m.Name)
			}
		}
		for _, attr := range extra {
			assert.Contains(t, attrs, attr, "metric %s", m.Name)
		}
	}
}

func TestRouteMiddlewareSpanNameFormatter(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /items/{id}", func(http.ResponseWriter, *http.Request) {})
	var patterns []string
	h := NewHandler(copyRouteRequest(RouteMiddleware(mux)), "server",
		WithTracerProvider(tp),
		WithSpanNameFormatter(func(operation string, r *http.Request) string {
			patterns = append(patterns, r.Pattern)
			return fmt.Sprintf("%s: %s", operation, r.Pattern)
		}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/items/42", http.NoBody))
	require.Len(t, recorder.Ended(), 1)
	assert.Equal(t, "server: GET /items/{id}", recorder.Ended()[0].Name())
	assert.Equal(t, []string{"", "GET /items/{id}"}, patterns, "format once at span start and once after routing")
}

func TestRouteMiddlewareNestedMux(t *testing.T) {
	t.Setenv("OTEL_METRICS_EXEMPLAR_FILTER", "always_off")
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	inner := http.NewServeMux()
	inner.HandleFunc("GET /api/items/{id}", func(http.ResponseWriter, *http.Request) {})
	outer := http.NewServeMux()
	outer.Handle("/api/", copyRouteRequest(RouteMiddleware(inner)))
	h := NewHandler(RouteMiddleware(outer), "server", WithTracerProvider(tp), WithMeterProvider(mp))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/items/42", http.NoBody))
	require.Len(t, recorder.Ended(), 1)
	assert.Equal(t, "GET /api/items/{id}", recorder.Ended()[0].Name())
	assert.Contains(t, recorder.Ended()[0].Attributes(), attribute.String("http.route", "/api/items/{id}"))
	assertRouteMetrics(t, reader, "/api/items/{id}")
}

func TestRouteMiddlewareUnmatched(t *testing.T) {
	for _, tt := range []struct {
		name   string
		method string
		path   string
		status int
	}{
		{name: "not found", method: http.MethodGet, path: "/missing/42", status: http.StatusNotFound},
		{name: "method not allowed", method: http.MethodPost, path: "/items/42", status: http.StatusMethodNotAllowed},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("OTEL_METRICS_EXEMPLAR_FILTER", "always_off")
			reader := sdkmetric.NewManualReader()
			mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
			recorder := tracetest.NewSpanRecorder()
			tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
			mux := http.NewServeMux()
			mux.HandleFunc("GET /items/{id}", func(http.ResponseWriter, *http.Request) {})
			h := NewHandler(copyRouteRequest(RouteMiddleware(mux)), "server", WithTracerProvider(tp), WithMeterProvider(mp))
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, httptest.NewRequestWithContext(t.Context(), tt.method, tt.path, http.NoBody))
			assert.Equal(t, tt.status, rr.Code)
			require.Len(t, recorder.Ended(), 1)
			assert.Equal(t, tt.method, recorder.Ended()[0].Name())
			for _, attr := range recorder.Ended()[0].Attributes() {
				assert.NotEqual(t, attribute.Key("http.route"), attr.Key)
			}
			assertRouteMetrics(t, reader, "")
		})
	}
}

func TestRouteMiddlewarePanic(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /items/{id}", func(http.ResponseWriter, *http.Request) { panic("handler panic") })
	h := NewHandler(copyRouteRequest(RouteMiddleware(mux)), "server", WithTracerProvider(tp))
	assert.PanicsWithValue(t, "handler panic", func() {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/items/42", http.NoBody))
	})
	require.Len(t, recorder.Ended(), 1)
	assert.Equal(t, "GET /items/{id}", recorder.Ended()[0].Name())
	assert.Contains(t, recorder.Ended()[0].Attributes(), attribute.String("http.route", "/items/{id}"))
}

func TestRouteMiddlewareRecoveredPanic(t *testing.T) {
	t.Setenv("OTEL_METRICS_EXEMPLAR_FILTER", "always_off")
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /items/{id}", func(http.ResponseWriter, *http.Request) { panic("handler panic") })
	next := copyRouteRequest(RouteMiddleware(mux))
	recoverPanic := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recover() != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
	h := NewHandler(recoverPanic, "server", WithTracerProvider(tp), WithMeterProvider(mp))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/items/42", http.NoBody))
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	require.Len(t, recorder.Ended(), 1)
	assert.Equal(t, "GET /items/{id}", recorder.Ended()[0].Name())
	assert.Contains(t, recorder.Ended()[0].Attributes(), attribute.String("http.route", "/items/{id}"))
	assert.Contains(t, recorder.Ended()[0].Attributes(), attribute.Int("http.response.status_code", http.StatusInternalServerError))
	assertRouteMetrics(t, reader, "/items/{id}")
}

func TestRouteMiddlewareNoPatternPreservesSpanName(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	h := NewHandler(RouteMiddleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		trace.SpanFromContext(r.Context()).SetName("application")
	})), "server", WithTracerProvider(tp))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/items/42", http.NoBody))
	require.Len(t, recorder.Ended(), 1)
	assert.Equal(t, "application", recorder.Ended()[0].Name())
	assert.NotContains(t, recorder.Ended()[0].Attributes(), attribute.String("http.route", "/items/42"))
}

func TestRouteMiddlewareWithoutHandler(t *testing.T) {
	var gotRequest *http.Request
	h := RouteMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRequest = r
		trace.SpanFromContext(r.Context()).SetName("application")
		w.WriteHeader(http.StatusAccepted)
	}))
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	ctx, span := tp.Tracer("test").Start(t.Context(), "initial")
	r := httptest.NewRequestWithContext(ctx, http.MethodGet, "/items/42", http.NoBody)
	r.Pattern = "GET /items/{id}"
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	span.End()
	assert.Same(t, r, gotRequest)
	assert.Equal(t, http.StatusAccepted, rr.Code)
	require.Len(t, recorder.Ended(), 1)
	assert.Equal(t, "application", recorder.Ended()[0].Name())
	assert.Empty(t, recorder.Ended()[0].Attributes())
}

func TestRouteMiddlewareChildSpan(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /items/{id}", func(http.ResponseWriter, *http.Request) {})
	withChildSpan := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tp.Tracer("test").Start(r.Context(), "child")
		defer span.End()
		RouteMiddleware(mux).ServeHTTP(w, r.WithContext(ctx))
	})
	h := NewHandler(withChildSpan, "server", WithTracerProvider(tp))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/items/42", http.NoBody))
	require.Len(t, recorder.Ended(), 2)
	assert.Equal(t, "child", recorder.Ended()[0].Name())
	assert.NotContains(t, recorder.Ended()[0].Attributes(), attribute.String("http.route", "/items/{id}"))
	assert.Equal(t, "GET /items/{id}", recorder.Ended()[1].Name())
	assert.Contains(t, recorder.Ended()[1].Attributes(), attribute.String("http.route", "/items/{id}"))
}

func TestRouteMiddlewareSharedLabeler(t *testing.T) {
	t.Setenv("OTEL_METRICS_EXEMPLAR_FILTER", "always_off")
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	labeler := &Labeler{}
	appAttr := attribute.String("application", "test")
	labeler.Add(appAttr)
	ctx := ContextWithLabeler(t.Context(), labeler)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /items/{id}", func(http.ResponseWriter, *http.Request) {})
	h := NewHandler(copyRouteRequest(RouteMiddleware(mux)), "server", WithMeterProvider(mp))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(ctx, http.MethodGet, "/items/42", http.NoBody))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(ctx, http.MethodGet, "/missing/42", http.NoBody))
	assert.Equal(t, []attribute.KeyValue{appAttr}, labeler.Get(), "route capture must not mutate a reusable Labeler")
	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(t.Context(), &rm))
	require.Len(t, rm.ScopeMetrics, 1)
	for _, m := range rm.ScopeMetrics[0].Metrics {
		var sets []attribute.Set
		switch data := m.Data.(type) {
		case metricdata.Histogram[int64]:
			for _, dp := range data.DataPoints {
				sets = append(sets, dp.Attributes)
			}
		case metricdata.Histogram[float64]:
			for _, dp := range data.DataPoints {
				sets = append(sets, dp.Attributes)
			}
		default:
			t.Fatalf("unexpected metric type %T", m.Data)
		}
		require.Len(t, sets, 2)
		for _, attrs := range sets {
			status, found := attrs.Value("http.response.status_code")
			require.True(t, found)
			route, matched := attrs.Value("http.route")
			if status.AsInt64() == http.StatusOK {
				assert.True(t, matched)
				assert.Equal(t, "/items/{id}", route.AsString())
			} else {
				assert.False(t, matched, "unmatched request metric %s must not inherit a route", m.Name)
			}
			assert.Contains(t, attrs.ToSlice(), appAttr)
		}
	}
}

func BenchmarkHandlerRouteMiddleware(b *testing.B) {
	tp := sdktrace.NewTracerProvider()
	mp := sdkmetric.NewMeterProvider()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /items/{id}", func(http.ResponseWriter, *http.Request) {})
	for _, bb := range []struct {
		name    string
		handler http.Handler
	}{
		{name: "copying middleware", handler: copyRouteRequest(mux)},
		{name: "otelhttp", handler: NewHandler(copyRouteRequest(mux), "server", WithTracerProvider(tp), WithMeterProvider(mp))},
		{name: "otelhttp with route middleware", handler: NewHandler(copyRouteRequest(RouteMiddleware(mux)), "server", WithTracerProvider(tp), WithMeterProvider(mp))},
	} {
		b.Run(bb.name, func(b *testing.B) {
			r := httptest.NewRequestWithContext(b.Context(), http.MethodGet, "/items/42", http.NoBody)
			rr := httptest.NewRecorder()
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				bb.handler.ServeHTTP(rr, r)
			}
		})
	}
}
