// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"bytes"
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

	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

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

	// Run a server and stop to make sure nothing is listening and force the error.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close()

	// Create our Transport and make request.
	tr := otelhttp.NewTransport(
		http.DefaultTransport,
		otelhttp.WithTracerProvider(provider),
	)
	c := http.Client{Transport: tr}
	r, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.Do(r)
	if err == nil {
		t.Fatal("transport should have returned an error, it didn't")
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("close response body: %v", err)
		}
	}()

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
	defer func() { assert.NoError(t, res.Body.Close()) }()

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
	defer func() { assert.NoError(t, res.Body.Close()) }()

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
	requestBody := []byte("john")
	responseBody := []byte("Hello, world!")

	t.Run("make http request and read entire response at once", func(t *testing.T) {
		reader := sdkmetric.NewManualReader()
		meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write(responseBody); err != nil {
				t.Fatal(err)
			}
		}))
		defer ts.Close()

		r, err := http.NewRequest(http.MethodGet, ts.URL, bytes.NewReader(requestBody))
		if err != nil {
			t.Fatal(err)
		}

		tr := otelhttp.NewTransport(
			http.DefaultTransport,
			otelhttp.WithMeterProvider(meterProvider),
		)

		c := http.Client{Transport: tr}
		res, err := c.Do(r)
		if err != nil {
			t.Fatal(err)
		}

		// Must read the body or else we won't get response metrics
		bodyBytes, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		require.Len(t, bodyBytes, 13)
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
			semconv.HTTPMethod("GET"),
			semconv.HTTPStatusCode(200),
		)
		assertClientScopeMetrics(t, rm.ScopeMetrics[0], attrs, 13)
	})

	t.Run("make http request and buffer response", func(t *testing.T) {
		reader := sdkmetric.NewManualReader()
		meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write(responseBody); err != nil {
				t.Fatal(err)
			}
		}))
		defer ts.Close()

		r, err := http.NewRequest(http.MethodGet, ts.URL, bytes.NewReader(requestBody))
		if err != nil {
			t.Fatal(err)
		}

		tr := otelhttp.NewTransport(
			http.DefaultTransport,
			otelhttp.WithMeterProvider(meterProvider),
		)

		c := http.Client{Transport: tr}
		res, err := c.Do(r)
		if err != nil {
			t.Fatal(err)
		}

		// Must read the body or else we won't get response metrics
		smallBuf := make([]byte, 10)

		// Read first 10 bytes
		bc, err := res.Body.Read(smallBuf)
		if err != nil {
			t.Fatal(err)
		}
		require.Equal(t, 10, bc)

		// reset byte array
		// Read last 3 bytes
		bc, err = res.Body.Read(smallBuf)
		require.Equal(t, io.EOF, err)
		require.Equal(t, 3, bc)

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
			semconv.HTTPMethod("GET"),
			semconv.HTTPStatusCode(200),
		)
		assertClientScopeMetrics(t, rm.ScopeMetrics[0], attrs, 13)
	})

	t.Run("make http request and close body before reading completely", func(t *testing.T) {
		reader := sdkmetric.NewManualReader()
		meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write(responseBody); err != nil {
				t.Fatal(err)
			}
		}))
		defer ts.Close()

		r, err := http.NewRequest(http.MethodGet, ts.URL, bytes.NewReader(requestBody))
		if err != nil {
			t.Fatal(err)
		}

		tr := otelhttp.NewTransport(
			http.DefaultTransport,
			otelhttp.WithMeterProvider(meterProvider),
		)

		c := http.Client{Transport: tr}
		res, err := c.Do(r)
		if err != nil {
			t.Fatal(err)
		}

		// Must read the body or else we won't get response metrics
		smallBuf := make([]byte, 10)

		// Read first 10 bytes
		bc, err := res.Body.Read(smallBuf)
		if err != nil {
			t.Fatal(err)
		}
		require.Equal(t, 10, bc)

		// close the response body early
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
			semconv.HTTPMethod("GET"),
			semconv.HTTPStatusCode(200),
		)
		assertClientScopeMetrics(t, rm.ScopeMetrics[0], attrs, 10)
	})
}

func assertClientScopeMetrics(t *testing.T, sm metricdata.ScopeMetrics, attrs attribute.Set, rxBytes int64) {
	assert.Equal(t, instrumentation.Scope{
		Name:    "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp",
		Version: Version(),
	}, sm.Scope)

	require.Len(t, sm.Metrics, 3)

	want := metricdata.Metrics{
		Name: "http.client.request.size",
		Data: metricdata.Sum[int64]{
			DataPoints:  []metricdata.DataPoint[int64]{{Attributes: attrs, Value: 4}},
			Temporality: metricdata.CumulativeTemporality,
			IsMonotonic: true,
		},
		Description: "Measures the size of HTTP request messages.",
		Unit:        "By",
	}
	metricdatatest.AssertEqual(t, want, sm.Metrics[0], metricdatatest.IgnoreTimestamp())

	want = metricdata.Metrics{
		Name: "http.client.response.size",
		Data: metricdata.Sum[int64]{
			DataPoints:  []metricdata.DataPoint[int64]{{Attributes: attrs, Value: rxBytes}},
			Temporality: metricdata.CumulativeTemporality,
			IsMonotonic: true,
		},
		Description: "Measures the size of HTTP response messages.",
		Unit:        "By",
	}
	metricdatatest.AssertEqual(t, want, sm.Metrics[1], metricdatatest.IgnoreTimestamp())

	want = metricdata.Metrics{
		Name: "http.client.duration",
		Data: metricdata.Histogram[float64]{
			DataPoints:  []metricdata.HistogramDataPoint[float64]{{Attributes: attrs}},
			Temporality: metricdata.CumulativeTemporality,
		},
		Description: "Measures the duration of outbound HTTP requests.",
		Unit:        "ms",
	}
	metricdatatest.AssertEqual(t, want, sm.Metrics[2], metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue())
}

func TestCustomAttributesHandling(t *testing.T) {
	var rm metricdata.ResourceMetrics
	const (
		clientRequestSize = "http.client.request.size"
		clientDuration    = "http.client.duration"
	)
	ctx := context.TODO()
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	defer func() {
		err := provider.Shutdown(ctx)
		if err != nil {
			t.Errorf("Error shutting down provider: %v", err)
		}
	}()

	transport := otelhttp.NewTransport(http.DefaultTransport, otelhttp.WithMeterProvider(provider))
	client := http.Client{Transport: transport}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	r, err := http.NewRequest(http.MethodGet, ts.URL, nil)
	require.NoError(t, err)
	labeler := &otelhttp.Labeler{}
	labeler.Add(attribute.String("foo", "fooValue"))
	labeler.Add(attribute.String("bar", "barValue"))
	ctx = otelhttp.ContextWithLabeler(ctx, labeler)
	r = r.WithContext(ctx)

	// test bonus: intententionally ignoring response to confirm that
	// http.client.response.size metric is not recorded
	// by the Transport.RoundTrip logic
	resp, err := client.Do(r)
	require.NoError(t, err)
	defer func() { assert.NoError(t, resp.Body.Close()) }()

	err = reader.Collect(ctx, &rm)
	assert.NoError(t, err)

	// http.client.response.size is not recorded so the assert.Len
	// above should be 2 instead of 3(test bonus)
	assert.Len(t, rm.ScopeMetrics[0].Metrics, 2)
	for _, m := range rm.ScopeMetrics[0].Metrics {
		switch m.Name {
		case clientRequestSize:
			d, ok := m.Data.(metricdata.Sum[int64])
			assert.True(t, ok)
			assert.Len(t, d.DataPoints, 1)
			attrSet := d.DataPoints[0].Attributes
			fooAtrr, ok := attrSet.Value(attribute.Key("foo"))
			assert.True(t, ok)
			assert.Equal(t, "fooValue", fooAtrr.AsString())
			barAtrr, ok := attrSet.Value(attribute.Key("bar"))
			assert.True(t, ok)
			assert.Equal(t, "barValue", barAtrr.AsString())
			assert.False(t, attrSet.HasValue(attribute.Key("baz")))
		case clientDuration:
			d, ok := m.Data.(metricdata.Histogram[float64])
			assert.True(t, ok)
			assert.Len(t, d.DataPoints, 1)
			attrSet := d.DataPoints[0].Attributes
			fooAtrr, ok := attrSet.Value(attribute.Key("foo"))
			assert.True(t, ok)
			assert.Equal(t, "fooValue", fooAtrr.AsString())
			barAtrr, ok := attrSet.Value(attribute.Key("bar"))
			assert.True(t, ok)
			assert.Equal(t, "barValue", barAtrr.AsString())
			assert.False(t, attrSet.HasValue(attribute.Key("baz")))
		}
	}
}

func BenchmarkTransportRoundTrip(b *testing.B) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello World")
	}))
	defer ts.Close()

	tp := sdktrace.NewTracerProvider()
	mp := sdkmetric.NewMeterProvider()

	r, err := http.NewRequest(http.MethodGet, ts.URL, nil)
	require.NoError(b, err)

	for _, bb := range []struct {
		name      string
		transport http.RoundTripper
	}{
		{
			name:      "without the otelhttp transport",
			transport: http.DefaultTransport,
		},
		{
			name: "with the otelhttp transport",
			transport: otelhttp.NewTransport(
				http.DefaultTransport,
				otelhttp.WithTracerProvider(tp),
				otelhttp.WithMeterProvider(mp),
			),
		},
	} {
		b.Run(bb.name, func(b *testing.B) {
			c := http.Client{Transport: bb.transport}

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				resp, _ := c.Do(r)
				resp.Body.Close()
			}
		})
	}
}
