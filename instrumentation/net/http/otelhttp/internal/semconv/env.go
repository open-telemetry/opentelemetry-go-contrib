// Code created by gotmpl. DO NOT MODIFY.
// source: internal/shared/semconv/env.go.tmpl

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
	"fmt"
	"net/http"
	"os"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type ResponseTelemetry struct {
	StatusCode int
	ReadBytes  int
	ReadError  error
	WriteBytes int
	WriteError error
}

type HTTPServer interface {
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
	TraceRequest(server string, req *http.Request) []attribute.KeyValue

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
	MetricsRequest(server string, req *http.Request) []attribute.KeyValue

	// TraceRequest returns trace attributes for telemetry from an HTTP response.
	//
	// If any of the fields in the ResponseTelemetry are not set the attribute will be omitted.
	TraceResponse(ResponseTelemetry) []attribute.KeyValue

	// Route returns the attribute for the route.
	Route(string) attribute.KeyValue
}

func NewHTTPServer() HTTPServer {
	env := strings.ToLower(os.Getenv("OTEL_HTTP_CLIENT_COMPATIBILITY_MODE"))
	switch env {
	// TODO: Add support for new semconv
	// case "http":
	// 	return compatibilityHttp
	case "http/dup":
		return dupHTTPServer{}
	default:
		return oldHTTPServer{}
	}
}

// ServerStatus returns a span status code and message for an HTTP status code
// value returned by a server. Status codes in the 400-499 range are not
// returned as errors.
func ServerStatus(code int) (codes.Code, string) {
	if code < 100 || code >= 600 {
		return codes.Error, fmt.Sprintf("Invalid HTTP status code %d", code)
	}
	if code >= 500 {
		return codes.Error, ""
	}
	return codes.Unset, ""
}
