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

package trace

import (
	"fmt"
	"net/http"

	"google.golang.org/grpc/codes"

	otelcore "go.opentelemetry.io/otel/api/core"
	otelkey "go.opentelemetry.io/otel/api/key"
)

// NetAttributesFromHTTPRequest generates attributes of the net
// namespace as specified by the OpenTelemetry specification for a
// span.  The network parameter is a string that net.Dial function
// from standard library can understand.
func NetAttributesFromHTTPRequest(network string, request *http.Request) []otelcore.KeyValue {
	transport := ""
	switch network {
	case "tcp", "tcp4", "tcp6":
		transport = "IP.TCP"
	case "udp", "udp4", "udp6":
		transport = "IP.UDP"
	case "ip", "ip4", "ip6":
		transport = "IP"
	case "unix", "unixgram", "unixpacket":
		transport = "Unix"
	default:
		transport = "other"
	}
	attrs := []otelcore.KeyValue{
		otelkey.String("net.transport", transport),
	}

	attrs = append(attrs, NetPeerAttrsFromString(request.RemoteAddr, HostParseStrict)...)
	for _, someHost := range []string{request.Host, request.Header.Get("Host"), request.URL.Host} {
		hostAttrs := NetHostAttrsFromString(someHost, HostParsePermissive)
		if len(hostAttrs) > 0 {
			attrs = append(attrs, hostAttrs...)
			break
		}
	}
	return attrs
}

// EndUserAttributesFromHTTPRequest generates attributes of the
// enduser namespace as specified by the OpenTelemetry specification
// for a span.
func EndUserAttributesFromHTTPRequest(request *http.Request) []otelcore.KeyValue {
	if username, _, ok := request.BasicAuth(); ok {
		return []otelcore.KeyValue{otelkey.String("enduser.id", username)}
	}
	return nil
}

// HTTPServerAttributesFromHTTPRequest generates attributes of the
// http namespace as specified by the OpenTelemetry specification for
// a span on the server side. Currently, only basic authentication is
// supported.
func HTTPServerAttributesFromHTTPRequest(serverName, route string, request *http.Request) []otelcore.KeyValue {
	attrs := []otelcore.KeyValue{
		otelkey.String("http.method", request.Method),
		otelkey.String("http.target", request.RequestURI),
	}
	if serverName != "" {
		attrs = append(attrs, otelkey.String("http.server_name", serverName))
	}
	scheme := ""
	if request.TLS != nil {
		scheme = "https"
	} else {
		scheme = "http"
	}
	attrs = append(attrs, otelkey.String("http.scheme", scheme))
	if route != "" {
		attrs = append(attrs, otelkey.String("http.route", route))
	}
	if request.Host != "" {
		attrs = append(attrs, otelkey.String("http.host", request.Host))
	}
	if ua := request.UserAgent(); ua != "" {
		attrs = append(attrs, otelkey.String("http.user_agent", ua))
	}
	if values, ok := request.Header["X-Forwarded-For"]; ok && len(values) > 0 {
		attrs = append(attrs, otelkey.String("http.client_ip", values[0]))
	}
	flavor := ""
	if request.ProtoMajor == 1 {
		flavor = fmt.Sprintf("1.%d", request.ProtoMinor)
	} else if request.ProtoMajor == 2 {
		flavor = "2"
	}
	if flavor != "" {
		attrs = append(attrs, otelkey.String("http.flavor", flavor))
	}
	return attrs
}

// HTTPAttributesFromHTTPStatusCode generates attributes of the http
// namespace as specified by the OpenTelemetry specification for a
// span.
func HTTPAttributesFromHTTPStatusCode(code int) []otelcore.KeyValue {
	attrs := []otelcore.KeyValue{
		otelkey.Int("http.status_code", code),
	}
	text := http.StatusText(code)
	if text != "" {
		attrs = append(attrs, otelkey.String("http.status_text", text))
	}
	return attrs
}

type codeRange struct {
	fromInclusive int
	toInclusive   int
}

func (r codeRange) contains(code int) bool {
	return r.fromInclusive <= code && code <= r.toInclusive
}

var validRangesPerCategory = map[int][]codeRange{
	1: {
		{http.StatusContinue, http.StatusEarlyHints},
	},
	2: {
		{http.StatusOK, http.StatusAlreadyReported},
		{http.StatusIMUsed, http.StatusIMUsed},
	},
	3: {
		{http.StatusMultipleChoices, http.StatusUseProxy},
		{http.StatusTemporaryRedirect, http.StatusPermanentRedirect},
	},
	4: {
		{http.StatusBadRequest, http.StatusTeapot}, // yes, teapot is so useful…
		{http.StatusMisdirectedRequest, http.StatusUpgradeRequired},
		{http.StatusPreconditionRequired, http.StatusTooManyRequests},
		{http.StatusRequestHeaderFieldsTooLarge, http.StatusRequestHeaderFieldsTooLarge},
		{http.StatusUnavailableForLegalReasons, http.StatusUnavailableForLegalReasons},
	},
	5: {
		{http.StatusInternalServerError, http.StatusLoopDetected},
		{http.StatusNotExtended, http.StatusNetworkAuthenticationRequired},
	},
}

// SpanStatusFromHTTPStatusCode generates a status code and a message
// as specified by the OpenTelemetry specification for a span.
func SpanStatusFromHTTPStatusCode(code int) (codes.Code, string) {
	spanCode := func() codes.Code {
		category := code / 100
		ranges, ok := validRangesPerCategory[category]
		if !ok {
			return codes.Unknown
		}
		ok = false
		for _, crange := range ranges {
			ok = crange.contains(code)
			if ok {
				break
			}
		}
		if !ok {
			return codes.Unknown
		}
		switch code {
		case http.StatusUnauthorized:
			return codes.Unauthenticated
		case http.StatusForbidden:
			return codes.PermissionDenied
		case http.StatusNotFound:
			return codes.NotFound
		case http.StatusTooManyRequests:
			return codes.ResourceExhausted
		case http.StatusNotImplemented:
			return codes.Unimplemented
		case http.StatusServiceUnavailable:
			return codes.Unavailable
		case http.StatusGatewayTimeout:
			return codes.DeadlineExceeded
		}
		if category > 0 && category < 4 {
			return codes.OK
		}
		if category == 4 {
			return codes.InvalidArgument
		}
		if category == 5 {
			return codes.Internal
		}
		// this really should not happen, if we get there then
		// it means that the code got out of sync with
		// validRangesPerCategory map
		return codes.Unknown
	}()
	if spanCode == codes.Unknown {
		return spanCode, fmt.Sprintf("Invalid HTTP status code %d", code)
	}
	return spanCode, fmt.Sprintf("HTTP status code: %d", code)
}
