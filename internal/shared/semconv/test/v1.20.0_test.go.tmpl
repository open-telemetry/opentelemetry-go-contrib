// Code created by gotmpl. DO NOT MODIFY.
// source: internal/shared/semconv/test/v1.20.0_test.go.tmpl

// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp/internal/semconv"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
)

func TestV120TraceRequest(t *testing.T) {
	// Anything but "http" or "http/dup" works.
	t.Setenv("OTEL_SEMCONV_STABILITY_OPT_IN", "old")
	serv := semconv.NewHTTPServer(nil)
	want := func(req testServerReq) []attribute.KeyValue {
		return []attribute.KeyValue{
			attribute.String("http.method", "GET"),
			attribute.String("http.scheme", "http"),
			attribute.String("net.host.name", req.hostname),
			attribute.Int("net.host.port", req.serverPort),
			attribute.String("net.sock.peer.addr", req.peerAddr),
			attribute.Int("net.sock.peer.port", req.peerPort),
			attribute.String("user_agent.original", "Go-http-client/1.1"),
			attribute.String("http.client_ip", req.clientIP),
			attribute.String("net.protocol.version", "1.1"),
			attribute.String("http.target", "/"),
		}
	}
	testTraceRequest(t, serv, want)
}

func TestV120TraceResponse(t *testing.T) {
	testCases := []struct {
		name string
		resp semconv.ResponseTelemetry
		want []attribute.KeyValue
	}{
		{
			name: "empty",
			resp: semconv.ResponseTelemetry{},
			want: nil,
		},
		{
			name: "no errors",
			resp: semconv.ResponseTelemetry{
				StatusCode: 200,
				ReadBytes:  701,
				WriteBytes: 802,
			},
			want: []attribute.KeyValue{
				attribute.Int("http.request_content_length", 701),
				attribute.Int("http.response_content_length", 802),
				attribute.Int("http.status_code", 200),
			},
		},
		{
			name: "with errors",
			resp: semconv.ResponseTelemetry{
				StatusCode: 200,
				ReadBytes:  701,
				ReadError:  fmt.Errorf("read error"),
				WriteBytes: 802,
				WriteError: fmt.Errorf("write error"),
			},
			want: []attribute.KeyValue{
				attribute.Int("http.request_content_length", 701),
				attribute.String("http.read_error", "read error"),
				attribute.Int("http.response_content_length", 802),
				attribute.String("http.write_error", "write error"),
				attribute.Int("http.status_code", 200),
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got := semconv.OldHTTPServer{}.ResponseTraceAttrs(tt.resp)
			assert.ElementsMatch(t, tt.want, got)
		})
	}
}

func TestV120RecordMetrics(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	server := semconv.NewHTTPServer(mp.Meter("test"))
	req, err := http.NewRequest("POST", "http://example.com", nil)
	assert.NoError(t, err)

	server.RecordMetrics(context.Background(), semconv.ServerMetricData{
		ServerName:   "stuff",
		ResponseSize: 200,
		MetricAttributes: semconv.MetricAttributes{
			Req:        req,
			StatusCode: 301,
			AdditionalAttributes: []attribute.KeyValue{
				attribute.String("key", "value"),
			},
		},
		MetricData: semconv.MetricData{
			RequestSize: 100,
			ElapsedTime: 300,
		},
	})

	rm := metricdata.ResourceMetrics{}
	require.NoError(t, reader.Collect(context.Background(), &rm))
	require.Len(t, rm.ScopeMetrics, 1)
	require.Len(t, rm.ScopeMetrics[0].Metrics, 3)

	attrs := attribute.NewSet(
		attribute.String("http.scheme", "http"),
		attribute.String("http.method", "POST"),
		attribute.Int64("http.status_code", 301),
		attribute.String("key", "value"),
		attribute.String("net.host.name", "stuff"),
		attribute.String("net.protocol.name", "http"),
		attribute.String("net.protocol.version", "1.1"),
	)

	expectedScopeMetric := metricdata.ScopeMetrics{
		Scope: instrumentation.Scope{
			Name: "test",
		},
		Metrics: []metricdata.Metrics{
			{
				Name:        "http.server.request.size",
				Description: "Measures the size of HTTP request messages.",
				Unit:        "By",
				Data: metricdata.Sum[int64]{
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: true,
					DataPoints: []metricdata.DataPoint[int64]{
						{
							Attributes: attrs,
						},
					},
				},
			},
			{
				Name:        "http.server.response.size",
				Description: "Measures the size of HTTP response messages.",
				Unit:        "By",
				Data: metricdata.Sum[int64]{
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: true,
					DataPoints: []metricdata.DataPoint[int64]{
						{
							Attributes: attrs,
						},
					},
				},
			},
			{
				Name:        "http.server.duration",
				Description: "Measures the duration of inbound HTTP requests.",
				Unit:        "ms",
				Data: metricdata.Histogram[float64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.HistogramDataPoint[float64]{
						{
							Attributes: attrs,
						},
					},
				},
			},
		},
	}

	metricdatatest.AssertEqual(t, expectedScopeMetric, rm.ScopeMetrics[0], metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue())
}

func TestV120ClientRequest(t *testing.T) {
	body := strings.NewReader("Hello, world!")
	url := "https://example.com:8888/foo/bar?stuff=morestuff"
	req, err := http.NewRequest("POST", url, body)
	assert.NoError(t, err)
	req.Header.Set("User-Agent", "go-test-agent")

	want := []attribute.KeyValue{
		attribute.String("http.method", "POST"),
		attribute.String("http.url", url),
		attribute.String("net.peer.name", "example.com"),
		attribute.Int("net.peer.port", 8888),
		attribute.Int("http.request_content_length", body.Len()),
		attribute.String("user_agent.original", "go-test-agent"),
	}
	got := semconv.OldHTTPClient{}.RequestTraceAttrs(req)
	assert.ElementsMatch(t, want, got)
}

func TestV120ClientResponse(t *testing.T) {
	resp := http.Response{
		StatusCode:    200,
		ContentLength: 123,
	}

	want := []attribute.KeyValue{
		attribute.Int("http.response_content_length", 123),
		attribute.Int("http.status_code", 200),
	}

	got := semconv.OldHTTPClient{}.ResponseTraceAttrs(&resp)
	assert.ElementsMatch(t, want, got)
}

func TestV120ClientMetrics(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	client := semconv.NewHTTPClient(mp.Meter("test"))
	req, err := http.NewRequest("POST", "http://example.com", nil)
	assert.NoError(t, err)

	opts := client.MetricOptions(semconv.MetricAttributes{
		Req:        req,
		StatusCode: 301,
		AdditionalAttributes: []attribute.KeyValue{
			attribute.String("key", "value"),
		},
	})

	ctx := context.Background()

	client.RecordResponseSize(ctx, 200, opts)

	client.RecordMetrics(ctx, semconv.MetricData{
		RequestSize: 100,
		ElapsedTime: 300,
	}, opts)

	rm := metricdata.ResourceMetrics{}
	require.NoError(t, reader.Collect(context.Background(), &rm))
	require.Len(t, rm.ScopeMetrics, 1)
	require.Len(t, rm.ScopeMetrics[0].Metrics, 3)

	attrs := attribute.NewSet(
		attribute.String("http.method", "POST"),
		attribute.Int64("http.status_code", 301),
		attribute.String("key", "value"),
		attribute.String("net.peer.name", "example.com"),
	)

	expectedScopeMetric := metricdata.ScopeMetrics{
		Scope: instrumentation.Scope{
			Name: "test",
		},
		Metrics: []metricdata.Metrics{
			{
				Name:        "http.client.request.size",
				Description: "Measures the size of HTTP request messages.",
				Unit:        "By",
				Data: metricdata.Sum[int64]{
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: true,
					DataPoints: []metricdata.DataPoint[int64]{
						{
							Attributes: attrs,
						},
					},
				},
			},
			{
				Name:        "http.client.response.size",
				Description: "Measures the size of HTTP response messages.",
				Unit:        "By",
				Data: metricdata.Sum[int64]{
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: true,
					DataPoints: []metricdata.DataPoint[int64]{
						{
							Attributes: attrs,
						},
					},
				},
			},
			{
				Name:        "http.client.duration",
				Description: "Measures the duration of outbound HTTP requests.",
				Unit:        "ms",
				Data: metricdata.Histogram[float64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.HistogramDataPoint[float64]{
						{
							Attributes: attrs,
						},
					},
				},
			},
		},
	}

	metricdatatest.AssertEqual(t, expectedScopeMetric, rm.ScopeMetrics[0], metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue())
}
