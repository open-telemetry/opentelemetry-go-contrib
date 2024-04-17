// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package semconv // import "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp/internal/semconv"

import (
	"net/http"
	"slices"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	semconvNew "go.opentelemetry.io/otel/semconv/v1.24.0"
)

type newHTTPServer struct{}

var _ HTTPServer = newHTTPServer{}

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
func (n newHTTPServer) RequestTraceAttrs(server string, req *http.Request) []attribute.KeyValue {
	const MaxAttributes = 11
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

	attrs[0] = semconvNew.ServerAddress(host)
	i := 1
	if hostPort := requiredHTTPPort(req.TLS != nil, p); hostPort > 0 {
		attrs[i] = semconvNew.ServerPort(hostPort)
		i++
	}
	i += n.method(req.Method, attrs[i:])     // Max 2
	i += n.scheme(req.TLS != nil, attrs[i:]) // Max 1

	if peer, peerPort := splitHostPort(req.RemoteAddr); peer != "" {
		// The Go HTTP server sets RemoteAddr to "IP:port", this will not be a
		// file-path that would be interpreted with a sock family.
		attrs[i] = semconvNew.NetworkPeerAddress(peer)
		i++
		if peerPort > 0 {
			attrs[i] = semconvNew.NetworkPeerPort(peerPort)
			i++
		}
	}

	if useragent := req.UserAgent(); useragent != "" {
		// This is the same between v1.20, and v1.24
		attrs[i] = semconvNew.UserAgentOriginal(useragent)
		i++
	}

	if clientIP := serverClientIP(req.Header.Get("X-Forwarded-For")); clientIP != "" {
		attrs[i] = semconvNew.ClientAddress(clientIP)
		i++
	}

	if req.URL != nil && req.URL.Path != "" {
		attrs[i] = semconvNew.URLPath(req.URL.Path)
		i++
	}

	protoName, protoVersion := netProtocol(req.Proto)
	if protoName != "" && protoName != "http" {
		attrs[i] = semconvNew.NetworkProtocolName(protoName)
		i++
	}
	if protoVersion != "" {
		attrs[i] = semconvNew.NetworkProtocolVersion(protoVersion)
		i++
	}

	return slices.Clip(attrs[:i])
}

func (n newHTTPServer) method(method string, attrs []attribute.KeyValue) int {
	if method == "" {
		attrs[0] = semconvNew.HTTPRequestMethodGet
		return 1
	}
	if attr, ok := methodLookup[method]; ok {
		attrs[0] = attr
		return 1
	}

	if attr, ok := methodLookup[strings.ToUpper(method)]; ok {
		attrs[0] = attr
	} else {
		// If the Original methos is not a standard HTTP method fallback to GET
		attrs[0] = semconvNew.HTTPRequestMethodGet
	}
	attrs[1] = semconvNew.HTTPRequestMethodOriginal(method)
	return 2
}

func (n newHTTPServer) scheme(https bool, attrs []attribute.KeyValue) int { // nolint:revive
	if https {
		attrs[0] = semconvNew.URLScheme("https")
		return 1
	}
	attrs[0] = semconvNew.URLScheme("http")
	return 1
}

// TraceResponse returns trace attributes for telemetry from an HTTP response.
//
// If any of the fields in the ResponseTelemetry are not set the attribute will be omitted.
func (n newHTTPServer) ResponseTraceAttrs(resp ResponseTelemetry) []attribute.KeyValue {
	attributes := []attribute.KeyValue{}

	if resp.ReadBytes > 0 {
		attributes = append(attributes,
			semconvNew.HTTPRequestBodySize(int(resp.ReadBytes)),
		)
	}
	if resp.WriteBytes > 0 {
		attributes = append(attributes,
			semconvNew.HTTPResponseBodySize(int(resp.WriteBytes)),
		)
	}
	if resp.StatusCode > 0 {
		attributes = append(attributes,
			semconvNew.HTTPResponseStatusCode(resp.StatusCode),
		)
	}

	return attributes
}

// Route returns the attribute for the route.
func (n newHTTPServer) Route(route string) attribute.KeyValue {
	return semconvNew.HTTPRoute(route)
}
