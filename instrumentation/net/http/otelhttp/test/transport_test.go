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
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"runtime"
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
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

func assertClientScopeMetrics(t *testing.T, sm metricdata.ScopeMetrics, attrs attribute.Set) {
	assert.Equal(t, instrumentation.Scope{
		Name:    "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp",
		Version: otelhttp.Version(),
	}, sm.Scope)

	require.Len(t, sm.Metrics, 3)

	want := metricdata.Metrics{
		Name: "http.client.request_content_length",
		Data: metricdata.Sum[int64]{
			DataPoints:  []metricdata.DataPoint[int64]{{Attributes: attrs, Value: 0}},
			Temporality: metricdata.CumulativeTemporality,
			IsMonotonic: true,
		},
	}
	metricdatatest.AssertEqual(t, want, sm.Metrics[0], metricdatatest.IgnoreTimestamp())

	want = metricdata.Metrics{
		Name: "http.client.response_content_length",
		Data: metricdata.Sum[int64]{
			DataPoints:  []metricdata.DataPoint[int64]{{Attributes: attrs, Value: 13}},
			Temporality: metricdata.CumulativeTemporality,
			IsMonotonic: true,
		},
	}
	metricdatatest.AssertEqual(t, want, sm.Metrics[1], metricdatatest.IgnoreTimestamp())

	// Duration value is not predictable.
	dur := sm.Metrics[2]
	assert.Equal(t, "http.client.duration", dur.Name)
	require.IsType(t, dur.Data, metricdata.Histogram[float64]{})
	hist := dur.Data.(metricdata.Histogram[float64])
	assert.Equal(t, metricdata.CumulativeTemporality, hist.Temporality)
	require.Len(t, hist.DataPoints, 1)
	dPt := hist.DataPoints[0]
	assert.Equal(t, attrs, dPt.Attributes, "attributes")
	assert.Equal(t, uint64(1), dPt.Count, "count")
	assert.Equal(t, []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000}, dPt.Bounds, "bounds")
}

func TestTransportUsesFormatter(t *testing.T) {
	prop := propagation.TraceContext{}
	spanRecorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	content := []byte("Hello, world!")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := prop.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		span := trace.SpanContextFromContext(ctx)
		if !span.IsValid() {
			t.Fatalf("invalid span wrapping handler: %#v", span)
		}
		if _, err := w.Write(content); err != nil {
			t.Fatal(err)
		}
	}))
	defer ts.Close()

	r, err := http.NewRequest(http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	tr := otelhttp.NewTransport(
		http.DefaultTransport,
		otelhttp.WithTracerProvider(provider),
		otelhttp.WithPropagators(prop),
	)

	c := http.Client{Transport: tr}
	res, err := c.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	require.NoError(t, res.Body.Close())

	spans := spanRecorder.Ended()
	spanName := spans[0].Name()
	expectedName := "HTTP GET"
	if spanName != expectedName {
		t.Fatalf("unexpected name: got %s, expected %s", spanName, expectedName)
	}
}

func TestTransportErrorStatus(t *testing.T) {
	// Prepare tracing stuff.
	spanRecorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))

	reader := metric.NewManualReader()
	meterProvider := metric.NewMeterProvider(metric.WithReader(reader))

	// Run a server and stop to make sure nothing is listening and force the error.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close()

	// Create our Transport and make request.
	tr := otelhttp.NewTransport(
		http.DefaultTransport,
		otelhttp.WithTracerProvider(provider),
		otelhttp.WithMeterProvider(meterProvider),
	)
	c := http.Client{Transport: tr}
	r, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Do(r)
	if err == nil {
		t.Fatal("transport should have returned an error, it didn't")
	}

	// Check span.
	spans := spanRecorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span; got: %d", len(spans))
	}
	span := spans[0]

	if span.EndTime().IsZero() {
		t.Errorf("span should be ended; it isn't")
	}

	if got := span.Status().Code; got != codes.Error {
		t.Errorf("expected error status code on span; got: %q", got)
	}

	errSubstr := "connect: connection refused"
	if runtime.GOOS == "windows" {
		// tls.Dial returns an error that does not contain the substring "connection refused"
		// on Windows machines
		//
		// ref: "dial tcp 127.0.0.1:50115: connectex: No connection could be made because the target machine actively refused it."
		errSubstr = "No connection could be made because the target machine actively refused it"
	}
	if got := span.Status().Description; !strings.Contains(got, errSubstr) {
		t.Errorf("expected error status message on span; got: %q", got)
	}

	// check metrics
	rm := metricdata.ResourceMetrics{}
	err = reader.Collect(context.Background(), &rm)
	require.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	require.Len(t, rm.ScopeMetrics[0].Metrics, 2) // response length isn't added on error

	metricdatatest.AssertHasAttributes(t, rm.ScopeMetrics[0].Metrics[0].Data.(metricdata.Sum[int64]),
		semconv.HTTPStatusCode(http.StatusBadRequest))
}

func TestTransportRequestWithTraceContext(t *testing.T) {
	spanRecorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(spanRecorder),
	)
	content := []byte("Hello, world!")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write(content)
		require.NoError(t, err)
	}))
	defer ts.Close()

	tracer := provider.Tracer("")
	ctx, span := tracer.Start(context.Background(), "test_span")

	r, err := http.NewRequest(http.MethodGet, ts.URL, nil)
	require.NoError(t, err)

	r = r.WithContext(ctx)

	tr := otelhttp.NewTransport(
		http.DefaultTransport,
	)

	c := http.Client{Transport: tr}
	res, err := c.Do(r)
	require.NoError(t, err)

	span.End()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	require.Equal(t, content, body)

	spans := spanRecorder.Ended()
	require.Len(t, spans, 2)

	assert.Equal(t, "test_span", spans[0].Name())
	assert.Equal(t, "HTTP GET", spans[1].Name())
	assert.NotEmpty(t, spans[1].Parent().SpanID())
	assert.Equal(t, spans[0].SpanContext().SpanID(), spans[1].Parent().SpanID())
}

func TestWithHTTPTrace(t *testing.T) {
	spanRecorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(spanRecorder),
	)
	content := []byte("Hello, world!")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write(content)
		require.NoError(t, err)
	}))
	defer ts.Close()

	tracer := provider.Tracer("")
	ctx, span := tracer.Start(context.Background(), "test_span")

	r, err := http.NewRequest(http.MethodGet, ts.URL, nil)
	require.NoError(t, err)

	r = r.WithContext(ctx)

	clientTracer := func(ctx context.Context) *httptrace.ClientTrace {
		var span trace.Span
		return &httptrace.ClientTrace{
			GetConn: func(_ string) {
				_, span = trace.SpanFromContext(ctx).TracerProvider().Tracer("").Start(ctx, "httptrace.GetConn")
			},
			GotConn: func(_ httptrace.GotConnInfo) {
				if span != nil {
					span.End()
				}
			},
		}
	}

	tr := otelhttp.NewTransport(
		http.DefaultTransport,
		otelhttp.WithClientTrace(clientTracer),
	)

	c := http.Client{Transport: tr}
	res, err := c.Do(r)
	require.NoError(t, err)

	span.End()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	require.Equal(t, content, body)

	spans := spanRecorder.Ended()
	require.Len(t, spans, 3)

	assert.Equal(t, "httptrace.GetConn", spans[0].Name())
	assert.Equal(t, "test_span", spans[1].Name())
	assert.Equal(t, "HTTP GET", spans[2].Name())
	assert.NotEmpty(t, spans[0].Parent().SpanID())
	assert.NotEmpty(t, spans[2].Parent().SpanID())
	assert.Equal(t, spans[2].SpanContext().SpanID(), spans[0].Parent().SpanID())
	assert.Equal(t, spans[1].SpanContext().SpanID(), spans[2].Parent().SpanID())
}

func TestTransportMetrics(t *testing.T) {
	reader := metric.NewManualReader()
	meterProvider := metric.NewMeterProvider(metric.WithReader(reader))

	content := []byte("Hello, world!")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(content); err != nil {
			t.Fatal(err)
		}
	}))
	defer ts.Close()

	r, err := http.NewRequest(http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	tr := otelhttp.NewTransport(
		http.DefaultTransport,
		otelhttp.WithMeterProvider(meterProvider),
		otelhttp.WithRequestAttributeGetter(func(req *http.Request) []attribute.KeyValue {
			return []attribute.KeyValue{attribute.String("test_req", "attribute_req")}
		}),
		otelhttp.WithResponseAttributeGetter(func(req *http.Response) []attribute.KeyValue {
			return []attribute.KeyValue{attribute.String("test_resp", "attribute_resp")}
		}),
	)

	c := http.Client{Transport: tr}
	res, err := c.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	require.NoError(t, res.Body.Close())

	host, portStr, _ := net.SplitHostPort(r.Host)
	if host == "" {
		host = "127.0.0.1"
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		port = 0
	}

	rm := metricdata.ResourceMetrics{}
	err = reader.Collect(context.Background(), &rm)
	require.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	attrs := attribute.NewSet(
		semconv.NetPeerName(host),
		semconv.NetPeerPort(port),
		semconv.HTTPURL(ts.URL),
		semconv.HTTPFlavorKey.String(fmt.Sprintf("1.%d", r.ProtoMinor)),
		semconv.HTTPMethod("GET"),
		semconv.HTTPResponseContentLength(13),
		attribute.String("test_req", "attribute_req"),
		attribute.String("test_resp", "attribute_resp"),
		semconv.HTTPStatusCode(200),
	)
	assertClientScopeMetrics(t, rm.ScopeMetrics[0], attrs)
}
