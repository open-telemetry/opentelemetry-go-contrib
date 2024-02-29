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

package semconv

import (
	"net/http"
	"testing"

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
	// Anything but "http" or "http/dup" works
	t.Setenv("OTEL_HTTP_CLIENT_COMPATIBILITY_MODE", "http")
	serv := NewHTTPServer()
	want := []attribute.KeyValue{
		attribute.Int("http.request.body.size", 701),
		attribute.Int("http.response.body.size", 802),
		attribute.Int("http.response.status_code", 200),
	}
	testTraceResponse(t, serv, want)
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
