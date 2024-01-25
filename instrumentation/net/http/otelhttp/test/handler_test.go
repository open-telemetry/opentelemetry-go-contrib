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
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"
)

func assertScopeMetrics(t *testing.T, sm metricdata.ScopeMetrics, attrs attribute.Set) {
	assert.Equal(t, instrumentation.Scope{
		Name:    "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp",
		Version: otelhttp.Version(),
	}, sm.Scope)

	require.Len(t, sm.Metrics, 3)

	want := metricdata.Metrics{
		Name:        "http.server.request_content_length",
		Description: "Measures the size of HTTP request content length (uncompressed)",
		Unit:        "By",
		Data: metricdata.Sum[int64]{
			DataPoints:  []metricdata.DataPoint[int64]{{Attributes: attrs, Value: 0}},
			Temporality: metricdata.CumulativeTemporality,
			IsMonotonic: true,
		},
	}
	metricdatatest.AssertEqual(t, want, sm.Metrics[0], metricdatatest.IgnoreTimestamp())

	want = metricdata.Metrics{
		Name:        "http.server.response_content_length",
		Description: "Measures the size of HTTP response content length (uncompressed)",
		Unit:        "By",
		Data: metricdata.Sum[int64]{
			DataPoints:  []metricdata.DataPoint[int64]{{Attributes: attrs, Value: 11}},
			Temporality: metricdata.CumulativeTemporality,
			IsMonotonic: true,
		},
	}
	metricdatatest.AssertEqual(t, want, sm.Metrics[1], metricdatatest.IgnoreTimestamp())

	// Duration value is not predictable.
	dur := sm.Metrics[2]
	assert.Equal(t, "http.server.duration", dur.Name)
	require.IsType(t, dur.Data, metricdata.Histogram[float64]{})
	hist := dur.Data.(metricdata.Histogram[float64])
	assert.Equal(t, metricdata.CumulativeTemporality, hist.Temporality)
	require.Len(t, hist.DataPoints, 1)
	dPt := hist.DataPoints[0]
	assert.Equal(t, attrs, dPt.Attributes, "attributes")
	assert.Equal(t, uint64(1), dPt.Count, "count")
	assert.Equal(t, []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000}, dPt.Bounds, "bounds")
}

func TestHandlerBasics(t *testing.T) {
	rr := httptest.NewRecorder()

	spanRecorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))

	reader := metric.NewManualReader()
	meterProvider := metric.NewMeterProvider(metric.WithReader(reader))

	h := otelhttp.NewHandler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			l, _ := otelhttp.LabelerFromContext(r.Context())
			l.Add(attribute.String("test", "attribute"))

			if _, err := io.WriteString(w, "hello world"); err != nil {
				t.Fatal(err)
			}
		}), "test_handler",
		otelhttp.WithTracerProvider(provider),
		otelhttp.WithMeterProvider(meterProvider),
		otelhttp.WithPropagators(propagation.TraceContext{}),
	)

	r, err := http.NewRequest(http.MethodGet, "http://localhost/", strings.NewReader("foo"))
	if err != nil {
		t.Fatal(err)
	}
	h.ServeHTTP(rr, r)

	rm := metricdata.ResourceMetrics{}
	err = reader.Collect(context.Background(), &rm)
	require.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	attrs := attribute.NewSet(
		semconv.NetHostName(r.Host),
		semconv.HTTPSchemeHTTP,
		semconv.NetProtocolName("http"),
		semconv.NetProtocolVersion(fmt.Sprintf("1.%d", r.ProtoMinor)),
		semconv.HTTPMethod("GET"),
		attribute.String("test", "attribute"),
		semconv.HTTPStatusCode(200),
	)
	assertScopeMetrics(t, rm.ScopeMetrics[0], attrs)

	if got, expected := rr.Result().StatusCode, http.StatusOK; got != expected {
		t.Fatalf("got %d, expected %d", got, expected)
	}

	spans := spanRecorder.Ended()
	if got, expected := len(spans), 1; got != expected {
		t.Fatalf("got %d spans, expected %d", got, expected)
	}
	if !spans[0].SpanContext().IsValid() {
		t.Fatalf("invalid span created: %#v", spans[0].SpanContext())
	}

	d, err := io.ReadAll(rr.Result().Body)
	if err != nil {
		t.Fatal(err)
	}
	if got, expected := string(d), "hello world"; got != expected {
		t.Fatalf("got %q, expected %q", got, expected)
	}
}

func TestHandlerEmittedAttributes(t *testing.T) {
	testCases := []struct {
		name       string
		handler    func(http.ResponseWriter, *http.Request)
		attributes []attribute.KeyValue
	}{
		{
			name: "With a success handler",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			attributes: []attribute.KeyValue{
				attribute.Int("http.status_code", http.StatusOK),
			},
		},
		{
			name: "With a failing handler",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			},
			attributes: []attribute.KeyValue{
				attribute.Int("http.status_code", http.StatusBadRequest),
			},
		},
		{
			name: "With an empty handler",
			handler: func(w http.ResponseWriter, r *http.Request) {
			},
			attributes: []attribute.KeyValue{
				attribute.Int("http.status_code", http.StatusOK),
			},
		},
		{
			name: "With persisting initial failing status in handler with multiple WriteHeader calls",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.WriteHeader(http.StatusOK)
			},
			attributes: []attribute.KeyValue{
				attribute.Int("http.status_code", http.StatusInternalServerError),
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider()
			provider.RegisterSpanProcessor(sr)
			h := otelhttp.NewHandler(
				http.HandlerFunc(tc.handler), "test_handler",
				otelhttp.WithTracerProvider(provider),
			)

			h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))

			require.Len(t, sr.Ended(), 1, "should emit a span")
			attrs := sr.Ended()[0].Attributes()

			for _, a := range tc.attributes {
				assert.Contains(t, attrs, a)
			}
		})
	}
}

type respWriteHeaderCounter struct {
	http.ResponseWriter

	headersWritten []int
}

func (rw *respWriteHeaderCounter) WriteHeader(statusCode int) {
	rw.headersWritten = append(rw.headersWritten, statusCode)
	rw.ResponseWriter.WriteHeader(statusCode)
}

func TestHandlerPropagateWriteHeaderCalls(t *testing.T) {
	testCases := []struct {
		name                 string
		handler              func(http.ResponseWriter, *http.Request)
		expectHeadersWritten []int
	}{
		{
			name: "With a success handler",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			expectHeadersWritten: []int{http.StatusOK},
		},
		{
			name: "With a failing handler",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			},
			expectHeadersWritten: []int{http.StatusBadRequest},
		},
		{
			name: "With an empty handler",
			handler: func(w http.ResponseWriter, r *http.Request) {
			},

			expectHeadersWritten: nil,
		},
		{
			name: "With calling WriteHeader twice",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.WriteHeader(http.StatusOK)
			},
			expectHeadersWritten: []int{http.StatusInternalServerError, http.StatusOK},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider()
			provider.RegisterSpanProcessor(sr)
			h := otelhttp.NewHandler(
				http.HandlerFunc(tc.handler), "test_handler",
				otelhttp.WithTracerProvider(provider),
			)

			recorder := httptest.NewRecorder()
			rw := &respWriteHeaderCounter{ResponseWriter: recorder}
			h.ServeHTTP(rw, httptest.NewRequest("GET", "/", nil))
			require.EqualValues(t, tc.expectHeadersWritten, rw.headersWritten, "should propagate all WriteHeader calls to underlying ResponseWriter")
		})
	}
}

func TestHandlerRequestWithTraceContext(t *testing.T) {
	rr := httptest.NewRecorder()

	h := otelhttp.NewHandler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, err := w.Write([]byte("hello world"))
			require.NoError(t, err)
		}), "test_handler")

	r, err := http.NewRequest(http.MethodGet, "http://localhost/", nil)
	require.NoError(t, err)

	spanRecorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(spanRecorder),
	)
	tracer := provider.Tracer("")
	ctx, span := tracer.Start(context.Background(), "test_request")
	r = r.WithContext(ctx)

	h.ServeHTTP(rr, r)
	assert.Equal(t, http.StatusOK, rr.Result().StatusCode)

	span.End()

	spans := spanRecorder.Ended()
	require.Len(t, spans, 2)

	assert.Equal(t, "test_handler", spans[0].Name())
	assert.Equal(t, "test_request", spans[1].Name())
	assert.NotEmpty(t, spans[0].Parent().SpanID())
	assert.Equal(t, spans[1].SpanContext().SpanID(), spans[0].Parent().SpanID())
}

func TestWithPublicEndpoint(t *testing.T) {
	spanRecorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(spanRecorder),
	)
	remoteSpan := trace.SpanContextConfig{
		TraceID: trace.TraceID{0x01},
		SpanID:  trace.SpanID{0x01},
		Remote:  true,
	}
	prop := propagation.TraceContext{}
	h := otelhttp.NewHandler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s := trace.SpanFromContext(r.Context())
			sc := s.SpanContext()

			// Should be with new root trace.
			assert.True(t, sc.IsValid())
			assert.False(t, sc.IsRemote())
			assert.NotEqual(t, remoteSpan.TraceID, sc.TraceID())
		}), "test_handler",
		otelhttp.WithPublicEndpoint(),
		otelhttp.WithPropagators(prop),
		otelhttp.WithTracerProvider(provider),
	)

	r, err := http.NewRequest(http.MethodGet, "http://localhost/", nil)
	require.NoError(t, err)

	sc := trace.NewSpanContext(remoteSpan)
	ctx := trace.ContextWithSpanContext(context.Background(), sc)
	prop.Inject(ctx, propagation.HeaderCarrier(r.Header))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	assert.Equal(t, http.StatusOK, rr.Result().StatusCode)

	// Recorded span should be linked with an incoming span context.
	assert.NoError(t, spanRecorder.ForceFlush(ctx))
	done := spanRecorder.Ended()
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

	for _, tt := range []struct {
		name          string
		fn            func(*http.Request) bool
		handlerAssert func(*testing.T, trace.SpanContext)
		spansAssert   func(*testing.T, trace.SpanContext, []sdktrace.ReadOnlySpan)
	}{
		{
			name: "with the method returning true",
			fn: func(r *http.Request) bool {
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
			fn: func(r *http.Request) bool {
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
				require.Len(t, spans[0].Links(), 0, "should not contain link")
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			spanRecorder := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(
				sdktrace.WithSpanProcessor(spanRecorder),
			)

			h := otelhttp.NewHandler(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					s := trace.SpanFromContext(r.Context())
					tt.handlerAssert(t, s.SpanContext())
				}), "test_handler",
				otelhttp.WithPublicEndpointFn(tt.fn),
				otelhttp.WithPropagators(prop),
				otelhttp.WithTracerProvider(provider),
			)

			r, err := http.NewRequest(http.MethodGet, "http://localhost/", nil)
			require.NoError(t, err)

			sc := trace.NewSpanContext(remoteSpan)
			ctx := trace.ContextWithSpanContext(context.Background(), sc)
			prop.Inject(ctx, propagation.HeaderCarrier(r.Header))

			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, r)
			assert.Equal(t, http.StatusOK, rr.Result().StatusCode)

			// Recorded span should be linked with an incoming span context.
			assert.NoError(t, spanRecorder.ForceFlush(ctx))
			spans := spanRecorder.Ended()
			tt.spansAssert(t, sc, spans)
		})
	}
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
			h := otelhttp.NewHandler(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tc.httpStatusCode)
				}), "test_handler",
				otelhttp.WithTracerProvider(provider),
			)

			h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))

			require.Len(t, sr.Ended(), 1, "should emit a span")
			assert.Equal(t, sr.Ended()[0].Status().Code, tc.wantSpanStatus, "should only set Error status for HTTP statuses >= 500")
		})
	}
}

func TestWithRouteTag(t *testing.T) {
	route := "/some/route"

	spanRecorder := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider()
	tracerProvider.RegisterSpanProcessor(spanRecorder)

	metricReader := metric.NewManualReader()
	meterProvider := metric.NewMeterProvider(metric.WithReader(metricReader))

	h := otelhttp.NewHandler(
		otelhttp.WithRouteTag(
			route,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusTeapot)
			}),
		),
		"test_handler",
		otelhttp.WithTracerProvider(tracerProvider),
		otelhttp.WithMeterProvider(meterProvider),
	)

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	want := semconv.HTTPRouteKey.String(route)

	require.Len(t, spanRecorder.Ended(), 1, "should emit a span")
	gotSpan := spanRecorder.Ended()[0]
	require.Contains(t, gotSpan.Attributes(), want, "should add route to span attributes")

	rm := metricdata.ResourceMetrics{}
	err := metricReader.Collect(context.Background(), &rm)
	require.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1, "should emit metrics for one scope")
	gotMetrics := rm.ScopeMetrics[0].Metrics

	for _, m := range gotMetrics {
		switch d := m.Data.(type) {
		case metricdata.Sum[int64]:
			require.Len(t, d.DataPoints, 1, "metric '%v' should have exactly one data point", m.Name)
			require.Contains(t, d.DataPoints[0].Attributes.ToSlice(), want, "should add route to attributes for metric '%v'", m.Name)

		case metricdata.Sum[float64]:
			require.Len(t, d.DataPoints, 1, "metric '%v' should have exactly one data point", m.Name)
			require.Contains(t, d.DataPoints[0].Attributes.ToSlice(), want, "should add route to attributes for metric '%v'", m.Name)

		case metricdata.Histogram[int64]:
			require.Len(t, d.DataPoints, 1, "metric '%v' should have exactly one data point", m.Name)
			require.Contains(t, d.DataPoints[0].Attributes.ToSlice(), want, "should add route to attributes for metric '%v'", m.Name)

		case metricdata.Histogram[float64]:
			require.Len(t, d.DataPoints, 1, "metric '%v' should have exactly one data point", m.Name)
			require.Contains(t, d.DataPoints[0].Attributes.ToSlice(), want, "should add route to attributes for metric '%v'", m.Name)

		case metricdata.Gauge[int64]:
			require.Len(t, d.DataPoints, 1, "metric '%v' should have exactly one data point", m.Name)
			require.Contains(t, d.DataPoints[0].Attributes.ToSlice(), want, "should add route to attributes for metric '%v'", m.Name)

		case metricdata.Gauge[float64]:
			require.Len(t, d.DataPoints, 1, "metric '%v' should have exactly one data point", m.Name)
			require.Contains(t, d.DataPoints[0].Attributes.ToSlice(), want, "should add route to attributes for metric '%v'", m.Name)

		default:
			require.Fail(t, "metric has unexpected data type", "metric '%v' has unexpected data type %T", m.Name, m.Data)
		}
	}
}
