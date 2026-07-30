// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgrpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

func TestServerAddrAttrsFromCanonicalTarget(t *testing.T) {
	tests := []struct {
		name            string
		canonicalTarget string
		want            []attribute.KeyValue
	}{
		// passthrough scheme.
		{
			name:            "passthrough",
			canonicalTarget: "passthrough:///127.0.0.1:7777",
			want:            []attribute.KeyValue{semconv.ServerAddress("127.0.0.1"), semconv.ServerPort(7777)},
		},
		// dns scheme: empty authority.
		{
			name:            "dns empty authority",
			canonicalTarget: "dns:///example.com:443",
			want:            []attribute.KeyValue{semconv.ServerAddress("example.com"), semconv.ServerPort(443)},
		},
		// dns scheme: with authority.
		{
			name:            "dns with authority",
			canonicalTarget: "dns://ns.example.com/example.com:443",
			want:            []attribute.KeyValue{semconv.ServerAddress("example.com"), semconv.ServerPort(443)},
		},
		// dns scheme: plain host:port normalized by gRPC (e.g. "myservice:443" → "dns:///myservice:443").
		{
			name:            "dns normalized host:port",
			canonicalTarget: "dns:///myservice:443",
			want:            []attribute.KeyValue{semconv.ServerAddress("myservice"), semconv.ServerPort(443)},
		},
		// dns scheme: named port, non-numeric port omitted from attributes.
		{
			name:            "dns named port",
			canonicalTarget: "dns:///svc:https",
			want:            []attribute.KeyValue{semconv.ServerAddress("svc")},
		},
		// xds scheme: requires google.golang.org/grpc/xds to be imported in the
		// client binary; without it grpc.NewClient produces "dns:///xds:///..." instead.
		{
			name:            "xds",
			canonicalTarget: "xds:///my-service:50051",
			want:            []attribute.KeyValue{semconv.ServerAddress("my-service"), semconv.ServerPort(50051)},
		},
		// Unix socket: absolute path, leading slash preserved.
		{
			name:            "unix absolute path",
			canonicalTarget: "unix:///tmp/grpc.sock",
			want:            []attribute.KeyValue{semconv.ServerAddress("/tmp/grpc.sock")},
		},
		// unix-abstract scheme.
		{
			name:            "unix-abstract",
			canonicalTarget: "unix-abstract:///grpc.sock",
			want:            []attribute.KeyValue{semconv.ServerAddress("/grpc.sock")},
		},
		// passthrough bufconn: non-address endpoint, best-effort.
		{
			name:            "passthrough bufconn",
			canonicalTarget: "passthrough:///bufnet",
			want:            []attribute.KeyValue{semconv.ServerAddress("bufnet")},
		},
		// IPv6: dns scheme.
		{
			name:            "ipv6 dns",
			canonicalTarget: "dns:///[::1]:8080",
			want:            []attribute.KeyValue{semconv.ServerAddress("::1"), semconv.ServerPort(8080)},
		},
		// IPv6: passthrough scheme.
		{
			name:            "ipv6 passthrough",
			canonicalTarget: "passthrough:///[::1]:8080",
			want:            []attribute.KeyValue{semconv.ServerAddress("::1"), semconv.ServerPort(8080)},
		},
		// url.Parse error path: invalid percent-encoding makes url.Parse fail;
		// too many colons cause SplitHostPort to fail first, so we reach url.Parse.
		{
			name:            "invalid url escape",
			canonicalTarget: "dns:///example%zz.com:443",
			want:            []attribute.KeyValue{semconv.ServerAddress("dns:///example%zz.com:443")},
		},
		// Fast path: bare host:port with no scheme — SplitHostPort succeeds directly.
		{
			name:            "bare host:port",
			canonicalTarget: "127.0.0.1:7777",
			want:            []attribute.KeyValue{semconv.ServerAddress("127.0.0.1"), semconv.ServerPort(7777)},
		},
		// Fast path: bare IPv6 address with numeric port.
		{
			name:            "bare ipv6:port",
			canonicalTarget: "[::1]:8080",
			want:            []attribute.KeyValue{semconv.ServerAddress("::1"), semconv.ServerPort(8080)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := serverAddrAttrsFromCanonicalTarget(tt.canonicalTarget)
			assert.Equal(t, tt.want, got)
		})
	}
}
