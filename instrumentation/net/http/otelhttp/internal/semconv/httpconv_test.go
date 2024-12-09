// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package semconv

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/attribute"
)

func TestNewTraceRequest(t *testing.T) {
	t.Setenv(OTelSemConvStabilityOptIn, "http/dup")
	serv := NewHTTPServer(nil)
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

func TestNewTraceResponse(t *testing.T) {
	testCases := []struct {
		name string
		resp ResponseTelemetry
		want []attribute.KeyValue
	}{
		{
			name: "empty",
			resp: ResponseTelemetry{},
			want: nil,
		},
		{
			name: "no errors",
			resp: ResponseTelemetry{
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
			resp: ResponseTelemetry{
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
			got := newHTTPServer{}.ResponseTraceAttrs(tt.resp)
			assert.ElementsMatch(t, tt.want, got)
		})
	}
}

func TestNewRecordMetrics(t *testing.T) {
	t.Setenv(OTelSemConvStabilityOptIn, "http/dup")
	server := NewTestHTTPServer()
	server.duplicate = true
	req, err := http.NewRequest("POST", "http://example.com", nil)
	assert.NoError(t, err)

	server.RecordMetrics(context.Background(), ServerMetricData{
		ServerName:   "stuff",
		ResponseSize: 200,
		MetricAttributes: MetricAttributes{
			Req:        req,
			StatusCode: 301,
			AdditionalAttributes: []attribute.KeyValue{
				attribute.String("key", "value"),
			},
		},
		MetricData: MetricData{
			RequestSize: 100,
			ElapsedTime: 300,
		},
	})

	assert.Equal(t, int64(100), server.requestBodySizeHistogram.(*testRecorder[int64]).value)
	assert.Equal(t, int64(200), server.responseBodySizeHistogram.(*testRecorder[int64]).value)
	assert.Equal(t, float64(300), server.requestDurationHistogram.(*testRecorder[float64]).value)

	want := []attribute.KeyValue{
		attribute.String("http.scheme", "http"),
		attribute.String("http.method", "POST"),
		attribute.Int64("http.status_code", 301),
		attribute.String("key", "value"),
		attribute.String("net.host.name", "stuff"),
		attribute.String("net.protocol.name", "http"),
		attribute.String("net.protocol.version", "1.1"),
	}
	_ = want

	// assert.ElementsMatch(t, want, server.requestBodySizeHistogram.(*testRecorder[int64]).attributes)
	// assert.ElementsMatch(t, want, server.responseBodySizeHistogram.(*testRecorder[int64]).attributes)
	// assert.ElementsMatch(t, want, server.requestDurationHistogram.(*testRecorder[float64]).attributes)
}

func TestNewMethod(t *testing.T) {
	testCases := []struct {
		method   string
		n        int
		want     attribute.KeyValue
		wantOrig attribute.KeyValue
	}{
		{
			method: http.MethodPost,
			n:      1,
			want:   attribute.String("http.request.method", "POST"),
		},
		{
			method:   "Put",
			n:        2,
			want:     attribute.String("http.request.method", "PUT"),
			wantOrig: attribute.String("http.request.method_original", "Put"),
		},
		{
			method:   "Unknown",
			n:        2,
			want:     attribute.String("http.request.method", "GET"),
			wantOrig: attribute.String("http.request.method_original", "Unknown"),
		},
	}

	for _, tt := range testCases {
		t.Run(tt.method, func(t *testing.T) {
			got, gotOrig := newHTTPServer{}.method(tt.method)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantOrig, gotOrig)
		})
	}
}

func TestNewTraceRequest_Client(t *testing.T) {
	t.Setenv(OTelSemConvStabilityOptIn, "http/dup")
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
	client := NewHTTPClient(nil)
	assert.ElementsMatch(t, want, client.RequestTraceAttrs(req))
}

func TestNewTraceResponse_Client(t *testing.T) {
	t.Setenv(OTelSemConvStabilityOptIn, "http/dup")
	testcases := []struct {
		resp http.Response
		want []attribute.KeyValue
	}{
		{resp: http.Response{StatusCode: 200, ContentLength: 123}, want: []attribute.KeyValue{attribute.Int("http.response.status_code", 200), attribute.Int("http.status_code", 200), attribute.Int("http.response_content_length", 123)}},
		{resp: http.Response{StatusCode: 404, ContentLength: 0}, want: []attribute.KeyValue{attribute.Int("http.response.status_code", 404), attribute.Int("http.status_code", 404), attribute.String("error.type", "404")}},
	}

	for _, tt := range testcases {
		client := NewHTTPClient(nil)
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
	got := newHTTPClient{}.RequestTraceAttrs(req)
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
		got := newHTTPClient{}.ResponseTraceAttrs(&tt.resp)
		assert.ElementsMatch(t, tt.want, got)
	}
}

func TestRequestErrorType(t *testing.T) {
	testcases := []struct {
		err  error
		want attribute.KeyValue
	}{
		{err: errors.New("http: nil Request.URL"), want: attribute.String("error.type", "*errors.errorString")},
		{err: customError{}, want: attribute.String("error.type", "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp/internal/semconv.customError")},
	}

	for _, tt := range testcases {
		got := newHTTPClient{}.ErrorType(tt.err)
		assert.Equal(t, tt.want, got)
	}
}

type customError struct{}

func (customError) Error() string {
	return "custom error"
}
