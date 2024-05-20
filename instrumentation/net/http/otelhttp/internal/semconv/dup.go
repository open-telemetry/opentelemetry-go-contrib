// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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
	count := 6 // HostName/ServerAdddress, Scheme, Method
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

	hostPort := requiredHTTPPort(req.TLS != nil, p)
	if hostPort > 0 {
		count += 2
	}

	methodOld, methodNew, methodOriginal := d.method(req.Method)
	if methodOriginal != (attribute.KeyValue{}) {
		count++
	}

	schemeOld, schemeNew := d.scheme(req.TLS != nil)

	peer, peerPort := splitHostPort(req.RemoteAddr)
	if peer != "" {
		// The Go HTTP server sets RemoteAddr to "IP:port", this will not be a
		// file-path that would be interpreted with a sock family.
		count += 2
		if peerPort > 0 {
			count += 2
		}
	}
	useragent := req.UserAgent()
	if useragent != "" {
		// This is the same between v1.20, and v1.24
		count++
	}

	clientIP := serverClientIP(req.Header.Get("X-Forwarded-For"))
	if clientIP != "" {
		count += 2
	}

	if req.URL != nil && req.URL.Path != "" {
		count += 2
	}

	protoName, protoVersion := netProtocol(req.Proto)
	if protoName != "" && protoName != "http" {
		count += 2
	}
	if protoVersion != "" {
		count += 2
	}

	attrs := make([]attribute.KeyValue, 0, count)
	attrs = append(attrs,
		semconvOld.NetHostName(host),
		semconvNew.ServerAddress(host),
		methodOld,
		methodNew,
		schemeOld,
		schemeNew,
	)

	if hostPort > 0 {
		attrs = append(attrs,
			semconvOld.NetHostPort(hostPort),
			semconvNew.ServerPort(hostPort),
		)
	}
	if methodOriginal != (attribute.KeyValue{}) {
		attrs = append(attrs, methodOriginal)
	}

	if peer != "" {
		attrs = append(attrs,
			semconvOld.NetSockPeerAddr(peer),
			semconvNew.NetworkPeerAddress(peer),
		)
		if peerPort > 0 {
			attrs = append(attrs,
				semconvOld.NetSockPeerPort(peerPort),
				semconvNew.NetworkPeerPort(peerPort),
			)
		}
	}

	if useragent != "" {
		// This is the same between v1.20, and v1.24
		attrs = append(attrs, semconvNew.UserAgentOriginal(useragent))
	}

	if clientIP != "" {
		attrs = append(attrs,
			semconvOld.HTTPClientIP(clientIP),
			semconvNew.ClientAddress(clientIP),
		)
	}

	if req.URL != nil && req.URL.Path != "" {
		attrs = append(attrs,
			semconvOld.HTTPTarget(req.URL.Path),
			semconvNew.URLPath(req.URL.Path),
		)
	}

	if protoName != "" && protoName != "http" {
		attrs = append(attrs,
			semconvOld.NetProtocolName(protoName),
			semconvNew.NetworkProtocolName(protoName),
		)
	}
	if protoVersion != "" {
		attrs = append(attrs,
			semconvOld.NetProtocolVersion(protoVersion),
			semconvNew.NetworkProtocolVersion(protoVersion),
		)
	}

	return attrs
}

func (d dupHTTPServer) method(method string) (attribute.KeyValue, attribute.KeyValue, attribute.KeyValue) {
	if method == "" {
		return semconvOld.HTTPMethod(http.MethodGet), semconvNew.HTTPRequestMethodGet, attribute.KeyValue{}
	}

	attr, found := methodLookup[method]
	if found {
		return semconvOld.HTTPMethod(method), attr, attribute.KeyValue{}
	}

	attr, found = methodLookup[strings.ToUpper(method)]
	if !found {
		attr = semconvNew.HTTPRequestMethodGet
	}

	return semconvOld.HTTPMethod(method), attr, semconvNew.HTTPRequestMethodOriginal(method)
}

func (d dupHTTPServer) scheme(https bool) (attribute.KeyValue, attribute.KeyValue) { // nolint:revive
	if https {
		return semconvOld.HTTPSchemeHTTPS, semconvNew.URLScheme("https")
	}
	return semconvOld.HTTPSchemeHTTP, semconvNew.URLScheme("http")
}

// ResponseTraceAttrs returns trace attributes for telemetry from an HTTP response.
//
// If any of the fields in the ResponseTelemetry are not set the attribute will be omitted.
func (d dupHTTPServer) ResponseTraceAttrs(resp ResponseTelemetry) []attribute.KeyValue {
	count := 0

	if resp.ReadBytes > 0 {
		count += 2
	}
	if resp.ReadError != nil && resp.ReadError != io.EOF {
		// This is not in the semantic conventions, but is historically provided
		count++
	}
	if resp.WriteBytes > 0 {
		count += 2
	}
	if resp.WriteError != nil && resp.WriteError != io.EOF {
		// This is not in the semantic conventions, but is historically provided
		count++
	}
	if resp.StatusCode > 0 {
		count += 2
	}

	attributes := make([]attribute.KeyValue, 0, count)

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
