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
	"testing"

	"go.opentelemetry.io/otel/attribute"
)

func TestV120TraceRequest(t *testing.T) {
	// Anything but "http" or "http/dup" works
	t.Setenv("OTEL_HTTP_CLIENT_COMPATIBILITY_MODE", "old")
	serv := NewHTTPServer()
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
	// Anything but "http" or "http/dup" works.
	t.Setenv("OTEL_HTTP_CLIENT_COMPATIBILITY_MODE", "old")
	serv := NewHTTPServer()
	want := []attribute.KeyValue{
		attribute.Int("http.request_content_length", 701),
		attribute.String("http.read_error", "read error"),
		attribute.Int("http.response_content_length", 802),
		attribute.String("http.write_error", "write error"),
		attribute.Int("http.status_code", 200),
	}
	testTraceResponse(t, serv, want)
}
