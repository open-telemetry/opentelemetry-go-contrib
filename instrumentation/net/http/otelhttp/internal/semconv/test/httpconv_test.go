// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp/internal/semconv"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
)

func TestNewTraceRequest(t *testing.T) {
	t.Setenv("OTEL_SEMCONV_STABILITY_OPT_IN", "http/dup")
	serv := semconv.NewHTTPServer(nil)
	want := func(req testServerReq) []attribute.KeyValue {
		return []attribute.KeyValue{
			attribute.String("http.request.method", "GET"),
			attribute.String("url.scheme", "http"),
			attribute.String("server.address", req.hostname),
			attribute.Int("server.port", req.serverPort),
			attribute.String("network.peer.address", req.peerAddr),
			attribute.Int("network.peer.port", req.peerPort),
			attribute.String("user_agent.original", "Go-http-client/1.1"),
			attribute.String("client.address", req.clientIP),
			attribute.String("network.protocol.version", "1.1"),
			attribute.String("url.path", "/"),
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

func TestNewRecordMetrics(t *testing.T) {
	oldAttrs := attribute.NewSet(
		attribute.String("http.scheme", "http"),
		attribute.String("http.method", "POST"),
		attribute.Int64("http.status_code", 301),
		attribute.String("key", "value"),
		attribute.String("net.host.name", "stuff"),
		attribute.String("net.protocol.name", "http"),
		attribute.String("net.protocol.version", "1.1"),
	)

	currAttrs := attribute.NewSet(
		attribute.String("http.request.method", "POST"),
		attribute.Int64("http.response.status_code", 301),
		attribute.String("key", "value"),
		attribute.String("network.protocol.name", "http"),
		attribute.String("network.protocol.version", "1.1"),
		attribute.String("server.address", "stuff"),
		attribute.String("url.scheme", "http"),
	)

	// The OldHTTPServer version
	expectedOldScopeMetric := metricdata.ScopeMetrics{
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
							Attributes: oldAttrs,
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
							Attributes: oldAttrs,
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
							Attributes: oldAttrs,
						},
					},
				},
			},
		},
	}

	// the CurrentHTTPServer version
	expectedCurrentScopeMetric := expectedOldScopeMetric
	expectedCurrentScopeMetric.Metrics = append(expectedCurrentScopeMetric.Metrics, []metricdata.Metrics{
		{
			Name:        "http.server.request.body.size",
			Description: "Size of HTTP server request bodies.",
			Unit:        "By",
			Data: metricdata.Histogram[int64]{
				Temporality: metricdata.CumulativeTemporality,
				DataPoints: []metricdata.HistogramDataPoint[int64]{
					{
						Attributes: currAttrs,
					},
				},
			},
		},
		{
			Name:        "http.server.response.body.size",
			Description: "Size of HTTP server response bodies.",
			Unit:        "By",
			Data: metricdata.Histogram[int64]{
				Temporality: metricdata.CumulativeTemporality,
				DataPoints: []metricdata.HistogramDataPoint[int64]{
					{
						Attributes: currAttrs,
					},
				},
			},
		},
		{
			Name:        "http.server.request.duration",
			Description: "Duration of HTTP server requests.",
			Unit:        "s",
			Data: metricdata.Histogram[float64]{
				Temporality: metricdata.CumulativeTemporality,
				DataPoints: []metricdata.HistogramDataPoint[float64]{
					{
						Attributes: currAttrs,
					},
				},
			},
		},
	}...)

	tests := []struct {
		name       string
		setEnv     bool
		serverFunc func(metric.MeterProvider) semconv.HTTPServer
		wantFunc   func(t *testing.T, rm metricdata.ResourceMetrics)
	}{
		{
			name:   "No environment variable set, and no Meter",
			setEnv: false,
			serverFunc: func(metric.MeterProvider) semconv.HTTPServer {
				return semconv.NewHTTPServer(nil)
			},
			wantFunc: func(t *testing.T, rm metricdata.ResourceMetrics) {
				assert.Empty(t, rm.ScopeMetrics)
			},
		},
		{
			name:   "No environment variable set, but with Meter",
			setEnv: false,
			serverFunc: func(mp metric.MeterProvider) semconv.HTTPServer {
				return semconv.NewHTTPServer(mp.Meter("test"))
			},
			wantFunc: func(t *testing.T, rm metricdata.ResourceMetrics) {
				require.Len(t, rm.ScopeMetrics, 1)

				// because of OldHTTPServer
				require.Len(t, rm.ScopeMetrics[0].Metrics, 3)
				metricdatatest.AssertEqual(t, expectedOldScopeMetric, rm.ScopeMetrics[0], metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue())
			},
		},
		{
			name:   "Set environment variable, but no Meter",
			setEnv: true,
			serverFunc: func(metric.MeterProvider) semconv.HTTPServer {
				return semconv.NewHTTPServer(nil)
			},
			wantFunc: func(t *testing.T, rm metricdata.ResourceMetrics) {
				assert.Empty(t, rm.ScopeMetrics)
			},
		},
		{
			name:   "Set environment variable and Meter",
			setEnv: true,
			serverFunc: func(mp metric.MeterProvider) semconv.HTTPServer {
				return semconv.NewHTTPServer(mp.Meter("test"))
			},
			wantFunc: func(t *testing.T, rm metricdata.ResourceMetrics) {
				require.Len(t, rm.ScopeMetrics, 1)
				require.Len(t, rm.ScopeMetrics[0].Metrics, 6)
				metricdatatest.AssertEqual(t, expectedCurrentScopeMetric, rm.ScopeMetrics[0], metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				t.Setenv(semconv.OTelSemConvStabilityOptIn, "http/dup")
			}

			reader := sdkmetric.NewManualReader()
			mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

			server := tt.serverFunc(mp)
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
			tt.wantFunc(t, rm)
		})
	}
}

func TestNewTraceResponse(t *testing.T) {
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
				attribute.Int("http.request.body.size", 701),
				attribute.Int("http.response.body.size", 802),
				attribute.Int("http.response.status_code", 200),
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
				attribute.Int("http.request.body.size", 701),
				attribute.Int("http.response.body.size", 802),
				attribute.Int("http.response.status_code", 200),
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got := semconv.CurrentHTTPServer{}.ResponseTraceAttrs(tt.resp)
			assert.ElementsMatch(t, tt.want, got)
		})
	}
}

func TestNewTraceRequest_Client(t *testing.T) {
	t.Setenv("OTEL_SEMCONV_STABILITY_OPT_IN", "http/dup")
	body := strings.NewReader("Hello, world!")
	url := "https://example.com:8888/foo/bar?stuff=morestuff"
	req := httptest.NewRequest("pOST", url, body)
	req.Header.Set("User-Agent", "go-test-agent")

	want := []attribute.KeyValue{
		attribute.String("http.request.method", "POST"),
		attribute.String("http.request.method_original", "pOST"),
		attribute.String("http.method", "pOST"),
		attribute.String("url.full", url),
		attribute.String("http.url", url),
		attribute.String("server.address", "example.com"),
		attribute.Int("server.port", 8888),
		attribute.String("network.protocol.version", "1.1"),
		attribute.String("net.peer.name", "example.com"),
		attribute.Int("net.peer.port", 8888),
		attribute.String("user_agent.original", "go-test-agent"),
		attribute.Int("http.request_content_length", 13),
	}
	client := semconv.NewHTTPClient(nil)
	assert.ElementsMatch(t, want, client.RequestTraceAttrs(req))
}

func TestNewTraceResponse_Client(t *testing.T) {
	t.Setenv("OTEL_SEMCONV_STABILITY_OPT_IN", "http/dup")
	testcases := []struct {
		resp http.Response
		want []attribute.KeyValue
	}{
		{resp: http.Response{StatusCode: 200, ContentLength: 123}, want: []attribute.KeyValue{attribute.Int("http.response.status_code", 200), attribute.Int("http.status_code", 200), attribute.Int("http.response_content_length", 123)}},
		{resp: http.Response{StatusCode: 404, ContentLength: 0}, want: []attribute.KeyValue{attribute.Int("http.response.status_code", 404), attribute.Int("http.status_code", 404), attribute.String("error.type", "404")}},
	}

	for _, tt := range testcases {
		client := semconv.NewHTTPClient(nil)
		assert.ElementsMatch(t, tt.want, client.ResponseTraceAttrs(&tt.resp))
	}
}

func TestClientRequest(t *testing.T) {
	body := strings.NewReader("Hello, world!")
	url := "https://example.com:8888/foo/bar?stuff=morestuff"
	req := httptest.NewRequest("pOST", url, body)
	req.Header.Set("User-Agent", "go-test-agent")

	want := []attribute.KeyValue{
		attribute.String("http.request.method", "POST"),
		attribute.String("http.request.method_original", "pOST"),
		attribute.String("url.full", url),
		attribute.String("server.address", "example.com"),
		attribute.Int("server.port", 8888),
		attribute.String("network.protocol.version", "1.1"),
	}
	got := semconv.CurrentHTTPClient{}.RequestTraceAttrs(req)
	assert.ElementsMatch(t, want, got)
}

func TestClientResponse(t *testing.T) {
	testcases := []struct {
		resp http.Response
		want []attribute.KeyValue
	}{
		{resp: http.Response{StatusCode: 200, ContentLength: 123}, want: []attribute.KeyValue{attribute.Int("http.response.status_code", 200)}},
		{resp: http.Response{StatusCode: 404, ContentLength: 0}, want: []attribute.KeyValue{attribute.Int("http.response.status_code", 404), attribute.String("error.type", "404")}},
	}

	for _, tt := range testcases {
		got := semconv.CurrentHTTPClient{}.ResponseTraceAttrs(&tt.resp)
		assert.ElementsMatch(t, tt.want, got)
	}
}

func TestRequestErrorType(t *testing.T) {
	testcases := []struct {
		err  error
		want attribute.KeyValue
	}{
		{err: errors.New("http: nil Request.URL"), want: attribute.String("error.type", "*errors.errorString")},
		{err: customError{}, want: attribute.String("error.type", "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp/internal/semconv/test.customError")},
	}

	for _, tt := range testcases {
		got := semconv.CurrentHTTPClient{}.ErrorType(tt.err)
		assert.Equal(t, tt.want, got)
	}
}

type customError struct{}

func (customError) Error() string {
	return "custom error"
}
