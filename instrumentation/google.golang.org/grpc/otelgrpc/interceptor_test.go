// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgrpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"

	"go.opentelemetry.io/otel/attribute"
)

func TestServerAddrAttrsFromDialTarget(t *testing.T) {
	tests := []struct {
		name   string
		target string
		want   []attribute.KeyValue
	}{
		// Plain host:port — no scheme, fast path.
		{
			name:   "plain host:port",
			target: "myservice:443",
			want:   []attribute.KeyValue{semconv.ServerAddress("myservice"), semconv.ServerPort(443)},
		},
		{
			name:   "plain IP:port",
			target: "192.168.1.1:9090",
			want:   []attribute.KeyValue{semconv.ServerAddress("192.168.1.1"), semconv.ServerPort(9090)},
		},
		{
			name:   "localhost",
			target: "localhost:8080",
			want:   []attribute.KeyValue{semconv.ServerAddress("localhost"), semconv.ServerPort(8080)},
		},
		// passthrough scheme — short form (scheme:host:port).
		{
			name:   "passthrough short form",
			target: "passthrough:127.0.0.1:7777",
			want:   []attribute.KeyValue{semconv.ServerAddress("127.0.0.1"), semconv.ServerPort(7777)},
		},
		// passthrough scheme — full URI form (scheme:///host:port).
		{
			name:   "passthrough full URI",
			target: "passthrough:///127.0.0.1:7777",
			want:   []attribute.KeyValue{semconv.ServerAddress("127.0.0.1"), semconv.ServerPort(7777)},
		},
		// dns scheme — empty authority.
		{
			name:   "dns empty authority",
			target: "dns:///example.com:443",
			want:   []attribute.KeyValue{semconv.ServerAddress("example.com"), semconv.ServerPort(443)},
		},
		// dns scheme — with authority.
		{
			name:   "dns with authority",
			target: "dns://ns.example.com/example.com:443",
			want:   []attribute.KeyValue{semconv.ServerAddress("example.com"), semconv.ServerPort(443)},
		},
		// xds scheme.
		{
			name:   "xds scheme",
			target: "xds:///my-service:50051",
			want:   []attribute.KeyValue{semconv.ServerAddress("my-service"), semconv.ServerPort(50051)},
		},
		// Unix socket — absolute path, leading slash must be preserved.
		{
			name:   "unix absolute path",
			target: "unix:///tmp/grpc.sock",
			want:   []attribute.KeyValue{semconv.ServerAddress("/tmp/grpc.sock")},
		},
		// Unix socket — relative path.
		{
			name:   "unix relative path",
			target: "unix:tmp/grpc.sock",
			want:   []attribute.KeyValue{semconv.ServerAddress("tmp/grpc.sock")},
		},
		// unix-abstract scheme.
		{
			name:   "unix-abstract",
			target: "unix-abstract:///grpc.sock",
			want:   []attribute.KeyValue{semconv.ServerAddress("/grpc.sock")},
		},
		// Non-address target (bufconn) — best-effort, return whatever endpoint we find.
		{
			name:   "passthrough bufconn",
			target: "passthrough:bufnet",
			want:   []attribute.KeyValue{semconv.ServerAddress("bufnet")},
		},
		// IPv6 — plain bracket notation handled by fast path before url.Parse.
		{
			name:   "ipv6 plain",
			target: "[::1]:8080",
			want:   []attribute.KeyValue{semconv.ServerAddress("::1"), semconv.ServerPort(8080)},
		},
		// IPv6 — passthrough short form, url.Parse places address in Opaque.
		{
			name:   "ipv6 passthrough short form",
			target: "passthrough:[::1]:8080",
			want:   []attribute.KeyValue{semconv.ServerAddress("::1"), semconv.ServerPort(8080)},
		},
		// IPv6 — passthrough full URI, address in Path.
		{
			name:   "ipv6 passthrough full URI",
			target: "passthrough:///[::1]:8080",
			want:   []attribute.KeyValue{semconv.ServerAddress("::1"), semconv.ServerPort(8080)},
		},
		// IPv6 — dns scheme.
		{
			name:   "ipv6 dns",
			target: "dns:///[::1]:8080",
			want:   []attribute.KeyValue{semconv.ServerAddress("::1"), semconv.ServerPort(8080)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := serverAddrAttrsFromDialTarget(tt.target)
			assert.Equal(t, tt.want, got)
		})
	}
}
