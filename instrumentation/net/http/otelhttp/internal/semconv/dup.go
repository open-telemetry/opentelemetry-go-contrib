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
	"io"
	"net/http"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	semconvOld "go.opentelemetry.io/otel/semconv/v1.20.0"
	semconvNew "go.opentelemetry.io/otel/semconv/v1.24.0"
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
	// old http.target http.scheme net.host.name net.host.port http.scheme net.host.name net.host.port http.method net.sock.peer.addr net.sock.peer.port user_agent.original http.method http.status_code net.protocol.version
	// new http.request.header server.address server.port network.local.address network.local.port client.address client.port url.path url.query url.scheme user_agent.original server.address server.port url.scheme http.request.method http.response.status_code error.type network.protocol.name network.protocol.version http.request.method_original http.response.header http.request.method network.peer.address network.peer.port network.transport http.request.method http.response.status_code error.type network.protocol.name network.protocol.version

	const MaxAttributes = 24
	attrs := make([]attribute.KeyValue, MaxAttributes)
	var host string
	var p int
	if server == "" {
		host, p = splitHostPort(req.Host)
	} else {
		// Prioritize the primary server name.
		host, p = splitHostPort(server)
		if p < 0 {
			_, p = splitHostPort(req.Host)
		}
	}

	attrs[0] = semconvOld.NetHostName(host)
	attrs[1] = semconvNew.ServerAddress(host)
	i := 2
	if hostPort := requiredHTTPPort(req.TLS != nil, p); hostPort > 0 {
		attrs[i] = semconvOld.NetHostPort(hostPort)
		attrs[i+1] = semconvNew.ServerPort(hostPort)
		i += 2
	}
	i += d.method(req.Method, attrs[i:])     // Max 3
	i += d.scheme(req.TLS != nil, attrs[i:]) // Max 2

	if peer, peerPort := splitHostPort(req.RemoteAddr); peer != "" {
		// The Go HTTP server sets RemoteAddr to "IP:port", this will not be a
		// file-path that would be interpreted with a sock family.
		attrs[i] = semconvOld.NetSockPeerAddr(peer)
		attrs[i+1] = semconvNew.NetworkPeerAddress(peer)
		i += 2
		if peerPort > 0 {
			attrs[i] = semconvOld.NetSockPeerPort(peerPort)
			attrs[i+1] = semconvNew.NetworkPeerPort(peerPort)
			i += 2
		}
	}

	if useragent := req.UserAgent(); useragent != "" {
		// This is the same between v1.20, and v1.24
		attrs[i] = semconvNew.UserAgentOriginal(useragent)
		i++
	}

	if clientIP := serverClientIP(req.Header.Get("X-Forwarded-For")); clientIP != "" {
		attrs[i] = semconvOld.HTTPClientIP(clientIP)
		attrs[i+1] = semconvNew.ClientAddress(clientIP)
		i += 2
	}

	if req.URL != nil && req.URL.Path != "" {
		attrs[i] = semconvOld.HTTPTarget(req.URL.Path)
		attrs[i+1] = semconvNew.URLPath(req.URL.Path)
		i += 2
	}

	protoName, protoVersion := netProtocol(req.Proto)
	if protoName != "" && protoName != "http" {
		attrs[i] = semconvOld.NetProtocolName(protoName)
		attrs[i+1] = semconvNew.NetworkProtocolName(protoName)
		i += 2
	}
	if protoVersion != "" {
		attrs[i] = semconvOld.NetProtocolVersion(protoVersion)
		attrs[i+1] = semconvNew.NetworkProtocolVersion(protoVersion)
		i += 2
	}

	// // TODO: When we drop go1.20 support use slices.clip().
	return attrs[:i:i]
}

func (d dupHTTPServer) method(method string, attrs []attribute.KeyValue) int {
	if method == "" {
		attrs[0] = semconvOld.HTTPMethod(http.MethodGet)
		attrs[1] = semconvNew.HTTPRequestMethodGet
		return 2
	}
	attrs[0] = semconvOld.HTTPMethod(method)
	if attr, ok := methodLookup[method]; ok {
		attrs[1] = attr
		return 2
	}

	if attr, ok := methodLookup[strings.ToUpper(method)]; ok {
		attrs[1] = attr
	} else {
		// If the Original method is not a standard HTTP method fallback to GET
		attrs[1] = semconvNew.HTTPRequestMethodGet
	}
	attrs[2] = semconvNew.HTTPRequestMethodOriginal(method)
	return 3
}

func (d dupHTTPServer) scheme(https bool, attrs []attribute.KeyValue) int { // nolint:revive
	if https {
		attrs[0] = semconvOld.HTTPSchemeHTTPS
		attrs[1] = semconvNew.URLScheme("https")
		return 2
	}
	attrs[0] = semconvOld.HTTPSchemeHTTP
	attrs[1] = semconvNew.URLScheme("http")
	return 2
}

// TraceRequest returns trace attributes for telemetry from an HTTP response.
//
// If any of the fields in the ResponseTelemetry are not set the attribute will be omitted.
func (d dupHTTPServer) TraceResponse(resp ResponseTelemetry) []attribute.KeyValue {
	attributes := []attribute.KeyValue{}

	if resp.ReadBytes > 0 {
		attributes = append(attributes,
			semconvOld.HTTPRequestContentLength(resp.ReadBytes),
			semconvNew.HTTPRequestBodySize(resp.ReadBytes),
		)
	}
	if resp.ReadError != nil && resp.ReadError != io.EOF {
		// This is not in the semantic conventions, but is historically provided
		attributes = append(attributes, attribute.String("http.read_error", resp.ReadError.Error()))
	}
	if resp.WriteBytes > 0 {
		attributes = append(attributes,
			semconvOld.HTTPResponseContentLength(resp.WriteBytes),
			semconvNew.HTTPResponseBodySize(resp.WriteBytes),
		)
	}
	if resp.StatusCode > 0 {
		attributes = append(attributes,
			semconvOld.HTTPStatusCode(resp.StatusCode),
			semconvNew.HTTPResponseStatusCode(resp.StatusCode),
		)
	}
	if resp.WriteError != nil && resp.WriteError != io.EOF {
		// This is not in the semantic conventions, but is historically provided
		attributes = append(attributes, attribute.String("http.write_error", resp.WriteError.Error()))
	}

	return attributes
}

// Route returns the attribute for the route.
func (d dupHTTPServer) Route(route string) attribute.KeyValue {
	return semconvNew.HTTPRoute(route)
}
