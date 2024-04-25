// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package semconv // import "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp/internal/semconv"

import (
	"io"
	"net/http"
	"slices"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	semconvOld "go.opentelemetry.io/otel/semconv/v1.20.0"
	semconvNew "go.opentelemetry.io/otel/semconv/v1.24.0"
)

type dupHTTPServer struct{}

var _ HTTPServer = dupHTTPServer{}

// RequestTraceAttrs returns trace attributes for an HTTP request received by a
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
func (d dupHTTPServer) RequestTraceAttrs(server string, req *http.Request) []attribute.KeyValue {
	const maxAttributes = 24
	attrs := make([]attribute.KeyValue, 0, maxAttributes)
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

	attrs = append(attrs, semconvOld.NetHostName(host))
	attrs = append(attrs, semconvNew.ServerAddress(host))
	if hostPort := requiredHTTPPort(req.TLS != nil, p); hostPort > 0 {
		attrs = append(attrs, semconvOld.NetHostPort(hostPort))
		attrs = append(attrs, semconvNew.ServerPort(hostPort))
	}
	attrs = d.method(req.Method, attrs)
	attrs = d.scheme(req.TLS != nil, attrs)

	if peer, peerPort := splitHostPort(req.RemoteAddr); peer != "" {
		// The Go HTTP server sets RemoteAddr to "IP:port", this will not be a
		// file-path that would be interpreted with a sock family.
		attrs = append(attrs, semconvOld.NetSockPeerAddr(peer))
		attrs = append(attrs, semconvNew.NetworkPeerAddress(peer))
		if peerPort > 0 {
			attrs = append(attrs, semconvOld.NetSockPeerPort(peerPort))
			attrs = append(attrs, semconvNew.NetworkPeerPort(peerPort))
		}
	}

	if useragent := req.UserAgent(); useragent != "" {
		// This is the same between v1.20, and v1.24
		attrs = append(attrs, semconvNew.UserAgentOriginal(useragent))
	}

	if clientIP := serverClientIP(req.Header.Get("X-Forwarded-For")); clientIP != "" {
		attrs = append(attrs, semconvOld.HTTPClientIP(clientIP))
		attrs = append(attrs, semconvNew.ClientAddress(clientIP))
	}

	if req.URL != nil && req.URL.Path != "" {
		attrs = append(attrs, semconvOld.HTTPTarget(req.URL.Path))
		attrs = append(attrs, semconvNew.URLPath(req.URL.Path))
	}

	protoName, protoVersion := netProtocol(req.Proto)
	if protoName != "" && protoName != "http" {
		attrs = append(attrs, semconvOld.NetProtocolName(protoName))
		attrs = append(attrs, semconvNew.NetworkProtocolName(protoName))
	}
	if protoVersion != "" {
		attrs = append(attrs, semconvOld.NetProtocolVersion(protoVersion))
		attrs = append(attrs, semconvNew.NetworkProtocolVersion(protoVersion))
	}

	return slices.Clip(attrs)
}

func (d dupHTTPServer) method(method string, attrs []attribute.KeyValue) []attribute.KeyValue {
	if method == "" {
		attrs = append(attrs, semconvOld.HTTPMethod(http.MethodGet))
		attrs = append(attrs, semconvNew.HTTPRequestMethodGet)
		return attrs
	}
	attrs = append(attrs, semconvOld.HTTPMethod(method))
	if attr, ok := methodLookup[method]; ok {
		attrs = append(attrs, attr)
		return attrs
	}

	if attr, ok := methodLookup[strings.ToUpper(method)]; ok {
		attrs = append(attrs, attr)
	} else {
		// If the Original method is not a standard HTTP method fallback to GET
		attrs = append(attrs, semconvNew.HTTPRequestMethodGet)
	}
	attrs = append(attrs, semconvNew.HTTPRequestMethodOriginal(method))
	return attrs
}

func (d dupHTTPServer) scheme(https bool, attrs []attribute.KeyValue) []attribute.KeyValue { // nolint:revive
	if https {
		attrs = append(attrs, semconvOld.HTTPSchemeHTTPS)
		attrs = append(attrs, semconvNew.URLScheme("https"))
		return attrs
	}
	attrs = append(attrs, semconvOld.HTTPSchemeHTTP)
	attrs = append(attrs, semconvNew.URLScheme("http"))
	return attrs
}

// ResponseTraceAttrs returns trace attributes for telemetry from an HTTP response.
//
// If any of the fields in the ResponseTelemetry are not set the attribute will be omitted.
func (d dupHTTPServer) ResponseTraceAttrs(resp ResponseTelemetry) []attribute.KeyValue {
	attributes := []attribute.KeyValue{}

	if resp.ReadBytes > 0 {
		attributes = append(attributes,
			semconvOld.HTTPRequestContentLength(int(resp.ReadBytes)),
			semconvNew.HTTPRequestBodySize(int(resp.ReadBytes)),
		)
	}
	if resp.ReadError != nil && resp.ReadError != io.EOF {
		// This is not in the semantic conventions, but is historically provided
		attributes = append(attributes, attribute.String("http.read_error", resp.ReadError.Error()))
	}
	if resp.WriteBytes > 0 {
		attributes = append(attributes,
			semconvOld.HTTPResponseContentLength(int(resp.WriteBytes)),
			semconvNew.HTTPResponseBodySize(int(resp.WriteBytes)),
		)
	}
	if resp.WriteError != nil && resp.WriteError != io.EOF {
		// This is not in the semantic conventions, but is historically provided
		attributes = append(attributes, attribute.String("http.write_error", resp.WriteError.Error()))
	}
	if resp.StatusCode > 0 {
		attributes = append(attributes,
			semconvOld.HTTPStatusCode(resp.StatusCode),
			semconvNew.HTTPResponseStatusCode(resp.StatusCode),
		)
	}

	return attributes
}

// Route returns the attribute for the route.
func (d dupHTTPServer) Route(route string) attribute.KeyValue {
	return semconvNew.HTTPRoute(route)
}
