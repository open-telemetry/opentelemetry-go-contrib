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
	// Anything but "http" or "http/dup" works
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
