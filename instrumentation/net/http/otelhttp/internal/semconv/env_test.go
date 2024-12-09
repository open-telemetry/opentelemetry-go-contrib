// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package semconv

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/embedded"
	"go.opentelemetry.io/otel/metric/noop"
)

func TestHTTPServerDoesNotPanic(t *testing.T) {
	testCases := []struct {
		name   string
		server HTTPServer
	}{
		{
			name:   "empty",
			server: HTTPServer{},
		},
		{
			name:   "nil meter",
			server: NewHTTPServer(nil),
		},
		{
			name:   "with Meter",
			server: NewHTTPServer(noop.Meter{}),
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			require.NotPanics(t, func() {
				req, err := http.NewRequest("GET", "http://example.com", nil)
				require.NoError(t, err)

				_ = tt.server.RequestTraceAttrs("stuff", req)
				_ = tt.server.ResponseTraceAttrs(ResponseTelemetry{StatusCode: 200})
				tt.server.RecordMetrics(context.Background(), ServerMetricData{
					ServerName: "stuff",
					MetricAttributes: MetricAttributes{
						Req: req,
					},
				})
			})
		})
	}
}

func TestHTTPClientDoesNotPanic(t *testing.T) {
	testCases := []struct {
		name   string
		client HTTPClient
	}{
		{
			name:   "empty",
			client: HTTPClient{},
		},
		{
			name:   "nil meter",
			client: NewHTTPClient(nil),
		},
		{
			name:   "with Meter",
			client: NewHTTPClient(noop.Meter{}),
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			require.NotPanics(t, func() {
				req, err := http.NewRequest("GET", "http://example.com", nil)
				require.NoError(t, err)

				_ = tt.client.RequestTraceAttrs(req)
				_ = tt.client.ResponseTraceAttrs(&http.Response{StatusCode: 200})

				opts := tt.client.MetricOptions(MetricAttributes{
					Req:        req,
					StatusCode: 200,
				})
				tt.client.RecordResponseSize(context.Background(), 40, opts.AddOptions())
				tt.client.RecordMetrics(context.Background(), MetricData{
					RequestSize: 20,
					ElapsedTime: 1,
				}, opts)
			})
		})
	}
}

type testRecorder[T any] struct {
	embedded.Int64Counter
	embedded.Int64Histogram
	embedded.Float64Histogram

	value      T
	attributes []attribute.KeyValue
}

var (
	_ metric.Int64Counter     = (*testRecorder[int64])(nil)
	_ metric.Float64Histogram = (*testRecorder[float64])(nil)
	_ metric.Int64Histogram   = (*testRecorder[int64])(nil)
	_ metric.Float64Histogram = (*testRecorder[float64])(nil)
)

func (t *testRecorder[T]) Add(_ context.Context, incr T, options ...metric.AddOption) {
	t.value = incr
	cfg := metric.NewAddConfig(options)
	attr := cfg.Attributes()
	t.attributes = attr.ToSlice()
}

func (t *testRecorder[T]) Record(_ context.Context, value T, options ...metric.RecordOption) {
	t.value = value
	cfg := metric.NewRecordConfig(options)
	attr := cfg.Attributes()
	t.attributes = attr.ToSlice()
}

func NewTestHTTPServer() HTTPServer {
	return HTTPServer{
		requestBytesCounter:       &testRecorder[int64]{},
		responseBytesCounter:      &testRecorder[int64]{},
		serverLatencyMeasure:      &testRecorder[float64]{},
		requestBodySizeHistogram:  &testRecorder[int64]{},
		responseBodySizeHistogram: &testRecorder[int64]{},
		requestDurationHistogram:  &testRecorder[float64]{},
	}
}

func NewTestHTTPClient() HTTPClient {
	return HTTPClient{
		requestBytesCounter:  &testRecorder[int64]{},
		responseBytesCounter: &testRecorder[int64]{},
		latencyMeasure:       &testRecorder[float64]{},
	}
}
