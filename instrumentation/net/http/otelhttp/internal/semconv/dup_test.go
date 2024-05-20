// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package semconv

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/attribute"
)

func TestDupTraceRequest(t *testing.T) {
	t.Setenv("OTEL_HTTP_CLIENT_COMPATIBILITY_MODE", "http/dup")
	serv := NewHTTPServer()
	want := func(req testServerReq) []attribute.KeyValue {
		return []attribute.KeyValue{
			attribute.String("http.method", "GET"),
			attribute.String("http.request.method", "GET"),
			attribute.String("http.scheme", "http"),
			attribute.String("url.scheme", "http"),
			attribute.String("net.host.name", req.hostname),
			attribute.String("server.address", req.hostname),
			attribute.Int("net.host.port", req.serverPort),
			attribute.Int("server.port", req.serverPort),
			attribute.String("net.sock.peer.addr", req.peerAddr),
			attribute.String("network.peer.address", req.peerAddr),
			attribute.Int("net.sock.peer.port", req.peerPort),
			attribute.Int("network.peer.port", req.peerPort),
			attribute.String("user_agent.original", "Go-http-client/1.1"),
			attribute.String("http.client_ip", req.clientIP),
			attribute.String("client.address", req.clientIP),
			attribute.String("net.protocol.version", "1.1"),
			attribute.String("network.protocol.version", "1.1"),
			attribute.String("http.target", "/"),
			attribute.String("url.path", "/"),
		}
	}
	testTraceRequest(t, serv, want)
}

func TestDupServerRequestTraceAttrs(t *testing.T) {
	// This test covers edge cases not covered by the test above.

	testCases := []struct {
		name    string
		server  string
		request *http.Request
		want    []attribute.KeyValue
	}{
		{
			name:   "Server No port, request with Port",
			server: "server",
			request: &http.Request{
				Host: "127.0.5.6:8080",
			},
			want: []attribute.KeyValue{
				attribute.String("net.host.name", "server"),
				attribute.String("server.address", "server"),
				attribute.Int("net.host.port", 8080),
				attribute.Int("server.port", 8080),
			},
		},
		{
			name: "Proto is not HTTP",
			request: &http.Request{
				Proto: "ftp/1.0",
			},
			want: []attribute.KeyValue{
				attribute.String("net.protocol.name", "ftp"),
				attribute.String("network.protocol.name", "ftp"),
			},
		},
		{
			name: "Method is empty",
			request: &http.Request{
				Method: "",
			},
			want: []attribute.KeyValue{
				attribute.String("http.method", "GET"),
				attribute.String("http.request.method", "GET"),
			},
		},
		{
			name: "https schema",
			request: &http.Request{
				TLS: &tls.ConnectionState{},
			},
			want: []attribute.KeyValue{
				attribute.String("http.scheme", "https"),
				attribute.String("url.scheme", "https"),
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got := dupHTTPServer{}.RequestTraceAttrs(tt.server, tt.request)
			for _, want := range tt.want {
				assert.Contains(t, got, want)
			}
		})
	}
}

func TestDupMethod(t *testing.T) {
	testCases := []struct {
		method   string
		n        int
		wantOld  attribute.KeyValue
		wantNew  attribute.KeyValue
		wantOrig attribute.KeyValue
	}{
		{
			method: http.MethodPost,
			n:      2,

			wantOld: attribute.String("http.method", "POST"),
			wantNew: attribute.String("http.request.method", "POST"),
		},
		{
			method:   "Put",
			n:        3,
			wantOld:  attribute.String("http.method", "Put"),
			wantNew:  attribute.String("http.request.method", "PUT"),
			wantOrig: attribute.String("http.request.method_original", "Put"),
		},
		{
			method: "Unknown",
			n:      3,

			wantOld:  attribute.String("http.method", "Unknown"),
			wantNew:  attribute.String("http.request.method", "GET"),
			wantOrig: attribute.String("http.request.method_original", "Unknown"),
		},
	}

	for _, tt := range testCases {
		t.Run(tt.method, func(t *testing.T) {
			gotOld, gotNew, gotOrig := dupHTTPServer{}.method(tt.method)
			assert.Equal(t, tt.wantOld, gotOld)
			assert.Equal(t, tt.wantNew, gotNew)
			assert.Equal(t, tt.wantOrig, gotOrig)
		})
	}
}

func TestDupTraceResponse(t *testing.T) {
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
				attribute.Int("http.request_content_length", 701),
				attribute.Int("http.request.body.size", 701),
				attribute.Int("http.response_content_length", 802),
				attribute.Int("http.response.body.size", 802),
				attribute.Int("http.status_code", 200),
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
				attribute.Int("http.request_content_length", 701),
				attribute.Int("http.request.body.size", 701),
				attribute.String("http.read_error", "read error"),
				attribute.Int("http.response_content_length", 802),
				attribute.Int("http.response.body.size", 802),
				attribute.String("http.write_error", "write error"),
				attribute.Int("http.status_code", 200),
				attribute.Int("http.response.status_code", 200),
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got := dupHTTPServer{}.ResponseTraceAttrs(tt.resp)
			assert.ElementsMatch(t, tt.want, got)
		})
	}
}

var benchDupAttrs []attribute.KeyValue

func BenchmarkDupServerRequestTraceAttrs(b *testing.B) {
	b.Run("NoPort", benchDupServerRequestTraceAttrs("server"))
	b.Run("WithPort", benchDupServerRequestTraceAttrs("server:8080"))
}

func benchDupServerRequestTraceAttrs(server string) func(*testing.B) {
	return func(b *testing.B) {
		b.ReportAllocs()
		serv := dupHTTPServer{}
		req := &http.Request{}
		for i := 0; i < b.N; i++ {
			benchDupAttrs = serv.RequestTraceAttrs(server, req)
		}
		b.ReportMetric(float64(len(benchDupAttrs)), "attrs")
	}
}
