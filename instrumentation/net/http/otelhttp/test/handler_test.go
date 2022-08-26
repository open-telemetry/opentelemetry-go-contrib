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
	"go.opentelemetry.io/otel/sdk/metric/metrictest"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
)

func assertMetricAttributes(t *testing.T, expectedAttributes []attribute.KeyValue, expRec []metrictest.ExportRecord) {
	for _, r := range expRec {
		assert.ElementsMatch(t, expectedAttributes, r.Attributes)
	}
}

func TestHandlerBasics(t *testing.T) {
	rr := httptest.NewRecorder()

	spanRecorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))

	meterProvider, metricExporter := metrictest.NewTestMeterProvider()

	operation := "test_handler"

	h := otelhttp.NewHandler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			l, _ := otelhttp.LabelerFromContext(r.Context())
			l.Add(attribute.String("test", "attribute"))

			if _, err := io.WriteString(w, "hello world"); err != nil {
				t.Fatal(err)
			}
		}), operation,
		otelhttp.WithTracerProvider(provider),
		otelhttp.WithMeterProvider(meterProvider),
		otelhttp.WithPropagators(propagation.TraceContext{}),
	)

	r, err := http.NewRequest(http.MethodGet, "http://localhost/", strings.NewReader("foo"))
	if err != nil {
		t.Fatal(err)
	}
	h.ServeHTTP(rr, r)

	require.NoError(t, metricExporter.Collect(context.Background()))
	if len(metricExporter.GetRecords()) == 0 {
		t.Fatalf("got 0 recorded measurements, expected 1 or more")
	}

	attributesToVerify := []attribute.KeyValue{
		semconv.HTTPServerNameKey.String(operation),
		semconv.HTTPSchemeHTTP,
		semconv.HTTPHostKey.String(r.Host),
		semconv.HTTPFlavorKey.String(fmt.Sprintf("1.%d", r.ProtoMinor)),
		semconv.HTTPMethodKey.String("GET"),
		attribute.String("test", "attribute"),
	}

	assertMetricAttributes(t, attributesToVerify, metricExporter.GetRecords())

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
	assert.Equal(t, 200, rr.Result().StatusCode)

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
	assert.Equal(t, 200, rr.Result().StatusCode)

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
			assert.Equal(t, 200, rr.Result().StatusCode)

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
		{200, codes.Unset},
		{400, codes.Unset},
		{500, codes.Error},
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
