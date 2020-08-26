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

package mongo

import "go.opentelemetry.io/otel/label"

const (
	TargetHostKey   = label.Key("out.host")
	TargetPortKey   = label.Key("out.port")
	HTTPMethodKey   = label.Key("http.method")
	HTTPCodeKey     = label.Key("http.code")
	HTTPURLKey      = label.Key("http.url")
	SpanTypeKey     = label.Key("span.type")
	ServiceNameKey  = label.Key("service.name")
	ResourceNameKey = label.Key("resource.name")
	ErrorKey        = label.Key("error")
	ErrorMsgKey     = label.Key("error.msg")
)

// TargetHost sets the target host address.
func TargetHost(targetHost string) label.KeyValue {
	return TargetHostKey.String(targetHost)
}

// TargetPort sets the target host port.
func TargetPort(targetPort string) label.KeyValue {
	return TargetPortKey.String(targetPort)
}

// HTTPMethod specifies the HTTP method used in a span.
func HTTPMethod(httpMethod string) label.KeyValue {
	return HTTPMethodKey.String(httpMethod)
}

// HTTPCode sets the HTTP status code as a attribute.
func HTTPCode(httpCode string) label.KeyValue {
	return HTTPCodeKey.String(httpCode)
}

// HTTPURL sets the HTTP URL for a span.
func HTTPURL(httpURL string) label.KeyValue {
	return HTTPURLKey.String(httpURL)
}

// SpanType defines the Span type (web, db, cache).
func SpanType(spanType string) label.KeyValue {
	return SpanTypeKey.String(spanType)
}

// ServiceName defines the Service name for this Span.
func ServiceName(serviceName string) label.KeyValue {
	return ServiceNameKey.String(serviceName)
}

// ResourceName defines the Resource name for the Span.
func ResourceName(resourceName string) label.KeyValue {
	return ResourceNameKey.String(resourceName)
}

// Error specifies whether an error occurred.
func Error(err bool) label.KeyValue {
	return ErrorKey.Bool(err)
}

// ErrorMsg specifies the error message.
func ErrorMsg(errorMsg string) label.KeyValue {
	return ErrorMsgKey.String(errorMsg)
}
