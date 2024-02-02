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

package semconv // import "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp/internal/semconv"

import (
	"net/http"

	"go.opentelemetry.io/otel/attribute"
	semconvOld "go.opentelemetry.io/otel/semconv/v1.20.0"
)

type dupHTTPServer struct{}

var _ HTTPServer = dupHTTPServer{}

// TraceRequest returns trace attributes for an HTTP request received by a
// server.
//
// The server must be the primary server name if it is known. For example this
// would be the ServerName directive
// (https://httpd.apache.org/docs/2.4/mod/core.html#servername) for an Apache
// server, and the server_name directive
// (http://nginx.org/en/docs/http/ngx_http_core_module.html#server_name) for an
// nginx server. More generically, the primary server name would be the host
// header value that matches the default virtual host of an HTTP server. It
// should include the host identifier and if a port is used to route to the
// server that port identifier should be included as an appropriate port
// suffix.
//
// If the primary server name is not known, server should be an empty string.
// The req Host will be used to determine the server instead.
func (d dupHTTPServer) TraceRequest(server string, req *http.Request) []attribute.KeyValue {
	// var host string
	// var p int
	// if server == "" {
	// 	host, p = splitHostPort(req.Host)
	// } else {
	// 	// Prioritize the primary server name.
	// 	host, p = splitHostPort(server)
	// 	if p < 0 {
	// 		_, p = splitHostPort(req.Host)
	// 	}
	// }

	const MaxAttributes = 12
	attrs := make([]attribute.KeyValue, MaxAttributes)
	// i := c.NetConv.HostName(host, attrs)
	// attrs[i] = c.method(req.Method)
	// i++
	// attrs[i] = c.scheme(req.TLS != nil)
	// i++

	// if hostPort := requiredHTTPPort(req.TLS != nil, p); hostPort > 0 {
	// 	i += c.NetConv.HostPort(hostPort, attrs[i:])
	// }

	// if peer, peerPort := splitHostPort(req.RemoteAddr); peer != "" {
	// 	// The Go HTTP server sets RemoteAddr to "IP:port", this will not be a
	// 	// file-path that would be interpreted with a sock family.
	// 	attrs[i] = c.NetConv.SockPeerAddr(peer)
	// 	i++
	// 	if peerPort > 0 {
	// 		attrs[i] = c.NetConv.SockPeerPort(peerPort)
	// 		i++
	// 	}
	// }

	// if useragent := req.UserAgent(); useragent != "" {
	// 	attrs[i] = c.UserAgentOriginalKey.String(useragent)
	// 	i++
	// }

	// if clientIP := serverClientIP(req.Header.Get("X-Forwarded-For")); clientIP != "" {
	// 	attrs[i] = c.HTTPClientIPKey.String(clientIP)
	// 	i++
	// }

	// if req.URL != nil && req.URL.Path != "" {
	// 	attrs[i] = c.HTTPTargetKey.String(req.URL.Path)
	// 	i++
	// }

	// protoName, protoVersion := netProtocol(req.Proto)
	// if protoName != "" && protoName != "http" {
	// 	attrs[i] = c.NetConv.NetProtocolName.String(protoName)
	// 	i++
	// }
	// if protoVersion != "" {
	// 	attrs[i] = c.NetConv.NetProtocolVersion.String(protoVersion)
	// 	i++
	// }

	// // TODO: When we drop go1.20 support use slices.clip().
	// return attrs[:i:i]
	return attrs[:0:0]
}

// MetricsRequest returns metric attributes for an HTTP request received by a
// server.
//
// The server must be the primary server name if it is known. For example this
// would be the ServerName directive
// (https://httpd.apache.org/docs/2.4/mod/core.html#servername) for an Apache
// server, and the server_name directive
// (http://nginx.org/en/docs/http/ngx_http_core_module.html#server_name) for an
// nginx server. More generically, the primary server name would be the host
// header value that matches the default virtual host of an HTTP server. It
// should include the host identifier and if a port is used to route to the
// server that port identifier should be included as an appropriate port
// suffix.
//
// If the primary server name is not known, server should be an empty string.
// The req Host will be used to determine the server instead.
func (d dupHTTPServer) MetricsRequest(server string, req *http.Request) []attribute.KeyValue {
	return nil
}

// TraceRequest returns trace attributes for telemetry from an HTTP response.
//
// If any of the fields in the ResponseTelemetry are not set the attribute will be omitted.
func (d dupHTTPServer) TraceResponse(_ ResponseTelemetry) []attribute.KeyValue {
	return nil
}

// Route returns the attribute for the route.
func (d dupHTTPServer) Route(route string) attribute.KeyValue {
	return semconvOld.HTTPRoute(route)
}
