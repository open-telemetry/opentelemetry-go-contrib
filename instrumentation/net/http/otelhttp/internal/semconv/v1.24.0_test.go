// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package semconv

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/attribute"
)

func TestNewTraceRequest(t *testing.T) {
	t.Setenv("OTEL_HTTP_CLIENT_COMPATIBILITY_MODE", "http")
	serv := NewHTTPServer()
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

func TestNewMethod(t *testing.T) {
	testCases := []struct {
		method string
		n      int
		want   []attribute.KeyValue
	}{
		{
			method: http.MethodPost,
			n:      1,
			want: []attribute.KeyValue{
				attribute.String("http.request.method", "POST"),
			},
		},
		{
			method: "Put",
			n:      2,
			want: []attribute.KeyValue{
				attribute.String("http.request.method", "PUT"),
				attribute.String("http.request.method_original", "Put"),
			},
		},
		{
			method: "Unknown",
			n:      2,
			want: []attribute.KeyValue{
				attribute.String("http.request.method", "GET"),
				attribute.String("http.request.method_original", "Unknown"),
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.method, func(t *testing.T) {
			attrs := make([]attribute.KeyValue, 5)
			n := newHTTPServer{}.method(tt.method, attrs[1:])
			require.Equal(t, tt.n, n, "Length doesn't match")
			require.ElementsMatch(t, tt.want, attrs[1:n+1])
		})
	}
}
